package worker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type TranscodingJob struct {
	ID       string `json:"id"`
	VideoID  string `json:"video_id"`
	Filename string `json:"filename"`
}

type TranscodingResult struct {
	JobID    string        `json:"job_id"`
	VideoID  string        `json:"video_id"`
	Status   string        `json:"status"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

type VideoTranscodingService struct {
	workers        int
	batchTimeout   time.Duration
	processingTime time.Duration
}

func NewVideoTranscodingService(workers int, batchTimeout, processingTime time.Duration) *VideoTranscodingService {
	if workers < 1 {
		workers = 1
	}
	return &VideoTranscodingService{workers: workers, batchTimeout: batchTimeout, processingTime: processingTime}
}

func (s *VideoTranscodingService) TranscodeBatch(ctx context.Context, batch []TranscodingJob) ([]TranscodingResult, error) {
	batchCtx, cancel := context.WithTimeout(ctx, s.batchTimeout)
	defer cancel()

	jobs := make(chan TranscodingJob)
	results := make(chan TranscodingResult)
	var workers sync.WaitGroup

	workers.Add(s.workers)
	for workerID := 0; workerID < s.workers; workerID++ {
		go func() {
			defer workers.Done()
			s.transcodingWorker(batchCtx, jobs, results)
		}()
	}

	// Channel lifecycle:
	// 1. This feeder is the sole sender/owner of jobs, therefore only it closes jobs.
	// 2. Closing jobs makes every worker's `range jobs` finish after queued work.
	// 3. Cancellation also stops the feeder, so it cannot leak while trying to send.
	go func() {
		defer close(jobs)
		for _, job := range batch {
			select {
			case jobs <- job:
			case <-batchCtx.Done():
				return
			}
		}
	}()

	// Workers are the result senders, but no single worker knows when all peers have
	// exited. This coordinator waits for the whole WaitGroup, then closes results.
	// Therefore no worker can ever send after results is closed.
	go func() {
		workers.Wait()
		close(results)
	}()

	collected := make([]TranscodingResult, 0, len(batch))
	// The receiver never closes results. range terminates when the coordinator closes it.
	for result := range results {
		collected = append(collected, result)
	}
	if err := batchCtx.Err(); err != nil {
		return collected, fmt.Errorf("transcoding batch: %w", err)
	}
	return collected, nil
}

func (s *VideoTranscodingService) transcodingWorker(
	ctx context.Context,
	jobs <-chan TranscodingJob,
	results chan<- TranscodingResult,
) {
	// Go note: directional channel types document that workers receive jobs and send results.
	for job := range jobs {
		started := time.Now()
		timer := time.NewTimer(s.processingTime)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}

		result := TranscodingResult{
			JobID: job.ID, VideoID: job.VideoID, Status: "completed", Duration: time.Since(started),
		}
		select {
		case results <- result:
		case <-ctx.Done():
			return
		}
	}
}
