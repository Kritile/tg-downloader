package transport

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/bot/internal/worker"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

const (
	callbackDownloadBest = "download_best"
	callbackModeAuto     = "mode_auto"
	callbackModeManual   = "mode_manual"
)

type Message struct {
	Platform string
	UserID   int64
	ChatID   int64
	Username string
	Text     string
}
type Callback struct {
	Platform string
	ID       string
	UserID   int64
	ChatID   int64
	Username string
	Data     string
}
type Button struct {
	Text     string
	Callback string
}

type Messenger interface {
	SendText(context.Context, int64, string) error
	SendButtons(context.Context, int64, string, [][]Button) error
	AnswerCallback(context.Context, string) error
	SendVideo(context.Context, int64, string) error
}

type Bot struct {
	platform      string
	messenger     Messenger
	urlValidator  domain.URLValidator
	queueService  *worker.QueueService
	userService   domain.UserService
	permissionSvc domain.PermissionService
	limitSvc      domain.LimitService
	formatSvc     domain.FormatService
	pendingURLs   map[int64]*pendingDownload
	mu            sync.Mutex
}
type pendingDownload struct {
	UserID   int64
	ChatID   int64
	URL      string
	Source   string
	Username string
}

func NewBot(platform string, m Messenger, v domain.URLValidator, q *worker.QueueService, u domain.UserService, p domain.PermissionService, l domain.LimitService, f domain.FormatService, _ string) *Bot {
	return &Bot{platform: platform, messenger: m, urlValidator: v, queueService: q, userService: u, permissionSvc: p, limitSvc: l, formatSvc: f, pendingURLs: map[int64]*pendingDownload{}}
}

func (b *Bot) HandleMessage(ctx context.Context, message Message) {
	text := strings.TrimSpace(message.Text)
	if text == "" {
		return
	}
	switch text {
	case "/start":
		b.start(ctx, message)
	case "/limits":
		b.limits(ctx, message)
	case "/mode":
		b.mode(ctx, message)
	default:
		b.url(ctx, message)
	}
}
func (b *Bot) HandleCallback(ctx context.Context, callback Callback) {
	_ = b.messenger.AnswerCallback(ctx, callback.ID)
	switch {
	case callback.Data == callbackDownloadBest:
		b.selectFormat(ctx, callback.ChatID, "")
	case callback.Data == callbackModeAuto || callback.Data == callbackModeManual:
		b.modeSelected(ctx, callback, callback.Data)
	case strings.HasPrefix(callback.Data, "format_"):
		b.selectFormat(ctx, callback.ChatID, strings.TrimPrefix(callback.Data, "format_"))
	}
}

func (b *Bot) start(ctx context.Context, m Message) {
	user, err := b.userService.GetOrCreate(ctx, m.UserID, m.Username)
	if err != nil {
		b.error(ctx, m.ChatID, fmt.Errorf("failed to initialize profile"))
		return
	}
	b.text(ctx, m.ChatID, "👋 Welcome to MediaHarvester Bot!\n\nSend me a YouTube, TikTok, or Instagram link and I'll download the video for you.\n\nCommands:\n/mode — choose download mode\n/limits — show your limits and usage")
	if user.AutoBestDownload == nil {
		b.modeButtons(ctx, m.ChatID, "Choose download mode:")
	}
}
func (b *Bot) mode(ctx context.Context, m Message) {
	if _, err := b.userService.GetOrCreate(ctx, m.UserID, m.Username); err != nil {
		b.error(ctx, m.ChatID, err)
		return
	}
	b.modeButtons(ctx, m.ChatID, "You can change the download mode at any time:")
}
func (b *Bot) modeSelected(ctx context.Context, c Callback, selected string) {
	user, err := b.userService.GetOrCreate(ctx, c.UserID, c.Username)
	if err != nil {
		b.error(ctx, c.ChatID, err)
		return
	}
	if err = b.userService.SetAutoBestDownload(ctx, user.ID, selected == callbackModeAuto); err != nil {
		b.error(ctx, c.ChatID, err)
		return
	}
	if selected == callbackModeAuto {
		b.text(ctx, c.ChatID, "✅ Mode updated: automatic best quality.")
	} else {
		b.text(ctx, c.ChatID, "✅ Mode updated: choose quality manually.")
	}
}
func (b *Bot) limits(ctx context.Context, m Message) {
	user, err := b.userService.GetOrCreate(ctx, m.UserID, m.Username)
	if err != nil {
		b.error(ctx, m.ChatID, err)
		return
	}
	_, d, dl, err := b.limitSvc.CheckDailyLimit(ctx, user)
	if err != nil {
		b.error(ctx, m.ChatID, err)
		return
	}
	_, mo, ml, err := b.limitSvc.CheckMonthlyLimit(ctx, user)
	if err != nil {
		b.error(ctx, m.ChatID, err)
		return
	}
	b.text(ctx, m.ChatID, fmt.Sprintf("📊 Limits:\n• Today: %d/%d\n• This month: %d/%d", d, dl, mo, ml))
}
func (b *Bot) modeButtons(ctx context.Context, chatID int64, title string) {
	_ = b.messenger.SendButtons(ctx, chatID, title, [][]Button{{{Text: "⚡ Automatic (best quality)", Callback: callbackModeAuto}}, {{Text: "🎛️ Choose quality manually", Callback: callbackModeManual}}})
}

func (b *Bot) url(ctx context.Context, m Message) {
	source, err := b.urlValidator.ValidateAndDetectSource(m.Text)
	if err != nil {
		b.error(ctx, m.ChatID, err)
		return
	}
	user, err := b.userService.GetOrCreate(ctx, m.UserID, m.Username)
	if err != nil {
		b.error(ctx, m.ChatID, err)
		return
	}
	if user.AutoBestDownload == nil {
		b.modeButtons(ctx, m.ChatID, "Choose automatic or manual quality before the first download.")
		return
	}
	ok, err := b.permissionSvc.CheckPermission(ctx, user, source)
	if err != nil {
		b.error(ctx, m.ChatID, err)
		return
	}
	if !ok {
		b.text(ctx, m.ChatID, "❌ You don't have permission to download from this source.")
		return
	}
	daily, current, limit, err := b.limitSvc.CheckDailyLimit(ctx, user)
	if err != nil {
		b.error(ctx, m.ChatID, err)
		return
	}
	if !daily {
		b.text(ctx, m.ChatID, fmt.Sprintf("⚠️ Daily limit exceeded: %d/%d.", current, limit))
		return
	}
	monthly, current, limit, err := b.limitSvc.CheckMonthlyLimit(ctx, user)
	if err != nil {
		b.error(ctx, m.ChatID, err)
		return
	}
	if !monthly {
		b.text(ctx, m.ChatID, fmt.Sprintf("⚠️ Monthly limit exceeded: %d/%d.", current, limit))
		return
	}
	if source == models.SourceReels || *user.AutoBestDownload {
		b.queue(ctx, m, string(source), "")
		return
	}
	b.mu.Lock()
	b.pendingURLs[m.ChatID] = &pendingDownload{UserID: user.ID, ChatID: m.ChatID, URL: m.Text, Source: string(source), Username: m.Username}
	b.mu.Unlock()
	b.text(ctx, m.ChatID, "🔍 Fetching available formats... Please wait.")
	formats, err := b.formatSvc.ListFormats(ctx, m.Text, source)
	if err != nil || len(formats) == 0 {
		b.text(ctx, m.ChatID, "⚠️ Could not fetch formats. Downloading with best quality...")
		b.queue(ctx, m, string(source), "")
		return
	}
	_ = b.SendFormatSelection(ctx, m.ChatID, formats)
}

func (b *Bot) SendFormatSelection(ctx context.Context, chatID int64, formats []models.VideoFormat) error {
	max := len(formats)
	if max > 10 {
		max = 10
	}
	rows := make([][]Button, 0)
	for i := 0; i < max; i += 2 {
		row := []Button{{Text: formats[i].DisplayName, Callback: "format_" + formats[i].FormatID}}
		if i+1 < max {
			row = append(row, Button{Text: formats[i+1].DisplayName, Callback: "format_" + formats[i+1].FormatID})
		}
		rows = append(rows, row)
	}
	rows = append(rows, []Button{{Text: "⚡ Best Quality (Auto)", Callback: callbackDownloadBest}})
	return b.messenger.SendButtons(ctx, chatID, "📹 Select video quality:", rows)
}
func (b *Bot) selectFormat(ctx context.Context, chatID int64, format string) {
	b.mu.Lock()
	p, ok := b.pendingURLs[chatID]
	if ok {
		delete(b.pendingURLs, chatID)
	}
	b.mu.Unlock()
	if !ok {
		b.text(ctx, chatID, "❌ Session expired. Please send the URL again.")
		return
	}
	b.queue(ctx, Message{UserID: p.UserID, ChatID: p.ChatID, Username: p.Username, Text: p.URL}, p.Source, format)
}
func (b *Bot) queue(ctx context.Context, m Message, source, format string) {
	if err := b.queueService.PushJob(ctx, &models.DownloadJob{Platform: b.platform, UserID: m.UserID, ChatID: m.ChatID, URL: m.Text, Source: source, Username: m.Username, Format: format}); err != nil {
		b.error(ctx, m.ChatID, err)
		return
	}
	b.text(ctx, m.ChatID, "⬇️ Downloading your video... Please wait.")
}
func (b *Bot) text(ctx context.Context, chatID int64, s string) {
	if err := b.messenger.SendText(ctx, chatID, s); err != nil {
		log.Printf("send message: %v", err)
	}
}
func (b *Bot) error(ctx context.Context, chatID int64, err error) {
	text := "❌ An error occurred: " + err.Error()
	if err == domain.ErrInvalidURL {
		text = "❌ Invalid URL. Send a valid YouTube, TikTok, or Instagram link."
	}
	b.text(ctx, chatID, text)
}
func (b *Bot) SendVideo(platform string, chatID int64, path string) error {
	if platform != b.platform {
		return fmt.Errorf("platform %s is not handled by %s transport", platform, b.platform)
	}
	return b.messenger.SendVideo(context.Background(), chatID, path)
}
func (b *Bot) SendFileTooLarge(platform string, chatID int64) {
	if platform == b.platform {
		message := "❌ File is too large. Maximum size is 50MB."
		if platform == "telegram" {
			message = "❌ File is too large. Telegram Local Bot API supports files up to 2000MB."
		}
		b.text(context.Background(), chatID, message)
	}
}
func (b *Bot) SendDownloadFailed(platform string, chatID int64) {
	if platform == b.platform {
		b.text(context.Background(), chatID, "❌ Failed to download video. Please try another link.")
	}
}

type NotifierRouter struct{ bots map[string]*Bot }

func NewNotifierRouter(bots ...*Bot) *NotifierRouter {
	result := &NotifierRouter{bots: map[string]*Bot{}}
	for _, bot := range bots {
		if bot != nil {
			result.bots[bot.platform] = bot
		}
	}
	return result
}
func (r *NotifierRouter) SendVideo(platform string, chatID int64, path string) error {
	bot := r.bots[platform]
	if bot == nil {
		return fmt.Errorf("no notifier for platform %s", platform)
	}
	return bot.SendVideo(platform, chatID, path)
}
func (r *NotifierRouter) SendFileTooLarge(platform string, chatID int64) {
	if bot := r.bots[platform]; bot != nil {
		bot.SendFileTooLarge(platform, chatID)
	}
}
func (r *NotifierRouter) SendDownloadFailed(platform string, chatID int64) {
	if bot := r.bots[platform]; bot != nil {
		bot.SendDownloadFailed(platform, chatID)
	}
}
