package usecase

import (
	"context"
	"errors"
	"strings"

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
		return nil, domain.ErrInvalidCredentials
	}

	return admin, nil
}

func (s *adminUserService) CreateAdmin(ctx context.Context, username, password string) error {
	_, err := s.adminRepo.GetByUsername(ctx, username)
	if err == nil {
		return errors.New("admin already exists")
	}

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

func (s *adminUserService) ChangePassword(ctx context.Context, adminID int64, currentPassword, newPassword string) error {
	if len(newPassword) < 12 {
		return errors.New("new password must contain at least 12 characters")
	}
	if strings.EqualFold(currentPassword, newPassword) {
		return errors.New("new password must be different from current password")
	}

	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(currentPassword)); err != nil {
		return domain.ErrInvalidCredentials
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.adminRepo.UpdatePasswordHash(ctx, adminID, string(hashedPassword))
}
