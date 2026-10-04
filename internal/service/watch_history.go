package service

import (
	"context"
	"fmt"
	"time"

	"streamingplatform/internal/domain"
	"streamingplatform/internal/repository"
)

type WatchHistoryService struct {
	history repository.WatchHistoryRepository
	newID   IDGenerator
}

func NewWatchHistoryService(history repository.WatchHistoryRepository, newID IDGenerator) *WatchHistoryService {
	return &WatchHistoryService{history: history, newID: newID}
}

func (s *WatchHistoryService) Save(ctx context.Context, item *domain.WatchHistory) (*domain.WatchHistory, error) {
	if item.UserID == "" {
		return nil, &domain.ValidationError{Field: "user_id", Value: item.UserID, Message: "must not be empty"}
	}
	if item.EpisodeID == "" {
		return nil, &domain.ValidationError{Field: "episode_id", Value: item.EpisodeID, Message: "must not be empty"}
	}
	if item.WatchedPercentage < 0 || item.WatchedPercentage > 100 {
		return nil, &domain.ValidationError{Field: "watched_percentage", Value: item.WatchedPercentage, Message: "must be between 0 and 100"}
	}
	item.ID = s.newID()
	item.UpdatedAt = time.Now().UTC()
	if err := s.history.Upsert(ctx, item); err != nil {
		return nil, fmt.Errorf("save watch history: %w", err)
	}
	return item, nil
}

func (s *WatchHistoryService) List(ctx context.Context, userID string) ([]domain.WatchHistory, error) {
	history, err := s.history.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("watch history service list: %w", err)
	}
	return history, nil
}
