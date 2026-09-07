package bot

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/go-telegram/bot/models"
)

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
