package usecase

import (
	"context"
	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type permissionService struct {
	settingsRepo domain.SettingsRepository
}

func NewPermissionService(settingsRepo domain.SettingsRepository) domain.PermissionService {
	return &permissionService{
		settingsRepo: settingsRepo,
	}
}

func (s *permissionService) CheckPermission(ctx context.Context, user *models.User, source models.VideoSource) (bool, error) {
	// If user has explicit permission set, use it
	if user.CanYoutube != nil && source == models.SourceYoutube {
		return *user.CanYoutube, nil
	}
	if user.CanTiktok != nil && source == models.SourceTiktok {
		return *user.CanTiktok, nil
	}

	// Otherwise, use global settings
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return false, err
	}

	switch source {
	case models.SourceYoutube:
		return settings.DefaultYoutubeAllowed, nil
	case models.SourceTiktok:
		return settings.DefaultTiktokAllowed, nil
	default:
		return false, domain.ErrUnsupportedSource
	}
}
