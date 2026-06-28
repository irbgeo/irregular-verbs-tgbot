# Технический дизайн: ввод всех форм (past/participle)

Дата: 2026-06-29. Правка проверки текстового ответа.

## Цель

Когда у формы несколько вариантов (`be` past = `was/were`), при текстовом
вводе нужно ввести **все** формы. Одной формы недостаточно.

## Область

Применяется к **текстовому вводу** past и participle:
- Тест: шаг 1 (past), шаг 2 (participle) — `checkAnswer`.
- Учить, формат `input` (режим 2 / повторение): цель = past или participle —
  `checkTarget`.

**Не затрагивает:**
- `base` — одна форма (логика `normBase`/«to» без изменений).
- `translation` — синонимы, достаточно любого одного (`anyEqual`, как сейчас).
- Режим 1 (`choice`) — тап одной кнопки; правильная = первая форма
  (`correctOption`), ввода нет.
- Показ форм — без изменений (`was/were` через «/»).

## Проверка `allFormsMatch(input, options) bool`

- Разбить `input` по разделителям: пробел, `/`, запятая (любой набор подряд),
  через `strings.FieldsFunc`.
- Нормализовать каждый токен (`norm`: lower+trim).
- Множество введённых токенов должно **точно** совпасть с множеством
  `options` (по `norm`): все формы присутствуют, лишних нет, порядок не важен,
  дубликаты схлопываются.
- Пустой ввод или пустой `options` → `false`.

Примеры (`be` past = `[was, were]`):

| Ввод | Итог |
|---|---|
| `was were`, `were was`, `was/were`, `was, were` | ✅ |
| `was` | ❌ (не все) |
| `was were eaten` | ❌ (лишнее) |

Одноформенные (`go` past = `[went]`): `went` ✅ — без изменений.

## Файлы

- `internal/service/check.go` — `allFormsMatch` (+ `isFormSep`); `checkAnswer`
  шаг 1 (past) и default (participle) → `allFormsMatch`. Импорт `unicode`.
- `internal/service/learn.go` — `checkTarget` ветки `KindPast`/`KindParticiple`
  → `allFormsMatch`.
- Обновить тесты: `check_test.go`, `learn_check_test.go` (одиночные `was`/`were`
  заменить на строгие случаи).

## Тесты (TDD)

- `allFormsMatch`: все приведённые примеры (мульти и одиночная форма, лишнее,
  порядок, разделители `/`, `,`, пробел).
- `checkAnswer` шаг 1/2 и `checkTarget` past/participle: «was were» ✓, «was» ✗,
  «been» ✓ (одна форма).
- Перевод (`checkTarget` translation) и base — поведение не изменилось.

## Вне объёма

Перевод, base, режим 1, тексты подсказок, `data/verbs.json`.
