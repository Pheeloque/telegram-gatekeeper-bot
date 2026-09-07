package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	telegram "github.com/go-telegram/bot"
	appbot "telegram-gatekeeper-bot/internal/bot"
	"telegram-gatekeeper-bot/internal/config"
	"telegram-gatekeeper-bot/internal/moderation"
	"telegram-gatekeeper-bot/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	store, err := storage.New(cfg.StoragePath)
	if err != nil {
		log.Fatalf("load storage: %v", err)
	}
	defer store.Close()

	moderationService := moderation.New(store)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tg, err := telegram.New(
		cfg.TelegramBotToken,
		telegram.WithAllowedUpdates(telegram.AllowedUpdates{"message", "callback_query", "my_chat_member"}),
		telegram.WithDefaultHandler(appbot.NewHandler(store, moderationService).Handle),
	)
	if err != nil {
		log.Fatalf("create bot: %v", err)
	}

	me, err := tg.GetMe(ctx)
	if err != nil {
		log.Fatalf("get bot info: %v", err)
	}

	log.Printf("bot started as @%s", me.Username)
	log.Printf("storage: %s", cfg.StoragePath)

	tg.Start(ctx)
	log.Println("bot stopped")
}
