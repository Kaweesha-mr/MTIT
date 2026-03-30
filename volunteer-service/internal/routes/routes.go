package routes

import (
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

	repo := repositories.VolunteerRepository(repositories.NewInMemoryVolunteerRepository())

	useDB := strings.EqualFold(strings.TrimSpace(os.Getenv("USE_DB")), "true")
	if useDB {
		dbHost := getEnv("DB_HOST", "localhost")
		dbPort := getEnv("DB_PORT", "5432")
		dbUser := getEnv("DB_USER", "volunteer_user")
		dbPassword := getEnv("DB_PASSWORD", "volunteer_pass")
		dbName := getEnv("DB_NAME", "volunteers_db")
		dbSSLMode := getEnv("DB_SSLMODE", "disable")

		dsn := repositories.BuildPostgresDSN(dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
		dbRepo, err := repositories.NewPostgresVolunteerRepository(dsn)
		if err != nil {
			log.Printf("failed to initialize postgres repository, falling back to in-memory: %v", err)
		} else {
			repo = dbRepo
			log.Print("volunteer-service using postgres repository")
		}
	}

	incidentServiceURL := os.Getenv("INCIDENT_SERVICE_URL")
	if incidentServiceURL == "" {
		incidentServiceURL = "http://localhost:8081"
	}

	logisticsServiceURL := os.Getenv("LOGISTICS_SERVICE_URL")
	if logisticsServiceURL == "" {
		logisticsServiceURL = "http://localhost:8083/logistics"
	}

	incidentVerifier := services.NewHTTPIncidentVerifier(incidentServiceURL)
	logisticsChecker := services.NewHTTPLogisticsChecker(logisticsServiceURL)
	service := services.NewVolunteerService(repo, incidentVerifier, logisticsChecker)
	h := handlers.NewVolunteerHandler(service)

	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/volunteers", h.VolunteersCollection)
	mux.HandleFunc("/volunteers/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/assign") {
			h.AssignVolunteer(w, r)
			return
		}

		h.VolunteerByID(w, r)
	})

	return mux
}

func getEnv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
