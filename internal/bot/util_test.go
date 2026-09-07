package bot

import (
	"context"
	"testing"

	"telegram-gatekeeper-bot/internal/storage"

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

func TestParseChannelIdentifier(t *testing.T) {
	cases := []struct {
		input string
		ref   channelRef
		ok    bool
	}{
		{"@example_channel", channelRef{username: "example_channel"}, true},
		{"example", channelRef{username: "example"}, true},
		{"https://t.me/example", channelRef{username: "example"}, true},
		{"-1001234567890", channelRef{id: -1001234567890, byID: true}, true},
		{"1001234567", channelRef{id: 1001234567, byID: true}, true},
		{"ab", channelRef{}, false},
		{"has space", channelRef{}, false},
		{"", channelRef{}, false},
	}
	for _, tc := range cases {
		got, ok := parseChannelIdentifier(tc.input)
		if ok != tc.ok {
			t.Errorf("parseChannelIdentifier(%q) ok = %v, want %v", tc.input, ok, tc.ok)
			continue
		}
		if !ok {
			continue
		}
		if got != tc.ref {
			t.Errorf("parseChannelIdentifier(%q) = %+v, want %+v", tc.input, got, tc.ref)
		}
	}
}

func TestNormalizeChannelID(t *testing.T) {
	cases := map[int64]int64{
		-1001418440636: -1001418440636,
		1418440636:     -1001418440636,
		0:              0,
		-123456:        -123456,
	}
	for input, want := range cases {
		if got := normalizeChannelID(input); got != want {
			t.Errorf("normalizeChannelID(%d) = %d, want %d", input, got, want)
		}
	}
}

type fakeChatService struct {
	resolved []string
}

func (f *fakeChatService) IsAdmin(ctx context.Context, chatID, userID int64) bool { return true }

func (f *fakeChatService) ResolveChannel(ctx context.Context, username string) (storage.Channel, bool) {
	f.resolved = append(f.resolved, username)
	return storage.Channel{ID: 7, Username: "chan"}, true
}

func TestResolveIdentifier(t *testing.T) {
	fake := &fakeChatService{}
	h := &Handler{chat: fake}

	channel, ok := h.resolveIdentifier(context.Background(), channelRef{id: 1418440636, byID: true})
	if !ok || channel.ID != -1001418440636 {
		t.Fatalf("expected ID -1001418440636, got %d (ok=%v)", channel.ID, ok)
	}

	channel, ok = h.resolveIdentifier(context.Background(), channelRef{id: -1001418440636, byID: true})
	if !ok || channel.ID != -1001418440636 {
		t.Fatalf("expected ID -1001418440636 unchanged, got %d (ok=%v)", channel.ID, ok)
	}

	channel, ok = h.resolveIdentifier(context.Background(), channelRef{username: "chan"})
	if !ok || channel.ID != 7 {
		t.Fatalf("expected username resolved through chat service, got ID=%d (ok=%v)", channel.ID, ok)
	}
	if len(fake.resolved) != 1 || fake.resolved[0] != "@chan" {
		t.Fatalf("expected username path to use chat service, got %v", fake.resolved)
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

func TestChannelFromForward(t *testing.T) {
	h := &Handler{}
	if _, ok := h.channelFromForward(&models.Message{}); ok {
		t.Fatal("message without forward origin should not resolve")
	}

	forwarded := &models.Message{
		ForwardOrigin: &models.MessageOrigin{
			MessageOriginChannel: &models.MessageOriginChannel{
				Chat: models.Chat{ID: -1001234567890, Username: "chan", Title: "Channel"},
			},
		},
	}
	channel, ok := h.channelFromForward(forwarded)
	if !ok || channel.ID != -1001234567890 || channel.Username != "chan" || channel.Title != "Channel" {
		t.Fatalf("unexpected channel from forward: %+v (ok=%v)", channel, ok)
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
