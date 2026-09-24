package transport

import (
	"context"
	"log"
	"os"

	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	maxmodel "github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/mediaharvester/tg-downloader/bot/internal/usecase"
)

type MaxTransport struct{ api *maxbot.Api }

func NewMaxTransport(api *maxbot.Api) *MaxTransport { return &MaxTransport{api: api} }
func (m *MaxTransport) SendText(ctx context.Context, chatID int64, text string) (MessageRef, error) {
	result, err := m.api.Messages.Send(ctx, maxbot.NewMessage().SetChat(chatID).SetText(text))
	if err != nil { return "", err }
	return MessageRef(result.Message.Body.Mid), nil
}
func (m *MaxTransport) SendButtons(ctx context.Context, chatID int64, text string, rows [][]Button) (MessageRef, error) {
	keyboard := maxmodel.NewKeyboard()
	for _, row := range rows {
		r := keyboard.AddRow()
		for _, button := range row {
			r.AddCallback(button.Text, maxmodel.IntentDefault, button.Callback)
		}
	}
	result, err := m.api.Messages.Send(ctx, maxbot.NewMessage().SetChat(chatID).SetText(text).AddKeyboard(keyboard))
	if err != nil { return "", err }
	return MessageRef(result.Message.Body.Mid), nil
}
func (m *MaxTransport) DeleteMessage(ctx context.Context, _ int64, ref MessageRef) error {
	_, err := m.api.Messages.DeleteMessage(ctx, string(ref))
	return err
}
func (m *MaxTransport) AnswerCallback(ctx context.Context, id string) error {
	_, err := m.api.Messages.AnswerOnCallback(ctx, id, maxmodel.CallbackAnswer{})
	return err
}
func (m *MaxTransport) SendVideo(ctx context.Context, chatID int64, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	token, err := m.api.Upload.Upload(ctx, maxmodel.UploadVideo, file, info.Name(), info.Size())
	if err == nil {
		_, err = m.api.Messages.Send(ctx, maxbot.NewMessage().SetChat(chatID).SetText("🎬 Here's your video!").AddAttachByToken(token, maxmodel.AttachVideo))
	}
	if cleanupErr := usecase.CleanupDownloadedFile(path); cleanupErr != nil {
		log.Printf("cleanup %s: %v", path, cleanupErr)
	}
	return err
}
func (m *MaxTransport) Start(ctx context.Context, bot *Bot) error {
	var marker int64
	for {
		updates, next, err := m.api.Subscriptions.GetUpdates(ctx, marker)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("MAX updates: %v", err)
			continue
		}
		marker = next
		for _, update := range updates {
			if update.UpdateType == maxmodel.UpdateMessageCreated || update.UpdateType == maxmodel.UpdateMessageEdited {
				msg := update.GetMessage()
				bot.HandleMessage(ctx, Message{Platform: "max", UserID: update.UserID, ChatID: update.ChatID, Username: update.User.Username, Text: msg.Body.Text})
			}
			if update.UpdateType == maxmodel.UpdateMessageCallback && update.Callback != nil {
				callback := update.Callback
				bot.HandleCallback(ctx, Callback{Platform: "max", ID: callback.CallbackID, UserID: update.UserID, ChatID: update.ChatID, Data: callback.Payload})
			}
		}
	}
}
