package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Pepegakac123/goCmsAssistant/internal/auth"
	"github.com/Pepegakac123/goCmsAssistant/internal/database"
)

type User struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	CreatedAt    any    `json:"created_at"`
	UpdatedAt    any    `json:"updated_at"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

func (cfg *apiConfig) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}
	type params struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	var p params
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse json", err)
		return
	}
	hashPwd, err := auth.HashPassword(p.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash password", err)
		return
	}
	p.Password = hashPwd

	validRoles := map[string]bool{"admin": true, "user": true}
	if !validRoles[p.Role] {
		respondWithError(w, http.StatusBadRequest, "Invalid role", nil)
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Name:           p.Username,
		HashedPassword: hashPwd,
		Role:           p.Role,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't register user", err)
		return
	}

	const defaultExpirationTime = time.Hour
	token, err := auth.MakeJWT(int(user.ID), cfg.token, defaultExpirationTime)
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
		Token:  refreshToken,
		UserID: user.ID,
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

func (cfg *apiConfig) createDefaultUser(req *http.Request) (User, error) {
	if cfg.platform != "dev" {
		return User{}, fmt.Errorf("you can not create default user anywhere else than in dev platform")
	}

	const defaultPassword = "admin"
	const defaultExpirationTime = time.Hour

	// Hash password
	hashPwd, err := auth.HashPassword(defaultPassword)
	if err != nil {
		return User{}, fmt.Errorf("couldn't hash password: %w", err)
	}

	// Create user
	user, err := cfg.db.CreateUser(req.Context(), database.CreateUserParams{
		Name:           "admin",
		HashedPassword: hashPwd,
		Role:           "admin",
	})
	if err != nil {
		return User{}, fmt.Errorf("couldn't create user: %w", err)
	}

	// Create JWT with user ID
	token, err := auth.MakeJWT(int(user.ID), cfg.token, defaultExpirationTime)
	if err != nil {
		return User{}, fmt.Errorf("couldn't create JWT: %w", err)
	}

	// Create refresh token
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		return User{}, fmt.Errorf("couldn't create refresh token: %w", err)
	}

	_, err = cfg.db.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: user.ID,
	})
	if err != nil {
		return User{}, fmt.Errorf("couldn't save refresh token: %w", err)
	}

	return User{
		ID:           int(user.ID),
		Name:         user.Name,
		Role:         user.Role,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Token:        token,
		RefreshToken: refreshToken,
	}, nil
}

func (cfg *apiConfig) resetAdminHandler(w http.ResponseWriter, req *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, http.StatusUnauthorized, "You can not reset user anywhere else than in dev PLATFORM", nil)
		return
	}

	err := cfg.db.DeleteAllUsers(req.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't reset users", err)
		return
	}

	defaultUser, err := cfg.createDefaultUser(req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create default user", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]any{
		"message": "Users reset successfully",
		"user":    defaultUser,
	})
}
