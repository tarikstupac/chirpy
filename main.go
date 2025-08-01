package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/tarikstupac/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	secretKey      string
	polkaKey       string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(
		fmt.Appendf(nil, `
		<html>
		
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
		
		</html>
			`, cfg.fileserverHits.Load()))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, req *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Not dev environment", nil)
		return
	}
	cfg.fileserverHits.Swap(0)
	err := cfg.db.DeleteAllUsers(req.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting users", err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func healthHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func removeProfanityFromMessage(msg string) string {
	splitString := strings.Split(msg, " ")
	for i, v := range splitString {
		switch strings.ToLower(v) {
		case "kerfuffle", "sharbert", "fornax":
			splitString[i] = "****"
		}
	}
	cleanedMessage := strings.Join(splitString, " ")
	return cleanedMessage
}

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")
	dbUrl := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	secretKey := os.Getenv("SECRET_KEY")
	polkaKey := os.Getenv("POLKA_KEY")
	if dbUrl == "" || port == "" || platform == "" || secretKey == "" || polkaKey == "" {
		log.Fatal("Env variables must be set")
	}

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("Failed to connect to the DB: %s", err)
	}
	dbQueries := database.New(db)

	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	apiCfg := &apiConfig{db: dbQueries, platform: platform, secretKey: secretKey, polkaKey: polkaKey}

	// API routes
	mux.HandleFunc("GET /api/healthz", healthHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)

	// Chirp routes
	mux.HandleFunc("POST /api/chirps", apiCfg.createChirpHandler)
	mux.HandleFunc("GET /api/chirps", apiCfg.retrieveChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{ID}", apiCfg.retrieveChirpByIdHandler)
	mux.HandleFunc("DELETE /api/chirps/{ID}", apiCfg.deleteChirpByIdHandler)

	// User routes
	mux.HandleFunc("POST /api/users", apiCfg.createUserHandler)
	mux.HandleFunc("PUT /api/users", apiCfg.updateUserEmailPasswordHandler)

	// Auth routes
	mux.HandleFunc("POST /api/login", apiCfg.loginHandler)
	mux.HandleFunc("POST /api/refresh", apiCfg.refreshHandler)
	mux.HandleFunc("POST /api/revoke", apiCfg.revokeHandler)

	// Webhook routes
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.polkaWebhookHandler)

	// static routes
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}
