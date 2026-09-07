package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	telegram "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
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
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("close storage: %v", err)
		}
	}()

	moderationService := moderation.New(store)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var handler *appbot.Handler

	tg, err := telegram.New(
		cfg.TelegramBotToken,
		telegram.WithAllowedUpdates(telegram.AllowedUpdates{"message", "callback_query", "my_chat_member"}),
		telegram.WithDefaultHandler(func(ctx context.Context, b *telegram.Bot, update *models.Update) {
			handler.Handle(ctx, b, update)
		}),
	)
	if err != nil {
		log.Fatalf("create bot: %v", err)
	}

	handler = appbot.NewHandler(moderationService, appbot.NewChatService(tg))

	me, err := tg.GetMe(ctx)
	if err != nil {
		log.Fatalf("get bot info: %v", err)
	}

	log.Printf("bot started as @%s", me.Username)
	log.Printf("storage: %s", cfg.StoragePath)

	tg.Start(ctx)
	log.Println("bot stopped")
}
