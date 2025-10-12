package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Pepegakac123/goCmsAssistant/internal/auth"
	"github.com/Pepegakac123/goCmsAssistant/internal/database"
)

const defaultAccessTokenExpiration = time.Hour
const defaultRefreshTokenExpiration = 24 * 28 * time.Hour // 28 dni

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var p params
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse json", err)
		return
	}
	user, err := cfg.db.GetUserByName(r.Context(), p.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Zwróć 401 i ukryj, czy problem to brak użytkownika, czy złe hasło
			respondWithError(w, http.StatusUnauthorized, "Username or password is wrong", nil)
			return
		}
		// Wszystkie inne błędy bazy danych traktuj jako błąd serwera
		respondWithError(w, http.StatusInternalServerError, "Couldn't get user", err)
		return
	}
	match, err := auth.CheckPasswordHash(p.Password, user.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't check password", err)
		return
	}
	if !match {
		respondWithError(w, http.StatusUnauthorized, "Username or password is wrong", nil)
		return
	}
	token, err := auth.MakeJWT(int(user.ID), cfg.token, defaultAccessTokenExpiration)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create JWT", err)
		return
	}
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create refresh token", err)
		return
	}
	_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(defaultRefreshTokenExpiration),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create refresh token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, User{
		ID:           int(user.ID),
		Name:         user.Name,
		Role:         user.Role,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Token:        token,
		RefreshToken: refreshToken,
	})
}

func (cfg *apiConfig) logoutHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Missing refresh token", err)
		return
	}

	err = cfg.db.RevokeToken(r.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Token nie istnieje lub już był unieważniony
			w.WriteHeader(http.StatusNoContent)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to revoke token", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (cfg *apiConfig) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Pobranie zweryfikowanego Refresh Tokena z kontekstu
	dbTokenRaw := r.Context().Value(contextKeyRefreshToken)
	dbToken, ok := dbTokenRaw.(database.RefreshToken)
	if !ok {
		respondWithError(w, http.StatusInternalServerError, "Internal server error: token not in context", nil)
		return
	}

	// 2. Generowanie NOWEGO Access Tokena (JWT - 1h)
	newAccessToken, err := auth.MakeJWT(int(dbToken.UserID), cfg.token, defaultAccessTokenExpiration)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create new Access Token", err)
		return
	}

	// 3. Generowanie NOWEGO Refresh Tokena (28 dni)
	newRefreshTokenStr, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create new Refresh Token string", err)
		return
	}

	_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     newRefreshTokenStr,
		UserID:    dbToken.UserID,
		ExpiresAt: time.Now().Add(defaultRefreshTokenExpiration),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save new refresh token", err)
		return
	}
	err = cfg.db.RevokeToken(r.Context(), dbToken.Token)
	if err != nil {
		// Loguj błąd, ale nie blokuj - nowy token już istnieje
		log.Printf("Warning: failed to revoke old token: %v", err)
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshTokenStr,
		"expires_in":    fmt.Sprintf("%v", defaultAccessTokenExpiration),
	})
}

func (cfg *apiConfig) revokeTokenHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Missing refresh token", err)
		return
	}

	err = cfg.db.RevokeToken(r.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Token nie istnieje lub już był unieważniony
			w.WriteHeader(http.StatusNoContent)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to revoke token", err)
		return
	}
	w.WriteHeader(http.StatusNoContent) // 204 No Content - standard dla udanego usunięcia/unieważnienia
}
