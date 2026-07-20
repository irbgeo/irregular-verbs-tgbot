package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	tgbot "github.com/irbgeo/go-tgbot"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/bot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/config"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/store"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/worker"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	st, err := store.Connect(connectCtx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		return err
	}
	defer st.Disconnect(context.Background()) // nolint:errcheck

	list, err := service.LoadVerbs(cfg.VerbsPath)
	if err != nil {
		return err
	}
	if err := service.SeedVerbs(ctx, st.Verbs, list); err != nil {
		return err
	}
	log.Printf("seeded %d verbs", len(list))

	svc := service.New(st.Users, list)

	client, err := tgbot.NewClient(cfg.BotToken)
	if err != nil {
		return err
	}

	tgSender := bot.NewTelegramSender(client)
	router := bot.New(svc, tgSender)

	worker.New(svc, router).Start(ctx)

	log.Println("bot started (long polling)")
	return client.Poll(ctx, tgbot.PollOptions{
		Timeout: 10,
		OnError: func(err error) { log.Printf("poll: %v", err) },
	}, router.Handle)
}
