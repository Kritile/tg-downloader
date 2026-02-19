package usecase

import (
	"context"

	"github.com/mediaharvester/tg-downloader/admin/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type adminUserManagementService struct {
	userRepo     domain.UserRepository
	settingsRepo domain.SettingsRepository
}

func NewAdminUserManagementService(
	userRepo domain.UserRepository,
	settingsRepo domain.SettingsRepository,
) domain.AdminUserManagementService {
	return &adminUserManagementService{
		userRepo:     userRepo,
		settingsRepo: settingsRepo,
	}
}

func (s *adminUserManagementService) GetUserByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	return s.userRepo.GetByTelegramID(ctx, telegramID)
}

func (s *adminUserManagementService) UpdateUserPermissions(
	ctx context.Context,
	userID int64,
	canYoutube, canTiktok *bool,
) error {
	user, err := s.userRepo.GetByTelegramID(ctx, userID)
	if err != nil {
		return err
	}

	if canYoutube != nil {
		user.CanYoutube = canYoutube
	}
	if canTiktok != nil {
		user.CanTiktok = canTiktok
	}

	return s.userRepo.Update(ctx, user)
}

func (s *adminUserManagementService) UpdateUserLimits(
	ctx context.Context,
	userID int64,
	dailyLimit, monthlyLimit *int,
) error {
	user, err := s.userRepo.GetByTelegramID(ctx, userID)
	if err != nil {
		return err
	}

	if dailyLimit != nil {
		user.DailyLimit = dailyLimit
	}
	if monthlyLimit != nil {
		user.MonthlyLimit = monthlyLimit
	}

	return s.userRepo.Update(ctx, user)
}

func (s *adminUserManagementService) GetAllUsers(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	// TODO: Implement pagination in repository
	// For now, return a placeholder
	return []*models.User{}, 0, nil
}

func (s *adminUserManagementService) SearchUsers(ctx context.Context, query string, limit int) ([]*models.User, error) {
	// TODO: Implement search in repository
	// For now, return empty
	return []*models.User{}, nil
}
