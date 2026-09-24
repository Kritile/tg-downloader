package usecase

import (
	"context"
	"strconv"

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
	canYoutube, canInstagram, canTiktok *bool,
) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if canYoutube != nil {
		user.CanYoutube = canYoutube
	}
	if canInstagram != nil {
		user.CanInstagram = canInstagram
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
	user, err := s.userRepo.GetByID(ctx, userID)
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
	return s.userRepo.GetAll(ctx, limit, offset)
}

func (s *adminUserManagementService) SearchUsers(ctx context.Context, query string, limit int) ([]*models.User, error) {
	telegramID, err := strconv.ParseInt(query, 10, 64)
	if err != nil {
		return []*models.User{}, nil
	}
	return s.userRepo.SearchByTelegramID(ctx, telegramID, limit)
}

func (s *adminUserManagementService) SearchUsersPage(ctx context.Context, query string, limit, offset int) ([]*models.User, int, error) {
	return s.userRepo.Search(ctx, query, limit, offset)
}

func (s *adminUserManagementService) SetUserBlocked(ctx context.Context, userID int64, blocked bool) error {
	return s.userRepo.SetBlocked(ctx, userID, blocked)
}
