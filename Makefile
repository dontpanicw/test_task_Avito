# Переопределяемые переменные
GO ?= go
OAPI_CODEGEN ?= oapi-codegen
GOLANGCI_LINT ?= golangci-lint

# Пути
GOPATH_BIN := $(shell $(GO) env GOPATH)/bin
OAPI := $(GOPATH_BIN)/$(OAPI_CODEGEN)
GOLANGCI := $(GOPATH_BIN)/$(GOLANGCI_LINT)

# Файлы для генерации
OPENAPI_YAML := backend/api/openapi.yaml
GEN_TYPES_SERVER := backend/internal/input/http/gen/types.go
GEN_SERVER := backend/internal/input/http/gen/server.go
GEN_SPEC := backend/internal/input/http/gen/spec.go
GEN_TYPES_CLIENT := backend/pkg/client/http/types.go
GEN_CLIENT := backend/pkg/client/http/http_client.go

# Цели (phony)
.PHONY: deps generate build run test lint \
        docker-up docker-down docker-down-volumes \
        clean tools help

## Скачать зависимости
deps:
	@echo "Downloading Go modules..."
	@$(GO) mod download

## Сгенерировать код из OpenAPI
generate: $(OAPI) | $(dir $(GEN_TYPES_SERVER)) $(dir $(GEN_TYPES_CLIENT))
	@echo "Generating server types -> $(GEN_TYPES_SERVER)"
	@PATH=$(GOPATH_BIN):$$PATH $(OAPI) -generate types -package gen $(OPENAPI_YAML) > $(GEN_TYPES_SERVER)

	@echo "Generating chi-server -> $(GEN_SERVER)"
	@PATH=$(GOPATH_BIN):$$PATH $(OAPI) -generate chi-server,strict-server -package gen $(OPENAPI_YAML) > $(GEN_SERVER)

	@echo "Generating spec -> $(GEN_SPEC)"
	@PATH=$(GOPATH_BIN):$$PATH $(OAPI) -generate spec -package gen $(OPENAPI_YAML) > $(GEN_SPEC)

	@echo "Generating client types -> $(GEN_TYPES_CLIENT)"
	@PATH=$(GOPATH_BIN):$$PATH $(OAPI) -generate types -package client $(OPENAPI_YAML) > $(GEN_TYPES_CLIENT)

	@echo "Generating HTTP client -> $(GEN_CLIENT)"
	@PATH=$(GOPATH_BIN):$$PATH $(OAPI) -generate client -package client $(OPENAPI_YAML) > $(GEN_CLIENT)

## Сборка бинарника
build:
	@echo "Building server -> bin/server"
	@$(GO) build -o bin/server ./backend/cmd

## Запуск приложения
run: build
	@echo "Running server..."
	@./bin/server

## Запуск тестов
test:
	@echo "Running tests..."
	@$(GO) test ./...

## Линтинг
lint: $(GOLANGCI)
	@echo "Running linter..."
	@$(GOLANGCI) run ./...

## Поднять Docker
docker-up:
	@echo "Starting Docker services..."
	@docker compose up --build

## Остановить Docker
docker-down:
	@echo "Stopping Docker services..."
	@docker compose down

## Остановить Docker с удалением volume'ов
docker-down-volumes:
	@echo "Stopping Docker and removing volumes..."
	@docker compose down -v

## Очистка
clean:
	@echo "Cleaning bin/ and Go cache..."
	@rm -rf bin/
	@$(GO) clean

## Установка инструментов
tools: $(OAPI) $(GOLANGCI)