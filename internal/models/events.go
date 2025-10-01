package models

import "time"

type EventType string

const (
	UserRegistered     EventType = "user.registered"
	UserLoggedIn       EventType = "user.logged.in"
	UserProfileViewed  EventType = "user.profile.viewed"
	UserProfileUpdated EventType = "user.profile.updated"
)

type Event struct {
	ID        string      `json:"id"`
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Source    string      `json:"source"`
}

type UserRegisteredEvent struct {
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type UserLoggedInEvent struct {
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	LoginTime time.Time `json:"login_time"`
	IPAddress string    `json:"ip_address,omitempty"`
}

type UserProfileViewedEvent struct {
	UserID    int       `json:"user_id"`
	ViewedBy  int       `json:"viewed_by,omitempty"`
	ViewTime  time.Time `json:"view_time"`
	UserAgent string    `json:"user_agent,omitempty"`
}

type UserProfileUpdatedEvent struct {
	UserID        int       `json:"user_id"`
	UpdatedAt     time.Time `json:"updated_at"`
	UpdatedFields []string  `json:"updated_fields"`
}
