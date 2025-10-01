package main

import (
	_ "auth-app/docs"
	"auth-app/internal/config"
	"auth-app/internal/handlers"
	"auth-app/internal/middleware"
	"auth-app/internal/models"
	"auth-app/internal/repository"
	"auth-app/internal/service"
	"context"
	"github.com/swaggo/http-swagger"
	"log"
	"net/http"
	"time"
)

// @title Auth API
// @version 1.0
// @description JWT Authentication API with PostgreSQL
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@authapp.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Authorization header using the Bearer scheme
func main() {
	// Загрузка конфигурации
	cfg := config.Load()

	log.Printf("=== CONFIG DEBUG ===")
	log.Printf("DB: %s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	log.Printf("Kafka: Enabled=%v, Brokers=%s", cfg.KafkaEnabled, cfg.KafkaBrokers)
	log.Printf("====================")

	// Инициализация базы данных
	db, err := repository.InitDB(cfg)
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer db.Close()

	// Создание таблиц
	if err := repository.CreateTables(db); err != nil {
		log.Fatalf("Error creating tables: %v", err)
	}

	// Инициализация Kafka
	kafkaService := service.NewKafkaService(cfg)
	defer kafkaService.Close()

	// Запуск consumers
	ctx := context.Background()

	// Consumer для событий регистрации
	kafkaService.ConsumeEvents(ctx, "user-registered", func(event models.Event) error {
		log.Printf("Received user registration event: %+v", event)
		// Здесь можно добавить обработку события
		// Например, отправка welcome email, создание профиля и т.д.
		return nil
	})

	// Consumer для событий входа
	kafkaService.ConsumeEvents(ctx, "user-logged-in", func(event models.Event) error {
		log.Printf("Received user login event: %+v", event)
		// Например, обновление last_login, аналитика и т.д.
		return nil
	})

	// Инициализация сервисов
	jwtService := service.NewJWTService(cfg.JWTSecret, cfg.JWTIssuer)
	userRepo := repository.NewUserRepository(db)
	// Инициализация handlers с Kafka
	authHandler := handlers.NewAuthHandler(jwtService, userRepo, kafkaService)

	// Настройка маршрутов
	mux := http.NewServeMux()

	// Swagger документация
	if cfg.EnableSwagger {
		mux.Handle("GET /swagger/", httpSwagger.Handler(
			httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
		))
	}

	// Публичные маршруты
	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("GET /health", authHandler.HealthCheck)

	// Защищенные маршруты
	protectedProfile := http.HandlerFunc(authHandler.Profile)
	mux.Handle("GET /profile", middleware.AuthMiddleware(jwtService)(protectedProfile))

	protectedUpdate := http.HandlerFunc(authHandler.UpdateProfile)
	mux.Handle("PUT /profile", middleware.AuthMiddleware(jwtService)(protectedUpdate))

	// Запуск сервера
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on :%s", cfg.ServerPort)
	log.Printf("Environment: %s", cfg.Env)
	if cfg.EnableSwagger {
		log.Printf("Swagger UI: http://localhost:%s/swagger/index.html", cfg.ServerPort)
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
