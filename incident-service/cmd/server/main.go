package main

import (
	"log"
	"net/http"
	"os"

	"incident-service/internal/routes"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	handler := routes.NewRouter()
	addr := ":" + port

	log.Printf("incident-service running on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
