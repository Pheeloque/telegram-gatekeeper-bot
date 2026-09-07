package bot

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestNormalizeChannelUsername(t *testing.T) {
	cases := map[string]string{
		"@example_channel":     "example_channel",
		"https://t.me/example": "example",
		"http://t.me/example/": "example",
		"t.me/example":         "example",
		"example":              "example",
		"ab":                   "",
		"has space":            "",
	}
	for input, want := range cases {
		if got := normalizeChannelUsername(input); got != want {
			t.Errorf("normalizeChannelUsername(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseCommand(t *testing.T) {
	cases := map[string]struct {
		cmd string
		ok  bool
	}{
		"/start":        {"start", true},
		"/GROUPS":       {"groups", true},
		"/groups@mybot": {"groups", true},
		"hello":         {"", false},
		"":              {"", false},
	}
	for input, want := range cases {
		cmd, ok := parseCommand(input)
		if cmd != want.cmd || ok != want.ok {
			t.Errorf("parseCommand(%q) = (%q, %v), want (%q, %v)", input, cmd, ok, want.cmd, want.ok)
		}
	}
}

func TestForwardChannelID(t *testing.T) {
	if _, ok := forwardChannelID(nil); ok {
		t.Fatal("nil origin should return not ok")
	}

	origin := &models.MessageOrigin{
		MessageOriginChannel: &models.MessageOriginChannel{
			Chat: models.Chat{ID: 123},
		},
	}
	id, ok := forwardChannelID(origin)
	if !ok || id != 123 {
		t.Fatalf("expected id=123 ok=true, got id=%d ok=%v", id, ok)
	}
}

func TestHasAdminRights(t *testing.T) {
	admin := &models.ChatMember{Type: models.ChatMemberTypeAdministrator}
	owner := &models.ChatMember{Type: models.ChatMemberTypeOwner}
	member := &models.ChatMember{Type: models.ChatMemberTypeMember}

	if !isAdminMember(admin) {
		t.Fatal("administrator should have admin rights")
	}
	if !isAdminMember(owner) {
		t.Fatal("owner should have admin rights")
	}
	if isAdminMember(member) {
		t.Fatal("member should not have admin rights")
	}
}
