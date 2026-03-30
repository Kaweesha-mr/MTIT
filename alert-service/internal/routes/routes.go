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

	return mux
}
