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
	formatSvc      domain.FormatService
	downloadPath   string
	pendingURLs    map[int64]*pendingDownload // chatID -> pending download info
}

type pendingDownload struct {
	UserID   int64
	ChatID   int64
	URL      string
	Source   string
	Username string
}

func NewBot(
	api *tgbotapi.BotAPI,
	urlValidator domain.URLValidator,
	queueService *worker.QueueService,
	userService domain.UserService,
	permissionSvc domain.PermissionService,
	limitSvc domain.LimitService,
	formatSvc domain.FormatService,
	downloadPath string,
) *Bot {
	return &Bot{
		api:            api,
		urlValidator:   urlValidator,
		queueService:   queueService,
		userService:    userService,
		permissionSvc:  permissionSvc,
		limitSvc:       limitSvc,
		formatSvc:      formatSvc,
		downloadPath:   downloadPath,
		pendingURLs:    make(map[int64]*pendingDownload),
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
			} else if update.CallbackQuery != nil {
				b.handleCallbackQuery(ctx, update.CallbackQuery)
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

func (b *Bot) handleCallbackQuery(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	// Acknowledge callback
	answer := tgbotapi.CallbackConfig{
		CallbackQueryID: callback.ID,
		Text:            "",
		ShowAlert:       false,
	}
	b.api.Request(answer)

	// Parse callback data: "format_<formatID>" or "download_best"
	data := callback.Data
	chatID := callback.Message.Chat.ID

	if data == "download_best" {
		b.handleDownloadBest(ctx, chatID)
		return
	}

	if strings.HasPrefix(data, "format_") {
		formatID := strings.TrimPrefix(data, "format_")
		b.handleFormatSelected(ctx, chatID, formatID)
		return
	}
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

	// Store pending download info
	b.pendingURLs[message.Chat.ID] = &pendingDownload{
		UserID:   user.ID,
		ChatID:   message.Chat.ID,
		URL:      text,
		Source:   string(source),
		Username: message.From.UserName,
	}

	// Fetch available formats
	b.sendMessage(message.Chat.ID, "🔍 Fetching available formats... Please wait.")

	formats, err := b.formatSvc.ListFormats(ctx, text, source)
	if err != nil {
		log.Printf("Failed to list formats: %v", err)
		b.sendMessage(message.Chat.ID, "⚠️ Could not fetch formats. Downloading with best quality...")
		b.queueDownload(ctx, message.Chat.ID, user.ID, text, string(source), "")
		return
	}

	if len(formats) == 0 {
		b.sendMessage(message.Chat.ID, "⚠️ No formats found. Downloading with best quality...")
		b.queueDownload(ctx, message.Chat.ID, user.ID, text, string(source), "")
		return
	}

	// Send format selection message
	b.SendFormatSelection(message.Chat.ID, formats)
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

// SendFormatSelection sends a message with format selection buttons
func (b *Bot) SendFormatSelection(chatID int64, formats []models.VideoFormat) error {
	// Limit to top 10 formats to avoid overwhelming the user
	maxFormats := 10
	if len(formats) < maxFormats {
		maxFormats = len(formats)
	}

	// Create inline keyboard with format options
	keyboard := make([][]tgbotapi.InlineKeyboardButton, 0)

	// Add format buttons (2 per row)
	for i := 0; i < maxFormats; i += 2 {
		row := []tgbotapi.InlineKeyboardButton{}
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(
			formats[i].DisplayName,
			fmt.Sprintf("format_%s", formats[i].FormatID),
		))
		if i+1 < maxFormats {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(
				formats[i+1].DisplayName,
				fmt.Sprintf("format_%s", formats[i+1].FormatID),
			))
		}
		keyboard = append(keyboard, row)
	}

	// Add "Best Quality" button
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⚡ Best Quality (Auto)", "download_best"),
	})

	// Create message
	msg := tgbotapi.NewMessage(chatID, "📹 Select video quality:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	_, err := b.api.Send(msg)
	return err
}

// handleDownloadBest handles the "Best Quality" selection
func (b *Bot) handleDownloadBest(ctx context.Context, chatID int64) {
	pending, ok := b.pendingURLs[chatID]
	if !ok {
		b.sendMessage(chatID, "❌ Session expired. Please send the URL again.")
		return
	}
	delete(b.pendingURLs, chatID)

	b.sendMessage(chatID, "⬇️ Downloading with best quality... Please wait.")
	b.queueDownload(ctx, pending.ChatID, pending.UserID, pending.URL, pending.Source, "")
}

// handleFormatSelected handles format selection from inline button
func (b *Bot) handleFormatSelected(ctx context.Context, chatID int64, formatID string) {
	pending, ok := b.pendingURLs[chatID]
	if !ok {
		b.sendMessage(chatID, "❌ Session expired. Please send the URL again.")
		return
	}
	delete(b.pendingURLs, chatID)

	b.sendMessage(chatID, "⬇️ Downloading with selected format... Please wait.")
	b.queueDownload(ctx, pending.ChatID, pending.UserID, pending.URL, pending.Source, formatID)
}

// queueDownload adds a download job to the queue
func (b *Bot) queueDownload(ctx context.Context, chatID, userID int64, url, source, format string) {
	job := &models.DownloadJob{
		UserID:   userID,
		ChatID:   chatID,
		URL:      url,
		Source:   source,
		Format:   format,
	}

	if err := b.queueService.PushJob(ctx, job); err != nil {
		log.Printf("Failed to push job to queue: %v", err)
		b.sendError(chatID, fmt.Errorf("failed to queue download"))
		return
	}

	b.sendMessage(chatID, "⬇️ Downloading your video... Please wait.")
}
