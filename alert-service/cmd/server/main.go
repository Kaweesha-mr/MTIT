package main

import (
	"log"
	"net/http"
	"os"

	"alert-service/internal/routes"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	handler := routes.NewRouter()
	addr := ":" + port

	log.Printf("alert-service running on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
