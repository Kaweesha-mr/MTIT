package routes

import (
	"log"
	"net/http"
	"os"

	"alert-service/internal/clients"
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

	// Initialize Incident Service client
	incidentServiceURL := os.Getenv("INCIDENT_SERVICE_URL")
	if incidentServiceURL == "" {
		incidentServiceURL = "http://localhost:8081"
	}
	incidentClient := clients.NewIncidentClient(incidentServiceURL)

	repo := repositories.NewPostgresAlertRepository(conn)
	service := services.NewAlertService(repo, incidentClient)
	h := handlers.NewAlertHandler(service, conn.PingContext, "postgres")

	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/alerts", h.AlertsCollection)
	mux.HandleFunc("/alerts/", h.AlertByID)
	mux.HandleFunc("/openapi.yaml", serveOpenAPI)
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/openapi.yaml", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/docs", swaggerDocs)
	mux.HandleFunc("/docs/", swaggerDocs)
	mux.HandleFunc("/swagger", swaggerDocs)
	mux.HandleFunc("/swagger/", swaggerDocs)

	// Wrap with CORS and tracing middleware
	corsHandler := withCORS(mux)
	return corsHandler
}

func serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "api/openapi.yaml")
}

func swaggerDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerUIHTML))
}

const swaggerUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Alert Service API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`

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
