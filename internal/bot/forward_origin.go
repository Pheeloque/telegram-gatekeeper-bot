package bot

import "github.com/go-telegram/bot/models"

func forwardChannelID(origin *models.MessageOrigin) (int64, bool) {
	if origin == nil || origin.MessageOriginChannel == nil {
		return 0, false
	}
	return origin.MessageOriginChannel.Chat.ID, true
}
