package config

import (
	"errors"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramBotToken string
	StoragePath      string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return Config{}, errors.New("TELEGRAM_BOT_TOKEN is not set")
	}

	storagePath := strings.TrimSpace(os.Getenv("STORAGE_PATH"))
	if storagePath == "" {
		storagePath = "data.db"
	}

	return Config{TelegramBotToken: token, StoragePath: storagePath}, nil
}
