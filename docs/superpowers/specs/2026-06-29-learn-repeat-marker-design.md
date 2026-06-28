# Дизайн: пометка повторения для выученных слов в «Учить»

Дата: 2026-06-29. Маленькая правка показа.

## Цель

Когда в «Учить» выпадает слово со статусом `learned` (повторение), у строки
слова показываем 🔁 вместо 🎓.

## Решение

- `QuizView` получает поле `Repeat bool`.
- `learnQuestion` (`internal/service/learn.go`): `Repeat = u.Words[sess.Base].Status == StatusLearned`.
- `learnPrompt` (`internal/bot/quiz.go`): иконка строки слова = `🔁 ` если `Repeat`, иначе `🎓 `. Остальное (значение якоря, маркер «to» у base, вопрос про цель) без изменений.

Пример (learned): `🔁 written\n\nВведите past:`; (study): `🎓 written\n\nВведите past:`.

## Файлы
- `internal/service/types.go` — `QuizView.Repeat`.
- `internal/service/learn.go` — `learnQuestion`.
- `internal/bot/quiz.go` — `learnPrompt`.

## Тесты (TDD)
- `learnQuestion`: `Repeat=true` для learned-слова; `false` для study.
- `learnPrompt`: `Repeat` → строка начинается с `🔁 `; иначе `🎓 `.

## Вне объёма
Логика лестницы/выбора, пороги, прочие экраны.
