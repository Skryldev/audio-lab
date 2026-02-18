package audiolab

import (
	"context"

	"github.com/Skryldev/audio-lab/application/usecase"
	"github.com/Skryldev/audio-lab/domain/model"
	"github.com/Skryldev/audio-lab/domain/ports"
	"github.com/Skryldev/audio-lab/infrastructure/ffmpeg"
	"github.com/Skryldev/audio-lab/infrastructure/storage"
	"github.com/Skryldev/audio-lab/pkg/logger"
	"github.com/Skryldev/audio-lab/pkg/progress"
	"github.com/Skryldev/audio-lab/pkg/retry"
	"go.uber.org/zap"
)

// Re-export types for convenient use by callers
type (
	Codec          = model.Codec
	BitrateMode    = model.BitrateMode
	ProcessingResult = model.ProcessingResult
	AudioMetadata  = model.AudioMetadata
	BatchJob       = model.BatchJob
	BatchResult    = model.BatchResult
	ProgressUpdate = progress.Update
	ProgressStage  = progress.Stage
)

// Re-export codec constants
const (
	CodecOpus = model.CodecOpus
	CodecAAC  = model.CodecAAC
	CodecMP3  = model.CodecMP3

	BitrateModeVBR = model.BitrateModeVBR
	BitrateModeCBR = model.BitrateCBR

	StageProbe     = progress.StageProbe
	StageNormalize = progress.StageNormalize
	StageEncode    = progress.StageEncode
	StageDone      = progress.StageDone
)

// Re-export option functions
var (
	WithCodec          = ports.WithCodec
	WithBitrate        = ports.WithBitrate
	WithBitrateMode    = ports.WithBitrateMode
	WithSampleRate     = ports.WithSampleRate
	WithNormalization  = ports.WithNormalization
	WithLoudnessTarget = ports.WithLoudnessTarget
	WithHighpass       = ports.WithHighpass
	WithLowpass        = ports.WithLowpass
	WithWorkers        = ports.WithWorkers
)

// Config holds top-level configuration for the processor
type Config struct {
	// FFmpegPath is the path to ffmpeg binary (auto-detected if empty)
	FFmpegPath string

	// FFprobePath is the path to ffprobe binary (auto-detected if empty)
	FFprobePath string

	// Logger is an optional custom logger. Uses production zap if nil.
	Logger *logger.Logger

	// ZapLogger allows passing a *zap.Logger directly
	ZapLogger *zap.Logger

	// ProgressCh is an optional channel for receiving progress updates
	ProgressCh chan<- ProgressUpdate

	// Workers sets the number of parallel batch workers (default: 4)
	Workers int

	// RetryConfig overrides default retry behavior
	RetryConfig *retry.Config
}

// Processor is the main entry point
type Processor struct {
	service *usecase.AudioService
	log     *logger.Logger
}

// New creates a new Processor with the given configuration
func New(cfg Config) (*Processor, error) {
	log := cfg.Logger
	if log == nil && cfg.ZapLogger != nil {
		log = logger.FromZap(cfg.ZapLogger)
	}
	if log == nil {
		var err error
		log, err = logger.New(false)
		if err != nil {
			return nil, err
		}
	}

	exec, err := ffmpeg.NewExecutor(ffmpeg.ExecutorConfig{
		FFmpegPath:  cfg.FFmpegPath,
		FFprobePath: cfg.FFprobePath,
		Logger:      log,
	})
	if err != nil {
		return nil, err
	}

	store := storage.NewLocalStorage()

	var reporter progress.Reporter = progress.NoopReporter{}
	if cfg.ProgressCh != nil {
		reporter = progress.NewChannelReporter(cfg.ProgressCh)
	}

	retryCfg := retry.DefaultConfig()
	if cfg.RetryConfig != nil {
		retryCfg = *cfg.RetryConfig
	}

	workers := cfg.Workers
	if workers <= 0 {
		workers = 4
	}

	svc, err := usecase.NewAudioService(usecase.Config{
		Executor:    exec,
		Storage:     store,
		Reporter:    reporter,
		Logger:      log,
		Workers:     workers,
		RetryConfig: retryCfg,
	})
	if err != nil {
		return nil, err
	}

	return &Processor{
		service: svc,
		log:     log,
	}, nil
}

// ProcessAudio processes a single audio file
func (p *Processor) ProcessAudio(ctx context.Context, inputPath, outputPath string, opts ...ports.Option) (*ProcessingResult, error) {
	return p.service.ProcessAudio(ctx, inputPath, outputPath, opts...)
}

// ProcessBatch processes multiple jobs concurrently
func (p *Processor) ProcessBatch(ctx context.Context, jobs []BatchJob) (<-chan BatchResult, error) {
	return p.service.ProcessBatch(ctx, jobs)
}

// ProbeAudio returns metadata about an audio file without processing
func (p *Processor) ProbeAudio(ctx context.Context, inputPath string) (*AudioMetadata, error) {
	return p.service.ProbeAudio(ctx, inputPath)
}

// Close flushes the logger and releases resources
func (p *Processor) Close() {
	_ = p.log.Sync()
}