package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	pkgerrors "github.com/Skryldev/audio-lab/pkg/errors"
	"github.com/Skryldev/audio-lab/pkg/logger"
	"go.uber.org/zap"
)

// Executor implements ports.FFmpegExecutor
type Executor struct {
	ffmpegPath  string
	ffprobePath string
	mu          sync.Mutex // guards concurrent ffmpeg invocations if needed
	log         *logger.Logger
}

// ExecutorConfig holds configuration for the FFmpeg executor
type ExecutorConfig struct {
	FFmpegPath  string
	FFprobePath string
	Logger      *logger.Logger
}

// NewExecutor creates a new FFmpeg executor
func NewExecutor(cfg ExecutorConfig) (*Executor, error) {
	ffmpegPath := cfg.FFmpegPath
	if ffmpegPath == "" {
		var err error
		ffmpegPath, err = exec.LookPath("ffmpeg")
		if err != nil {
			return nil, fmt.Errorf("ffmpeg not found in PATH: %w", err)
		}
	}

	ffprobePath := cfg.FFprobePath
	if ffprobePath == "" {
		var err error
		ffprobePath, err = exec.LookPath("ffprobe")
		if err != nil {
			return nil, fmt.Errorf("ffprobe not found in PATH: %w", err)
		}
	}

	log := cfg.Logger
	if log == nil {
		log, _ = logger.New(false)
	}

	return &Executor{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		log:         log,
	}, nil
}

// Execute runs ffmpeg with the given arguments
func (e *Executor) Execute(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, e.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	e.log.Debug("executing ffmpeg",
		zap.Strings("args", args),
	)

	if err := cmd.Run(); err != nil {
		exitCode := -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return pkgerrors.NewFFmpegError(
			"ffmpeg execution failed",
			args,
			exitCode,
			stderr.String(),
			err,
		)
	}

	return nil
}

// Probe runs ffprobe and returns JSON output
func (e *Executor) Probe(ctx context.Context, inputPath string) ([]byte, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, e.ffprobePath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		exitCode := -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return nil, pkgerrors.NewFFmpegError(
			"ffprobe execution failed",
			args,
			exitCode,
			stderr.String(),
			err,
		)
	}

	return stdout.Bytes(), nil
}

// BuildFilterChain constructs an ffmpeg audio filter string
type FilterChainBuilder struct {
	filters []string
}

func NewFilterChainBuilder() *FilterChainBuilder {
	return &FilterChainBuilder{}
}

func (b *FilterChainBuilder) AddHighpass(freq int) *FilterChainBuilder {
	b.filters = append(b.filters, fmt.Sprintf("highpass=f=%d", freq))
	return b
}

func (b *FilterChainBuilder) AddLowpass(freq int) *FilterChainBuilder {
	b.filters = append(b.filters, fmt.Sprintf("lowpass=f=%d", freq))
	return b
}

func (b *FilterChainBuilder) AddLoudnorm(targetLUFS, truePeak, LRA float64) *FilterChainBuilder {
	filter := fmt.Sprintf("loudnorm=I=%.1f:TP=%.1f:LRA=%.1f", targetLUFS, truePeak, LRA)
	b.filters = append(b.filters, filter)
	return b
}

func (b *FilterChainBuilder) AddResample(hz int) *FilterChainBuilder {
	b.filters = append(b.filters, fmt.Sprintf("aresample=%d", hz))
	return b
}

func (b *FilterChainBuilder) Build() string {
	return strings.Join(b.filters, ",")
}

func (b *FilterChainBuilder) IsEmpty() bool {
	return len(b.filters) == 0
}