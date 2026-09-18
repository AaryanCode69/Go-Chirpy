package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/AaryanCode69/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	const port = "8080"
	const filepathRoot = "."

	// 1. Load settings from the .env file (if there is one).
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}
	platform := os.Getenv("PLATFORM")
	jwtSecret := os.Getenv("JWT_SECRET")
	polkaKey := os.Getenv("POLKA_KEY")

	// 2. Connect to the database.
	// sql.Open only checks the settings. Ping actually talks to Postgres,
	// so a wrong URL makes the app stop right away instead of on the first request.
	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %s", err)
	}
	if err := dbConn.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}

	// 3. Build the shared config that handlers use.
	apiCfg := &apiConfig{
		db:        database.New(dbConn),
		platform:  platform,
		jwtSecret: jwtSecret,
		polkaKey:  polkaKey,
	}

	// 4. Register routes.
	mux := http.NewServeMux()

	fileServer := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fileServer))

	mux.HandleFunc("GET /api/healthz", handlerReadiness)

	mux.HandleFunc("POST /api/chirps", apiCfg.handlerValidateAndSaveChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetAllChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirp)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handleDeleteChirpByID)

	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	mux.HandleFunc("POST /api/login", apiCfg.handlerLoginUser)
	mux.HandleFunc("PUT /api/users", apiCfg.handleUserUpdate)

	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handleUserUpgrade)

	mux.HandleFunc("POST /api/refresh", apiCfg.handleTokenRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevokeToken)

	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	// 5. Start the server.
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second, // don't wait forever on slow clients
	}

	log.Printf("Serving files from %s on port: %s", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}
