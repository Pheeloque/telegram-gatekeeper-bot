package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"telegram-gatekeeper-bot/internal/storage"
)

func (h *Handler) groupsKeyboard(ctx context.Context, tg *bot.Bot, userID int64) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton
	for _, group := range h.store.GroupsSnapshot() {
		if !h.isAdmin(ctx, tg, userID, group.ID) {
			continue
		}
		rows = append(rows, []models.InlineKeyboardButton{{Text: group.Title, CallbackData: "group:" + strconv.FormatInt(group.ID, 10)}})
	}
	if len(rows) == 0 {
		rows = append(rows, []models.InlineKeyboardButton{{Text: "Нет доступных групп", CallbackData: "noop"}})
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (h *Handler) settingsKeyboard(groupID int64) *models.InlineKeyboardMarkup {
	id := strconv.FormatInt(groupID, 10)
	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "➕ Добавить канал", CallbackData: "add:" + id}},
		{{Text: "📋 Чёрный список", CallbackData: "list:" + id}},
		{{Text: "➖ Удалить канал", CallbackData: "remove_menu:" + id}},
		{{Text: "⬅ К группам", CallbackData: "back"}},
	}}
}

func (h *Handler) listText(groupID int64) string {
	channels := h.moderation.Channels(groupID)
	if len(channels) == 0 {
		return "Чёрный список пуст."
	}
	var b strings.Builder
	b.WriteString("Запрещённые каналы:\n\n")
	for i, channel := range channels {
		fmt.Fprintf(&b, "%d. %s", i+1, storage.DisplayChannel(channel))
		if channel.Title != "" {
			fmt.Fprintf(&b, " — %s", channel.Title)
		}
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}
