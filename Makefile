.PHONY: run build swag clean docker-up docker-down install-deps setup

# Переменные
BINARY_NAME=auth-app
GOBIN=$(shell go env GOBIN)
GOPATH=$(shell go env GOPATH)
SWAG_CMD=$(GOPATH)/bin/swag

# Проверка и установка swag
check-swag:
	@if [ ! -f "$(SWAG_CMD)" ]; then \
		echo "Swag not found. Installing..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi

# Установка зависимостей
install-deps:
	go mod tidy

# Запуск приложения
run:
	go run cmd/server/main.go

# Сборка
build:
	go build -o bin/$(BINARY_NAME) cmd/server/main.go

# Генерация Swagger документации
swag: check-swag
	$(SWAG_CMD) init -g cmd/server/main.go --output docs/ --parseDependency --parseInternal

# Запуск с генерацией Swagger
run-with-swag: swag run

# Очистка
clean:
	rm -rf bin/
	rm -rf docs/

# Запуск Docker
docker-up:
	docker-compose up -d

# Остановка Docker
docker-down:
	docker-compose down

# Установка всех зависимостей (первый запуск)
setup: install-deps swag

# Тесты
test:
	go test ./...

# Прямой запуск swag (если make все еще не работает)
swag-direct:
	go install github.com/swaggo/swag/cmd/swag@latest
	$(GOPATH)/bin/swag init -g cmd/server/main.go --output docs/ --parseDependency --parseInternal

help:
	@echo "Available targets:"
	@echo "  setup         - First time setup (install deps and generate swagger)"
	@echo "  install-deps  - Install dependencies"
	@echo "  run           - Run the application"
	@echo "  build         - Build the application"
	@echo "  swag          - Generate Swagger documentation"
	@echo "  swag-direct   - Direct swag generation (if make swag fails)"
	@echo "  run-with-swag - Run with Swagger generation"
	@echo "  docker-up     - Start PostgreSQL in Docker"
	@echo "  docker-down   - Stop PostgreSQL"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build artifacts"
