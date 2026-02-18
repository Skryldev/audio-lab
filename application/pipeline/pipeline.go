package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Skryldev/audio-lab/domain/model"
	"github.com/Skryldev/audio-lab/domain/ports"
	pkgerrors "github.com/Skryldev/audio-lab/pkg/errors"
	"github.com/Skryldev/audio-lab/pkg/logger"
	"github.com/Skryldev/audio-lab/pkg/progress"
	"github.com/Skryldev/audio-lab/infrastructure/ffmpeg"
	"go.uber.org/zap"
)

// ffprobeOutput maps key fields from ffprobe JSON
type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
		Size     string `json:"size"`
		FormatName string `json:"format_name"`
	} `json:"format"`
	Streams []struct {
		CodecName   string `json:"codec_name"`
		SampleRate  string `json:"sample_rate"`
		Channels    int    `json:"channels"`
		BitRate     string `json:"bit_rate"`
	} `json:"streams"`
}

// Stage represents a single pipeline stage function
type Stage func(ctx context.Context, job *Job) error

// Job holds the state of a single processing operation
type Job struct {
	ID         string
	InputPath  string
	OutputPath string
	TempPath   string // intermediate temp file path if needed
	Options    *model.ProcessingOptions
	Reporter   progress.Reporter
	Log        *logger.Logger
}

// Pipeline orchestrates audio processing stages
type Pipeline struct {
	executor ports.FFmpegExecutor
	storage  ports.StorageProvider
	stages   []namedStage
	log      *logger.Logger
}

type namedStage struct {
	name  string
	stage Stage
}

// NewPipeline creates a new audio processing pipeline
func NewPipeline(executor ports.FFmpegExecutor, storage ports.StorageProvider, log *logger.Logger) *Pipeline {
	p := &Pipeline{
		executor: executor,
		storage:  storage,
		log:      log,
	}
	return p
}

// Run executes the full pipeline for a job
func (p *Pipeline) Run(ctx context.Context, job *Job) (*model.ProcessingResult, error) {
	start := time.Now()

	// Validate input
	if err := p.validateInput(ctx, job); err != nil {
		return nil, err
	}

	// Probe input metadata
	inputMeta, err := p.probeFile(ctx, job.InputPath)
	if err != nil {
		return nil, pkgerrors.NewProcessingError("probe", "failed to probe input file", err)
	}

	job.report(progress.StageProbe, 5, "input probed")

	// Build and execute FFmpeg command
	if err := p.runFFmpeg(ctx, job); err != nil {
		return nil, err
	}

	job.report(progress.StageEncode, 90, "encoding complete")

	// Probe output
	outputMeta, err := p.probeFile(ctx, job.OutputPath)
	if err != nil {
		// non-fatal: output probe failure shouldn't fail the whole operation
		p.log.Warn("failed to probe output file", zap.Error(err))
		outputMeta = &model.AudioMetadata{}
	}

	job.report(progress.StageDone, 100, "done")

	return &model.ProcessingResult{
		InputPath:   job.InputPath,
		OutputPath:  job.OutputPath,
		InputMeta:   inputMeta,
		OutputMeta:  outputMeta,
		Duration:    time.Since(start),
		ProcessedAt: time.Now(),
	}, nil
}

func (p *Pipeline) validateInput(ctx context.Context, job *Job) error {
	if job.InputPath == "" {
		return pkgerrors.NewValidationError("inputPath", "", "input path must not be empty")
	}
	if job.OutputPath == "" {
		return pkgerrors.NewValidationError("outputPath", "", "output path must not be empty")
	}

	exists, err := p.storage.Exists(ctx, job.InputPath)
	if err != nil {
		return pkgerrors.NewProcessingError("validate", "failed to check input file", err)
	}
	if !exists {
		return pkgerrors.NewValidationError("inputPath", job.InputPath, "input file does not exist")
	}

	opts := job.Options
	if opts.Bitrate <= 0 {
		return pkgerrors.NewValidationError("bitrate", opts.Bitrate, "bitrate must be positive")
	}
	if opts.SampleRate <= 0 {
		return pkgerrors.NewValidationError("sampleRate", opts.SampleRate, "sample rate must be positive")
	}

	return nil
}

func (p *Pipeline) runFFmpeg(ctx context.Context, job *Job) error {
	opts := job.Options
	args := []string{"-y", "-i", job.InputPath}

	// Build audio filter chain
	fb := ffmpeg.NewFilterChainBuilder()

	if opts.HighpassEnabled {
		fb.AddHighpass(opts.HighpassFreq)
	}
	if opts.LowpassEnabled {
		fb.AddLowpass(opts.LowpassFreq)
	}
	if opts.NormalizationEnabled {
		fb.AddLoudnorm(opts.LoudnessTarget, opts.TruePeakLimit, opts.LoudnessRange)
	}

	filterStr := fb.Build()
	if filterStr != "" {
		args = append(args, "-af", filterStr)
	}

	// Sample rate
	args = append(args, "-ar", fmt.Sprintf("%d", opts.SampleRate))

	// Codec-specific encoding arguments
	codecArgs, err := buildCodecArgs(opts)
	if err != nil {
		return pkgerrors.NewProcessingError("encode", "failed to build codec args", err)
	}
	args = append(args, codecArgs...)

	// Output path
	args = append(args, job.OutputPath)

	job.report(progress.StageEncode, 20, "encoding started")

	return p.executor.Execute(ctx, args)
}

func buildCodecArgs(opts *model.ProcessingOptions) ([]string, error) {
	bitrate := fmt.Sprintf("%dk", opts.Bitrate/1000)

	switch opts.Codec {
	case model.CodecOpus:
		args := []string{"-c:a", "libopus"}
		if opts.BitrateMode == model.BitrateModeVBR {
			args = append(args, "-vbr", "on", "-b:a", bitrate)
		} else {
			args = append(args, "-vbr", "off", "-b:a", bitrate)
		}
		return args, nil

	case model.CodecAAC:
		args := []string{"-c:a", "aac"}
		if opts.BitrateMode == model.BitrateModeVBR {
			// AAC VBR uses quality scale 1-5
			args = append(args, "-q:a", "2")
		} else {
			args = append(args, "-b:a", bitrate)
		}
		return args, nil

	case model.CodecMP3:
		args := []string{"-c:a", "libmp3lame"}
		if opts.BitrateMode == model.BitrateModeVBR {
			args = append(args, "-q:a", "2")
		} else {
			args = append(args, "-b:a", bitrate)
		}
		return args, nil

	default:
		return nil, fmt.Errorf("unsupported codec: %s", opts.Codec)
	}
}

func (p *Pipeline) probeFile(ctx context.Context, path string) (*model.AudioMetadata, error) {
	data, err := p.executor.Probe(ctx, path)
	if err != nil {
		return nil, err
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	meta := &model.AudioMetadata{
		Format: probe.Format.FormatName,
	}

	// Parse duration
	var durationSec float64
	if _, err := fmt.Sscanf(probe.Format.Duration, "%f", &durationSec); err == nil {
		meta.Duration = time.Duration(durationSec * float64(time.Second))
	}

	// Parse size
	fmt.Sscanf(probe.Format.Size, "%d", &meta.Size)

	// Parse stream info
	for _, s := range probe.Streams {
		meta.Codec = s.CodecName
		meta.Channels = s.Channels
		fmt.Sscanf(s.SampleRate, "%d", &meta.SampleRate)
		fmt.Sscanf(s.BitRate, "%d", &meta.Bitrate)
		break // take first audio stream
	}

	return meta, nil
}

// ProbeFile probes audio metadata for a path.
func (p *Pipeline) ProbeFile(ctx context.Context, path string) (*model.AudioMetadata, error) {
	return p.probeFile(ctx, path)
}

// report is a helper to emit progress updates
func (j *Job) report(stage progress.Stage, percent float64, msg string) {
	if j.Reporter == nil {
		return
	}
	j.Reporter.Report(progress.Update{
		JobID:   j.ID,
		Stage:   stage,
		Percent: percent,
		Message: msg,
	})
}
