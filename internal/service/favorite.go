package service

import (
	"context"
	"fmt"
	"time"

	"streamingplatform/internal/domain"
	"streamingplatform/internal/repository"
)

type FavoriteService struct {
	favorites repository.FavoriteRepository
	newID     IDGenerator
}

func NewFavoriteService(favorites repository.FavoriteRepository, newID IDGenerator) *FavoriteService {
	return &FavoriteService{favorites: favorites, newID: newID}
}

func (s *FavoriteService) Add(ctx context.Context, userID, contentID string) (*domain.Favorite, error) {
	if userID == "" {
		return nil, &domain.ValidationError{Field: "user_id", Value: userID, Message: "must not be empty"}
	}
	if contentID == "" {
		return nil, &domain.ValidationError{Field: "content_id", Value: contentID, Message: "must not be empty"}
	}
	favorite := domain.Favorite{UserID: userID, ContentID: contentID}
	event := domain.AnalyticsEvent{
		ID: s.newID(), UserID: userID, ContentID: contentID,
		Type: domain.EventAddedToFavorites, CreatedAt: time.Now().UTC(),
	}
	if err := s.favorites.AddWithAnalytics(ctx, favorite, event); err != nil {
		return nil, fmt.Errorf("add favorite: %w", err)
	}
	return &favorite, nil
}

func (s *FavoriteService) List(ctx context.Context, userID string) ([]domain.Favorite, error) {
	favorites, err := s.favorites.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("favorite service list: %w", err)
	}
	return favorites, nil
}
