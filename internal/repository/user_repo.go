package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"auth-app/internal/config"
	"auth-app/internal/models"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func InitDB(cfg *config.Config) (*sqlx.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	// Проверяем соединение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	// Устанавливаем настройки пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Successfully connected to PostgreSQL database")
	return db, nil
}

func CreateTables(db *sqlx.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			password TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating tables: %w", err)
	}

	log.Println("Database tables created/verified successfully")
	return nil
}

func (r *UserRepository) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (username, email, password) 
		VALUES ($1, $2, $3) 
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(
		query,
		user.Username,
		user.Email,
		user.Password,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, email, password, created_at, updated_at FROM users WHERE username = $1`

	err := r.db.Get(&user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, email, password, created_at, updated_at FROM users WHERE email = $1`

	err := r.db.Get(&user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUserByID(id int) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, email, password, created_at, updated_at FROM users WHERE id = $1`

	err := r.db.Get(&user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}
