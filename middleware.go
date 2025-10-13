package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Pepegakac123/goCmsAssistant/internal/auth"
)

type contextKey string

const (
	contextKeyUserID       contextKey = "userID"
	contextKeyUserRole     contextKey = "userRole"
	contextKeyRefreshToken contextKey = "refreshToken"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "The token is invalid", err)
			return
		}
		userId, err := auth.ValidateJWT(token, cfg.token)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "The token is invalid", err)
			return
		}
		// ✅ Sprawdź czy użytkownik nadal istnieje
		_, err = cfg.db.GetUser(r.Context(), int64(userId))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				respondWithError(w, http.StatusUnauthorized, "User not found", nil)
				return
			}
			respondWithError(w, http.StatusInternalServerError, "Database error", err)
			return
		}
		ctx := context.WithValue(r.Context(), contextKeyUserID, userId)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
func (cfg *apiConfig) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		userID, ok := r.Context().Value(contextKeyUserID).(int)

		if !ok {
			respondWithError(w, http.StatusUnauthorized, "User context error", nil)
			return
		}

		if userID == 0 {
			respondWithError(w, http.StatusUnauthorized, "Invalid user ID", nil)
			return
		}

		role, err := cfg.db.GetUserRoleById(r.Context(), int64(userID))
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Could not determine user role", nil)
			return
		}

		if role != "admin" {
			// Użytkownik ma rolę, ale nie jest adminem
			respondWithError(w, http.StatusForbidden, "Access Forbidden: User is not an administrator", nil)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyUserRole, role)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
func (cfg *apiConfig) refreshTokenValidationMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing refresh token in Authorization header", err)
			return
		}
		dbToken, err := cfg.db.GetRefreshToken(r.Context(), refreshToken)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				respondWithError(w, http.StatusUnauthorized, "Invalid refresh token", nil)
				return
			}
			respondWithError(w, http.StatusInternalServerError, "Database error during token lookup", err)
			return
		}

		if dbToken.RevokedAt != nil {
			respondWithError(w, http.StatusUnauthorized, "Refresh token has been revoked", nil)
			return
		}

		if time.Now().After(dbToken.ExpiresAt) {
			respondWithError(w, http.StatusUnauthorized, "Refresh token has expired", nil)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyRefreshToken, dbToken)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
