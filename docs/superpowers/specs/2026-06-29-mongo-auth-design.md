# Дизайн: MongoDB с аутентификацией

Дата: 2026-06-29. Инфраструктурная правка (без изменений Go-кода).

## Цель

MongoDB запускается **с паролем** (включена аутентификация). Бот подключается
с кредами. Порт Mongo не публикуется наружу.

## Решения

- Пользователь — **root через INITDB** (`MONGO_INITDB_ROOT_USERNAME/PASSWORD`).
- Порт Mongo **не публикуется** — доступ только внутри compose-сети.
- Креды — в `.env` (gitignored), подставляются в compose.

## Изменения

### `docker-compose.yml`
- **mongo:**
  - добавить `environment` с `MONGO_INITDB_ROOT_USERNAME: ${MONGO_USERNAME:?...}`
    и `MONGO_INITDB_ROOT_PASSWORD: ${MONGO_PASSWORD:?...}`;
  - удалить блок `ports` (Mongo больше не виден с хоста);
  - healthcheck `db.adminCommand('ping')` оставить — `ping` разрешён без
    авторизации даже при включённом auth.
- **bot:** заменить `MONGO_URI` на
  `mongodb://${MONGO_USERNAME}:${MONGO_PASSWORD}@mongo:27017/?authSource=admin`
  (`authSource=admin`, root-юзер в admin; данные — в `MONGO_DB=irregular_verbs`).

### `.env` (gitignored)
Добавить `MONGO_USERNAME` и `MONGO_PASSWORD` (стойкий пароль). В git не попадает.

### `.env.example` (новый, коммитим)
Шаблон без значений: `BOT_TOKEN=`, `MONGO_USERNAME=`, `MONGO_PASSWORD=`.

### `README.md`
В «⚙️ Запуск»: про `.env` с кредами Mongo, закрытый порт, требование чистого
тома при первом включении auth, и как гонять store-тесты.

## Эксплуатация

- Go-код не меняется: `store.Connect(MONGO_URI, MONGO_DB)` — креды/`authSource`
  в URI, БД данных — `MONGO_DB`.
- **Включение auth требует чистого тома:** `MONGO_INITDB_*` создаёт пользователя
  только при первой инициализации пустого `mongo-data`. На существующем томе
  нужно `docker compose down -v && docker compose up -d`.
- **Store-тесты с хоста** теперь скипаются (порт закрыт). Чтобы прогнать —
  задать `MONGO_URI` с кредами на доступный Mongo или временно опубликовать порт.

## Тестирование

Юнит-тестов нет (compose/env). Проверка ручная: `docker compose down -v &&
docker compose up -d` → бот стартует и подключается (нет ошибок auth в логах).

## Вне объёма

Least-privilege app-пользователь, TLS до Mongo, ротация секретов, секрет-менеджер.
