package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Pepegakac123/goCmsAssistant/internal/auth"
)

type contextKey string

const (
	contextKeyUserID   contextKey = "userID"
	contextKeyUserRole contextKey = "userRole"
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
		ctx := context.WithValue(r.Context(), contextKeyUserID, userId)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
func (cfg *apiConfig) userAdminMiddleware(next http.Handler) http.Handler {
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

		role, err := cfg.db.GetUserRoleById(r.Context(), int32(userID))
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
