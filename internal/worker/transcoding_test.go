package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestVideoTranscodingServiceCompletesBatch(t *testing.T) {
	transcoder := NewVideoTranscodingService(2, time.Second, 0)
	jobs := []TranscodingJob{
		{ID: "job-1", VideoID: "video-1", Filename: "one.mp4"},
		{ID: "job-2", VideoID: "video-2", Filename: "two.mp4"},
	}
	results, err := transcoder.TranscodeBatch(context.Background(), jobs)
	if err != nil {
		t.Fatalf("transcode: %v", err)
	}
	if len(results) != len(jobs) {
		t.Fatalf("results=%d jobs=%d", len(results), len(jobs))
	}
}

func TestVideoTranscodingServiceCancelsBatch(t *testing.T) {
	transcoder := NewVideoTranscodingService(1, time.Millisecond, time.Second)
	_, err := transcoder.TranscodeBatch(context.Background(), []TranscodingJob{{ID: "job-1"}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline, got %v", err)
	}
}
