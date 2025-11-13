.PHONY: build run test clean docker-up docker-down generate

# Генерация кода из OpenAPI спецификации
generate:
	@echo "Generating code from OpenAPI spec..."
	@go generate ./api

# Запуск тестов
test:
	@echo "Running tests..."
	@go test ./...

# Запуск через docker-compose
docker-up:
	@echo "Starting services with docker-compose..."
	@docker-compose up --build

# Остановка docker-compose
docker-down:
	@echo "Stopping services..."
	@docker-compose down

# Остановка и удаление volumes
docker-down-volumes:
	@echo "Stopping services and removing volumes..."
	@docker-compose down -v

# Установка зависимостей
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Установка инструментов для разработки
install-tools:
	@echo "Installing development tools..."
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

