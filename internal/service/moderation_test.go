package service

import (
	"errors"
	"testing"

	"streamingplatform/internal/domain"
)

func TestModerationErrorSupportsErrorsAs(t *testing.T) {
	err := (ModerationService{}).CheckContentStatus("video-1", "rejected")
	var moderationErr *domain.ModerationError
	if !errors.As(err, &moderationErr) {
		t.Fatal("expected *ModerationError")
	}
	if moderationErr.ContentID != "video-1" {
		t.Fatalf("unexpected content ID: %s", moderationErr.ContentID)
	}
}

func TestTypedNilInterfacePitfall(t *testing.T) {
	var moderationErr *domain.ModerationError = nil
	var err error = moderationErr
	// Go note: an interface contains (dynamic type, dynamic value). The value is nil,
	// but the dynamic type is *ModerationError, so the interface itself is not nil.
	if err == nil {
		t.Fatal("typed nil stored in an interface is expected to be non-nil")
	}
}
