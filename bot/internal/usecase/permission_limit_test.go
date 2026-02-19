package usecase_test

import (
	"context"
	"testing"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/bot/internal/usecase"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

// MockSettingsRepository implements domain.SettingsRepository for testing
type MockSettingsRepository struct {
	settings *models.Settings
	err      error
}

func (m *MockSettingsRepository) Get(ctx context.Context) (*models.Settings, error) {
	return m.settings, m.err
}

func (m *MockSettingsRepository) Update(ctx context.Context, settings *models.Settings) error {
	return nil
}

func TestPermissionService_CheckPermission(t *testing.T) {
	tests := []struct {
		name             string
		user             *models.User
		source           models.VideoSource
		settings         *models.Settings
		expectedAllowed  bool
	}{
		{
			name: "User with explicit YouTube permission",
			user: &models.User{
				CanYoutube: boolPtr(true),
			},
			source:          models.SourceYoutube,
			settings:        &models.Settings{DefaultYoutubeAllowed: false},
			expectedAllowed: true,
		},
		{
			name: "User with explicit TikTok denial",
			user: &models.User{
				CanTiktok: boolPtr(false),
			},
			source:          models.SourceTiktok,
			settings:        &models.Settings{DefaultTiktokAllowed: true},
			expectedAllowed: false,
		},
		{
			name: "User without explicit permission uses global settings",
			user: &models.User{
				CanYoutube: nil,
			},
			source:          models.SourceYoutube,
			settings:        &models.Settings{DefaultYoutubeAllowed: true},
			expectedAllowed: true,
		},
		{
			name: "User without explicit permission denied by global settings",
			user: &models.User{
				CanYoutube: nil,
			},
			source:          models.SourceYoutube,
			settings:        &models.Settings{DefaultYoutubeAllowed: false},
			expectedAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockSettingsRepository{settings: tt.settings}
			service := usecase.NewPermissionService(mockRepo)

			allowed, err := service.CheckPermission(context.Background(), tt.user, tt.source)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if allowed != tt.expectedAllowed {
				t.Errorf("expected %v, got %v", tt.expectedAllowed, allowed)
			}
		})
	}
}

func TestLimitService_CheckDailyLimit(t *testing.T) {
	tests := []struct {
		name            string
		user            *models.User
		dailyCount      int
		settings        *models.Settings
		expectedAllowed bool
	}{
		{
			name: "Under daily limit",
			user: &models.User{
				DailyLimit: intPtr(10),
			},
			dailyCount:      5,
			settings:        &models.Settings{DefaultDailyLimit: 10},
			expectedAllowed: true,
		},
		{
			name: "At daily limit",
			user: &models.User{
				DailyLimit: intPtr(10),
			},
			dailyCount:      10,
			settings:        &models.Settings{DefaultDailyLimit: 10},
			expectedAllowed: false,
		},
		{
			name: "Over daily limit",
			user: &models.User{
				DailyLimit: intPtr(10),
			},
			dailyCount:      15,
			settings:        &models.Settings{DefaultDailyLimit: 10},
			expectedAllowed: false,
		},
		{
			name: "User limit nil uses global",
			user: &models.User{
				DailyLimit: nil,
			},
			dailyCount:      5,
			settings:        &models.Settings{DefaultDailyLimit: 10},
			expectedAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDownloadRepo := &MockDownloadRepository{dailyCount: tt.dailyCount}
			mockSettingsRepo := &MockSettingsRepository{settings: tt.settings}
			service := usecase.NewLimitService(mockDownloadRepo, mockSettingsRepo)

			allowed, current, limit, err := service.CheckDailyLimit(context.Background(), tt.user)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if allowed != tt.expectedAllowed {
				t.Errorf("expected allowed=%v, got %v", tt.expectedAllowed, allowed)
			}

			if current != tt.dailyCount {
				t.Errorf("expected current=%d, got %d", tt.dailyCount, current)
			}

			expectedLimit := 10 // Default fallback
			if tt.user.DailyLimit != nil {
				expectedLimit = *tt.user.DailyLimit
			}
			if limit != expectedLimit {
				t.Errorf("expected limit=%d, got %d", expectedLimit, limit)
			}
		})
	}
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

// MockDownloadRepository implements domain.DownloadRepository for testing
type MockDownloadRepository struct {
	dailyCount   int
	monthlyCount int
}

func (m *MockDownloadRepository) Create(ctx context.Context, download *models.Download) error {
	return nil
}

func (m *MockDownloadRepository) GetDailyCount(ctx context.Context, userID int64) (int, error) {
	return m.dailyCount, nil
}

func (m *MockDownloadRepository) GetMonthlyCount(ctx context.Context, userID int64) (int, error) {
	return m.monthlyCount, nil
}

func (m *MockDownloadRepository) GetUserDownloads(ctx context.Context, userID int64, limit, offset int) ([]*models.Download, error) {
	return []*models.Download{}, nil
}
