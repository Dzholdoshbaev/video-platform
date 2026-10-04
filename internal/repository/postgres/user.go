package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"streamingplatform/internal/domain"
)

type UserRepository struct{ pool *pgxpool.Pool }

func NewUserRepository(pool *pgxpool.Pool) *UserRepository { return &UserRepository{pool: pool} }

func (r *UserRepository) Create(ctx context.Context, profile *domain.Profile) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, username, email, age, country, premium)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		profile.ID, profile.Username, profile.Email, profile.Age, profile.Country, profile.Premium,
	)
	if err != nil {
		return mapError("create user", err, domain.ErrUserNotFound)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.Profile, error) {
	profile := &domain.Profile{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, age, country, premium
		FROM users WHERE id = $1`, id,
	).Scan(&profile.ID, &profile.Username, &profile.Email, &profile.Age, &profile.Country, &profile.Premium)
	if err != nil {
		return nil, mapError("get user", err, domain.ErrUserNotFound)
	}
	return profile, nil
}
