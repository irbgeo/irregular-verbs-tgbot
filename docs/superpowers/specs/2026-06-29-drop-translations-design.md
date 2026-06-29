# Дизайн: убрать переводы из заданий; всегда одна из 3 форм

Дата: 2026-06-29.

## Цель

В заданиях участвуют только **base / past / participle**. Перевод нигде не
показывается и не спрашивается. В каждом задании даётся одна форма-якорь.

`Verb.Translations` в данных/структуре **остаётся** (не используется в
заданиях) — данные не трогаем.

## Тест (новый раунд)

На каждое слово:
- случайный **якорь** ∈ {base, past, participle} (через `s.rng`), показываем его
  значение;
- спрашиваем **две оставшиеся** формы по очереди, в каноническом порядке
  base→past→participle без якоря (2 под-вопроса);
- проверка ответа — `checkTarget` (base: «to»/без; past/participle: все формы);
- **ошибка / «Помощь»** → показать формы (`correctText`), слово в изучение
  (`study/mode1/box0`), следующее слово;
- **обе формы верно без помощи** → экран `test_result` («В изучение / Скип»);
- кнопки Теста (Помощь/Скип/Меню) — без изменений.

Сессия теста: хранит `AnchorKind`; цели выводятся из якоря
(`testTargets(anchor)`), `Step` (0..1) индексирует их.

## Учить

- Якорь ∈ {base, past, participle}, цель ∈ {base, past, participle} (минус
  перевод; якорь-форма может совпасть с целью).
- Режим 1 (выбор): цель всегда форма → 4 кнопки (`formOptions`). Ветка
  «перевод = 5 кнопок» убирается.

## Фидбэк

`correctText` → `to <base> — <past> — <participle>` (без перевода).

## Изменения кода

- `internal/service/check.go`:
  - `checkTarget` — ветки только `base/past/participle` (translation удалён);
  - удалить `checkAnswer` (Тест переходит на `checkTarget`);
  - `correctText` — без `Translations`;
  - удалить `anyEqual`, если больше не используется.
- `internal/service/learn.go`:
  - `buildRound` — якорь/цель из 3 форм; для choice всегда `formOptions`;
  - удалить `translationOptions`.
- `internal/service/test_flow.go`:
  - `testTargets(anchor) []string`, выбор случайного якоря в `StartTest`/
    `advance`; `testQuestion(u, sess)` строит `QuizView` (Mode `test`,
    `AnchorKind`/`AnchorValue`/`TargetKind`, Format input);
  - `Answer` — проверка `checkTarget(v, targets[Step], …)`; `Step < 1` → `Step++`,
    иначе `test_result`; ошибка → study+advance; `Help` → reveal+study+advance.
- `internal/service/types.go`:
  - удалить `QuizView.Translations`; удалить неиспользуемый `KindTranslation`.
- `internal/bot/quiz.go`:
  - `quizPrompt` показывает якорь (с «to» для base) + «Введите `<форма>`:»;
  - убрать `kindLabel["translation"]`.

## Тесты (TDD)

- `checkTarget`: base/past/participle (translation-кейсы убрать).
- Новый Тест-раунд: фикс якоря через `rng`, проверить 2 цели, результат после
  второй; ошибка/Помощь → study+advance; конец очереди → done.
- Учить: якорь/цель из 3 форм; choice — 4 кнопки.
- Убрать `TestTranslationOptions`, translation-кейсы в `checkTarget`/`correctText`.
- Обновить bot-тесты промпта Теста (новый текст «якорь → Введите …»).
- `markSolved`-тест Теста — под новый раунд.

## Вне объёма

Удаление `Translations` из `data/verbs.json`; смена кнопок Теста; пороги.
