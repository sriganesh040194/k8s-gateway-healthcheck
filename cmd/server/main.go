package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s-gateway-healthcheck/internal/health"
)

const (
	defaultPort            = "8080"
	defaultShutdownTimeout = 10 * time.Second
	defaultReadTimeout     = 5 * time.Second
	defaultWriteTimeout    = 10 * time.Second
	defaultIdleTimeout     = 60 * time.Second
)

func main() {
	port := getEnv("PORT", defaultPort)
	appName := getEnv("APP_NAME", "gateway-healthcheck")
	appVersion := getEnv("APP_VERSION", "1.0.0")
	environment := getEnv("ENVIRONMENT", "production")
	livenessPath := getEnv("LIVENESS_PATH", "/healthz")
	readinessPath := getEnv("READINESS_PATH", "/readyz")
	startupPath := getEnv("STARTUP_PATH", "/startupz")
	fullHealthPath := getEnv("FULL_HEALTH_PATH", "/health")

	logger := log.New(os.Stdout, "", 0)
	checker := health.NewChecker(appName, appVersion, environment)

	mux := http.NewServeMux()

	// Liveness probe — is the process alive?
	mux.HandleFunc(livenessPath, livenessHandler(checker))

	// Readiness probe — is the app ready to serve traffic?
	mux.HandleFunc(readinessPath, readinessHandler(checker))

	// Startup probe — has the app finished initializing?
	mux.HandleFunc(startupPath, startupHandler(checker))

	// Full status endpoint for App Gateway / monitoring dashboards
	mux.HandleFunc(fullHealthPath, fullHealthHandler(checker))

	// Root info
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"service": appName,
			"version": appVersion,
			"status":  "running",
		})
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      loggingMiddleware(logger)(corsMiddleware()(mux)),
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	go func() {
		logger.Printf(`{"level":"info","msg":"Server starting","port":"%s","app":"%s","version":"%s","env":"%s"}`,
			port, appName, appVersion, environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf(`{"level":"fatal","msg":"Server failed","error":"%s"}`, err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Printf(`{"level":"info","msg":"Shutting down gracefully"}`)
	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf(`{"level":"fatal","msg":"Forced shutdown","error":"%s"}`, err)
	}
	logger.Printf(`{"level":"info","msg":"Server stopped"}`)
}

func livenessHandler(checker *health.Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := checker.Liveness()
		writeJSON(w, result.HTTPStatus(), result)
	}
}

func readinessHandler(checker *health.Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := checker.Readiness()
		writeJSON(w, result.HTTPStatus(), result)
	}
}

func startupHandler(checker *health.Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := checker.Startup()
		writeJSON(w, result.HTTPStatus(), result)
	}
}

func fullHealthHandler(checker *health.Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := checker.Full()
		writeJSON(w, result.HTTPStatus(), result)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func loggingMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			logger.Printf(`{"level":"info","method":"%s","path":"%s","status":%d,"duration_ms":%d,"remote_addr":"%s"}`,
				r.Method, r.URL.Path, rw.status, time.Since(start).Milliseconds(), r.RemoteAddr)
		})
	}
}

func corsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
