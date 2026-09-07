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

func TestForwardChannelName(t *testing.T) {
	mk := func(chat models.Chat) *models.MessageOrigin {
		return &models.MessageOrigin{MessageOriginChannel: &models.MessageOriginChannel{Chat: chat}}
	}

	if got := forwardChannelName(mk(models.Chat{Username: "chan", Title: "T"})); got != "@chan" {
		t.Fatalf("expected @chan, got %q", got)
	}
	if got := forwardChannelName(mk(models.Chat{Title: "Channel"})); got != "Channel" {
		t.Fatalf("expected Channel, got %q", got)
	}
	if got := forwardChannelName(mk(models.Chat{})); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if got := forwardChannelName(nil); got != "" {
		t.Fatalf("expected empty for nil, got %q", got)
	}
}

func TestUserDisplayName(t *testing.T) {
	withUsername := &models.User{Username: "alice", FirstName: "A", LastName: "B"}
	if got := userDisplayName(withUsername); got != "@alice" {
		t.Fatalf("expected @alice, got %q", got)
	}

	noUsername := &models.User{FirstName: "Alice", LastName: "Smith"}
	if got := userDisplayName(noUsername); got != "Alice Smith" {
		t.Fatalf("expected 'Alice Smith', got %q", got)
	}

	empty := &models.User{ID: 42}
	if got := userDisplayName(empty); got != "42" {
		t.Fatalf("expected 42, got %q", got)
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
