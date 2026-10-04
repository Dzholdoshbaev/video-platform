package repository

import (
	"context"

	"streamingplatform/internal/domain"
)

// Go note: interfaces live near their consumers and are satisfied implicitly;
// PostgreSQL repositories never need an "implements" declaration.
type UserRepository interface {
	Create(ctx context.Context, profile *domain.Profile) error
	GetByID(ctx context.Context, id string) (*domain.Profile, error)
}

type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *domain.Subscription) error
	GetByID(ctx context.Context, id string) (*domain.Subscription, error)
}

type WatchHistoryRepository interface {
	Upsert(ctx context.Context, history *domain.WatchHistory) error
	ListByUser(ctx context.Context, userID string) ([]domain.WatchHistory, error)
}

type FavoriteRepository interface {
	// AddWithAnalytics is one atomic use case: either both rows commit or neither does.
	AddWithAnalytics(ctx context.Context, favorite domain.Favorite, event domain.AnalyticsEvent) error
	ListByUser(ctx context.Context, userID string) ([]domain.Favorite, error)
}
