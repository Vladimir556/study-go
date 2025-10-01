package handlers

import (
	"auth-app/internal/middleware"
	"auth-app/internal/models"
	"auth-app/internal/repository"
	"auth-app/internal/service"
	"context"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type AuthHandler struct {
	jwtService   *service.JWTService
	userRepo     *repository.UserRepository
	kafkaService *service.KafkaService
}

func NewAuthHandler(jwtService *service.JWTService, userRepo *repository.UserRepository, kafkaService *service.KafkaService) *AuthHandler {
	return &AuthHandler{
		jwtService:   jwtService,
		userRepo:     userRepo,
		kafkaService: kafkaService,
	}
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param user body models.User true "User registration data"
// @Success 201 {object} map[string]interface{} "User created successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid request body"
// @Failure 409 {object} models.ErrorResponse "User already exists"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if user.Username == "" || user.Password == "" || user.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Username, password and email are required")
		return
	}

	// Проверяем существование пользователя
	existingUser, err := h.userRepo.GetUserByUsername(user.Username)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error checking user existence")
		return
	}
	if existingUser != nil {
		respondWithError(w, http.StatusConflict, "Username already exists")
		return
	}

	// Проверяем email
	existingUser, err = h.userRepo.GetUserByEmail(user.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error checking email existence")
		return
	}
	if existingUser != nil {
		respondWithError(w, http.StatusConflict, "Email already exists")
		return
	}

	// Хешируем пароль
	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user")
		return
	}

	user.Password = hashedPassword

	// Сохраняем в БД
	if err := h.userRepo.CreateUser(&user); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user")
		return
	}

	// Отправляем событие в Kafka
	if h.kafkaService != nil {
		eventData := models.UserRegisteredEvent{
			UserID:    user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		}

		go func() {
			ctx := context.Background()
			if err := h.kafkaService.ProduceEvent(ctx, models.UserRegistered, eventData); err != nil {
				log.Printf("Failed to produce registration event: %v", err)
			}
		}()
	}

	// Очищаем пароль в ответе
	user.Password = ""

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User created successfully",
		"user":    user,
	})
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.AuthResponse "Login successful"
// @Failure 400 {object} models.ErrorResponse "Invalid request body"
// @Failure 401 {object} models.ErrorResponse "Invalid credentials"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var loginReq models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Находим пользователя
	user, err := h.userRepo.GetUserByUsername(loginReq.Username)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error finding user")
		return
	}
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Проверяем пароль
	if !checkPasswordHash(loginReq.Password, user.Password) {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Генерируем токен
	token, err := h.jwtService.GenerateToken(user.ID, user.Username)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error generating token")
		return
	}

	// Создаем ответ без пароля
	userResponse := models.User{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	response := models.AuthResponse{
		Token: token,
		User:  userResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Отправляем событие в Kafka
	if h.kafkaService != nil {
		eventData := models.UserLoggedInEvent{
			UserID:    user.ID,
			Username:  user.Username,
			LoginTime: time.Now(),
			IPAddress: getIPAddress(r),
		}

		go func() {
			ctx := context.Background()
			if err := h.kafkaService.ProduceEvent(ctx, models.UserLoggedIn, eventData); err != nil {
				log.Printf("Failed to produce login event: %v", err)
			}
		}()
	}
}

// Profile godoc
// @Summary Get user profile
// @Description Get current user profile information
// @Tags user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User "User profile"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 404 {object} models.ErrorResponse "User not found"
// @Router /profile [get]
func (h *AuthHandler) Profile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r)
	if claims == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Отправляем событие просмотра профиля
	if h.kafkaService != nil {
		eventData := models.UserProfileViewedEvent{
			UserID:    claims.UserID,
			ViewTime:  time.Now(),
			UserAgent: r.UserAgent(),
		}

		go func() {
			ctx := context.Background()
			if err := h.kafkaService.ProduceEvent(ctx, models.UserProfileViewed, eventData); err != nil {
				log.Printf("Failed to produce profile view event: %v", err)
			}
		}()
	}

	user, err := h.userRepo.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Очищаем пароль
	user.Password = ""

	json.NewEncoder(w).Encode(user)
}

// Вспомогательная функция для получения IP
func getIPAddress(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	return r.RemoteAddr
}

// HealthCheck godoc
// @Summary Health check
// @Description Check API health status
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "API is healthy"
// @Failure 503 {object} map[string]string "Service unavailable"
// @Router /health [get]
func (h *AuthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// Здесь можно добавить проверку подключения к БД
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "message": "API is running"})
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.ErrorResponse{Error: message})
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update current user profile information
// @Tags user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param updateData body models.UpdateProfileRequest true "Profile update data"
// @Success 200 {object} models.UpdateProfileResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /profile [put]
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r)
	if claims == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var updateReq models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация - хотя бы одно поле должно быть заполнено
	if updateReq.Username == "" && updateReq.Email == "" && updateReq.Password == "" {
		respondWithError(w, http.StatusBadRequest, "At least one field (username, email, or password) must be provided")
		return
	}

	// Подготавливаем данные для обновления
	updateData := make(map[string]interface{})

	// Проверка и подготовка username
	if updateReq.Username != "" {
		if len(updateReq.Username) < 3 {
			respondWithError(w, http.StatusBadRequest, "Username must be at least 3 characters long")
			return
		}

		// Проверяем что username не занят другим пользователем
		exists, err := h.userRepo.CheckUsernameExists(updateReq.Username, claims.UserID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error checking username availability")
			return
		}
		if exists {
			respondWithError(w, http.StatusConflict, "Username already taken")
			return
		}

		updateData["username"] = updateReq.Username
	}

	// Проверка и подготовка email
	if updateReq.Email != "" {
		if !isValidEmail(updateReq.Email) {
			respondWithError(w, http.StatusBadRequest, "Invalid email format")
			return
		}

		// Проверяем что email не занят другим пользователем
		exists, err := h.userRepo.CheckEmailExists(updateReq.Email, claims.UserID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error checking email availability")
			return
		}
		if exists {
			respondWithError(w, http.StatusConflict, "Email already registered")
			return
		}

		updateData["email"] = updateReq.Email
	}

	// Подготовка password
	if updateReq.Password != "" {
		if len(updateReq.Password) < 6 {
			respondWithError(w, http.StatusBadRequest, "Password must be at least 6 characters long")
			return
		}

		hashedPassword, err := hashPassword(updateReq.Password)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error processing password")
			return
		}
		updateData["password"] = hashedPassword
	}

	// Обновляем пользователя в БД
	if err := h.userRepo.UpdateUser(claims.UserID, updateData); err != nil {
		log.Printf("Error updating user: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Error updating profile")
		return
	}

	// Отправляем событие в Kafka (если есть изменения)
	if h.kafkaService != nil && (updateReq.Username != "" || updateReq.Email != "") {
		eventData := models.UserProfileUpdatedEvent{
			UserID:        claims.UserID,
			UpdatedAt:     time.Now(),
			UpdatedFields: getUpdatedFields(updateReq),
		}

		go func() {
			ctx := context.Background()
			if err := h.kafkaService.ProduceEvent(ctx, "user.profile.updated", eventData); err != nil {
				log.Printf("Failed to produce profile update event: %v", err)
			}
		}()
	}

	// Получаем обновленные данные пользователя
	updatedUser, err := h.userRepo.GetUserByID(claims.UserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching updated profile")
		return
	}

	// Очищаем пароль в ответе
	updatedUser.Password = ""

	response := models.UpdateProfileResponse{
		Message: "Profile updated successfully",
		User:    *updatedUser,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Вспомогательные функции
func isValidEmail(email string) bool {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(emailRegex, email)
	return matched
}

func getUpdatedFields(updateReq models.UpdateProfileRequest) []string {
	var fields []string
	if updateReq.Username != "" {
		fields = append(fields, "username")
	}
	if updateReq.Email != "" {
		fields = append(fields, "email")
	}
	if updateReq.Password != "" {
		fields = append(fields, "password")
	}
	return fields
}
