package routes

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"volunteer-service/internal/handlers"
	"volunteer-service/internal/repositories"
	"volunteer-service/internal/services"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	repo := mustInitPostgresRepo()

	incidentServiceURL := os.Getenv("INCIDENT_SERVICE_URL")
	if incidentServiceURL == "" {
		incidentServiceURL = "http://localhost:8081"
	}

	logisticsServiceURL := os.Getenv("LOGISTICS_SERVICE_URL")
	if logisticsServiceURL == "" {
		logisticsServiceURL = "http://localhost:8086"
	}

	incidentVerifier := services.NewHTTPIncidentVerifier(incidentServiceURL)
	logisticsChecker := services.NewHTTPLogisticsChecker(logisticsServiceURL)
	service := services.NewVolunteerService(repo, incidentVerifier, logisticsChecker)
	h := handlers.NewVolunteerHandler(service, nil, "postgres")

	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/volunteers", h.VolunteersCollection)
	mux.HandleFunc("/volunteers/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/assign") {
			h.AssignVolunteer(w, r)
			return
		}

		h.VolunteerByID(w, r)
	})
	mux.HandleFunc("/openapi.yaml", serveOpenAPI)
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/openapi.yaml", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/docs", swaggerDocs)
	mux.HandleFunc("/docs/", swaggerDocs)

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
	// Try to read the file from different possible locations
	paths := []string{
		"api/openapi.yaml",
		"./api/openapi.yaml",
		"../api/openapi.yaml",
		"../../api/openapi.yaml",
	}

	for _, path := range paths {
		if content, err := ioutil.ReadFile(path); err == nil {
			w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content)
			return
		}
	}

	// If file not found, return embedded spec
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(openAPISpec))
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
	<title>Volunteer Service API Docs</title>
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

const openAPISpec = `openapi: 3.0.3
info:
  title: Volunteer Service API
  version: 1.0.0
servers:
  - url: http://localhost:8082
paths:
  /health:
    get:
      summary: Health check
      responses:
        '200':
          description: Service is healthy
  /volunteers:
    get:
      summary: List all volunteers
      responses:
        '200':
          description: List of volunteers retrieved
    post:
      summary: Create a new volunteer
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/VolunteerInput'
      responses:
        '201':
          description: Volunteer created
  /volunteers/{id}:
    get:
      summary: Get volunteer by ID
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: Volunteer found
        '404':
          description: Volunteer not found
    put:
      summary: Update volunteer
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateVolunteerRequest'
      responses:
        '200':
          description: Volunteer updated
        '404':
          description: Volunteer not found
    delete:
      summary: Delete volunteer
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '204':
          description: Volunteer deleted
        '404':
          description: Volunteer not found
  /volunteers/{id}/assign:
    put:
      summary: Assign volunteer to incident
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/AssignVolunteerRequest'
      responses:
        '200':
          description: Volunteer assigned
        '404':
          description: Volunteer not found
components:
  schemas:
    VolunteerInput:
      type: object
      properties:
        name:
          type: string
        role:
          type: string
        phone:
          type: string
      required: [name, role, phone]
    UpdateVolunteerRequest:
      type: object
      properties:
        name:
          type: string
        role:
          type: string
        phone:
          type: string
      required: [name, role, phone]
    VolunteerResponse:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        role:
          type: string
        phone:
          type: string
        status:
          type: string
    AssignVolunteerRequest:
      type: object
      properties:
        incidentId:
          type: integer
        role:
          type: string
      required: [incidentId]
    AssignVolunteerResponse:
      type: object
      properties:
        id:
          type: integer
        assignedTo:
          type: integer
        status:
          type: string
`

func mustInitPostgresRepo() repositories.VolunteerRepository {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5434")
	dbUser := getEnv("DB_USER", "volunteer_user")
	dbPassword := getEnv("DB_PASSWORD", "volunteer_pass")
	dbName := getEnv("DB_NAME", "volunteers_db")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	dsn := repositories.BuildPostgresDSN(dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	repo, err := repositories.NewPostgresVolunteerRepository(dsn)
	if err != nil {
		log.Fatalf("failed to initialize postgres repository: %v", err)
	}

	log.Print("volunteer-service using postgres repository")
	return repo
}

func getEnv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
