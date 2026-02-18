package progress

import (
	"sync"
	"time"
)

// Stage represents a pipeline stage
type Stage string

const (
	StageProbe     Stage = "probe"
	StagePreprocess Stage = "preprocess"
	StageNormalize  Stage = "normalize"
	StageFilter     Stage = "filter"
	StageEncode     Stage = "encode"
	StageDone       Stage = "done"
)

// Update holds a progress update
type Update struct {
	JobID     string
	Stage     Stage
	Percent   float64
	Message   string
	Timestamp time.Time
}

// Reporter is the interface for progress reporting
type Reporter interface {
	Report(update Update)
}

// ChannelReporter sends updates to a channel
type ChannelReporter struct {
	ch chan<- Update
}

// NewChannelReporter creates a reporter that sends updates to ch
func NewChannelReporter(ch chan<- Update) *ChannelReporter {
	return &ChannelReporter{ch: ch}
}

func (r *ChannelReporter) Report(update Update) {
	select {
	case r.ch <- update:
	default: // non-blocking: drop if channel is full
	}
}

// MultiReporter fans out to multiple reporters
type MultiReporter struct {
	mu        sync.RWMutex
	reporters []Reporter
}

func NewMultiReporter(reporters ...Reporter) *MultiReporter {
	return &MultiReporter{reporters: reporters}
}

func (m *MultiReporter) Add(r Reporter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reporters = append(m.reporters, r)
}

func (m *MultiReporter) Report(update Update) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.reporters {
		r.Report(update)
	}
}

// NoopReporter discards all updates
type NoopReporter struct{}

func (n NoopReporter) Report(_ Update) {}