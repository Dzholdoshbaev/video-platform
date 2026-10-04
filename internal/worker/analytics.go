package worker

import (
	"context"
	"fmt"
	"sync"

	"streamingplatform/internal/domain"
)

type AnalyticsProcessor struct {
	ctx    context.Context
	cancel context.CancelFunc
	events chan domain.AnalyticsEvent

	workers sync.WaitGroup
	mu      sync.Mutex
	// mu protects processedEvents. Every worker can increment concurrently; without
	// the mutex those read-modify-write operations form a data race and lose updates.
	processedEvents int
	changed         chan struct{}
}

func NewAnalyticsProcessor(parent context.Context, workerCount, bufferSize int) *AnalyticsProcessor {
	if workerCount < 1 {
		workerCount = 1
	}
	if bufferSize < 1 {
		bufferSize = 1
	}
	ctx, cancel := context.WithCancel(parent)
	processor := &AnalyticsProcessor{
		ctx: ctx, cancel: cancel, events: make(chan domain.AnalyticsEvent, bufferSize),
		changed: make(chan struct{}),
	}
	processor.workers.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		// Go note: a goroutine is a lightweight concurrent function, not a Java Thread object.
		go processor.runWorker()
	}
	return processor
}

func (p *AnalyticsProcessor) Submit(ctx context.Context, event domain.AnalyticsEvent) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("submit analytics event: %w", ctx.Err())
	case <-p.ctx.Done():
		return fmt.Errorf("submit analytics event: processor stopped")
	case p.events <- event:
		return nil
	}
}

func (p *AnalyticsProcessor) runWorker() {
	defer p.workers.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.events:
			p.mu.Lock()
			p.processedEvents++
			close(p.changed)
			p.changed = make(chan struct{})
			p.mu.Unlock()
		}
	}
}

func (p *AnalyticsProcessor) WaitForCount(ctx context.Context, target int) error {
	for {
		p.mu.Lock()
		if p.processedEvents >= target {
			p.mu.Unlock()
			return nil
		}
		changed := p.changed
		p.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}

func (p *AnalyticsProcessor) ProcessedCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.processedEvents
}

func (p *AnalyticsProcessor) Shutdown() {
	// Cancellation is used instead of closing events, avoiding a race between an HTTP
	// Submit and close(events). Wait guarantees that all worker goroutines have exited.
	p.cancel()
	p.workers.Wait()
}
