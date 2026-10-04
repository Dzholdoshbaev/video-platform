package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"streamingplatform/internal/domain"
)

type WatchHistoryRepository struct{ pool *pgxpool.Pool }

func NewWatchHistoryRepository(pool *pgxpool.Pool) *WatchHistoryRepository {
	return &WatchHistoryRepository{pool: pool}
}

func (r *WatchHistoryRepository) Upsert(ctx context.Context, history *domain.WatchHistory) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO watch_history
			(id, user_id, episode_id, watched_percentage, last_position, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, episode_id) DO UPDATE SET
			watched_percentage = EXCLUDED.watched_percentage,
			last_position = EXCLUDED.last_position,
			updated_at = EXCLUDED.updated_at`,
		history.ID, history.UserID, history.EpisodeID, history.WatchedPercentage,
		history.LastPosition, history.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert watch history: %w", err)
	}
	return nil
}

func (r *WatchHistoryRepository) ListByUser(ctx context.Context, userID string) ([]domain.WatchHistory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, episode_id, watched_percentage, last_position, updated_at
		FROM watch_history WHERE user_id = $1 ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list watch history: %w", err)
	}
	defer rows.Close()

	history := make([]domain.WatchHistory, 0)
	for rows.Next() {
		var item domain.WatchHistory
		if err := rows.Scan(&item.ID, &item.UserID, &item.EpisodeID, &item.WatchedPercentage, &item.LastPosition, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan watch history: %w", err)
		}
		history = append(history, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watch history: %w", err)
	}
	return history, nil
}
