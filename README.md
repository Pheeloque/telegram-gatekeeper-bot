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

После выбора группы можно добавить публичный канал по `@username`, посмотреть чёрный список или удалить канал.

## Примечание о пересылках

Модерация использует `Message.forward_origin` и ищет `MessageOriginChannel`. Ограничения Bot API на доступность этой информации определяются самим Telegram; они не снимаются кодом бота.

## Хранение

Настройки сохраняются в `data.json`. Для production с несколькими экземплярами бота лучше заменить JSON на PostgreSQL.

## Проверка

После установки зависимостей рекомендуется выполнить:

```bash
go test ./...
```
