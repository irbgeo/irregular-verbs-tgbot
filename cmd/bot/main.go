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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	st, err := store.Connect(connectCtx, &cfg)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Disconnect(context.Background()) // nolint:errcheck

	list, err := service.LoadVerbs(cfg.VerbsPath)
	if err != nil {
		log.Fatalf("verbs: %v", err)
	}
	seedVerbsParams := service.SeedVerbsParams{
		Repo:  st.Verbs,
		Verbs: list,
	}
	err = service.SeedVerbs(ctx, seedVerbsParams)
	if err != nil {
		log.Fatalf("seed verbs: %v", err)
	}
	log.Printf("seeded %d verbs", len(list))

	svc := service.New(st.Users, list)

	client, err := tgbot.NewClient(cfg.BotToken)
	if err != nil {
		log.Fatalf("telegram client: %v", err)
	}

	tgSender := bot.NewTelegramSender(client)
	router := bot.New(svc, tgSender)

	w := worker.New(svc, router)
	w.Start(ctx)

	log.Println("bot started (long polling)")
	pollOptions := tgbot.PollOptions{
		Timeout: 10,
		OnError: func(err error) {
			log.Printf("poll: %v", err)
		},
	}
	err = client.Poll(ctx, pollOptions, router.Handle)
	if err != nil {
		log.Fatalf("poll: %v", err)
	}
}
