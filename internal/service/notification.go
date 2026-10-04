package service

import (
	"context"
	"errors"
	"fmt"
)

type Notifier interface {
	Send(ctx context.Context, userID string, message string) error
	ChannelName() string
}

type EmailNotifier struct{}

func (EmailNotifier) Send(ctx context.Context, userID, message string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (EmailNotifier) ChannelName() string { return "email" }

type PushNotifier struct{}

func (PushNotifier) Send(ctx context.Context, userID, message string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (PushNotifier) ChannelName() string { return "push" }

type NotificationService struct{ notifiers []Notifier }

func NewNotificationService(notifiers []Notifier) *NotificationService {
	return &NotificationService{notifiers: append([]Notifier(nil), notifiers...)}
}

func (s *NotificationService) SendAll(ctx context.Context, userID, message string) error {
	var sendErrors []error
	for _, notifier := range s.notifiers {
		if err := notifier.Send(ctx, userID, message); err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("send %s notification: %w", notifier.ChannelName(), err))
		}
	}
	// errors.Join keeps every wrapped cause discoverable via errors.Is/errors.As while
	// still attempting all configured channels after one provider fails.
	return errors.Join(sendErrors...)
}
