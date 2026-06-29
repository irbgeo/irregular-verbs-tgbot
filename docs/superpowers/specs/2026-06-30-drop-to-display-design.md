# Дизайн: убрать «to » у инфинитивов в выводе

Дата: 2026-06-30.

## Изменение
Инфинитив показывается без маркера «to » во всех местах:
- промпт Теста, якорь «Учить», кнопки-варианты (base), фидбэк (`correctText`),
  список слов.
- Хелпер `BaseLabel` удаляется (показываем `base` как есть).

Ввод **не меняется**: «go» и «to go» оба принимаются (`normBase` срезает
необязательный «to»).

## Файлы
- `internal/service/check.go` — удалить `BaseLabel`; `correctText` → `v.Base`.
- `internal/bot/quiz.go` — `quizPrompt`/`learnPrompt` без `BaseLabel`.
- `internal/bot/screens.go` — кнопки base и список без `BaseLabel`.
- Тесты: убрать «to go»/«to be» из ожиданий; удалить тест `BaseLabel`.

## Вне объёма
Приём ввода с «to»; прочее форматирование.
