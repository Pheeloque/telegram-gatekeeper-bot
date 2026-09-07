package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"telegram-gatekeeper-bot/internal/moderation"
	"telegram-gatekeeper-bot/internal/storage"
)

type Handler struct {
	store      *storage.Store
	moderation *moderation.Service
	session    *Session
	api        *apiClient
}

func NewHandler(store *storage.Store, moderationService *moderation.Service, api *apiClient) *Handler {
	return &Handler{store: store, moderation: moderationService, session: newSession(), api: api}
}

func (h *Handler) Handle(ctx context.Context, tg *bot.Bot, update *models.Update) {
	if update.MyChatMember != nil && isGroupChat(update.MyChatMember.Chat.Type) {
		chat := update.MyChatMember.Chat
		if err := h.store.UpsertGroup(chat.ID, chat.Title); err != nil {
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
		if err := h.store.UpsertGroup(msg.Chat.ID, msg.Chat.Title); err != nil {
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
	channelID, ok := extractForwardedChannelID(msg.ForwardOrigin)
	if !ok || !h.moderation.IsForbidden(msg.Chat.ID, channelID) {
		return
	}

	if _, err := tg.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID}); err != nil {
		log.Printf("delete message chat=%d message=%d: %v", msg.Chat.ID, msg.ID, err)
		return
	}
	log.Printf("deleted forwarded message chat=%d message=%d from channel=%d", msg.Chat.ID, msg.ID, channelID)
}

func extractForwardedChannelID(origin *models.MessageOrigin) (int64, bool) {
	raw, err := json.Marshal(origin)
	if err != nil {
		return 0, false
	}
	var value struct {
		Type string `json:"type"`
		Chat *struct {
			ID int64 `json:"id"`
		} `json:"chat,omitempty"`
	}
	if err := json.Unmarshal(raw, &value); err != nil || value.Type != "channel" || value.Chat == nil || value.Chat.ID == 0 {
		return 0, false
	}
	return value.Chat.ID, true
}

func (h *Handler) handlePrivateMessage(ctx context.Context, tg *bot.Bot, msg *models.Message) {
	userID := msg.From.ID
	if command, ok := parseCommand(msg.Text); ok {
		switch command {
		case "start", "help", "groups":
			h.session.SetAwaiting(userID, InputNone)
			h.sendPrivate(ctx, tg, userID, "Выберите группу, которой хотите управлять:", h.groupsKeyboard(ctx, userID))
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
	_ = h.answerCallback(ctx, tg, q.ID)
	if q.Data == "" {
		return
	}
	userID := q.From.ID

	switch {
	case q.Data == "back":
		h.session.SetAwaiting(userID, InputNone)
		h.editCallback(ctx, tg, q, "Выберите группу:", h.groupsKeyboard(ctx, userID))
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
	if !h.isSelectedAdmin(ctx, userID, groupID) && !h.isAdmin(ctx, userID, groupID) {
		h.editCallback(ctx, tg, q, "У вас нет прав администратора в этой группе.", nil)
		return
	}
	h.session.SetSelectedGroup(userID, groupID)
	h.session.SetAwaiting(userID, InputNone)
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
		rows = append(rows, []models.InlineKeyboardButton{{Text: "❌ " + storage.DisplayChannel(channel), CallbackData: fmt.Sprintf("remove:%d:%d", channel.ID, groupID)}})
	}
	rows = append(rows, []models.InlineKeyboardButton{{Text: "⬅ Назад", CallbackData: fmt.Sprintf("group:%d", groupID)}})
	h.editCallback(ctx, tg, q, "Выберите канал для удаления:", &models.InlineKeyboardMarkup{InlineKeyboard: rows})
}

func (h *Handler) processAddChannel(ctx context.Context, tg *bot.Bot, msg *models.Message) {
	userID := msg.From.ID
	groupID, ok := h.session.GetSelectedGroup(userID)
	if !ok || !h.isAdmin(ctx, userID, groupID) {
		h.session.SetAwaiting(userID, InputNone)
		h.sendPrivate(ctx, tg, userID, "Сначала выберите группу через /start и убедитесь, что вы её администратор.", nil)
		return
	}
	username := normalizeChannelUsername(msg.Text)
	if username == "" {
		h.sendPrivate(ctx, tg, userID, "Нужен публичный канал в формате @channel_username.", nil)
		return
	}
	chat, err := h.api.GetChat(ctx, "@"+username)
	if err != nil || chat.Type != "channel" {
		h.sendPrivate(ctx, tg, userID, "Не удалось найти публичный канал по этому username.", nil)
		return
	}
	channel := storage.Channel{ID: chat.ID, Username: chat.Username, Title: chat.Title}
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
	if !ok || !h.isAdmin(ctx, userID, groupID) {
		h.session.SetAwaiting(userID, InputNone)
		h.sendPrivate(ctx, tg, userID, "У вас больше нет прав администратора этой группы.", nil)
		return
	}
	chat, err := h.api.GetChat(ctx, strings.TrimSpace(msg.Text))
	if err != nil || chat.Type != "channel" {
		h.sendPrivate(ctx, tg, userID, "Пришлите @username канала, который нужно удалить из списка.", nil)
		return
	}
	if err := h.moderation.RemoveChannel(groupID, chat.ID); err != nil {
		h.sendPrivate(ctx, tg, userID, "Этот канал не найден в чёрном списке.", h.settingsKeyboard(groupID))
		return
	}
	h.session.SetAwaiting(userID, InputNone)
	h.sendPrivate(ctx, tg, userID, "Канал удалён из чёрного списка.", h.settingsKeyboard(groupID))
}

func (h *Handler) groupsKeyboard(ctx context.Context, userID int64) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton
	for _, group := range h.store.GroupsSnapshot() {
		isAdmin, err := h.api.IsAdmin(ctx, group.ID, userID)
		if err != nil || !isAdmin {
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

func (h *Handler) sendPrivate(ctx context.Context, tg *bot.Bot, userID int64, text string, markup *models.InlineKeyboardMarkup) {
	params := &bot.SendMessageParams{ChatID: userID, Text: text}
	if markup != nil {
		params.ReplyMarkup = markup
	}
	if _, err := tg.SendMessage(ctx, params); err != nil {
		log.Printf("send private message: %v", err)
	}
}

func (h *Handler) editCallback(ctx context.Context, tg *bot.Bot, q *models.CallbackQuery, text string, markup *models.InlineKeyboardMarkup) {
	if q.Message.Message == nil {
		log.Printf("callback message is inaccessible: data=%q", q.Data)
		h.sendPrivate(ctx, tg, q.From.ID, text, markup)
		return
	}

	if _, err := tg.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      q.Message.Message.Chat.ID,
		MessageID:   q.Message.Message.ID,
		Text:        text,
		ReplyMarkup: markup,
	}); err != nil {
		log.Printf("edit callback message: %v", err)
		h.sendPrivate(ctx, tg, q.From.ID, text, markup)
	}
}

func (h *Handler) answerCallback(ctx context.Context, tg *bot.Bot, id string) error {
	_, err := tg.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: id})
	if err != nil {
		log.Printf("answer callback: %v", err)
	}
	return err
}

func (h *Handler) isAdmin(ctx context.Context, userID, groupID int64) bool {
	ok, err := h.api.IsAdmin(ctx, groupID, userID)
	return err == nil && ok
}

func (h *Handler) selectAndVerify(ctx context.Context, userID, groupID int64) bool {
	if !h.isAdmin(ctx, userID, groupID) {
		return false
	}
	h.session.SetSelectedGroup(userID, groupID)
	return true
}

func (h *Handler) isSelectedAdmin(ctx context.Context, userID, groupID int64) bool {
	selected, ok := h.session.GetSelectedGroup(userID)
	return ok && selected == groupID && h.isAdmin(ctx, userID, groupID)
}

func (h *Handler) groupTitle(groupID int64) string {
	for _, group := range h.store.GroupsSnapshot() {
		if group.ID == groupID {
			return group.Title
		}
	}
	return strconv.FormatInt(groupID, 10)
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

func isGroupChat(chatType models.ChatType) bool {
	return chatType == models.ChatTypeGroup || chatType == models.ChatTypeSupergroup
}

func parseCallbackID(data, prefix string) (int64, error) {
	return strconv.ParseInt(strings.TrimPrefix(data, prefix), 10, 64)
}

func parseRemoveCallback(data string) (channelID, groupID int64, ok bool) {
	parts := strings.Split(strings.TrimPrefix(data, "remove:"), ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	channelID, err1 := strconv.ParseInt(parts[0], 10, 64)
	groupID, err2 := strconv.ParseInt(parts[1], 10, 64)
	return channelID, groupID, err1 == nil && err2 == nil
}

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9_]{5,}$`)

func normalizeChannelUsername(input string) string {
	value := strings.TrimSpace(input)
	for _, prefix := range []string{"https://t.me/", "http://t.me/", "t.me/"} {
		value = strings.TrimPrefix(value, prefix)
	}
	value = strings.TrimPrefix(value, "@")
	value = strings.TrimSuffix(value, "/")
	if !usernameRe.MatchString(value) {
		return ""
	}
	return value
}

func parseCommand(text string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return "", false
	}
	command := strings.TrimPrefix(fields[0], "/")
	if at := strings.IndexByte(command, '@'); at >= 0 {
		command = command[:at]
	}
	return strings.ToLower(command), command != ""
}
