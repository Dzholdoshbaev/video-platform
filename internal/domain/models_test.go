package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestValidationErrorSupportsIsAndAs(t *testing.T) {
	original := &ValidationError{Field: "age", Value: 10, Message: "too young"}
	err := fmt.Errorf("create profile: %w", original)

	if !errors.Is(err, ErrValidation) {
		t.Fatal("expected errors.Is to find ErrValidation")
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatal("expected errors.As to extract *ValidationError")
	}
	if validationErr.Field != "age" {
		t.Fatalf("unexpected field: %s", validationErr.Field)
	}
}

func TestSentinelErrorWrapping(t *testing.T) {
	err := fmt.Errorf("repository lookup: %w", ErrUserNotFound)
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatal("wrapped sentinel must remain discoverable")
	}
}
