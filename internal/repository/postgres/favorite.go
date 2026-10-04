package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"streamingplatform/internal/domain"
)

type FavoriteRepository struct{ pool *pgxpool.Pool }

func NewFavoriteRepository(pool *pgxpool.Pool) *FavoriteRepository {
	return &FavoriteRepository{pool: pool}
}

func (r *FavoriteRepository) AddWithAnalytics(ctx context.Context, favorite domain.Favorite, event domain.AnalyticsEvent) (err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin add favorite transaction: %w", err)
	}
	// Go note: defer is registered now and runs on every return path. Rollback after a
	// successful Commit is a documented no-op, so this keeps all failure paths safe.
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			// pgx.ErrTxClosed is expected after Commit; any other rollback failure is relevant.
			if err == nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
				err = fmt.Errorf("rollback add favorite transaction: %w", rollbackErr)
			}
		}
	}()

	var exists bool
	if scanErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, favorite.UserID).Scan(&exists); scanErr != nil {
		return fmt.Errorf("check favorite user: %w", scanErr)
	}
	if !exists {
		return fmt.Errorf("check favorite user: %w", domain.ErrUserNotFound)
	}

	if _, execErr := tx.Exec(ctx, `INSERT INTO favorites (user_id, content_id) VALUES ($1, $2)`, favorite.UserID, favorite.ContentID); execErr != nil {
		return mapError("insert favorite", execErr, domain.ErrUserNotFound)
	}
	if _, execErr := tx.Exec(ctx, `
		INSERT INTO analytics_events (id, user_id, content_id, type, created_at)
		VALUES ($1, $2, $3, $4, $5)`, event.ID, event.UserID, event.ContentID, event.Type, event.CreatedAt); execErr != nil {
		return fmt.Errorf("insert favorite analytics event: %w", execErr)
	}
	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("commit add favorite transaction: %w", commitErr)
	}
	return nil
}

func (r *FavoriteRepository) ListByUser(ctx context.Context, userID string) ([]domain.Favorite, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, content_id FROM favorites WHERE user_id = $1 ORDER BY content_id`, userID)
	if err != nil {
		return nil, fmt.Errorf("list favorites: %w", err)
	}
	defer rows.Close()

	favorites := make([]domain.Favorite, 0)
	for rows.Next() {
		var favorite domain.Favorite
		if err := rows.Scan(&favorite.UserID, &favorite.ContentID); err != nil {
			return nil, fmt.Errorf("scan favorite: %w", err)
		}
		favorites = append(favorites, favorite)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate favorites: %w", err)
	}
	return favorites, nil
}
