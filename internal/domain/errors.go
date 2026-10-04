package domain

import (
	"errors"
	"fmt"
)

// Go note: sentinel errors are stable identities. Callers inspect them with errors.Is,
// even after another layer wraps them with fmt.Errorf("...: %w", err).
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrEpisodeNotFound      = errors.New("episode not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrValidation           = errors.New("validation failed")
	ErrConflict             = errors.New("resource conflict")
)

type ValidationError struct {
	Field   string
	Value   any
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %q (value %v): %s", e.Field, e.Value, e.Message)
}

func (e *ValidationError) Unwrap() error { return ErrValidation }

type ModerationError struct {
	ContentID string
	Reason    string
}

func (e *ModerationError) Error() string {
	if e == nil {
		return "<nil moderation error>"
	}
	return fmt.Sprintf("content %s was rejected: %s", e.ContentID, e.Reason)
}
