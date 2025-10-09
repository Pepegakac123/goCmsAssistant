package main

import (
	"fmt"
	"image"
	_ "image/jpeg" // Ważne! Zarejestruj dekodery
	_ "image/png"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chai2010/webp"
	"github.com/google/uuid"
	"github.com/jdeng/goheif"
	"github.com/nfnt/resize"
)

type Images struct {
	Images []ImageInfo `json:"images"`
}

type ImageInfo struct {
	OriginalSize int    `json:"originalSize"`
	WebpSize     int    `json:"webpSize"`
	Filename     string `json:"filename"`
}

// Główny handler
func (cfg *apiConfig) uploadImagesHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hitted\n")
	const uploadLimit = 1 << 30 // 1 GB

	r.Body = http.MaxBytesReader(w, r.Body, uploadLimit)
	err := r.ParseMultipartForm(uploadLimit)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		respondWithError(w, http.StatusBadRequest, "No images provided", nil)
		return
	}

	var results []ImageInfo

	for _, fileHeader := range files {
		imageInfo, err := cfg.processImage(fileHeader)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error(), err)
			return
		}
		results = append(results, imageInfo)
	}

	response := Images{Images: results}
	respondWithJSON(w, http.StatusOK, response)
}

// Przetwarzanie pojedynczego obrazu
func (cfg *apiConfig) processImage(fileHeader *multipart.FileHeader) (ImageInfo, error) {
	start := time.Now()
	defer func() {
		log.Printf("Przetworzono %s w %v\n", fileHeader.Filename, time.Since(start))
	}()

	const maxWidth = 1920
	const maxHeight = 1080

	log.Printf("1. Walidacja typu...")
	mediaType, err := validateImageType(fileHeader)
	if err != nil {
		return ImageInfo{}, err
	}
	log.Printf("   Typ: %s (czas: %v)\n", mediaType, time.Since(start))

	log.Printf("2. Otwieranie pliku...")
	file, err := fileHeader.Open()
	if err != nil {
		return ImageInfo{}, fmt.Errorf("couldn't open file: %w", err)
	}
	defer file.Close()
	log.Printf("   Otwarto (czas: %v)\n", time.Since(start))

	originalSize := fileHeader.Size

	log.Printf("3. Dekodowanie i resize...")
	decodeStart := time.Now()
	img, err := decodeAndResize(file, maxWidth, maxHeight, mediaType)
	if err != nil {
		return ImageInfo{}, err
	}
	log.Printf("   Dekodowanie zajęło: %v\n", time.Since(decodeStart))

	log.Printf("4. Zapisywanie jako WebP...")
	encodeStart := time.Now()
	filename := fmt.Sprintf("%s.webp", uuid.New().String())
	webpSize, err := cfg.saveAsWebP(img, filename)
	if err != nil {
		return ImageInfo{}, err
	}
	log.Printf("   Encoding WebP zajął: %v\n", time.Since(encodeStart))

	return ImageInfo{
		OriginalSize: int(originalSize),
		WebpSize:     int(webpSize),
		Filename:     filename,
	}, nil
}

// Walidacja typu pliku
func validateImageType(fileHeader *multipart.FileHeader) (string, error) {
	contentType := fileHeader.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", fmt.Errorf("couldn't parse media type: %w", err)
	}

	validTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
		"image/heic": true,
		"image/heif": true,
	}

	if !validTypes[mediaType] {
		return "", fmt.Errorf("unsupported file type: %s", mediaType)
	}

	return mediaType, nil
}

// Dekodowanie i resize obrazu
func decodeAndResize(file multipart.File, maxWidth, maxHeight uint, mediaType string) (image.Image, error) {
	// Dekoduj
	var img image.Image
	var err error

	if mediaType == "image/heic" || mediaType == "image/heif" {
		img, err = goheif.Decode(file)
		if err != nil {
			log.Fatalf("Failed to parse file: %v\n", err)
		}
	} else {
		img, _, err = image.Decode(file)
		if err != nil {
			return nil, fmt.Errorf("couldn't decode image: %w", err)
		}
	}

	// Sprawdź czy resize jest potrzebny
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width > int(maxWidth) || height > int(maxHeight) {
		img = resize.Thumbnail(maxWidth, maxHeight, img, resize.Lanczos3)
	}

	return img, nil
}

// Zapisz obraz jako WebP
func (cfg *apiConfig) saveAsWebP(img image.Image, filename string) (int64, error) {
	outputPath := filepath.Join(cfg.tempRoot, filename)

	outFile, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("couldn't create output file: %w", err)
	}
	defer outFile.Close()

	options := &webp.Options{
		Lossless: false,
		Quality:  80,
	}

	if err := webp.Encode(outFile, img, options); err != nil {
		os.Remove(outputPath) // Cleanup on error
		return 0, fmt.Errorf("couldn't encode to WebP: %w", err)
	}

	// Sprawdź rozmiar
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		return 0, fmt.Errorf("couldn't stat WebP file: %w", err)
	}

	return fileInfo.Size(), nil
}
