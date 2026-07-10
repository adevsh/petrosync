package bot

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/adevsh/petrosync/internal/service"
	"github.com/adevsh/petrosync/internal/telegram"
)

type Replier interface {
	SendMessage(ctx context.Context, chatID int64, text string) (int64, error)
}

func HandleUpdate(ctx context.Context, upd telegram.Update, linker *service.TelegramLinkService, replier Replier) error {
	if upd.Message == nil || upd.Message.From == nil {
		return nil
	}

	text := strings.TrimSpace(upd.Message.Text)
	if text == "" {
		return nil
	}

	fields := strings.Fields(text)
	if len(fields) == 0 {
		return nil
	}

	cmd := fields[0]
	if !strings.HasPrefix(cmd, "/link") {
		return nil
	}
	if cmd != "/link" && !strings.HasPrefix(cmd, "/link@") {
		return nil
	}

	chatID := upd.Message.Chat.ID
	fromID := upd.Message.From.ID

	if len(fields) < 2 {
		if replier != nil {
			_, _ = replier.SendMessage(ctx, chatID, "Usage: /link <token>")
		}
		return nil
	}

	token := strings.TrimSpace(fields[1])
	if len(token) != 64 {
		if replier != nil {
			_, _ = replier.SendMessage(ctx, chatID, "Invalid token format. Usage: /link <token>")
		}
		return nil
	}

	row, err := linker.LinkByToken(ctx, fromID, token)
	if err != nil {
		msg := "Linking failed. Please request a new token."
		if errors.Is(err, service.ErrInvalidOrExpiredLinkToken) {
			msg = "Invalid or expired token. Please request a new token."
		} else if errors.Is(err, service.ErrUserAlreadyLinked) {
			msg = "This user is already linked."
		} else if errors.Is(err, service.ErrTelegramAlreadyLinked) {
			msg = "This Telegram account is already linked to another user."
		}
		if replier != nil {
			_, _ = replier.SendMessage(ctx, chatID, msg)
		}
		return nil
	}

	_ = row
	if replier != nil {
		_, _ = replier.SendMessage(ctx, chatID, "Linked successfully.")
	}
	return nil
}

type TelegramBot struct {
	client *telegram.Client
	linker *service.TelegramLinkService
}

func NewTelegramBot(client *telegram.Client, linker *service.TelegramLinkService) *TelegramBot {
	return &TelegramBot{client: client, linker: linker}
}

func (b *TelegramBot) Run(ctx context.Context) error {
	var offset int64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		updates, err := b.client.GetUpdates(ctx, offset, 30)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		for _, upd := range updates {
			if upd.UpdateID >= offset {
				offset = upd.UpdateID + 1
			}
			_ = HandleUpdate(ctx, upd, b.linker, b.client)
		}
	}
}
