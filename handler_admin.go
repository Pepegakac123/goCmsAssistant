package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Pepegakac123/goCmsAssistant/internal/auth"
	"github.com/Pepegakac123/goCmsAssistant/internal/database"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
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
	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Name:           p.Username,
		HashedPassword: hashPwd,
		Role:           p.Role,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't register user", err)
		return
	}
	respondWithJSON(w, http.StatusOK, User{
		ID:   int(user.ID),
		Name: user.Name,
		Role: user.Role,
	})
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
	err = cfg.createDefaultUser(req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create default user", err)
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Users reset successfully",
	})
}

func (cfg *apiConfig) createDefaultUser(req *http.Request) error {
	if cfg.platform != "dev" {
		return fmt.Errorf("You can not create default user anywhere else than in dev PLATFORM")
	}
	hashPwd, err := auth.HashPassword("admin")
	if err != nil {
		return err
	}
	_, err = cfg.db.CreateUser(req.Context(), database.CreateUserParams{
		Name:           "admin",
		HashedPassword: hashPwd,
		Role:           "admin",
	})
	if err != nil {
		return err
	}
	return nil
}
