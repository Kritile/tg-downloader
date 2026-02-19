package usecase

import (
	"context"
	"errors"

	"github.com/mediaharvester/tg-downloader/admin/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
	"golang.org/x/crypto/bcrypt"
)

type adminUserService struct {
	adminRepo domain.AdminUserRepository
}

func NewAdminUserService(adminRepo domain.AdminUserRepository) domain.AdminUserService {
	return &adminUserService{
		adminRepo: adminRepo,
	}
}

func (s *adminUserService) Authenticate(ctx context.Context, username, password string) (*models.Admin, error) {
	admin, err := s.adminRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	return admin, nil
}

func (s *adminUserService) CreateAdmin(ctx context.Context, username, password string) error {
	// Check if admin already exists
	_, err := s.adminRepo.GetByUsername(ctx, username)
	if err == nil {
		return errors.New("admin already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := &models.Admin{
		Username:     username,
		PasswordHash: string(hashedPassword),
	}

	return s.adminRepo.Create(ctx, admin)
}
