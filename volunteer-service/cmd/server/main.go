package main

import (
	"log"
	"net/http"
	"os"

	"volunteer-service/internal/routes"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	handler := routes.NewRouter()
	addr := ":" + port

	log.Printf("volunteer-service running on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
