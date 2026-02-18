package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/Skryldev/audio-lab/domain/model"
	"github.com/Skryldev/audio-lab/pkg/logger"
	"github.com/Skryldev/audio-lab/pkg/progress"
	"go.uber.org/zap"
)

// WorkerPool manages concurrent job execution
type WorkerPool struct {
	pipeline *Pipeline
	workers  int
	log      *logger.Logger
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(p *Pipeline, workers int, log *logger.Logger) *WorkerPool {
	if workers <= 0 {
		workers = 4
	}
	return &WorkerPool{
		pipeline: p,
		workers:  workers,
		log:      log,
	}
}

// Run processes batch jobs concurrently and sends results to returned channel
// The channel is closed when all jobs are complete or context is canceled
func (wp *WorkerPool) Run(ctx context.Context, jobs []model.BatchJob, reporter progress.Reporter) (<-chan model.BatchResult, error) {
	results := make(chan model.BatchResult, len(jobs))

	go func() {
		defer close(results)

		jobCh := make(chan model.BatchJob, len(jobs))
		for _, j := range jobs {
			jobCh <- j
		}
		close(jobCh)

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, wp.workers)

		for job := range jobCh {
			select {
			case <-ctx.Done():
				results <- model.BatchResult{
					JobID: job.ID,
					Err:   ctx.Err(),
				}
				continue
			case semaphore <- struct{}{}:
			}

			wg.Add(1)
			go func(j model.BatchJob) {
				defer wg.Done()
				defer func() { <-semaphore }()

				result, err := wp.processJob(ctx, j, reporter)
				results <- model.BatchResult{
					JobID:  j.ID,
					Result: result,
					Err:    err,
				}
			}(job)
		}

		wg.Wait()
	}()

	return results, nil
}

func (wp *WorkerPool) processJob(ctx context.Context, job model.BatchJob, reporter progress.Reporter) (*model.ProcessingResult, error) {
	opts := job.Options
	if opts == nil {
		opts = model.DefaultProcessingOptions()
	}

	pipelineJob := &Job{
		ID:         job.ID,
		InputPath:  job.InputPath,
		OutputPath: job.OutputPath,
		Options:    opts,
		Reporter:   reporter,
		Log:        wp.log.With(zap.String("job_id", job.ID)),
	}

	wp.log.Info("processing batch job",
		zap.String("job_id", job.ID),
		zap.String("input", job.InputPath),
	)

	result, err := wp.pipeline.Run(ctx, pipelineJob)
	if err != nil {
		wp.log.Error("batch job failed",
			zap.String("job_id", job.ID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("job %s failed: %w", job.ID, err)
	}

	return result, nil
}