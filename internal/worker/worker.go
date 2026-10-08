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

// worker owns the background loops.
type worker struct {
	svc    *service.Service
	router *bot.Router
}

// New creates a worker.
func New(
	svc *service.Service,
	router *bot.Router,
) *worker {
	return &worker{
		svc:    svc,
		router: router,
	}
}

// Start launches the background loops as goroutines; they stop when ctx is
// cancelled.
func (s *worker) Start(ctx context.Context) {
	go s.remindLoop(ctx)
}

// remindLoop periodically sends a learn task to users idle for over 24h.
func (s *worker) remindLoop(ctx context.Context) {
	t := time.NewTicker(reminderTick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.remindAll(ctx)
		}
	}
}

// remindAll sends a reminder to every user due one now.
func (s *worker) remindAll(ctx context.Context) {
	ids, err := s.svc.DueReminders(ctx)
	if err != nil {
		log.Printf("reminders: due query: %v", err)
		return
	}
	for _, id := range ids {
		s.remind(ctx, id)
	}
}

// remind builds and delivers one user's reminder, logging any failure.
func (s *worker) remind(ctx context.Context, userID int64) {
	v, ok, err := s.svc.Remind(ctx, userID)
	if err != nil {
		log.Printf("reminders: build %d: %v", userID, err)
		return
	}
	if !ok {
		return
	}
	deliverParams := bot.DeliverParams{
		ChatID: userID,
		View:   &v,
	}
	err = s.router.Deliver(ctx, deliverParams)
	if err != nil {
		log.Printf("reminders: deliver %d: %v", userID, err)
	}
}
