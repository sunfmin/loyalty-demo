package main

//go:generate protoc --go_out=. --go_opt=paths=source_relative ../../api/v1/loyalty.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative ../../api/v1/transaction.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative ../../api/v1/reward.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative ../../api/v1/campaign.proto

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	
	"github.com/opentracing/opentracing-go"
)

func main() {
	// Initialize NoopTracer for development (production will use Jaeger/Zipkin)
	opentracing.SetGlobalTracer(opentracing.NoopTracer{})
	
	// TODO: Load configuration from environment
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	
	// Create HTTP multiplexer
	mux := http.NewServeMux()
	
	// Register health check endpoint
	mux.HandleFunc("/health", healthCheckHandler)
	
	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	
	log.Printf("Starting loyalty API server on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","version":"1.0.0","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

