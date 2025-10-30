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
	"sync"
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

// Job reprezentuje jedno zadanie do przetworzenia
type Job struct {
	FileHeader *multipart.FileHeader
	Index      int
}

// Result reprezentuje wynik przetworzenia
type Result struct {
	ImageInfo ImageInfo
	Error     error
	Index     int
}

// Główny handler
func (cfg *apiConfig) uploadImagesHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hitted\n")
	startTime := time.Now()

	const uploadLimit = 1 << 30 // 1 GB
	const numWorkers = 4        // Liczba równoczesnych przetwarzań

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

	log.Printf("Przetwarzam %d plików używając %d workerów...\n", len(files), numWorkers)

	// Kanały do komunikacji
	jobs := make(chan Job, len(files))
	results := make(chan Result, len(files))

	// WaitGroup do czekania na zakończenie wszystkich workerów
	var wg sync.WaitGroup

	// Uruchom workerów
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				log.Printf("[Worker %d] Przetwarzam %s...\n", workerID, job.FileHeader.Filename)

				imageInfo, err := cfg.processImage(job.FileHeader)
				results <- Result{
					ImageInfo: imageInfo,
					Error:     err,
					Index:     job.Index,
				}
			}
		}(i)
	}

	// Wyślij zadania do workerów
	go func() {
		for i, fileHeader := range files {
			jobs <- Job{
				FileHeader: fileHeader,
				Index:      i,
			}
		}
		close(jobs) // Zamknij kanał gdy wszystkie zadania wysłane
	}()

	// Zamknij kanał results gdy wszyscy workerzy skończą
	go func() {
		wg.Wait()
		close(results)
	}()

	// Zbierz wyniki (zachowaj oryginalną kolejność)
	resultSlice := make([]Result, len(files))
	for result := range results {
		resultSlice[result.Index] = result
	}

	// Sprawdź błędy i zbuduj odpowiedź
	var finalResults []ImageInfo
	for _, result := range resultSlice {
		if result.Error != nil {
			respondWithError(w, http.StatusBadRequest, result.Error.Error(), result.Error)
			return
		}
		finalResults = append(finalResults, result.ImageInfo)
	}

	elapsed := time.Since(startTime)
	log.Printf("✓ Przetworzono %d plików w %v (%.2f plików/s)\n",
		len(files), elapsed, float64(len(files))/elapsed.Seconds())

	response := Images{Images: finalResults}
	respondWithJSON(w, http.StatusOK, response)
}

// Przetwarzanie pojedynczego obrazu
func (cfg *apiConfig) processImage(fileHeader *multipart.FileHeader) (ImageInfo, error) {
	start := time.Now()
	defer func() {
		log.Printf("Przetworzono %s w %v\n", fileHeader.Filename, time.Since(start))
	}()

	const maxWidth = 2560
	const maxHeight = 1440

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
			return nil, fmt.Errorf("couldn't decode HEIC: %w", err)
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
