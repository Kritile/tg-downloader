package usecase

import (
	"context"

	"github.com/mediaharvester/tg-downloader/admin/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type adminStatsService struct {
	downloadRepo domain.DownloadRepository
	userRepo     domain.UserRepository
}

func NewAdminStatsService(
	downloadRepo domain.DownloadRepository,
	userRepo domain.UserRepository,
) domain.AdminStatsService {
	return &adminStatsService{
		downloadRepo: downloadRepo,
		userRepo:     userRepo,
	}
}

func (s *adminStatsService) GetDownloadsToday(ctx context.Context) (int, error) {
	// TODO: Implement in repository
	return 0, nil
}

func (s *adminStatsService) GetDownloadsThisMonth(ctx context.Context) (int, error) {
	// TODO: Implement in repository
	return 0, nil
}

func (s *adminStatsService) GetTopUsers(ctx context.Context, limit int) ([]*models.User, []int, error) {
	// TODO: Implement in repository
	return []*models.User{}, []int{}, nil
}

func (s *adminStatsService) GetTotalUsers(ctx context.Context) (int, error) {
	// TODO: Implement in repository
	return 0, nil
}
