package transport

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/bot/internal/usecase"
	"github.com/mediaharvester/tg-downloader/bot/internal/worker"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type Bot struct {
	api            *tgbotapi.BotAPI
	urlValidator   domain.URLValidator
	queueService   *worker.QueueService
	userService    domain.UserService
	permissionSvc  domain.PermissionService
	limitSvc       domain.LimitService
	downloadPath   string
}

func NewBot(
	api *tgbotapi.BotAPI,
	urlValidator domain.URLValidator,
	queueService *worker.QueueService,
	userService domain.UserService,
	permissionSvc domain.PermissionService,
	limitSvc domain.LimitService,
	downloadPath string,
) *Bot {
	return &Bot{
		api:            api,
		urlValidator:   urlValidator,
		queueService:   queueService,
		userService:    userService,
		permissionSvc:  permissionSvc,
		limitSvc:       limitSvc,
		downloadPath:   downloadPath,
	}
}

func (b *Bot) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	log.Println("Bot started, listening for messages...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			if update.Message != nil {
				b.handleMessage(ctx, update.Message)
			}
		}
	}
}

func (b *Bot) handleMessage(ctx context.Context, message *tgbotapi.Message) {
	// Ignore non-text messages
	if message.Text == "" {
		return
	}

	text := strings.TrimSpace(message.Text)

	// Handle /start command
	if text == "/start" {
		b.handleStartCommand(message)
		return
	}

	// Handle URL messages
	b.handleURL(ctx, message, text)
}

func (b *Bot) handleStartCommand(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, 
		"👋 Welcome to MediaHarvester Bot!\n\n"+
		"Send me a YouTube or TikTok link and I'll download the video for you.\n\n"+
		"⚠️ Note: Maximum file size is 50MB")

	b.api.Send(msg)
}

func (b *Bot) handleURL(ctx context.Context, message *tgbotapi.Message, text string) {
	// Validate URL and detect source
	source, err := b.urlValidator.ValidateAndDetectSource(text)
	if err != nil {
		b.sendError(message.Chat.ID, err)
		return
	}

	// Get or create user
	user, err := b.userService.GetOrCreate(ctx, message.From.ID, message.From.UserName)
	if err != nil {
		log.Printf("Failed to get/create user: %v", err)
		b.sendError(message.Chat.ID, fmt.Errorf("failed to process request"))
		return
	}

	// Check permission
	hasPermission, err := b.permissionSvc.CheckPermission(ctx, user, source)
	if err != nil {
		log.Printf("Failed to check permission: %v", err)
		b.sendError(message.Chat.ID, fmt.Errorf("failed to check permission"))
		return
	}
	if !hasPermission {
		b.sendMessage(message.Chat.ID, "❌ You don't have permission to download from this source.")
		return
	}

	// Check daily limit
	dailyOk, currentDaily, limitDaily, err := b.limitSvc.CheckDailyLimit(ctx, user)
	if err != nil {
		log.Printf("Failed to check daily limit: %v", err)
		b.sendError(message.Chat.ID, fmt.Errorf("failed to check limit"))
		return
	}
	if !dailyOk {
		b.sendMessage(message.Chat.ID, 
			fmt.Sprintf("⚠️ Daily limit exceeded. You've downloaded %d/%d videos today.", currentDaily, limitDaily))
		return
	}

	// Check monthly limit
	monthlyOk, currentMonthly, limitMonthly, err := b.limitSvc.CheckMonthlyLimit(ctx, user)
	if err != nil {
		log.Printf("Failed to check monthly limit: %v", err)
		b.sendError(message.Chat.ID, fmt.Errorf("failed to check limit"))
		return
	}
	if !monthlyOk {
		b.sendMessage(message.Chat.ID, 
			fmt.Sprintf("⚠️ Monthly limit exceeded. You've downloaded %d/%d videos this month.", currentMonthly, limitMonthly))
		return
	}

	// Push job to queue
	job := &models.DownloadJob{
		UserID:   user.ID,
		ChatID:   message.Chat.ID,
		URL:      text,
		Source:   string(source),
		Username: message.From.UserName,
	}

	if err := b.queueService.PushJob(ctx, job); err != nil {
		log.Printf("Failed to push job to queue: %v", err)
		b.sendError(message.Chat.ID, fmt.Errorf("failed to queue download"))
		return
	}

	// Notify user
	b.sendMessage(message.Chat.ID, "⬇️ Downloading your video... Please wait.")
}

func (b *Bot) sendError(chatID int64, err error) {
	errorMsg := "❌ An error occurred: " + err.Error()
	
	// Sanitize error message for users
	if err == domain.ErrInvalidURL {
		errorMsg = "❌ Invalid URL. Please send a valid YouTube or TikTok link."
	} else if err == domain.ErrUnsupportedSource {
		errorMsg = "❌ Unsupported source. Currently only YouTube and TikTok are supported."
	}

	b.sendMessage(chatID, errorMsg)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.api.Send(msg)
}

// SendVideo sends a video file to the user and deletes it afterward
func (b *Bot) SendVideo(chatID int64, filePath string) error {
	// Send video
	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
	video.Caption = "🎬 Here's your video!"

	_, err := b.api.Send(video)
	if err != nil {
		return fmt.Errorf("failed to send video: %w", err)
	}

	// Delete file immediately after sending
	if err := usecase.CleanupDownloadedFile(filePath); err != nil {
		log.Printf("Failed to delete file %s: %v", filePath, err)
	}

	return nil
}

// SendFileTooLarge sends a message about file size limit
func (b *Bot) SendFileTooLarge(chatID int64) {
	b.sendMessage(chatID, "❌ File is too large. Maximum size is 50MB.")
}

// SendDownloadFailed sends a message about download failure
func (b *Bot) SendDownloadFailed(chatID int64) {
	b.sendMessage(chatID, "❌ Failed to download video. Please try another link.")
}
