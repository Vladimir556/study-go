package handlers

import (
	"auth-app/internal/middleware"
	"auth-app/internal/models"
	"auth-app/internal/repository"
	"auth-app/internal/service"
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	jwtService *service.JWTService
	userRepo   *repository.UserRepository
}

func NewAuthHandler(jwtService *service.JWTService, userRepo *repository.UserRepository) *AuthHandler {
	return &AuthHandler{
		jwtService: jwtService,
		userRepo:   userRepo,
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

	user, err := h.userRepo.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Очищаем пароль
	user.Password = ""

	json.NewEncoder(w).Encode(user)
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
