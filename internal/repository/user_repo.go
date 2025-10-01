package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
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

// UpdateUser updates user profile information
func (r *UserRepository) UpdateUser(userID int, updateData map[string]interface{}) error {
	if len(updateData) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Базовый запрос
	query := "UPDATE users SET "

	// Собираем SET части
	setParts := []string{}
	params := []interface{}{}
	paramCount := 1

	for field, value := range updateData {
		if field == "password" {
			setParts = append(setParts, fmt.Sprintf("password = $%d", paramCount))
		} else {
			setParts = append(setParts, fmt.Sprintf("%s = $%d", field, paramCount))
		}
		params = append(params, value)
		paramCount++
	}

	// Добавляем updated_at
	setParts = append(setParts, "updated_at = CURRENT_TIMESTAMP")

	// Собираем полный запрос
	query += strings.Join(setParts, ", ")
	query += fmt.Sprintf(" WHERE id = $%d", paramCount)
	params = append(params, userID)

	result, err := r.db.Exec(query, params...)
	if err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// CheckUsernameExists checks if username exists (excluding current user)
func (r *UserRepository) CheckUsernameExists(username string, excludeUserID int) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND id != $2)"

	err := r.db.Get(&exists, query, username, excludeUserID)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// CheckEmailExists checks if email exists (excluding current user)
func (r *UserRepository) CheckEmailExists(email string, excludeUserID int) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND id != $2)"

	err := r.db.Get(&exists, query, email, excludeUserID)
	if err != nil {
		return false, err
	}

	return exists, nil
}
