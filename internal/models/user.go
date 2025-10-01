package models

import "time"

// User represents user model
// @Description User information
type User struct {
	// User ID
	ID int `json:"id" example:"1" db:"id"`
	// Username
	Username string `json:"username" example:"john_doe" db:"username"`
	// Email address
	Email string `json:"email" example:"john@example.com" db:"email"`
	// Password (will be hidden in responses)
	Password string `json:"password" db:"password"`
	// Creation timestamp
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T00:00:00Z" db:"created_at"`
	// Last update timestamp
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T00:00:00Z" db:"updated_at"`
}

// LoginRequest represents login credentials
// @Description Login request payload
type LoginRequest struct {
	// Username for login
	Username string `json:"username" example:"john_doe"`
	// Password for login
	Password string `json:"password" example:"securepassword123"`
}

// AuthResponse represents authentication response
// @Description Authentication response with JWT token
type AuthResponse struct {
	// JWT token for authentication
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	// User information
	User User `json:"user"`
}

// ErrorResponse represents error response
// @Description Error response format
type ErrorResponse struct {
	// Error message
	Error string `json:"error" example:"Error description"`
}

// UpdateProfileRequest represents profile update data
// @Description Profile update request payload
type UpdateProfileRequest struct {
	// Username for update
	Username string `json:"username,omitempty" example:"new_username"`
	// Email for update
	Email string `json:"email,omitempty" example:"new_email@example.com"`
	// Password for update
	Password string `json:"password,omitempty" example:"newpassword123"`
}

// UpdateProfileResponse represents profile update response
// @Description Profile update response
type UpdateProfileResponse struct {
	// Success message
	Message string `json:"message" example:"Profile updated successfully"`
	// Updated user information
	User User `json:"user"`
}
