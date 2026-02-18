package errors

import (
	"errors"
	"fmt"
)

// ErrorCode categorizes errors
type ErrorCode string

const (
	ErrCodeProcessing  ErrorCode = "PROCESSING_ERROR"
	ErrCodeFFmpeg      ErrorCode = "FFMPEG_ERROR"
	ErrCodeValidation  ErrorCode = "VALIDATION_ERROR"
	ErrCodeIO          ErrorCode = "IO_ERROR"
	ErrCodeTimeout     ErrorCode = "TIMEOUT_ERROR"
	ErrCodeCanceled    ErrorCode = "CANCELED_ERROR"
)

// MusicProcError is the base structured error
type MusicProcError struct {
	Code    ErrorCode
	Message string
	Cause   error
	Fields  map[string]interface{}
}

func (e *MusicProcError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *MusicProcError) Unwrap() error {
	return e.Cause
}

// ProcessingError represents a general audio processing failure
type ProcessingError struct {
	MusicProcError
	Stage string
}

func NewProcessingError(stage, message string, cause error) *ProcessingError {
	return &ProcessingError{
		MusicProcError: MusicProcError{
			Code:    ErrCodeProcessing,
			Message: message,
			Cause:   cause,
		},
		Stage: stage,
	}
}

func (e *ProcessingError) Error() string {
	base := e.MusicProcError.Error()
	return fmt.Sprintf("%s (stage=%s)", base, e.Stage)
}

// FFmpegError represents an FFmpeg execution failure
type FFmpegError struct {
	MusicProcError
	Args     []string
	ExitCode int
	Stderr   string
}

func NewFFmpegError(message string, args []string, exitCode int, stderr string, cause error) *FFmpegError {
	return &FFmpegError{
		MusicProcError: MusicProcError{
			Code:    ErrCodeFFmpeg,
			Message: message,
			Cause:   cause,
		},
		Args:     args,
		ExitCode: exitCode,
		Stderr:   stderr,
	}
}

func (e *FFmpegError) Error() string {
	return fmt.Sprintf("[%s] %s (exit=%d, stderr=%q): %v",
		e.Code, e.Message, e.ExitCode, truncate(e.Stderr, 200), e.Cause)
}

// ValidationError represents input validation failure
type ValidationError struct {
	MusicProcError
	Field string
	Value interface{}
}

func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		MusicProcError: MusicProcError{
			Code:    ErrCodeValidation,
			Message: message,
		},
		Field: field,
		Value: value,
	}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("[%s] field=%s value=%v: %s", e.Code, e.Field, e.Value, e.Message)
}

// Is enables errors.Is checks
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As enables errors.As checks
func As[T error](err error) (T, bool) {
	var target T
	ok := errors.As(err, &target)
	return target, ok
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}