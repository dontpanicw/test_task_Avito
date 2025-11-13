# PR Reviewer Assignment Service

Сервис для автоматического назначения ревьюверов на Pull Request'ы.

## Описание

Микросервис, который автоматически назначает ревьюверов на Pull Request'ы из команды автора, позволяет выполнять переназначение ревьюверов и получать список PR'ов, назначенных конкретному пользователю, а также управлять командами и активностью пользователей.

## Архитектура

Проект использует Clean Architecture с разделением на слои:
- **entity** - доменные модели
- **port** - интерфейсы (репозитории и use cases)
- **usecase** - бизнес-логика
- **adapter** - реализации (PostgreSQL репозитории)
- **input/http** - HTTP handlers

## Требования

- Go 1.21+
- PostgreSQL 15+
- Docker и Docker Compose (для запуска через docker-compose)

## Быстрый старт

### Запуск через Docker Compose

Самый простой способ запустить сервис:

```bash
make docker-up
```

Или напрямую:

```bash
docker-compose up --build
```

Сервис будет доступен на `http://localhost:8080`

### Локальный запуск

1. Установите зависимости:

```bash
make deps
```

2. Установите инструменты для генерации кода (опционально):

```bash
make install-tools
```

3. Сгенерируйте код из OpenAPI спецификации:

```bash
make generate
```

4. Запустите PostgreSQL (через docker-compose или локально):

```bash
docker-compose up -d postgres
```

5. Соберите и запустите приложение:

```bash
make build
make run
```

Или установите переменные окружения и запустите:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=pr_reviewer
export SERVER_PORT=8080

go run ./cmd/server
```

## API Документация

OpenAPI спецификация находится в `api/openapi.yaml`.

После запуска сервиса документация доступна по адресу: `http://localhost:8080/openapi.yaml` (если настроено) или в файле `api/openapi.yaml`.

## Основные эндпоинты

- `POST /team/add` - Создать команду с участниками
- `GET /team/get?team_name=<name>` - Получить команду с участниками
- `POST /users/setIsActive` - Установить флаг активности пользователя
- `GET /users/getReview?user_id=<id>` - Получить PR'ы, где пользователь назначен ревьювером
- `POST /pullRequest/create` - Создать PR и автоматически назначить до 2 ревьюверов
- `POST /pullRequest/merge` - Пометить PR как MERGED
- `POST /pullRequest/reassign` - Переназначить ревьювера
- `GET /health` - Health check

## Переменные окружения

- `DB_HOST` - Хост базы данных (по умолчанию: localhost)
- `DB_PORT` - Порт базы данных (по умолчанию: 5432)
- `DB_USER` - Пользователь БД (по умолчанию: postgres)
- `DB_PASSWORD` - Пароль БД (по умолчанию: postgres)
- `DB_NAME` - Имя БД (по умолчанию: pr_reviewer)
- `SERVER_PORT` - Порт сервера (по умолчанию: 8080)

## Бизнес-логика

1. При создании PR автоматически назначаются **до двух** активных ревьюверов из **команды автора**, исключая самого автора.
2. Переназначение заменяет одного ревьювера на случайного **активного** участника **из команды заменяемого** ревьювера.
3. После `MERGED` менять список ревьюверов **нельзя**.
4. Если доступных кандидатов меньше двух, назначается доступное количество (0/1).
5. Пользователь с `isActive = false` не должен назначаться на ревью.

## Структура проекта

```
.
├── api/                    # OpenAPI спецификация
├── cmd/server/             # Точка входа приложения
├── internal/
│   ├── entity/            # Доменные модели
│   ├── port/              # Интерфейсы
│   ├── usecase/           # Бизнес-логика
│   ├── adapter/           # Реализации (репозитории)
│   └── input/http/        # HTTP handlers
├── pkg/
│   ├── migration/         # SQL миграции
│   └── client/            # HTTP клиент (сгенерирован)
├── docker-compose.yaml    # Docker Compose конфигурация
├── Dockerfile             # Docker образ
└── Makefile              # Команды для сборки и запуска
```

## Миграции

Миграции применяются автоматически при первом запуске PostgreSQL через docker-compose (файлы из `pkg/migration/*.up.sql` копируются в `/docker-entrypoint-initdb.d/`).

## Тестирование

```bash
make test
```

## Остановка сервиса

```bash
make docker-down
```

Для полной очистки (включая volumes):

```bash
make docker-down-volumes
```

## Примечания

- Security схемы (AdminToken, UserToken) определены в OpenAPI, но middleware для их проверки не реализован (для упрощения тестового задания). В production необходимо добавить проверку токенов.
- Health check эндпоинт добавлен для удобства мониторинга, хотя его нет в OpenAPI спецификации.

