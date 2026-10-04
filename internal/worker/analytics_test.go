package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"streamingplatform/internal/domain"
)

func TestAnalyticsProcessorCountsConcurrentEvents(t *testing.T) {
	processor := NewAnalyticsProcessor(context.Background(), 4, 100)
	for i := 0; i < 50; i++ {
		event := domain.AnalyticsEvent{ID: fmt.Sprint(i)}
		if err := processor.Submit(context.Background(), event); err != nil {
			t.Fatalf("submit: %v", err)
		}
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := processor.WaitForCount(waitCtx, 50); err != nil {
		t.Fatalf("wait for processing: %v", err)
	}
	if got := processor.ProcessedCount(); got != 50 {
		t.Fatalf("processed %d events, want 50", got)
	}
	processor.Shutdown()
}
