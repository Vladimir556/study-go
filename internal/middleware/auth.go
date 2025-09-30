package middleware

import (
	"context"
	"net/http"

	"auth-app/internal/service"
)

func AuthMiddleware(jwtService *service.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := r.Header.Get("Authorization")
			if tokenString == "" {
				http.Error(w, `{"error": "Authorization header required"}`, http.StatusUnauthorized)
				return
			}

			if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
				tokenString = tokenString[7:]
			}

			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), "userClaims", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserFromContext(r *http.Request) *service.Claims {
	claims, ok := r.Context().Value("userClaims").(*service.Claims)
	if !ok {
		return nil
	}
	return claims
}
