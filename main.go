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

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Halo dari Go di Kubernetes! Version: %s\n", version)
}

// healthHandler dipakai oleh K8s liveness & readiness probe
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{Status: "ok", Version: version})
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HelloResponse{Name: "Huri", Message: "Testing deploy again"})
}

func envHandler(w http.ResponseWriter, r *http.Request) {
	customMsg := os.Getenv("CUSTOM_MESSAGE")
	if customMsg == "" {
		customMsg = "Default message (CUSTOM_MESSAGE env is not set)"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(EnvResponse{CustomMessage: customMsg})
}

func dbPingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	host := os.Getenv("PG_HOST")
	port := os.Getenv("PG_PORT")
	user := os.Getenv("PG_USERNAME")
	password := os.Getenv("PG_PASSWORD")
	dbname := os.Getenv("PG_DATABASE")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(DbPingResponse{
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
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(DbPingResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to open connection: %v", err),
		})
		return
	}
	defer db.Close()

	// Set timeout for Ping
	errChan := make(chan error, 1)
	go func() {
		errChan <- db.Ping()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(DbPingResponse{
				Status:  "error",
				Message: fmt.Sprintf("Failed to ping database: %v", err),
			})
			return
		}
	case <-time.After(5 * time.Second):
		w.WriteHeader(http.StatusGatewayTimeout)
		json.NewEncoder(w).Encode(DbPingResponse{
			Status:  "error",
			Message: "Database ping timed out (5s)",
		})
		return
	}

	json.NewEncoder(w).Encode(DbPingResponse{
		Status:  "success",
		Message: "Successfully connected and pinged PostgreSQL database!",
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

	addr := ":" + port
	log.Printf("server jalan di %s (version=%s)", addr, version)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
