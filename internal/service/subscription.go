package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"streamingplatform/internal/domain"
	"streamingplatform/internal/repository"
)

type SubscriptionService struct {
	users         repository.UserRepository
	subscriptions repository.SubscriptionRepository
	payments      PaymentProvider
	notifications *NotificationService
	newID         IDGenerator
}

func NewSubscriptionService(
	users repository.UserRepository,
	subscriptions repository.SubscriptionRepository,
	payments PaymentProvider,
	notifications *NotificationService,
	newID IDGenerator,
) *SubscriptionService {
	return &SubscriptionService{
		users: users, subscriptions: subscriptions, payments: payments,
		notifications: notifications, newID: newID,
	}
}

func (s *SubscriptionService) ActivateSubscription(
	ctx context.Context,
	userID, plan string,
	price float64,
	expirationDate string,
) (*domain.Subscription, error) {
	if _, err := s.users.GetByID(ctx, userID); err != nil {
		// Go note: errors.Is follows wrapped errors, unlike direct equality checks.
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, fmt.Errorf("activate subscription for %s: %w", userID, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("activate subscription lookup user: %w", err)
	}

	expiresAt, err := time.Parse(time.DateOnly, expirationDate)
	if err != nil {
		return nil, &domain.ValidationError{
			Field: "expiration_date", Value: expirationDate, Message: "must use YYYY-MM-DD",
		}
	}
	if err := s.payments.Authorize(ctx, userID, price); err != nil {
		return nil, fmt.Errorf("authorize subscription payment: %w", err)
	}

	subscription, err := domain.NewSubscription(s.newID(), userID, plan, price, expiresAt)
	if err != nil {
		var validationErr *domain.ValidationError
		if errors.As(err, &validationErr) {
			return nil, fmt.Errorf("validate subscription: %w", validationErr)
		}
		return nil, fmt.Errorf("build subscription: %w", err)
	}
	if err := s.subscriptions.Create(ctx, subscription); err != nil {
		return nil, fmt.Errorf("save subscription: %w", err)
	}
	if err := s.payments.Capture(ctx, userID, price); err != nil {
		return nil, fmt.Errorf("capture subscription payment: %w", err)
	}
	if err := s.notifications.SendAll(ctx, userID, "subscription activated: "+plan); err != nil {
		return nil, fmt.Errorf("notify subscription activation: %w", err)
	}
	return subscription, nil
}

func (s *SubscriptionService) Get(ctx context.Context, id string) (*domain.Subscription, error) {
	subscription, err := s.subscriptions.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("subscription service get: %w", err)
	}
	return subscription, nil
}
