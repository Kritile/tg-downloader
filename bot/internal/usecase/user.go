package usecase

import (
	"context"
	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type userService struct {
	userRepo domain.UserRepository
}

func NewUserService(userRepo domain.UserRepository) domain.UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) GetOrCreate(ctx context.Context, telegramID int64, username string) (*models.User, error) {
	return s.userRepo.GetOrCreate(ctx, telegramID, username)
}
