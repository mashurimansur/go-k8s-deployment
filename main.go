package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/hello", helloHandler)

	addr := ":" + port
	log.Printf("server jalan di %s (version=%s)", addr, version)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
