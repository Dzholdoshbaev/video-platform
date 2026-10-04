package worker

import (
	"context"
	"fmt"
	"sync"
)

const (
	PlayerEventPlay  = "play"
	PlayerEventPause = "pause"
	PlayerEventSeek  = "seek"
	PlayerEventStop  = "stop"
)

type PlayerEvent struct {
	Type     string `json:"type"`
	Position int    `json:"position,omitempty"`
}

type PlayerEventQueue struct {
	events    chan PlayerEvent
	closeOnce sync.Once
}

func NewPlayerEventQueue(capacity int) *PlayerEventQueue {
	return &PlayerEventQueue{events: make(chan PlayerEvent, capacity)}
}

func (q *PlayerEventQueue) Publish(ctx context.Context, event PlayerEvent) error {
	select {
	case q.events <- event:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("publish player event: %w", ctx.Err())
	}
}

func (q *PlayerEventQueue) Consume(ctx context.Context, handle func(PlayerEvent)) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, open := <-q.events:
			if !open {
				return nil
			}
			handle(event)
		}
	}
}

func (q *PlayerEventQueue) Close() {
	// The queue is the sending side's owner. Consumers only range/receive and never close it.
	q.closeOnce.Do(func() { close(q.events) })
}

// Range demonstrates the compact receive-until-close form.
func (q *PlayerEventQueue) Range(handle func(PlayerEvent)) {
	for event := range q.events {
		handle(event)
	}
}
