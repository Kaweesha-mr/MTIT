package routes

import (
	"log"
	"net/http"

	"alert-service/internal/db"
	"alert-service/internal/handlers"
	"alert-service/internal/repositories"
	"alert-service/internal/services"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	conn, err := db.NewPostgresConnection()
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	repo := repositories.NewPostgresAlertRepository(conn)
	service := services.NewAlertService(repo)
	h := handlers.NewAlertHandler(service)

	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/alerts", h.AlertsCollection)
	mux.HandleFunc("/alerts/", h.AlertByID)

	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
