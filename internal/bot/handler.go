package bot

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"telegram-gatekeeper-bot/internal/storage"
)

// Moderation describes the application logic the handler uses.
type Moderation interface {
	UpsertGroup(id int64, title string) error
	Groups() []storage.Group
	GroupTitle(id int64) (string, bool)
	IsForbidden(groupID, channelID int64) bool
	AddChannel(groupID int64, channel storage.Channel) error
	RemoveChannel(groupID, channelID int64) error
	Channels(groupID int64) []storage.Channel
}

// ChatService resolves group membership and channels through Telegram.
type ChatService interface {
	IsAdmin(ctx context.Context, chatID, userID int64) bool
	ResolveChannel(ctx context.Context, identifier string) (storage.Channel, bool)
}

type Handler struct {
	moderation Moderation
	chat       ChatService
	session    *Session
}

func NewHandler(moderation Moderation, chat ChatService) *Handler {
	return &Handler{moderation: moderation, chat: chat, session: newSession()}
}

func (h *Handler) Handle(ctx context.Context, tg *bot.Bot, update *models.Update) {
	if update.MyChatMember != nil && isGroupChat(update.MyChatMember.Chat.Type) {
		chat := update.MyChatMember.Chat
		if err := h.moderation.UpsertGroup(chat.ID, chat.Title); err != nil {
			log.Printf("store group from my_chat_member: %v", err)
		}
	}

	switch {
	case update.CallbackQuery != nil:
		h.handleCallback(ctx, tg, update.CallbackQuery)
	case update.Message != nil:
		h.handleMessage(ctx, tg, update.Message)
	}
}

func (h *Handler) handleMessage(ctx context.Context, tg *bot.Bot, msg *models.Message) {
	if isGroupChat(msg.Chat.Type) {
		if err := h.moderation.UpsertGroup(msg.Chat.ID, msg.Chat.Title); err != nil {
			log.Printf("store group: %v", err)
		}
		if msg.From != nil && msg.From.IsBot {
			return
		}
		h.handleModeration(ctx, tg, msg)
		return
	}

	if msg.Chat.Type == models.ChatTypePrivate && msg.From != nil {
		h.handlePrivateMessage(ctx, tg, msg)
	}
}

func (h *Handler) handleModeration(ctx context.Context, tg *bot.Bot, msg *models.Message) {
	if msg.ForwardOrigin == nil {
		return
	}
	channelID, ok := forwardChannelID(msg.ForwardOrigin)
	if !ok || !h.moderation.IsForbidden(msg.Chat.ID, channelID) {
		return
	}

	if _, err := tg.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID}); err != nil {
		log.Printf("delete message chat=%d message=%d: %v", msg.Chat.ID, msg.ID, err)
		return
	}
	log.Printf("deleted forwarded message chat=%d message=%d from channel=%d", msg.Chat.ID, msg.ID, channelID)

	h.notifyDeleted(ctx, tg, msg)
}

func (h *Handler) notifyDeleted(ctx context.Context, tg *bot.Bot, msg *models.Message) {
	var b strings.Builder
	b.WriteString("⚠️ Сообщение из недопустимого канала было удалено.")
	if channel := forwardChannelName(msg.ForwardOrigin); channel != "" {
		b.WriteString("\n📢 Канал: ")
		b.WriteString(channel)
	}
	if msg.From != nil {
		b.WriteString("\n👤 Отправитель: ")
		b.WriteString(userDisplayName(msg.From))
	}
	b.WriteString("\n\nПересылки из этого канала в группе запрещены.")
	if _, err := tg.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: b.String()}); err != nil {
		log.Printf("send moderation notice chat=%d: %v", msg.Chat.ID, err)
	}
}

func (h *Handler) handlePrivateMessage(ctx context.Context, tg *bot.Bot, msg *models.Message) {
	userID := msg.From.ID
	if command, ok := parseCommand(msg.Text); ok {
		switch command {
		case "start", "help", "groups":
			h.session.SetAwaiting(userID, InputNone)
			h.sendPrivate(ctx, tg, userID, "Выберите группу, которой хотите управлять:", h.groupsKeyboard(ctx, tg, userID))
			return
		}
	}

	switch h.session.GetAwaiting(userID) {
	case InputAddChannel:
		h.processAddChannel(ctx, tg, msg)
	case InputRemoveChannel:
		h.processRemoveChannel(ctx, tg, msg)
	default:
		h.sendPrivate(ctx, tg, userID, "Используйте /start или /groups, чтобы открыть настройки.", nil)
	}
}

func (h *Handler) handleCallback(ctx context.Context, tg *bot.Bot, q *models.CallbackQuery) {
	log.Printf("callback query: user=%d data=%q", q.From.ID, q.Data)
	h.answerCallback(ctx, tg, q.ID)
	if q.Data == "" {
		return
	}
	userID := q.From.ID

	switch {
	case q.Data == "back":
		h.session.SetAwaiting(userID, InputNone)
		h.editCallback(ctx, tg, q, "Выберите группу:", h.groupsKeyboard(ctx, tg, userID))
	case strings.HasPrefix(q.Data, "group:"):
		h.openGroup(ctx, tg, q, userID)
	case strings.HasPrefix(q.Data, "add:"):
		h.startAddChannel(ctx, tg, q, userID)
	case strings.HasPrefix(q.Data, "remove:"):
		h.removeChannelByCallback(ctx, tg, q, userID)
	case strings.HasPrefix(q.Data, "list:"):
		h.showList(ctx, tg, q, userID)
	case strings.HasPrefix(q.Data, "remove_menu:"):
		h.showRemoveMenu(ctx, tg, q, userID)
	}
}

func (h *Handler) openGroup(ctx context.Context, tg *bot.Bot, q *models.CallbackQuery, userID int64) {
	groupID, err := parseCallbackID(q.Data, "group:")
	if err != nil {
		h.editCallback(ctx, tg, q, "Некорректный идентификатор группы.", nil)
		return
	}
	if !h.chat.IsAdmin(ctx, groupID, userID) {
		h.editCallback(ctx, tg, q, "У вас нет прав администратора в этой группе.", nil)
		return
	}
	h.selectGroup(userID, groupID)
	h.editCallback(ctx, tg, q, "Группа: "+h.groupTitle(groupID), h.settingsKeyboard(groupID))
}

func (h *Handler) startAddChannel(ctx context.Context, tg *bot.Bot, q *models.CallbackQuery, userID int64) {
	groupID, err := parseCallbackID(q.Data, "add:")
	if err != nil || !h.selectAndVerify(ctx, userID, groupID) {
		h.editCallback(ctx, tg, q, "Нет доступа к этой группе.", nil)
		return
	}
	h.session.SetAwaiting(userID, InputAddChannel)
	h.editCallback(ctx, tg, q, "Пришлите @username публичного канала, например @example_channel", nil)
}

func (h *Handler) removeChannelByCallback(ctx context.Context, tg *bot.Bot, q *models.CallbackQuery, userID int64) {
	channelID, groupID, ok := parseRemoveCallback(q.Data)
	if !ok || !h.selectAndVerify(ctx, userID, groupID) {
		h.editCallback(ctx, tg, q, "Нет доступа к этой группе.", nil)
		return
	}
	if err := h.moderation.RemoveChannel(groupID, channelID); err != nil {
		h.editCallback(ctx, tg, q, "Не удалось удалить канал.", nil)
		return
	}
	h.editCallback(ctx, tg, q, "Канал удалён из чёрного списка.", h.settingsKeyboard(groupID))
}

func (h *Handler) showList(ctx context.Context, tg *bot.Bot, q *models.CallbackQuery, userID int64) {
	groupID, err := parseCallbackID(q.Data, "list:")
	if err != nil || !h.selectAndVerify(ctx, userID, groupID) {
		h.editCallback(ctx, tg, q, "Нет доступа к этой группе.", nil)
		return
	}
	h.editCallback(ctx, tg, q, h.listText(groupID), h.settingsKeyboard(groupID))
}

func (h *Handler) showRemoveMenu(ctx context.Context, tg *bot.Bot, q *models.CallbackQuery, userID int64) {
	groupID, err := parseCallbackID(q.Data, "remove_menu:")
	if err != nil || !h.selectAndVerify(ctx, userID, groupID) {
		h.editCallback(ctx, tg, q, "Нет доступа к этой группе.", nil)
		return
	}
	channels := h.moderation.Channels(groupID)
	if len(channels) == 0 {
		h.editCallback(ctx, tg, q, "Чёрный список пуст.", h.settingsKeyboard(groupID))
		return
	}
	rows := make([][]models.InlineKeyboardButton, 0, len(channels)+1)
	for _, channel := range channels {
		rows = append(rows, []models.InlineKeyboardButton{{
			Text:         "❌ " + storage.DisplayChannel(channel),
			CallbackData: removeCallbackID(channel.ID, groupID),
		}})
	}
	rows = append(rows, []models.InlineKeyboardButton{{Text: "⬅ Назад", CallbackData: "group:" + strconv.FormatInt(groupID, 10)}})
	h.editCallback(ctx, tg, q, "Выберите канал для удаления:", &models.InlineKeyboardMarkup{InlineKeyboard: rows})
}

func (h *Handler) processAddChannel(ctx context.Context, tg *bot.Bot, msg *models.Message) {
	userID := msg.From.ID
	groupID, ok := h.session.GetSelectedGroup(userID)
	if !ok || !h.chat.IsAdmin(ctx, groupID, userID) {
		h.session.SetAwaiting(userID, InputNone)
		h.sendPrivate(ctx, tg, userID, "Сначала выберите группу через /start и убедитесь, что вы её администратор.", nil)
		return
	}
	username := normalizeChannelUsername(msg.Text)
	if username == "" {
		h.sendPrivate(ctx, tg, userID, "Нужен публичный канал в формате @channel_username.", nil)
		return
	}
	channel, found := h.chat.ResolveChannel(ctx, "@"+username)
	if !found {
		h.sendPrivate(ctx, tg, userID, "Не удалось найти публичный канал по этому username.", nil)
		return
	}
	if err := h.moderation.AddChannel(groupID, channel); err != nil {
		h.sendPrivate(ctx, tg, userID, "Не удалось сохранить канал.", nil)
		return
	}
	h.session.SetAwaiting(userID, InputNone)
	h.sendPrivate(ctx, tg, userID, "Канал добавлен в чёрный список: "+storage.DisplayChannel(channel), h.settingsKeyboard(groupID))
}

func (h *Handler) processRemoveChannel(ctx context.Context, tg *bot.Bot, msg *models.Message) {
	userID := msg.From.ID
	groupID, ok := h.session.GetSelectedGroup(userID)
	if !ok || !h.chat.IsAdmin(ctx, groupID, userID) {
		h.session.SetAwaiting(userID, InputNone)
		h.sendPrivate(ctx, tg, userID, "У вас больше нет прав администратора этой группы.", nil)
		return
	}
	channel, found := h.chat.ResolveChannel(ctx, strings.TrimSpace(msg.Text))
	if !found {
		h.sendPrivate(ctx, tg, userID, "Пришлите @username канала, который нужно удалить из списка.", nil)
		return
	}
	if err := h.moderation.RemoveChannel(groupID, channel.ID); err != nil {
		h.sendPrivate(ctx, tg, userID, "Этот канал не найден в чёрном списке.", h.settingsKeyboard(groupID))
		return
	}
	h.session.SetAwaiting(userID, InputNone)
	h.sendPrivate(ctx, tg, userID, "Канал удалён из чёрного списка.", h.settingsKeyboard(groupID))
}

func (h *Handler) selectGroup(userID, groupID int64) {
	h.session.SetSelectedGroup(userID, groupID)
	h.session.SetAwaiting(userID, InputNone)
}

func (h *Handler) selectAndVerify(ctx context.Context, userID, groupID int64) bool {
	if !h.chat.IsAdmin(ctx, groupID, userID) {
		return false
	}
	h.selectGroup(userID, groupID)
	return true
}

func (h *Handler) groupTitle(groupID int64) string {
	if title, ok := h.moderation.GroupTitle(groupID); ok {
		return title
	}
	return strconv.FormatInt(groupID, 10)
}
