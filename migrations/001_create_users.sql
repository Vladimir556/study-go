-- migrations/001_create_users.sql

-- Создание таблицы пользователей
CREATE TABLE IF NOT EXISTS users (
                                     id SERIAL PRIMARY KEY,
                                     username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                             );

-- Индексы для улучшения производительности
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$ language 'plpgsql';

CREATE OR REPLACE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
                         FOR EACH ROW
                         EXECUTE FUNCTION update_updated_at_column();