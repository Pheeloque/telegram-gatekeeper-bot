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

func removeCallbackID(channelID, groupID int64) string {
	return "remove:" + strconv.FormatInt(channelID, 10) + ":" + strconv.FormatInt(groupID, 10)
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

// channelRef is a user-provided channel reference that has been parsed into a
// concrete form: either a channel username or a numeric chat ID.
type channelRef struct {
	username string
	id       int64
	byID     bool
}

// parseChannelIdentifier parses a user-provided channel reference. It accepts
// either a public channel username (with or without @ or t.me/ prefix) or a
// numeric channel ID.
func parseChannelIdentifier(input string) (channelRef, bool) {
	value := strings.TrimSpace(input)
	for _, prefix := range []string{"https://t.me/", "http://t.me/", "t.me/"} {
		value = strings.TrimPrefix(value, prefix)
	}
	value = strings.TrimSuffix(value, "/")
	if id, err := strconv.ParseInt(value, 10, 64); err == nil {
		return channelRef{id: id, byID: true}, true
	}
	if username := normalizeChannelUsername(value); username != "" {
		return channelRef{username: username}, true
	}
	return channelRef{}, false
}

// normalizeChannelID converts a user-visible numeric chat ID into the full Bot
// API ID for supergroups and channels. Telegram shows such IDs without the
// "-100" prefix, so a positive ID is treated as the numeric suffix.
func normalizeChannelID(id int64) int64 {
	if id <= 0 {
		return id
	}
	if withPrefix, err := strconv.ParseInt("-100"+strconv.FormatInt(id, 10), 10, 64); err == nil {
		return withPrefix
	}
	return id
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

func userDisplayName(u *models.User) string {
	if u.Username != "" {
		return "@" + u.Username
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name != "" {
		return name
	}
	return strconv.FormatInt(u.ID, 10)
}
