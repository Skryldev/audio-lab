package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	musicproc "github.com/Skryldev/audio-lab"
)

func main() {
	// ── Graceful shutdown via signal ──────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── Progress channel ──────────────────────────────────────────────────
	progressCh := make(chan musicproc.ProgressUpdate, 32)
	go func() {
		for upd := range progressCh {
			fmt.Printf("[%s] stage=%-10s %.0f%%  %s\n",
				upd.JobID[:8], upd.Stage, upd.Percent, upd.Message)
		}
	}()

	// ── Create processor ──────────────────────────────────────────────────
	processor, err := musicproc.New(musicproc.Config{
		Workers:    4,
		ProgressCh: progressCh,
	})
	if err != nil {
		log.Fatalf("failed to create processor: %v", err)
	}
	defer func() {
		close(progressCh)
		processor.Close()
	}()

	// ── Example 1: Single file processing ────────────────────────────────
	fmt.Println("\n── Example 1: Single File Processing ──")
	singleExample(ctx, processor)

	// ── Example 2: Batch processing ──────────────────────────────────────
	fmt.Println("\n── Example 2: Batch Processing ──")
	batchExample(ctx, processor)

	// ── Example 3: Probe audio ───────────────────────────────────────────
	fmt.Println("\n── Example 3: Probe Audio ──")
	probeExample(ctx, processor)
}

func singleExample(ctx context.Context, p *musicproc.Processor) {
	inputPath := os.Getenv("MUSICPROC_INPUT")
	if inputPath == "" {
		inputPath = "/tmp/sample.wav"
	}
	outputPath := "/tmp/output_opus.opus"

	result, err := p.ProcessAudio(ctx, inputPath, outputPath,
		musicproc.WithCodec(musicproc.CodecOpus),
		musicproc.WithBitrate(128_000),
		musicproc.WithBitrateMode(musicproc.BitrateModeCBR),
		musicproc.WithSampleRate(48_000),
		musicproc.WithNormalization(true),
		musicproc.WithLoudnessTarget(-16.0), // Streaming platforms target
		musicproc.WithHighpass(80),
	)
	if err != nil {
		fmt.Printf("processing failed: %v\n", err)
		return
	}

	fmt.Printf("Done! took=%s output=%s\n", result.Duration, result.OutputPath)
	if result.OutputMeta != nil {
		fmt.Printf("Output: codec=%s sampleRate=%d bitrate=%d\n",
			result.OutputMeta.Codec,
			result.OutputMeta.SampleRate,
			result.OutputMeta.Bitrate,
		)
	}
}

func batchExample(ctx context.Context, p *musicproc.Processor) {
	jobs := []musicproc.BatchJob{
		{
			ID:         "job-001",
			InputPath:  "/tmp/track1.wav",
			OutputPath: "/tmp/track1.aac",
			Options:    nil, // will use defaults
		},
		{
			ID:         "job-002",
			InputPath:  "/tmp/track2.wav",
			OutputPath: "/tmp/track2.mp3",
		},
	}

	resultsCh, err := p.ProcessBatch(ctx, jobs)
	if err != nil {
		fmt.Printf("batch failed to start: %v\n", err)
		return
	}

	successCount := 0
	for res := range resultsCh {
		if res.Err != nil {
			fmt.Printf("[%s] FAILED: %v\n", res.JobID, res.Err)
			continue
		}
		successCount++
		fmt.Printf("[%s] OK took=%s\n", res.JobID, res.Result.Duration)
	}

	fmt.Printf("Batch complete: %d/%d succeeded\n", successCount, len(jobs))
}

func probeExample(ctx context.Context, p *musicproc.Processor) {
	inputPath := os.Getenv("MUSICPROC_INPUT")
	if inputPath == "" {
		inputPath = "/tmp/sample.wav"
	}

	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	meta, err := p.ProbeAudio(probeCtx, inputPath)
	if err != nil {
		fmt.Printf("probe failed: %v\n", err)
		return
	}

	fmt.Printf("Probe result:\n")
	fmt.Printf("  Duration  : %s\n", meta.Duration)
	fmt.Printf("  Codec     : %s\n", meta.Codec)
	fmt.Printf("  SampleRate: %d Hz\n", meta.SampleRate)
	fmt.Printf("  Channels  : %d\n", meta.Channels)
	fmt.Printf("  Bitrate   : %d bps\n", meta.Bitrate)
	fmt.Printf("  Format    : %s\n", meta.Format)
	fmt.Printf("  Size      : %d bytes\n", meta.Size)
}