package service

import (
	"context"
	"time"
)

// reminderIdle is how long a user may go without solving a task before a
// reminder fires; it also throttles reminders to at most one per this window.
const reminderIdle = 24 * time.Hour

// markSolved records that the user just engaged with a task (resets the
// reminder timer).
func (s *Service) markSolved(u *User) { u.LastSolvedAt = s.now() }

// DueReminders returns the IDs of users who should get a reminder task now:
// idle/un-reminded for reminderIdle and with a non-empty learn pool.
func (s *Service) DueReminders(ctx context.Context) ([]int64, error) {
	before := s.now().Add(-reminderIdle)
	users, err := s.users.DueForReminder(ctx, before)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for _, u := range users {
		study, learned := s.learnPool(u)
		if len(study) == 0 && len(learned) == 0 {
			continue
		}
		ids = append(ids, u.ID)
	}
	return ids, nil
}

// Remind starts a learn session for the user and sends back the question View.
// ok is false (and nothing is changed) if the user has nothing to learn.
func (s *Service) Remind(ctx context.Context, userID int64) (View, bool, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, false, err
	}
	v, ok := s.beginLearn(u)
	if !ok {
		return View{}, false, nil
	}
	u.LastRemindedAt = s.now()
	if err := s.save(ctx, u); err != nil {
		return View{}, false, err
	}
	return v, true, nil
}
