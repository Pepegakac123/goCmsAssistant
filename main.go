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
	wpApi      wpApi
}
type tattooWpDestination struct {
	tattooUrl      string `json:"tattooUrl"`
	tattooHostname string `json:"tattooHostname"`
	tattooAppPwd   string `json:"tattooAppPwd"`
}

type threeDWpDestination struct {
	threeDUrl      string `json:"threeDUrl"`
	threeDHostname string `json:"threeDHostname"`
	threeAppPwd    string `json:"threeAppPwd"`
}

type wpApi struct {
	tattoo  tattooWpDestination `json:"tattoo"`
	threeD  threeDWpDestination `json:"threeD"`
	baseUrl string              `json:"baseUrl"`
	user    string              `json:"user"`
}

func main() {

	cfg := loadEnv()

	err := cfg.ensureDirs()
	if err != nil {
		log.Fatalf("Couldn't create assets directory: %v", err)
	}

	mux := http.NewServeMux()
	// mux.Handle("/",)
	assetsHandler := http.StripPrefix("/assets", http.FileServer(http.Dir(cfg.assetsRoot)))

	mux.Handle("/assets/", assetsHandler)
	mux.HandleFunc("GET /api", cfg.indexHandler)
	mux.HandleFunc("POST /api/images/upload", cfg.uploadImagesHandler)
	mux.HandleFunc("DELETE /api/images/delete/{filename}", cfg.deleteImageHandler)
	mux.HandleFunc("DELETE /api/images/tmp/cleanup", cfg.cleanupImagesHandler)
	mux.HandleFunc("POST /api/images/send", cfg.sendImagesHandler)

	srv := &http.Server{
		Addr:    "0.0.0.0:" + cfg.port, // ✅ Jawnie IPv4
		Handler: mux,
	}

	log.Printf("Serving on: http://localhost:%s/\n", cfg.port)
	log.Fatal(srv.ListenAndServe())
}
func (cfg *apiConfig) indexHandler(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Hello World",
	})
}
func loadEnv() apiConfig {
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

	wpTattooUrl := os.Getenv("WORDPRESS_TATTOO_URL")
	if wpTattooUrl == "" {
		log.Fatal("WORDPRESS_TATTOO_URL environment variable is not set")
	}
	wpTattooHostname := os.Getenv("WORDPRESS_TATTOO_HOSTNAME")
	if wpTattooHostname == "" {
		log.Fatal("WORDPRESS_TATTOO_HOSTNAME environment variable is not set")
	}
	wpTattooAppPwd := os.Getenv("WORDPRESS_TATTOO_APP_PWD")
	if wpTattooAppPwd == "" {
		log.Fatal("WORDPRESS_TATTOO_APP_PWD environment variable is not set")
	}
	wp3DUrl := os.Getenv("WORDPRESS_3D_URL")
	if wp3DUrl == "" {
		log.Fatal("WORDPRESS_3D_URL environment variable is not set")
	}
	wp3DHostname := os.Getenv("WORDPRESS_3D_HOSTNAME")
	if wp3DHostname == "" {
		log.Fatal("WORDPRESS_3D_HOSTNAME environment variable is not set")
	}
	wp3DAppPwd := os.Getenv("WORDPRESS_3D_APP_PWD")
	if wp3DAppPwd == "" {
		log.Fatal("WORDPRESS_3D_APP_PWD environment variable is not set")
	}
	wpUser := os.Getenv("WP_USERNAME")
	if wpUser == "" {
		log.Fatal("WP_USERNAME environment variable is not set")
	}
	wpBaseUrl := os.Getenv("WP_BASE_URL")
	if wpBaseUrl == "" {
		log.Fatal("WP_BASE_URL environment variable is not set")
	}
	wp := wpApi{
		tattoo: tattooWpDestination{
			tattooUrl:      wpTattooUrl,
			tattooHostname: wpTattooHostname,
			tattooAppPwd:   wpTattooAppPwd,
		},
		threeD: threeDWpDestination{
			threeDUrl:      wp3DUrl,
			threeDHostname: wp3DHostname,
			threeAppPwd:    wp3DAppPwd,
		},
		baseUrl: wpBaseUrl,
		user:    wpUser,
	}
	cfg := apiConfig{
		port:       port,
		assetsRoot: assetsRoot,
		tempRoot:   tempRoot,
		wpApi:      wp,
	}
	return cfg
}
