package model

import "time"

// Codec represents supported audio codecs
type Codec string

const (
	CodecOpus Codec = "opus"
	CodecAAC  Codec = "aac"
	CodecMP3  Codec = "mp3"
)

// BitrateMode represents bitrate encoding mode
type BitrateMode string

const (
	BitrateModeVBR BitrateMode = "vbr"
	BitrateCBR     BitrateMode = "cbr"
)

// AudioMetadata holds metadata of an audio file
type AudioMetadata struct {
	Duration   time.Duration
	SampleRate int
	Channels   int
	Bitrate    int
	Codec      string
	Format     string
	Size       int64
}

// ProcessingOptions holds all configuration for audio processing
type ProcessingOptions struct {
	// Codec settings
	Codec       Codec
	Bitrate     int
	BitrateMode BitrateMode
	SampleRate  int

	// Normalization
	NormalizationEnabled bool
	LoudnessTarget       float64 // LUFS (EBU R128), default: -23
	TruePeakLimit        float64 // dBTP, default: -1.0
	LoudnessRange        float64 // LU, default: 7.0

	// Filters
	HighpassEnabled bool
	HighpassFreq    int // Hz, default: 80

	LowpassEnabled bool
	LowpassFreq    int // Hz, default: 18000

	// Processing
	Timeout time.Duration
	Workers int

	// Retry
	MaxRetries  int
	RetryDelay  time.Duration
}

// DefaultProcessingOptions returns sane defaults
func DefaultProcessingOptions() *ProcessingOptions {
	return &ProcessingOptions{
		Codec:                CodecOpus,
		Bitrate:              128000,
		BitrateMode:          BitrateCBR,
		SampleRate:           48000,
		NormalizationEnabled: true,
		LoudnessTarget:       -23.0,
		TruePeakLimit:        -1.0,
		LoudnessRange:        7.0,
		HighpassEnabled:      false,
		HighpassFreq:         80,
		LowpassEnabled:       false,
		LowpassFreq:          18000,
		Timeout:              5 * time.Minute,
		Workers:              4,
		MaxRetries:           3,
		RetryDelay:           time.Second,
	}
}

// ProcessingResult holds the result of an audio processing operation
type ProcessingResult struct {
	InputPath    string
	OutputPath   string
	InputMeta    *AudioMetadata
	OutputMeta   *AudioMetadata
	Duration     time.Duration
	ProcessedAt  time.Time
}

// BatchJob represents a batch processing job
type BatchJob struct {
	ID         string
	InputPath  string
	OutputPath string
	Options    *ProcessingOptions
}

// BatchResult holds results of a batch operation
type BatchResult struct {
	JobID   string
	Result  *ProcessingResult
	Err     error
}