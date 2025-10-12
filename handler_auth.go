package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Pepegakac123/goCmsAssistant/internal/auth"
	"github.com/Pepegakac123/goCmsAssistant/internal/database"
)

const defaultAccessTokenExpiration = time.Hour
const defaultRefreshTokenExpiration = 24 * 28 * time.Hour // 28 dni

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Hello World",
	})
}

func (cfg *apiConfig) logoutHandler(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Hello World",
	})
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

	// 4. Unieważnienie STAREGO Tokena (dla jednokrotnego użycia refresh tokena - bezpieczeństwo)
	err = cfg.db.RevokeToken(r.Context(), dbToken.Token)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to revoke old refresh token", err)
		return
	}

	// 5. Zapisanie NOWEGO Refresh Tokena w bazie
	_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     newRefreshTokenStr,
		UserID:    dbToken.UserID,
		ExpiresAt: time.Now().Add(defaultRefreshTokenExpiration),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save new refresh token", err)
		return
	}

	// 6. Zwrócenie NOWEJ pary tokenów
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
		if err.Error() == "sql: no rows in result set" {
			// Zwracamy 404/204 dla niezmienionego stanu, by nie zdradzać istnienia tokena
			w.WriteHeader(http.StatusNoContent)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to revoke token", err)
		return
	}
	w.WriteHeader(http.StatusNoContent) // 204 No Content - standard dla udanego usunięcia/unieważnienia
}
