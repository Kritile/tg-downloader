package transport

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mediaharvester/tg-downloader/bot/internal/usecase"
)

type TelegramTransport struct{ api *tgbotapi.BotAPI }

func NewTelegramTransport(api *tgbotapi.BotAPI) *TelegramTransport {
	return &TelegramTransport{api: api}
}
func (t *TelegramTransport) SendText(_ context.Context, chatID int64, text string) error {
	_, err := t.api.Send(tgbotapi.NewMessage(chatID, text))
	return err
}
func (t *TelegramTransport) SendButtons(_ context.Context, chatID int64, text string, rows [][]Button) error {
	keyboard := make([][]tgbotapi.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		out := make([]tgbotapi.InlineKeyboardButton, 0, len(row))
		for _, button := range row {
			out = append(out, tgbotapi.NewInlineKeyboardButtonData(button.Text, button.Callback))
		}
		keyboard = append(keyboard, out)
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	_, err := t.api.Send(msg)
	return err
}
func (t *TelegramTransport) AnswerCallback(_ context.Context, id string) error {
	_, err := t.api.Request(tgbotapi.CallbackConfig{CallbackQueryID: id})
	return err
}
func (t *TelegramTransport) SendVideo(_ context.Context, chatID int64, path string) error {
	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(path))
	video.Caption = "🎬 Here's your video!"
	_, err := t.api.Send(video)
	if cleanupErr := usecase.CleanupDownloadedFile(path); cleanupErr != nil {
		log.Printf("cleanup %s: %v", path, cleanupErr)
	}
	if err != nil {
		return fmt.Errorf("failed to send video: %w", err)
	}
	return nil
}
func (t *TelegramTransport) Start(ctx context.Context, bot *Bot) error {
	updates := t.api.GetUpdatesChan(tgbotapi.UpdateConfig{Offset: 0, Timeout: 60})
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			if update.Message != nil {
				bot.HandleMessage(ctx, Message{Platform: "telegram", UserID: update.Message.From.ID, ChatID: update.Message.Chat.ID, Username: update.Message.From.UserName, Text: update.Message.Text})
			}
			if update.CallbackQuery != nil {
				bot.HandleCallback(ctx, Callback{Platform: "telegram", ID: update.CallbackQuery.ID, UserID: update.CallbackQuery.From.ID, ChatID: update.CallbackQuery.Message.Chat.ID, Username: update.CallbackQuery.From.UserName, Data: update.CallbackQuery.Data})
			}
		}
	}
}
