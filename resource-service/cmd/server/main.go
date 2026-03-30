package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"resource-service/internal/clients"
	"resource-service/internal/config"
	"resource-service/internal/handlers"
	"resource-service/internal/repositories"
	"resource-service/internal/routes"
	"resource-service/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	mongoConn, err := config.ConnectMongo(cfg)
	if err != nil {
		log.Fatalf("mongo connection error: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoConn.Client.Disconnect(ctx)
	}()

	repo := repositories.NewMongoResourceRepository(mongoConn.Client.Database(cfg.DBName), cfg.CollectionName)
	shelterClient := clients.NewShelterClient(cfg.ShelterServiceURL, cfg.ShelterTimeoutSeconds)
	service := services.NewResourceService(repo, shelterClient)
	handler := handlers.NewResourceHandler(service)
	router := routes.NewRouter(handler)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("resource-service running on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
