package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type apiConfig struct {
	port       string
	assetsRoot string
	tempRoot   string
}

func main() {
	godotenv.Load(".env")

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable is not set")
	}
	assetsRoot := os.Getenv("ASSETS_ROOT")
	if assetsRoot == "" {
		log.Fatal("ASSETS_ROOT environment variable is not set")
	}

	tempRoot := os.Getenv("TEMP_ROOT")
	if tempRoot == "" {
		log.Fatal("TEMP_ROOT environment variable is not set")
	}

	cfg := apiConfig{
		port:       port,
		assetsRoot: assetsRoot,
		tempRoot:   tempRoot,
	}

	err := cfg.ensureDirs()
	if err != nil {
		log.Fatalf("Couldn't create assets directory: %v", err)
	}

	mux := http.NewServeMux()
	// mux.Handle("/",)
	assetsHandler := http.StripPrefix("/assets", http.FileServer(http.Dir(assetsRoot)))

	mux.Handle("/assets/", assetsHandler)

	mux.HandleFunc("POST /api/images/upload", cfg.uploadImagesHandler)
	mux.HandleFunc("DELETE /api/images/delete/{filename}", cfg.deleteImageHandler)
	mux.HandleFunc("DELETE /api/images/tmp/cleanup", cfg.cleanupImagesHandler)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: loggingMiddleware(mux),
	}

	log.Printf("Serving on: http://localhost:%s/\n", port)
	log.Fatal(srv.ListenAndServe())
}
