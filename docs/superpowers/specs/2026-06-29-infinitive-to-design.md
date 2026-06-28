# Технический дизайн: «to » у инфинитива

Дата: 2026-06-29. Небольшая правка поверх Теста и «Учить».

## Цель

База (инфинитив) показывается пользователю с маркером **«to »** (`go` → `to go`).
При вводе инфинитива принимаем и с «to», и без.

## Показ — единый помощник

Экспортируемый помощник в `service` (один источник правила):

```go
func BaseLabel(base string) string { return "to " + base }
```

Применяется во всех местах показа base:

- **Учить, якорь** (`internal/bot/quiz.go`, `learnPrompt`): если `AnchorKind == KindBase`,
  показывать `service.BaseLabel(AnchorValue)`.
- **Учить, кнопки режима 1** (`internal/bot/screens.go`, ветка learn у `ScreenQuiz`):
  если `TargetKind == KindBase`, текст кнопки = `service.BaseLabel(opt)` для каждого
  варианта (включая дистракторы). **Callback `lc:<idx>` и сравнение в сервисе не
  меняются** — `Session.Options` остаются голыми формами, проверка ответа цела.
- **Фидбэк «правильно»** (`internal/service/check.go`, `correctText`): base-токен →
  `BaseLabel(v.Base)`. Покрывает Тест и Учить (неверно/Помощь/Показать).
- **Списки слов** (`internal/bot/screens.go`, `wordRows`): подпись base = `BaseLabel(it.Base)`.
  **Callback `tog:<base>` остаётся голым** — это идентификатор слова.

Маркер «to » — только для base. Past/participle/перевод — без изменений.

## Ввод — с «to» и без

Внутренний помощник в `internal/service/check.go`:

```go
func normBase(s string) string { return strings.TrimPrefix(norm(s), "to ") }
```

- **Тест, шаг 0** (`checkAnswer` case 0): `normBase(input) == norm(v.Base)`.
- **Учить, цель = base** (`checkTarget`, ветка `KindBase`): `normBase(input) == norm(v.Base)`.

Принимается «go» и «to go». «togo» (без пробела) не проходит.

## Файлы

- `internal/service/check.go` — `BaseLabel` (экспорт), `normBase`, `correctText`, `checkAnswer` case 0.
- `internal/service/learn.go` — `checkTarget` ветка `KindBase`.
- `internal/bot/quiz.go` — `learnPrompt` (якорь base).
- `internal/bot/screens.go` — кнопки вариантов learn (base-цель) + `wordRows` (списки).

## Тесты (TDD)

- `TestBaseLabel` — `BaseLabel("go") == "to go"`.
- `normBase`/ввод: «go», «to go» → верно; «togo», «to went» → неверно — для Теста (шаг 0)
  и Учить (цель base).
- `correctText` содержит `to <base>`.
- bot-рендер: якорь base = «to go»; кнопка base-цели = «to go» с callback `lc:0`;
  список = «to go — went — gone» с callback `tog:go`.
- Обновить существующие тесты, где сравнивается точный текст `correctText` или подписи
  списков, если они завязаны на голую base.

## Вне объёма

- `data/verbs.json` — не трогаем.
- Прочие формы/перевод — без «to».
