package bot

import "github.com/go-telegram/bot/models"

func forwardChannel(origin *models.MessageOrigin) (models.Chat, bool) {
	if origin == nil || origin.MessageOriginChannel == nil {
		return models.Chat{}, false
	}
	return origin.MessageOriginChannel.Chat, true
}

func forwardChannelID(origin *models.MessageOrigin) (int64, bool) {
	chat, ok := forwardChannel(origin)
	if !ok {
		return 0, false
	}
	return chat.ID, true
}

func forwardChannelName(origin *models.MessageOrigin) string {
	chat, ok := forwardChannel(origin)
	if !ok {
		return ""
	}
	if chat.Username != "" {
		return "@" + chat.Username
	}
	return chat.Title
}
