package service

import (
	"context"
	"fmt"
	"time"
)

type TasteSource func(ctx context.Context, userID string) ([]string, error)

type UserTasteService struct{ source TasteSource }

func NewUserTasteService(source TasteSource) *UserTasteService {
	return &UserTasteService{source: source}
}

func SlowTasteSource(ctx context.Context, _ string) ([]string, error) {
	timer := time.NewTimer(250 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return []string{"science-fiction", "documentary", "thriller"}, nil
	}
}

func (s *UserTasteService) Load(ctx context.Context, userID string) ([]string, error) {
	type result struct {
		values []string
		err    error
	}
	// Buffered size 1 lets the producer finish even if cancellation wins the select.
	resultChannel := make(chan result, 1)
	go func() {
		values, err := s.source(ctx, userID)
		resultChannel <- result{values: values, err: err}
	}()

	// Go note: select waits on whichever operation becomes ready first.
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("load user taste: %w", ctx.Err())
	case loaded := <-resultChannel:
		if loaded.err != nil {
			return nil, fmt.Errorf("load user taste source: %w", loaded.err)
		}
		return loaded.values, nil
	}
}

// SelectWithTimeAfter is intentionally educational. In request code prefer a
// context timeout so cancellation propagates through every called layer.
func SelectWithTimeAfter(ch <-chan string, timeout time.Duration) (string, error) {
	select {
	case result := <-ch:
		return result, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("timed out")
	}
}
