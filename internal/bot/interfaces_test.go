package bot

import (
	"testing"

	"telegram-gatekeeper-bot/internal/moderation"
)

func TestConcreteDependenciesSatisfyInterfaces(t *testing.T) {
	var _ Moderation = (*moderation.Service)(nil)
	var _ ChatService = (*telegramChat)(nil)
}
