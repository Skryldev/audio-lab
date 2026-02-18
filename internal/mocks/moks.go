package mocks

import (
	"context"
	"encoding/json"
)

// MockFFmpegExecutor is a test double for ports.FFmpegExecutor
type MockFFmpegExecutor struct {
	ExecuteFunc func(ctx context.Context, args []string) error
	ProbeFunc   func(ctx context.Context, inputPath string) ([]byte, error)
	ExecutedArgs [][]string
}

func (m *MockFFmpegExecutor) Execute(ctx context.Context, args []string) error {
	m.ExecutedArgs = append(m.ExecutedArgs, args)
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, args)
	}
	return nil
}

func (m *MockFFmpegExecutor) Probe(ctx context.Context, inputPath string) ([]byte, error) {
	if m.ProbeFunc != nil {
		return m.ProbeFunc(ctx, inputPath)
	}
	return defaultProbeResponse(), nil
}

func defaultProbeResponse() []byte {
	resp := map[string]interface{}{
		"format": map[string]interface{}{
			"duration":    "120.5",
			"bit_rate":    "192000",
			"size":        "2880000",
			"format_name": "wav",
		},
		"streams": []map[string]interface{}{
			{
				"codec_name":  "pcm_s16le",
				"sample_rate": "44100",
				"channels":    2,
				"bit_rate":    "1411200",
			},
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

// MockStorageProvider is a test double for ports.StorageProvider
type MockStorageProvider struct {
	ExistsFunc   func(ctx context.Context, path string) (bool, error)
	SizeFunc     func(ctx context.Context, path string) (int64, error)
	RemoveFunc   func(ctx context.Context, path string) error
	TempFileFunc func(ctx context.Context, dir, pattern string) (string, error)
}

func (m *MockStorageProvider) Exists(ctx context.Context, path string) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, path)
	}
	return true, nil
}

func (m *MockStorageProvider) Size(ctx context.Context, path string) (int64, error) {
	if m.SizeFunc != nil {
		return m.SizeFunc(ctx, path)
	}
	return 1024, nil
}

func (m *MockStorageProvider) Remove(ctx context.Context, path string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, path)
	}
	return nil
}

func (m *MockStorageProvider) TempFile(ctx context.Context, dir, pattern string) (string, error) {
	if m.TempFileFunc != nil {
		return m.TempFileFunc(ctx, dir, pattern)
	}
	return "/tmp/mock_temp_file", nil
}