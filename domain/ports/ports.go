package ports

import (
	"context"

	"github.com/Skryldev/audio-lab/domain/model"
)

// AudioProcessor defines the main processing interface
type AudioProcessor interface {
	// ProcessAudio processes a single audio file
	ProcessAudio(ctx context.Context, inputPath, outputPath string, opts ...Option) (*model.ProcessingResult, error)

	// ProcessBatch processes multiple audio files concurrently
	ProcessBatch(ctx context.Context, jobs []model.BatchJob) (<-chan model.BatchResult, error)

	// ProbeAudio returns metadata about an audio file without processing
	ProbeAudio(ctx context.Context, inputPath string) (*model.AudioMetadata, error)
}

// FFmpegExecutor is the abstraction for FFmpeg command execution
type FFmpegExecutor interface {
	// Execute runs an ffmpeg command with the given arguments
	Execute(ctx context.Context, args []string) error

	// Probe runs ffprobe and returns JSON output
	Probe(ctx context.Context, inputPath string) ([]byte, error)
}

// StorageProvider abstracts filesystem or object storage operations
type StorageProvider interface {
	// Exists checks if a file exists
	Exists(ctx context.Context, path string) (bool, error)

	// Size returns file size in bytes
	Size(ctx context.Context, path string) (int64, error)

	// Remove deletes a file
	Remove(ctx context.Context, path string) error

	// TempFile creates a temporary file and returns its path
	TempFile(ctx context.Context, dir, pattern string) (string, error)
}

// ProgressReporter allows callers to receive progress updates
type ProgressReporter interface {
	// Report sends a progress update
	Report(jobID string, percent float64, stage string)
}

// Option is the functional option type
type Option func(*model.ProcessingOptions)

// WithCodec sets the output codec
func WithCodec(codec model.Codec) Option {
	return func(o *model.ProcessingOptions) {
		o.Codec = codec
	}
}

// WithBitrate sets the target bitrate in bps
func WithBitrate(bitrate int) Option {
	return func(o *model.ProcessingOptions) {
		o.Bitrate = bitrate
	}
}

// WithBitrateMode sets VBR or CBR mode
func WithBitrateMode(mode model.BitrateMode) Option {
	return func(o *model.ProcessingOptions) {
		o.BitrateMode = mode
	}
}

// WithSampleRate sets the output sample rate
func WithSampleRate(hz int) Option {
	return func(o *model.ProcessingOptions) {
		o.SampleRate = hz
	}
}

// WithNormalization enables or disables EBU R128 loudness normalization
func WithNormalization(enabled bool) Option {
	return func(o *model.ProcessingOptions) {
		o.NormalizationEnabled = enabled
	}
}

// WithLoudnessTarget sets the target loudness in LUFS (EBU R128)
func WithLoudnessTarget(lufs float64) Option {
	return func(o *model.ProcessingOptions) {
		o.LoudnessTarget = lufs
	}
}

// WithHighpass enables highpass filter at given frequency
func WithHighpass(hz int) Option {
	return func(o *model.ProcessingOptions) {
		o.HighpassEnabled = true
		o.HighpassFreq = hz
	}
}

// WithLowpass enables lowpass filter at given frequency
func WithLowpass(hz int) Option {
	return func(o *model.ProcessingOptions) {
		o.LowpassEnabled = true
		o.LowpassFreq = hz
	}
}

// WithWorkers sets the number of concurrent workers for batch processing
func WithWorkers(n int) Option {
	return func(o *model.ProcessingOptions) {
		if n > 0 {
			o.Workers = n
		}
	}
}

// WithRetry sets retry configuration
func WithRetry(maxRetries int, delay ...interface{}) Option {
	return func(o *model.ProcessingOptions) {
		o.MaxRetries = maxRetries
	}
}

// WithProgressReporter attaches a progress reporter (stored externally)
func WithProgressReporter(_ ProgressReporter) Option {
	return func(_ *model.ProcessingOptions) {}
}