package main

import (
	"net/http"
	"os"
	"path/filepath"
)

func (cfg *apiConfig) cleanupImagesHandler(w http.ResponseWriter, r *http.Request) {
	cfg.cleanupImages()
	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Temp folder cleaned successfully",
	})
}

func (cfg *apiConfig) deleteImageHandler(w http.ResponseWriter, r *http.Request) {
	imgFilename := r.PathValue("filename")

	if imgFilename == "" {
		respondWithError(w, http.StatusBadRequest, "No filename provided", nil)
		return // ✅ DODAJ
	}

	filePath := filepath.Join(cfg.tempRoot, imgFilename)
	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			respondWithError(w, http.StatusNotFound, "File not found", err)
		} else {
			respondWithError(w, http.StatusInternalServerError, "Couldn't delete file", err)
		}
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{
		"message":  "File deleted successfully",
		"filename": imgFilename,
	})
}
