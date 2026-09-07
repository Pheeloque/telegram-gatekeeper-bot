package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	TelegramBotToken string
	StoragePath      string
}

func Load() (Config, error) {
	_ = loadEnvFile(".env")

	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return Config{}, errors.New("TELEGRAM_BOT_TOKEN is not set")
	}

	storagePath := strings.TrimSpace(os.Getenv("STORAGE_PATH"))
	if storagePath == "" {
		storagePath = "data.json"
	}

	return Config{TelegramBotToken: token, StoragePath: storagePath}, nil
}

func loadEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set env %s: %w", key, err)
			}
		}
	}
	return nil
}
