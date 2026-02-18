package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Skryldev/audio-lab/application/pipeline"
	"github.com/Skryldev/audio-lab/domain/model"
	"github.com/Skryldev/audio-lab/domain/ports"
	pkgerrors "github.com/Skryldev/audio-lab/pkg/errors"
	"github.com/Skryldev/audio-lab/pkg/logger"
	"github.com/Skryldev/audio-lab/pkg/progress"
	"github.com/Skryldev/audio-lab/pkg/retry"
	"go.uber.org/zap"
)

// AudioService is the main application service implementing ports.AudioProcessor
type AudioService struct {
	pipeline   *pipeline.Pipeline
	workerPool *pipeline.WorkerPool
	storage    ports.StorageProvider
	reporter   progress.Reporter
	log        *logger.Logger
	retryCfg   retry.Config
}

// Config holds AudioService configuration
type Config struct {
	Executor    ports.FFmpegExecutor
	Storage     ports.StorageProvider
	Reporter    progress.Reporter
	Logger      *logger.Logger
	Workers     int
	RetryConfig retry.Config
}

// NewAudioService creates a new AudioService
func NewAudioService(cfg Config) (*AudioService, error) {
	if cfg.Executor == nil {
		return nil, fmt.Errorf("FFmpegExecutor is required")
	}
	if cfg.Storage == nil {
		return nil, fmt.Errorf("StorageProvider is required")
	}

	log := cfg.Logger
	if log == nil {
		var err error
		log, err = logger.New(false)
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	}

	reporter := cfg.Reporter
	if reporter == nil {
		reporter = progress.NoopReporter{}
	}

	retryCfg := cfg.RetryConfig
	if retryCfg.MaxAttempts == 0 {
		retryCfg = retry.DefaultConfig()
	}

	workers := cfg.Workers
	if workers <= 0 {
		workers = 4
	}

	p := pipeline.NewPipeline(cfg.Executor, cfg.Storage, log)
	wp := pipeline.NewWorkerPool(p, workers, log)

	return &AudioService{
		pipeline:   p,
		workerPool: wp,
		storage:    cfg.Storage,
		reporter:   reporter,
		log:        log,
		retryCfg:   retryCfg,
	}, nil
}

// ProcessAudio processes a single audio file with optional configuration
func (s *AudioService) ProcessAudio(ctx context.Context, inputPath, outputPath string, opts ...ports.Option) (*model.ProcessingResult, error) {
	// Apply options on top of defaults
	options := model.DefaultProcessingOptions()
	for _, o := range opts {
		o(options)
	}

	// Apply timeout
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	s.log.Info("starting audio processing",
		zap.String("input", inputPath),
		zap.String("output", outputPath),
		zap.String("codec", string(options.Codec)),
		zap.Int("bitrate", options.Bitrate),
	)

	job := &pipeline.Job{
		ID:         generateJobID(inputPath),
		InputPath:  inputPath,
		OutputPath: outputPath,
		Options:    options,
		Reporter:   s.reporter,
		Log:        s.log,
	}

	var result *model.ProcessingResult

	err := retry.Do(ctx, retry.Config{
		MaxAttempts: options.MaxRetries,
		Delay:       options.RetryDelay,
		Multiplier:  2.0,
		MaxDelay:    30 * time.Second,
	}, func() error {
		var runErr error
		result, runErr = s.pipeline.Run(ctx, job)
		if runErr != nil {
			// Don't retry validation errors
			var valErr *pkgerrors.ValidationError
			if isValidationError(runErr, &valErr) {
				return nil // non-retryable: clear error to stop retries
			}
		}
		return runErr
	})

	if err != nil {
		s.log.Error("audio processing failed",
			zap.String("input", inputPath),
			zap.Error(err),
		)
		return nil, err
	}

	s.log.Info("audio processing completed",
		zap.String("output", outputPath),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// ProcessBatch processes multiple jobs concurrently
func (s *AudioService) ProcessBatch(ctx context.Context, jobs []model.BatchJob) (<-chan model.BatchResult, error) {
	if len(jobs) == 0 {
		ch := make(chan model.BatchResult)
		close(ch)
		return ch, nil
	}

	s.log.Info("starting batch processing",
		zap.Int("job_count", len(jobs)),
	)

	return s.workerPool.Run(ctx, jobs, s.reporter)
}

// ProbeAudio returns metadata about an audio file without processing it
func (s *AudioService) ProbeAudio(ctx context.Context, inputPath string) (*model.AudioMetadata, error) {
	exists, err := s.storage.Exists(ctx, inputPath)
	if err != nil {
		return nil, pkgerrors.NewProcessingError("probe", "failed to check file", err)
	}
	if !exists {
		return nil, pkgerrors.NewValidationError("inputPath", inputPath, "file does not exist")
	}

	// Probe via pipeline public API.
	return s.pipeline.ProbeFile(ctx, inputPath)
}

func isValidationError(err error, target **pkgerrors.ValidationError) bool {
	return errors.As(err, target)
}

func generateJobID(input string) string {
	return fmt.Sprintf("job-%d-%s", time.Now().UnixNano(), sanitize(input))
}

func sanitize(s string) string {
	if len(s) > 20 {
		s = s[len(s)-20:]
	}
	result := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}
