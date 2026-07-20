// Package worker runs the bot's background loops (the idle-user reminder
// scheduler), orchestrating the service and the bot router. It keeps this
// logic out of main, which only wires things up.
package worker

import (
	"context"
	"log"
	"time"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/bot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// reminderTick is how often the scheduler scans for users due a reminder.
const reminderTick = time.Hour

// Worker owns the background loops.
type Worker struct {
	svc    *service.Service
	router *bot.Router
}

// New creates a Worker.
func New(svc *service.Service, router *bot.Router) *Worker {
	return &Worker{svc: svc, router: router}
}

// Start launches the background loops as goroutines; they stop when ctx is
// cancelled.
func (w *Worker) Start(ctx context.Context) {
	go w.remindLoop(ctx)
}

// remindLoop periodically sends a learn task to users idle for over 24h.
func (w *Worker) remindLoop(ctx context.Context) {
	t := time.NewTicker(reminderTick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ids, err := w.svc.DueReminders(ctx)
			if err != nil {
				log.Printf("reminders: due query: %v", err)
				continue
			}
			for _, id := range ids {
				v, ok, err := w.svc.Remind(ctx, id)
				if err != nil {
					log.Printf("reminders: build %d: %v", id, err)
					continue
				}
				if !ok {
					continue
				}
				if err := w.router.Deliver(ctx, id, v); err != nil {
					log.Printf("reminders: deliver %d: %v", id, err)
				}
			}
		}
	}
}
