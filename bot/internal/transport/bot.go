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

const (
	callbackDownloadBest = "download_best"
	callbackModeAuto     = "mode_auto"
	callbackModeManual   = "mode_manual"
)

type Bot struct {
	api           *tgbotapi.BotAPI
	urlValidator  domain.URLValidator
	queueService  *worker.QueueService
	userService   domain.UserService
	permissionSvc domain.PermissionService
	limitSvc      domain.LimitService
	formatSvc     domain.FormatService
	downloadPath  string
	pendingURLs   map[int64]*pendingDownload
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
		api:           api,
		urlValidator:  urlValidator,
		queueService:  queueService,
		userService:   userService,
		permissionSvc: permissionSvc,
		limitSvc:      limitSvc,
		formatSvc:     formatSvc,
		downloadPath:  downloadPath,
		pendingURLs:   make(map[int64]*pendingDownload),
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
	if message.Text == "" {
		return
	}

	text := strings.TrimSpace(message.Text)

	switch text {
	case "/start":
		b.handleStartCommand(ctx, message)
		return
	case "/limits":
		b.handleLimitsCommand(ctx, message)
		return
	case "/mode":
		b.handleModeCommand(ctx, message)
		return
	}

	b.handleURL(ctx, message, text)
}

func (b *Bot) handleCallbackQuery(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	answer := tgbotapi.CallbackConfig{CallbackQueryID: callback.ID}
	b.api.Request(answer)

	data := callback.Data
	chatID := callback.Message.Chat.ID

	if data == callbackDownloadBest {
		b.handleDownloadBest(ctx, chatID)
		return
	}
	if data == callbackModeAuto || data == callbackModeManual {
		b.handleModeSelected(ctx, callback, data)
		return
	}
	if strings.HasPrefix(data, "format_") {
		formatID := strings.TrimPrefix(data, "format_")
		b.handleFormatSelected(ctx, chatID, formatID)
		return
	}
}

func (b *Bot) handleStartCommand(ctx context.Context, message *tgbotapi.Message) {
	user, err := b.userService.GetOrCreate(ctx, message.From.ID, message.From.UserName)
	if err != nil {
		b.sendError(message.Chat.ID, fmt.Errorf("failed to initialize profile"))
		return
	}

	b.sendMessage(message.Chat.ID,
		"👋 Welcome to MediaHarvester Bot!\n\n"+
			"Send me a YouTube or TikTok link and I'll download the video for you.\n\n"+
			"Commands:\n"+
			"/mode — choose download mode\n"+
			"/limits — show your limits and usage")

	if user.AutoBestDownload == nil {
		b.sendModeSelection(message.Chat.ID, "Выберите режим скачивания:")
	}
}

func (b *Bot) handleModeCommand(ctx context.Context, message *tgbotapi.Message) {
	_, err := b.userService.GetOrCreate(ctx, message.From.ID, message.From.UserName)
	if err != nil {
		b.sendError(message.Chat.ID, fmt.Errorf("failed to load profile"))
		return
	}
	b.sendModeSelection(message.Chat.ID, "Текущий режим можно изменить в любой момент:")
}

func (b *Bot) handleModeSelected(ctx context.Context, callback *tgbotapi.CallbackQuery, selected string) {
	user, err := b.userService.GetOrCreate(ctx, callback.From.ID, callback.From.UserName)
	if err != nil {
		b.sendError(callback.Message.Chat.ID, fmt.Errorf("failed to update mode"))
		return
	}

	autoBest := selected == callbackModeAuto
	if err := b.userService.SetAutoBestDownload(ctx, user.ID, autoBest); err != nil {
		b.sendError(callback.Message.Chat.ID, fmt.Errorf("failed to save mode"))
		return
	}

	if autoBest {
		b.sendMessage(callback.Message.Chat.ID, "✅ Режим обновлён: автоматическая загрузка в лучшем качестве.")
	} else {
		b.sendMessage(callback.Message.Chat.ID, "✅ Режим обновлён: ручной выбор качества перед загрузкой.")
	}
}

func (b *Bot) handleLimitsCommand(ctx context.Context, message *tgbotapi.Message) {
	user, err := b.userService.GetOrCreate(ctx, message.From.ID, message.From.UserName)
	if err != nil {
		b.sendError(message.Chat.ID, fmt.Errorf("failed to load limits"))
		return
	}

	_, currentDaily, dailyLimit, err := b.limitSvc.CheckDailyLimit(ctx, user)
	if err != nil {
		b.sendError(message.Chat.ID, fmt.Errorf("failed to load daily limit"))
		return
	}
	_, currentMonthly, monthlyLimit, err := b.limitSvc.CheckMonthlyLimit(ctx, user)
	if err != nil {
		b.sendError(message.Chat.ID, fmt.Errorf("failed to load monthly limit"))
		return
	}

	b.sendMessage(message.Chat.ID, fmt.Sprintf("📊 Ваши лимиты:\n• Сегодня: %d/%d\n• За месяц: %d/%d", currentDaily, dailyLimit, currentMonthly, monthlyLimit))
}

func (b *Bot) handleURL(ctx context.Context, message *tgbotapi.Message, text string) {
	source, err := b.urlValidator.ValidateAndDetectSource(text)
	if err != nil {
		b.sendError(message.Chat.ID, err)
		return
	}

	user, err := b.userService.GetOrCreate(ctx, message.From.ID, message.From.UserName)
	if err != nil {
		log.Printf("Failed to get/create user: %v", err)
		b.sendError(message.Chat.ID, fmt.Errorf("failed to process request"))
		return
	}

	if user.AutoBestDownload == nil {
		b.sendModeSelection(message.Chat.ID, "Перед первой загрузкой выберите режим: автоматический или ручной.")
		return
	}

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

	dailyOk, currentDaily, limitDaily, err := b.limitSvc.CheckDailyLimit(ctx, user)
	if err != nil {
		log.Printf("Failed to check daily limit: %v", err)
		b.sendError(message.Chat.ID, fmt.Errorf("failed to check limit"))
		return
	}
	if !dailyOk {
		b.sendMessage(message.Chat.ID, fmt.Sprintf("⚠️ Daily limit exceeded. You've downloaded %d/%d videos today.", currentDaily, limitDaily))
		return
	}

	monthlyOk, currentMonthly, limitMonthly, err := b.limitSvc.CheckMonthlyLimit(ctx, user)
	if err != nil {
		log.Printf("Failed to check monthly limit: %v", err)
		b.sendError(message.Chat.ID, fmt.Errorf("failed to check limit"))
		return
	}
	if !monthlyOk {
		b.sendMessage(message.Chat.ID, fmt.Sprintf("⚠️ Monthly limit exceeded. You've downloaded %d/%d videos this month.", currentMonthly, limitMonthly))
		return
	}

	if user.AutoBestDownload != nil && *user.AutoBestDownload {
		b.sendMessage(message.Chat.ID, "⬇️ Downloading with best quality... Please wait.")
		b.queueDownload(ctx, message.Chat.ID, user.ID, text, string(source), "")
		return
	}

	b.pendingURLs[message.Chat.ID] = &pendingDownload{UserID: user.ID, ChatID: message.Chat.ID, URL: text, Source: string(source), Username: message.From.UserName}
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

	b.SendFormatSelection(message.Chat.ID, formats)
}

func (b *Bot) sendModeSelection(chatID int64, title string) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚡ Автоматически (лучшее качество)", callbackModeAuto),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎛️ Выбирать качество вручную", callbackModeManual),
		),
	)
	msg := tgbotapi.NewMessage(chatID, title)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) sendError(chatID int64, err error) {
	errorMsg := "❌ An error occurred: " + err.Error()
	if err == domain.ErrInvalidURL {
		errorMsg = "❌ Invalid URL. Please send a valid YouTube, TikTok, or Instagram Reels link."
	} else if err == domain.ErrUnsupportedSource {
		errorMsg = "❌ Unsupported source. Currently YouTube, TikTok, and Instagram Reels are supported."
	}
	b.sendMessage(chatID, errorMsg)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.api.Send(msg)
}

func (b *Bot) SendVideo(chatID int64, filePath string) error {
	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
	video.Caption = "🎬 Here's your video!"

	_, err := b.api.Send(video)
	if err != nil {
		return fmt.Errorf("failed to send video: %w", err)
	}
	if err := usecase.CleanupDownloadedFile(filePath); err != nil {
		log.Printf("Failed to delete file %s: %v", filePath, err)
	}
	return nil
}

func (b *Bot) SendFileTooLarge(chatID int64) {
	b.sendMessage(chatID, "❌ File is too large. Maximum size is 50MB.")
}
func (b *Bot) SendDownloadFailed(chatID int64) {
	b.sendMessage(chatID, "❌ Failed to download video. Please try another link.")
}

func (b *Bot) SendFormatSelection(chatID int64, formats []models.VideoFormat) error {
	maxFormats := 10
	if len(formats) < maxFormats {
		maxFormats = len(formats)
	}

	keyboard := make([][]tgbotapi.InlineKeyboardButton, 0)
	for i := 0; i < maxFormats; i += 2 {
		row := []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(formats[i].DisplayName, fmt.Sprintf("format_%s", formats[i].FormatID))}
		if i+1 < maxFormats {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(formats[i+1].DisplayName, fmt.Sprintf("format_%s", formats[i+1].FormatID)))
		}
		keyboard = append(keyboard, row)
	}
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("⚡ Best Quality (Auto)", callbackDownloadBest)})

	msg := tgbotapi.NewMessage(chatID, "📹 Select video quality:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	_, err := b.api.Send(msg)
	return err
}

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

func (b *Bot) queueDownload(ctx context.Context, chatID, userID int64, url, source, format string) {
	job := &models.DownloadJob{UserID: userID, ChatID: chatID, URL: url, Source: source, Format: format}
	if err := b.queueService.PushJob(ctx, job); err != nil {
		log.Printf("Failed to push job to queue: %v", err)
		b.sendError(chatID, fmt.Errorf("failed to queue download"))
		return
	}
	b.sendMessage(chatID, "⬇️ Downloading your video... Please wait.")
}
