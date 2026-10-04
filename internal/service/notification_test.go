package service

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type recordingNotifier struct {
	channel string
	calls   int
	err     error
}

func (n *recordingNotifier) Send(_ context.Context, _, _ string) error {
	n.calls++
	return n.err
}

func (n *recordingNotifier) ChannelName() string { return n.channel }

func TestNotificationServiceUsesEveryChannel(t *testing.T) {
	email := &recordingNotifier{channel: "email"}
	push := &recordingNotifier{channel: "push"}
	notifications := NewNotificationService([]Notifier{email, push})

	if err := notifications.SendAll(context.Background(), "user-1", "hello"); err != nil {
		t.Fatalf("send notifications: %v", err)
	}
	if email.calls != 1 || push.calls != 1 {
		t.Fatalf("calls: email=%d push=%d", email.calls, push.calls)
	}
}

func TestNotificationServiceWrapsChannelError(t *testing.T) {
	sentinel := errors.New("provider unavailable")
	push := &recordingNotifier{channel: "push"}
	notifications := NewNotificationService([]Notifier{
		&recordingNotifier{channel: "email", err: sentinel},
		push,
	})
	err := notifications.SendAll(context.Background(), "user-1", "hello")
	if !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "email") {
		t.Fatalf("expected wrapped channel error, got %v", err)
	}
	if push.calls != 1 {
		t.Fatal("a failed channel must not prevent later channels from being attempted")
	}
}
