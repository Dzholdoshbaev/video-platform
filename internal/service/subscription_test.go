package service

import (
	"context"
	"errors"
	"testing"

	"streamingplatform/internal/domain"
)

type subscriptionUserRepository struct {
	profile *domain.Profile
	err     error
}

func (r subscriptionUserRepository) Create(context.Context, *domain.Profile) error { return nil }
func (r subscriptionUserRepository) GetByID(context.Context, string) (*domain.Profile, error) {
	return r.profile, r.err
}

type subscriptionRepositoryFake struct {
	created *domain.Subscription
	err     error
}

func (r *subscriptionRepositoryFake) Create(_ context.Context, subscription *domain.Subscription) error {
	r.created = subscription
	return r.err
}

func (r *subscriptionRepositoryFake) GetByID(context.Context, string) (*domain.Subscription, error) {
	return nil, domain.ErrSubscriptionNotFound
}

type paymentFake struct {
	authorized bool
	captured   bool
}

func (p *paymentFake) Authorize(context.Context, string, float64) error {
	p.authorized = true
	return nil
}

func (p *paymentFake) Capture(context.Context, string, float64) error {
	p.captured = true
	return nil
}

func TestSubscriptionServiceActivate(t *testing.T) {
	repository := &subscriptionRepositoryFake{}
	payment := &paymentFake{}
	notifier := &recordingNotifier{channel: "test"}
	service := NewSubscriptionService(
		subscriptionUserRepository{profile: &domain.Profile{ID: "user-1"}},
		repository,
		payment,
		NewNotificationService([]Notifier{notifier}),
		func() string { return "subscription-1" },
	)

	created, err := service.ActivateSubscription(context.Background(), "user-1", "premium", 12.50, "2030-01-31")
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if created.ID != "subscription-1" || !created.AutoRenew {
		t.Fatalf("unexpected subscription: %+v", created)
	}
	if repository.created == nil || !payment.authorized || !payment.captured || notifier.calls != 1 {
		t.Fatal("expected persistence, authorization, capture and notification")
	}
}

func TestSubscriptionServiceReturnsWrappedNotFound(t *testing.T) {
	service := NewSubscriptionService(
		subscriptionUserRepository{err: domain.ErrUserNotFound},
		&subscriptionRepositoryFake{}, &paymentFake{}, NewNotificationService(nil),
		func() string { return "unused" },
	)
	_, err := service.ActivateSubscription(context.Background(), "missing", "premium", 1, "2030-01-01")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestSubscriptionServiceReturnsValidationErrorForDate(t *testing.T) {
	service := NewSubscriptionService(
		subscriptionUserRepository{profile: &domain.Profile{ID: "user-1"}},
		&subscriptionRepositoryFake{}, &paymentFake{}, NewNotificationService(nil),
		func() string { return "unused" },
	)
	_, err := service.ActivateSubscription(context.Background(), "user-1", "premium", 1, "31-01-2030")
	var validationErr *domain.ValidationError
	if !errors.As(err, &validationErr) || validationErr.Field != "expiration_date" {
		t.Fatalf("expected expiration_date ValidationError, got %v", err)
	}
}
