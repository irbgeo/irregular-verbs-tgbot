# Дизайн: старт mode0-слов в «Учить» + убрать подпись формы у якоря

Дата: 2026-06-29. Багфикс + правка показа в «Учить».

## Фикс 1: «Учить» стартует study-слова с режима 1

**Проблема.** Список (📋 Мои слова / 📚 Список слов) кладёт слово как
`{status: study, mode: 0, box: 0}` (`internal/service/lists.go`, `CommitList`).
`wordFormat` считает режимом 1 (выбор) только `mode == 1`, поэтому слово с
`mode 0` уходит в текстовый ввод (режим 2), а лестница (`study && mode == 1`)
для него не срабатывает.

**Решение.** «Учить» инициализирует mode при первом показе study-слова:

```go
func (s *Service) startStudyWord(u *User, base string) {
	w := u.Words[base]
	if w.Status == StatusStudy && w.Mode == 0 {
		w.Mode = 1
		u.Words[base] = w
	}
}
```

Вызывается в `StartLearn` и `advanceLearn` сразу после выбора слова и установки
`sess.Base`, **до** `s.buildRound(...)` (чтобы `wordFormat` уже видел `mode 1`).
Существующий `save` в этих use-case'ах сохраняет изменение. Слова из Теста (уже
`mode 1`), `mode 2` и `learned` не меняются.

## Правка 2: у якоря нет подписи формы

`learnPrompt` (`internal/bot/quiz.go`) показывает только значение якоря, без
«(вид формы)». Для base сохраняется маркер «to » (`to go`). Вопрос про цель
(«Введите/Выберите `<форма>`:») не меняется.

- Было: `🎓 written (past participle)\n\nВведите past:`
- Стало: `🎓 written\n\nВведите past:`

`kindLabel` остаётся (используется для названия цели в вопросе).

## Файлы

- `internal/service/learn.go` — `startStudyWord` + вызовы в `StartLearn` и `advanceLearn`.
- `internal/bot/quiz.go` — `learnPrompt` (убрать подпись якоря).

## Тесты (TDD)

- `startStudyWord` / поток: study-слово с `mode 0` в «Учить» → формат `choice`
  (режим 1); верный ответ двигает `box` по лестнице mode1.
- `learnPrompt`: якорь без «(...)»; base-якорь = «to go»; в вопросе название
  формы остаётся.
- Обновить bot-тесты, где проверялись `«went (past)»` / `«to go (инфинитив)»`
  в тексте якоря.

## Вне объёма

Логика выбора слова и лестницы (кроме старта `mode 0`→`1`), Тест, списки,
`data/verbs.json`.
