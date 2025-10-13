package main

import (
	"context"
	"fmt"
	"log"
	"os"
)

func (cfg *apiConfig) cleanupImages() {
	os.RemoveAll(cfg.tempRoot)
	os.MkdirAll(cfg.tempRoot, 0755)
}

func (cfg *apiConfig) ensureDefaultAdmin(ctx context.Context) error {
	// Sprawdź czy są jacyś użytkownicy
	count, err := cfg.db.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("couldn't count users: %w", err)
	}

	// Jeśli baza jest pusta, utwórz domyślnego admina
	if count == 0 {
		log.Println("📝 No users found, creating default admin...")

		// ✅ Wykorzystujemy istniejącą funkcję!
		user, err := cfg.createDefaultUser(ctx)
		if err != nil {
			return fmt.Errorf("couldn't create default user: %w", err)
		}

		log.Printf("✅ Default admin created (ID: %d, username: admin, password: admin)\n", user.ID)
		log.Println("⚠️  CHANGE THE DEFAULT PASSWORD IMMEDIATELY!")
	}

	return nil
}
