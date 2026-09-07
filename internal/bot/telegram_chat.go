package bot

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"telegram-gatekeeper-bot/internal/storage"
)

type telegramChat struct {
	tg *bot.Bot
}

func NewChatService(tg *bot.Bot) ChatService {
	return &telegramChat{tg: tg}
}

func (c *telegramChat) IsAdmin(ctx context.Context, chatID, userID int64) bool {
	member, err := c.tg.GetChatMember(ctx, &bot.GetChatMemberParams{ChatID: chatID, UserID: userID})
	if err != nil {
		return false
	}
	return isAdminMember(member)
}

func (c *telegramChat) ResolveChannel(ctx context.Context, username string) (storage.Channel, bool) {
	return c.resolveChat(ctx, username)
}

func (c *telegramChat) ResolveChannelByID(ctx context.Context, id int64) (storage.Channel, bool) {
	return c.resolveChat(ctx, id)
}

func (c *telegramChat) resolveChat(ctx context.Context, chatID any) (storage.Channel, bool) {
	chat, err := c.tg.GetChat(ctx, &bot.GetChatParams{ChatID: chatID})
	if err != nil || chat.Type != models.ChatTypeChannel {
		return storage.Channel{}, false
	}
	return storage.Channel{ID: chat.ID, Username: chat.Username, Title: chat.Title}, true
}

func isAdminMember(member *models.ChatMember) bool {
	switch member.Type {
	case models.ChatMemberTypeOwner, models.ChatMemberTypeAdministrator:
		return true
	}
	return false
}
