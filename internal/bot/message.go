package bot

import (
	"context"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

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

	replyMarkup := models.ReplyMarkup(nil)
	if markup != nil {
		replyMarkup = markup
	}

	if _, err := tg.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      q.Message.Message.Chat.ID,
		MessageID:   q.Message.Message.ID,
		Text:        text,
		ReplyMarkup: replyMarkup,
	}); err != nil {
		// Telegram rejects edits that leave the message unchanged (same text and
		// same keyboard). This is expected when a user taps the same button
		// twice, so it is not an error and must not fall back to a duplicate
		// message.
		if strings.Contains(strings.ToLower(err.Error()), "message is not modified") {
			return
		}
		log.Printf("edit callback message: %v", err)
		h.sendPrivate(ctx, tg, q.From.ID, text, markup)
	}
}

func (h *Handler) answerCallback(ctx context.Context, tg *bot.Bot, id string) {
	_, err := tg.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: id})
	if err != nil {
		log.Printf("answer callback: %v", err)
	}
}
