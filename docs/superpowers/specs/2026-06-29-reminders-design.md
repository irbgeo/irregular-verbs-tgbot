# Технический дизайн: напоминания (этап 4)

Дата: 2026-06-29.

## Цель

Если у пользователя есть слова для учёбы и он **более 24 часов не решал
задание**, бот сам присылает ему задание «Учить» (вопрос), на который можно
ответить прямо в чате. Пока пользователь молчит — напоминаем не чаще раза в
сутки.

## Поток

Фоновый тикер в `cmd/bot` (раз в час) рядом с `poll`. На тик:
1. `ids := svc.DueReminders(ctx)` — должники.
2. для каждого `id`: `v, ok := svc.Remind(ctx, id)`; если `ok` —
   `router.Deliver(ctx, id, v)` (рендер + отправка в личку, `chatID == userID`).
3. ошибка отправки (заблокировал бота и т.п.) — логируем, продолжаем.

Ответ пользователя приходит обычным апдейтом и идёт через текущий роутер;
решение задания обновляет `LastSolvedAt`, и напоминания прекращаются, пока он
снова не затихнет на 24ч.

## Модель (`internal/service/types.go`, `User`)

- `LastSolvedAt time.Time` `bson:"last_solved_at"` — момент последнего решения задания.
- `LastRemindedAt time.Time` `bson:"last_reminded_at"` — момент последнего отправленного напоминания (троттлинг).

(Существующие `CreatedAt`, `LastActiveAt` не трогаем.)

## Условие «должник»

Пороговое время `before = now − 24ч`. Кандидат, если **все**:
- `CreatedAt ≤ before` — аккаунт старше суток (новичку даём фору);
- `LastSolvedAt ≤ before` — не решал задание ≥ 24ч (нулевое время = «никогда» проходит);
- `LastRemindedAt ≤ before` — не напоминали ≥ 24ч;
- есть слова (`words` непуст) **и** пул для учёбы непуст (`study ∪ learned`).

Первые четыре проверяются в Mongo; точная проверка пула (study∪learned, не
только skipped) — в сервисе на Go.

## Слой хранилища и сервиса

`UserRepository` (порт) — новый метод:
```go
DueForReminder(ctx context.Context, before time.Time) ([]*User, error)
```
Mongo-фильтр:
```go
bson.M{
  "created_at":       bson.M{"$lte": before},
  "last_solved_at":   bson.M{"$lte": before},
  "last_reminded_at": bson.M{"$lte": before},
  "words":            bson.M{"$exists": true, "$ne": bson.M{}},
}
```
Возвращает все совпавшие документы.

Сервис:
- `reminderIdle = 24 * time.Hour` (константа).
- `func (s *Service) DueReminders(ctx) ([]int64, error)` —
  `repo.DueForReminder(now − reminderIdle)`, оставляет только тех, у кого
  `learnPool` непуст; возвращает их ID.
- `func (s *Service) beginLearn(u *User) (View, bool)` — извлечён из текущего
  `StartLearn`: подобрать слово (`pickLearnWord`), `startStudyWord`,
  `buildRound`, поставить `State = quiz+session`; вернуть quiz-View и `true`;
  пул пуст → `(View{}, false)`, `u` не меняется.
- `StartLearn` рефакторится на `beginLearn` (поведение прежнее: пул пуст →
  экран `learn_empty`).
- `func (s *Service) Remind(ctx, userID) (View, bool, error)` — `load`;
  `v, ok := beginLearn(u)`; если `!ok` → вернуть `(View{}, false, nil)` без
  изменений; иначе `u.LastRemindedAt = now`, `save`, вернуть `(v, true, nil)`.
- `markSolved(u)` — `u.LastSolvedAt = s.now()`. Вызывается в use-case'ах ответа:
  - Тест: `Answer` (после прохождения `inQuiz`-гейта, тестовая ветка), `Help` (тестовая ветка);
  - Учить: `resolveLearn` (покрывает ввод, выбор и «Показать»).

## Бот

`internal/bot` — новый метод доставки View (рендер инкапсулирован в пакете):
```go
func (r *Router) Deliver(ctx context.Context, chatID int64, v service.View) error {
	text, kb := render(v)
	if text == "" { return nil }
	return r.sender.Send(ctx, chatID, text, kb)
}
```

## Планировщик (`cmd/bot`)

- Константа `reminderTick = time.Hour`.
- Горутина `remindLoop(ctx, svc, router)` с `time.NewTicker(reminderTick)`:
  на каждый тик — `DueReminders` → `Remind` → `router.Deliver`; ошибки логируются.
  Останов по `ctx.Done()`. Запускается через `go remindLoop(...)` перед `poll`.

## Тесты (TDD)

- `markSolved`: решение в Тесте (`Answer`) и в Учить (через `resolveLearn`) обновляет `LastSolvedAt`.
- `DueReminders` (фейковый репозиторий реализует тот же фильтр по времени):
  берёт должника; пропускает — кто решал недавно, кого недавно напоминали,
  у кого нет слов / только skipped, и новичка (`CreatedAt > before`).
- `Remind`: ставит `LastRemindedAt`, возвращает quiz-View и `ok=true`; пустой
  пул → `ok=false`, `LastRemindedAt` не меняется.
- `StartLearn` после рефакторинга — существующие тесты зелёные.
- Mongo `DueForReminder` — интеграционный тест в пакете `store` (Mongo поднят).
- `cmd/bot` `remindLoop` — без юнит-теста (тонкая обёртка, как `poll`).

## Вне объёма

Тихие часы / таймзоны (ночью тоже может прийти), настройка интервалов из UI,
отписка от напоминаний, дедуп при нескольких инстансах бота. Пороги — константы.
