package service

import (
	"context"
	"fmt"

	"streamingplatform/internal/domain"
	"streamingplatform/internal/repository"
)

type ProfileService struct {
	users repository.UserRepository
	newID IDGenerator
}

func NewProfileService(users repository.UserRepository, newID IDGenerator) *ProfileService {
	return &ProfileService{users: users, newID: newID}
}

func (s *ProfileService) Create(ctx context.Context, username, email string, age int, country string) (*domain.Profile, error) {
	profile, err := domain.NewProfile(s.newID(), username, email, age, country)
	if err != nil {
		return nil, fmt.Errorf("build profile: %w", err)
	}
	if err := s.users.Create(ctx, profile); err != nil {
		return nil, fmt.Errorf("persist profile: %w", err)
	}
	return profile, nil
}

func (s *ProfileService) Get(ctx context.Context, id string) (*domain.Profile, error) {
	profile, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("profile service get: %w", err)
	}
	return profile, nil
}
