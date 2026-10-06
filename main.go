package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// version di-inject saat build via ldflags, dari CI (git commit sha)
var version = "dev"

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type HelloResponse struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type EnvResponse struct {
	CustomMessage string `json:"custom_message"`
}

type DbPingResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type TriggerResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// writeJSON menulis response JSON dengan status code tertentu dan
// me-log error jika encoding/penulisan gagal.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := fmt.Fprintf(w, "Halo dari Go di Kubernetes! Version: %s\n", version); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

// healthHandler dipakai oleh K8s liveness & readiness probe
func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok", Version: version})
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HelloResponse{Name: "Huri", Message: "Testing deploy again"})
}

func envHandler(w http.ResponseWriter, r *http.Request) {
	customMsg := os.Getenv("CUSTOM_MESSAGE")
	if customMsg == "" {
		customMsg = "Default message (CUSTOM_MESSAGE env is not set)"
	}
	writeJSON(w, http.StatusOK, EnvResponse{CustomMessage: customMsg})
}

func dbPingHandler(w http.ResponseWriter, r *http.Request) {
	host := os.Getenv("PG_HOST")
	port := os.Getenv("PG_PORT")
	user := os.Getenv("PG_USERNAME")
	password := os.Getenv("PG_PASSWORD")
	dbname := os.Getenv("PG_DATABASE")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		writeJSON(w, http.StatusInternalServerError, DbPingResponse{
			Status:  "error",
			Message: "Database credentials are not fully configured in environment variables (PG_HOST, PG_PORT, etc.)",
		})
		return
	}

	// connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=5",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, DbPingResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to open connection: %v", err),
		})
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close db connection: %v", err)
		}
	}()

	// Set timeout for Ping
	errChan := make(chan error, 1)
	go func() {
		errChan <- db.Ping()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, DbPingResponse{
				Status:  "error",
				Message: fmt.Sprintf("Failed to ping database: %v", err),
			})
			return
		}
	case <-time.After(5 * time.Second):
		writeJSON(w, http.StatusGatewayTimeout, DbPingResponse{
			Status:  "error",
			Message: "Database ping timed out (5s)",
		})
		return
	}

	writeJSON(w, http.StatusOK, DbPingResponse{
		Status:  "success",
		Message: "Successfully connected and pinged PostgreSQL database!",
	})
}

func triggerPipelineHandler(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = "master"
	}

	writeJSON(w, http.StatusOK, TriggerResponse{
		Status:  "success",
		Message: fmt.Sprintf("Pipeline triggered successfully for ref: %s (simulation)", ref),
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/hello", helloHandler)
	mux.HandleFunc("/env", envHandler)
	mux.HandleFunc("/db-ping", dbPingHandler)
	mux.HandleFunc("/trigger-pipeline", triggerPipelineHandler)

	addr := ":" + port
	log.Printf("server jalan di %s (version=%s)", addr, version)

	// Wrap mux dengan loggingMiddleware
	loggedHandler := loggingMiddleware(mux)

	if err := http.ListenAndServe(addr, loggedHandler); err != nil {
		log.Fatal(err)
	}
}