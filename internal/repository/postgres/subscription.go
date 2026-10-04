package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"streamingplatform/internal/domain"
)

type SubscriptionRepository struct{ pool *pgxpool.Pool }

func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

func (r *SubscriptionRepository) Create(ctx context.Context, subscription *domain.Subscription) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subscriptions (id, user_id, plan, price, auto_renew, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		subscription.ID, subscription.UserID, subscription.Plan, subscription.Price,
		subscription.AutoRenew, subscription.ExpiresAt,
	)
	if err != nil {
		return mapError("create subscription", err, domain.ErrSubscriptionNotFound)
	}
	return nil
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, id string) (*domain.Subscription, error) {
	subscription := &domain.Subscription{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, plan, price, auto_renew, expires_at
		FROM subscriptions WHERE id = $1`, id,
	).Scan(
		&subscription.ID, &subscription.UserID, &subscription.Plan, &subscription.Price,
		&subscription.AutoRenew, &subscription.ExpiresAt,
	)
	if err != nil {
		return nil, mapError("get subscription", err, domain.ErrSubscriptionNotFound)
	}
	return subscription, nil
}
