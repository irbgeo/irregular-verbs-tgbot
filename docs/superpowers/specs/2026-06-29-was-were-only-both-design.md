# Дизайн: требовать обе формы только для was/were

Дата: 2026-06-29.

## Правило
При **текстовом вводе** многовариантной формы (past/participle):
- если варианты = `{was, were}` (past у `be`) → нужны **обе** формы;
- иначе (любые другие альтернативы: hung/hanged, burnt/burned, lit/lighted …)
  → достаточно **любой одной**.

Показ форм (список/инфо) и режим 1 «Учить» (тап одной кнопки) не меняются.

## Реализация (`internal/service/check.go`)
- `requiresAllVariants(forms) bool` = множество равно `{was, were}` (регистр
  неважен). Хардкод именно этой пары.
- Вернуть `anyEqual(input, options)` (был удалён) — «любая одна».
- `matchForm(input, forms) bool` = `requiresAllVariants` ? `allFormsMatch` :
  `anyEqual`.
- `checkTarget` (Учить, ввод): past/participle → `matchForm`.
- `checkAllFormsOrdered` (Тест): для каждой позиции — если `requiresAllVariants`,
  потребляем `len(forms)` токенов (как множество); иначе потребляем **1** токен
  (он должен быть одним из вариантов).

## Тесты
- `matchForm`: was/were — нужны обе; burnt/burned — любая одна; одиночная форма.
- `checkAllFormsOrdered`: be — обе past; глагол с альтернативой (burnt/burned) —
  одна форма на позицию; лишний токен → неверно.
- Существующие be-кейсы (`checkTarget`, ordered) остаются (be — особый случай).

## Вне объёма
Данные `verbs.json`; режим 1; показ форм.
