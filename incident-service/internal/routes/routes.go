package routes

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"incident-service/internal/handlers"
	"incident-service/internal/repositories"
	"incident-service/internal/services"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	repo := initRepository()
	alertURL := os.Getenv("ALERT_SERVICE_URL")
	if alertURL == "" {
		alertURL = "http://localhost:8085/alerts"
	}
	shelterURL := os.Getenv("SHELTER_SERVICE_URL")
	if shelterURL == "" {
		shelterURL = "http://localhost:8084/shelters"
	}

	service := services.NewIncidentService(repo, alertURL, shelterURL)
	h := handlers.NewIncidentHandler(service, nil, "postgres")

	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/incidents", h.Incidents)
	mux.HandleFunc("/incidents/", h.IncidentByID)
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

func serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	http.ServeFile(w, r, resolveOpenAPIPath())
}

func resolveOpenAPIPath() string {
	local := filepath.Join("api", "openapi.yaml")
	if _, err := os.Stat(local); err == nil {
		return local
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if ok {
		fromSourceTree := filepath.Join(filepath.Dir(thisFile), "..", "..", "api", "openapi.yaml")
		if _, err := os.Stat(fromSourceTree); err == nil {
			return fromSourceTree
		}
	}

	return local
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
	<title>Incident Service API Docs</title>
	<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
	<div id="swagger-ui"></div>
	<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
	<script>
		const path = window.location.pathname;
		let prefix = "";
		if (path.endsWith("/docs")) {
			prefix = path.slice(0, -5);
		} else if (path.endsWith("/docs/")) {
			prefix = path.slice(0, -6);
		}
		const specURL = (prefix || "") + "/openapi.yaml?v=" + Date.now();

		window.ui = SwaggerUIBundle({
			url: specURL,
			dom_id: "#swagger-ui",
			deepLinking: true,
			presets: [SwaggerUIBundle.presets.apis],
			layout: "BaseLayout"
		});
	</script>
</body>
</html>`

func initRepository() repositories.IncidentRepository {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5436")
	dbUser := getEnv("DB_USER", "incident_user")
	dbPassword := getEnv("DB_PASSWORD", "incident_pass")
	dbName := getEnv("DB_NAME", "incidents_db")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	dsn := repositories.BuildPostgresDSN(dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	repo, err := repositories.NewPostgresIncidentRepository(dsn)
	if err != nil {
		log.Fatalf("failed to initialize postgres repository: %v", err)
	}

	log.Print("incident-service using postgres repository")
	return repo
}

func getEnv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
