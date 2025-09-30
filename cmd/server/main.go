package main

import (
	_ "auth-app/docs"
	"auth-app/internal/config"
	"auth-app/internal/handlers"
	"auth-app/internal/middleware"
	"auth-app/internal/repository"
	"auth-app/internal/service"
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

	// Инициализация сервисов
	jwtService := service.NewJWTService(cfg.JWTSecret, cfg.JWTIssuer)
	userRepo := repository.NewUserRepository(db)
	authHandler := handlers.NewAuthHandler(jwtService, userRepo)

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
