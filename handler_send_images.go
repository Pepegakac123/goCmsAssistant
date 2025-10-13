package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type WPMediaResponse struct {
	ID        int    `json:"id"`
	SourceURL string `json:"source_url"`
	Title     struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	MediaType string `json:"media_type"`
	MimeType  string `json:"mime_type"`
}

type WebsiteType string

const (
	WebsiteTattoo WebsiteType = "tattoo"
	Website3D     WebsiteType = "3d"
)

func (w WebsiteType) IsValid() bool {
	return w == WebsiteTattoo || w == Website3D
}

type UploadResult struct {
	Filename     string `json:"filename"`
	WordPressID  int    `json:"wordpressId,omitempty"`
	WordPressURL string `json:"wordpressUrl,omitempty"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
}

func (cfg *apiConfig) sendImagesHandler(w http.ResponseWriter, r *http.Request) {
	// Parse website type
	webType, err := getWebsiteType(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid website type", err)
		return
	}

	// Read directory
	entries, err := os.ReadDir(cfg.tempRoot)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't read directory", err)
		return
	}

	var results []UploadResult

	// Upload each file
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Skip non-webp files (opcjonalnie)
		if filepath.Ext(entry.Name()) != ".webp" {
			continue
		}

		filePath := filepath.Join(cfg.tempRoot, entry.Name())

		// Upload to WordPress
		mediaResp, err := cfg.uploadToWordPress(filePath, webType)

		result := UploadResult{
			Filename: entry.Name(),
		}

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			fmt.Printf("Failed to upload %s: %v\n", entry.Name(), err)
		} else {
			result.Success = true
			result.WordPressID = mediaResp.ID
			result.WordPressURL = mediaResp.SourceURL
		}

		// if result.Success {
		// 	_, err = cfg.db.CreateUploadHistory(r.Context(), database.CreateUploadHistoryParams{
		// 		Filename:     result.Filename,
		// 		OriginalSize: int32(fileInfo.Size()),
		// 		WebpSize:     int32(webpSize), // musisz to śledzić
		// 		WordpressID:  sql.NullInt32{Int32: int32(mediaResp.ID), Valid: true},
		// 		WordpressURL: sql.NullString{String: mediaResp.SourceURL, Valid: true},
		// 		WebsiteType:  string(webType),
		// 		Success:      1,
		// 		UserID:       int32(userID), // z context
		// 	})
		// }

		results = append(results, result)
	}

	//Cleanup Temp folder
	cfg.cleanupImages()
	// Return results
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Upload process completed",
		"results": results,
	})
}

func (cfg *apiConfig) uploadToWordPress(filePath string, webType WebsiteType) (*WPMediaResponse, error) {
	// Select URL and password based on website type
	var url, appPwd string

	if webType == WebsiteTattoo {
		url = cfg.wpApi.tattoo.tattooUrl
		appPwd = cfg.wpApi.tattoo.tattooAppPwd
	} else {
		url = cfg.wpApi.threeD.threeDUrl
		appPwd = cfg.wpApi.threeD.threeAppPwd
	}

	// Construct full URL
	url = fmt.Sprintf("%s%s/media", url, cfg.wpApi.baseUrl)

	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't open file: %w", err)
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("couldn't read file: %w", err)
	}

	// Determine content type
	contentType := "image/jpeg"
	ext := filepath.Ext(filePath)
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	case ".gif":
		contentType = "image/gif"
	}

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewReader(fileBytes))
	if err != nil {
		return nil, fmt.Errorf("couldn't create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filePath)))
	req.SetBasicAuth(cfg.wpApi.user, appPwd)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code - WordPress returns 201 Created
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Decode response
	var mediaResponse WPMediaResponse
	err = json.NewDecoder(resp.Body).Decode(&mediaResponse) // ✅ resp.Body, nie req.Body!
	if err != nil {
		return nil, fmt.Errorf("couldn't decode response: %w", err)
	}

	return &mediaResponse, nil
}

func getWebsiteType(r *http.Request) (WebsiteType, error) {
	type params struct {
		Type string `json:"type"`
	}

	var p params
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		return "", fmt.Errorf("couldn't parse request body: %w", err)
	}

	webType := WebsiteType(p.Type)

	// Walidacja
	if !webType.IsValid() {
		return "", fmt.Errorf("invalid website type '%s' (use 'tattoo' or '3d')", p.Type)
	}

	return webType, nil
}

func (w *WPMediaResponse) GetTitle() string {
	return w.Title.Rendered
}
