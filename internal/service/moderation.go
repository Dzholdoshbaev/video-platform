package service

import "streamingplatform/internal/domain"

type ModerationService struct{}

func (ModerationService) CheckContentStatus(contentID, status string) error {
	if status == "approved" {
		return nil
	}
	return &domain.ModerationError{ContentID: contentID, Reason: "status is " + status}
}
