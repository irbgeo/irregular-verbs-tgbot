# Дизайн: Тест — ввод всех 3 форм по порядку

Дата: 2026-06-29. Заменяет недавний раунд Теста (якорь + 2 цели).

## Раунд
- Показываем инфинитив `to <base>`. Одним сообщением пользователь вводит **3
  формы по порядку: base → past → participle**.
- Раунд = ОДИН под-вопрос на слово (в Тесте больше нет Step/якоря).
- Ошибка / «Помощь» → показать формы (`correctText`), слово в изучение
  (`study/mode1/box0`), следующее слово. Всё верно → `test_result`
  («В изучение / Скип»). Кнопки Теста — без изменений.

## Проверка `checkAllFormsOrdered(v, input, variant)`
- Токенизируем `input` по любым разделителям (пробел, `/`, запятая, перенос
  строки) через `FieldsFunc(isFormSep)`, нормализуем токены (`norm`).
- Опционально срезаем ведущий токен `to` (маркер инфинитива).
- По порядку сверяем три группы: `[v.Base]`, `v.Past[variant]`,
  `v.Participle[variant]`. Каждая группа потребляет ровно `len(group)` токенов
  и сверяется как мультимножество (порядок ВНУТРИ группы не важен). Лишние или
  недостающие токены → неверно.
- **Порядок трёх форм важен** (базовая группа первой и т.д.).

Примеры: `go went gone` ✓; `went go gone` ✗; `to go went gone` ✓;
`be was were been` ✓; `be was/were been` ✓; `be been was were` ✗.

## Файлы
- `internal/service/check.go` — `checkAllFormsOrdered` + токенизатор `tokensOf`
  + `sameFormSet`.
- `internal/service/test_flow.go` — раунд = один ввод: `testQuestion(sess)`
  возвращает `QuizView{Base, Mode:"test"}`; `StartTest`/`advance` без
  Step/якоря; `Answer` использует `checkAllFormsOrdered`. Удалить добавленные
  ранее `testKinds`/`testTargets`/анкер-логику Теста (`AnchorKind` остаётся для
  Учить).
- `internal/bot/quiz.go` — `quizPrompt`: `to <base>` + «Введите 3 формы по
  порядку (инфинитив, past, participle):».

## Тесты
- `checkAllFormsOrdered`: примеры выше (порядок, многовариантность, «to»,
  лишнее/недостающее).
- Тест-раунд: один верный ввод → `test_result`; неверный → study+advance;
  «Помощь» → reveal+study+advance; конец очереди → done.

## Учить
Не трогаем (один якорь → одна цель).

## Вне объёма
`Verb.Translations`/`Session.Step` (неиспользуемые) — не удаляем здесь.
