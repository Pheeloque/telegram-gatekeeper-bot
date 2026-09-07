# telegram-gatekeeper-bot

Telegram-бот для модерации групп: он хранит отдельный чёрный список каналов для каждой группы и удаляет пересланные сообщения из запрещённых каналов.

## Структура проекта

```text
telegram-gatekeeper-bot/
├── cmd/
│   └── bot/
│       └── main.go
├── internal/
│   ├── bot/
│   │   ├── forward_origin.go
│   │   ├── handler.go
│   │   ├── keyboard.go
│   │   ├── message.go
│   │   ├── session.go
│   │   ├── telegram_chat.go
│   │   └── util.go
│   ├── config/
│   │   └── config.go
│   ├── moderation/
│   │   └── service.go
│   └── storage/
│       └── storage.go
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

`cmd/bot` только собирает зависимости и запускает приложение. Telegram-обработчики находятся в `internal/bot`, бизнес-логика — в `internal/moderation`, хранение — в `internal/storage`, конфигурация — в `internal/config`.

## Требования

- Go 1.23+;
- бот, созданный через `@BotFather`;
- бот добавлен в группу как администратор;
- боту выдано право удалять сообщения;
- пользователь, меняющий настройки, должен быть администратором группы.

## Установка

1. Скопируйте `.env.example` в `.env`.
2. Укажите токен:

```env
TELEGRAM_BOT_TOKEN=123456789:YOUR_TOKEN
STORAGE_PATH=data.json
```

3. Запустите:

```bash
go mod tidy
go run ./cmd/bot
```

## Управление

В личном чате:

- `/start` — открыть список групп;
- `/groups` — открыть список групп;
- `/help` — открыть список групп.

После выбора группы можно добавить канал в чёрный список по `@username`, числовому ID (например `1234567890`, как показывает Telegram) или просто переслав сообщение из этого канала. Посмотреть чёрный список или удалить канал можно аналогично.

## Примечание о пересылках

Модерация использует `Message.forward_origin` и ищет `MessageOriginChannel`. Ограничения Bot API на доступность этой информации определяются самим Telegram; они не снимаются кодом бота.

## Хранение

Настройки сохраняются в SQLite-файле (по умолчанию `data.db`, задаётся через `STORAGE_PATH`). Драйвер `modernc.org/sqlite` (чистый Go, без CGO). Для нескольких экземпляров бота замените SQLite на централизованную БД (например PostgreSQL).

## Проверка

После установки зависимостей рекомендуется выполнить:

```bash
go test ./...
```

## Docker

Проект поставляется с `Dockerfile` и `docker-compose.yml`.

### Локальный запуск

```bash
docker compose up --build
```

Токен бот читает из `.env` (как при обычном запуске). База `data.db` хранится в Docker-томе `bot-data` и переживает пересборку контейнера.

### Деплой (Railway)

1. Создайте проект Railway из этого репозитория (Dockerfile определяется автоматически).
2. Добавьте Volume и примонтируйте его в путь `/data` (в коде это объявлено через `VOLUME`).
3. Задайте переменные окружения:
   - `TELEGRAM_BOT_TOKEN` — токен от @BotFather;
   - `STORAGE_PATH=/data/data.db` — база на постоянном томе.
4. Railway сам перезапустит процесс при падении; long-polling держит соединение с Telegram.
