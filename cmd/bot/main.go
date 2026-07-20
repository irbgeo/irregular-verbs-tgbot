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
	defer st.Disconnect(context.Background())

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
	router := bot.New(svc, bot.TelegramSender{Client: client})

	log.Println("bot started (long polling)")
	go remindLoop(ctx, svc, router)
	return client.Poll(ctx, tgbot.PollOptions{
		Timeout: 10,
		OnError: func(err error) { log.Printf("poll: %v", err) },
	}, router.Handle)
}

// reminderTick is how often the scheduler scans for users due a reminder.
const reminderTick = time.Hour

// remindLoop periodically sends a learn task to users idle for over 24h.
func remindLoop(ctx context.Context, svc *service.Service, router *bot.Router) {
	t := time.NewTicker(reminderTick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ids, err := svc.DueReminders(ctx)
			if err != nil {
				log.Printf("reminders: due query: %v", err)
				continue
			}
			for _, id := range ids {
				v, ok, err := svc.Remind(ctx, id)
				if err != nil {
					log.Printf("reminders: build %d: %v", id, err)
					continue
				}
				if !ok {
					continue
				}
				if err := router.Deliver(ctx, id, v); err != nil {
					log.Printf("reminders: deliver %d: %v", id, err)
				}
			}
		}
	}
}
