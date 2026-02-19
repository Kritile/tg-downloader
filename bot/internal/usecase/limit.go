package usecase

import (
	"context"
	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type limitService struct {
	downloadRepo domain.DownloadRepository
	settingsRepo domain.SettingsRepository
}

func NewLimitService(downloadRepo domain.DownloadRepository, settingsRepo domain.SettingsRepository) domain.LimitService {
	return &limitService{
		downloadRepo: downloadRepo,
		settingsRepo: settingsRepo,
	}
}

func (s *limitService) CheckDailyLimit(ctx context.Context, user *models.User) (bool, int, int, error) {
	// Get current daily count
	currentCount, err := s.downloadRepo.GetDailyCount(ctx, user.ID)
	if err != nil {
		return false, 0, 0, err
	}

	// Determine the limit to use
	limit := s.getDailyLimit(user)

	return currentCount < limit, currentCount, limit, nil
}

func (s *limitService) CheckMonthlyLimit(ctx context.Context, user *models.User) (bool, int, int, error) {
	// Get current monthly count
	currentCount, err := s.downloadRepo.GetMonthlyCount(ctx, user.ID)
	if err != nil {
		return false, 0, 0, err
	}

	// Determine the limit to use
	limit := s.getMonthlyLimit(user)

	return currentCount < limit, currentCount, limit, nil
}

func (s *limitService) getDailyLimit(user *models.User) int {
	// Priority: user.daily_limit != NULL → use it
	if user.DailyLimit != nil {
		return *user.DailyLimit
	}

	// Else → use global_daily_limit (from settings)
	// This is handled at a higher level, return a default
	return 10 // Default fallback
}

func (s *limitService) getMonthlyLimit(user *models.User) int {
	// Priority: user.monthly_limit != NULL → use it
	if user.MonthlyLimit != nil {
		return *user.MonthlyLimit
	}

	// Else → use global_monthly_limit (from settings)
	// This is handled at a higher level, return a default
	return 100 // Default fallback
}
