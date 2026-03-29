package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	defaultPort              = "8083"
	defaultDBName            = "resource_db"
	defaultCollectionName    = "resources"
	defaultShelterServiceURL = "http://localhost:8084"
	defaultMongoTimeout      = 10
	defaultShelterTimeout    = 5
)

type Config struct {
	Port                  string
	MongoURI              string
	DBName                string
	CollectionName        string
	ShelterServiceURL     string
	MongoTimeoutSeconds   int
	ShelterTimeoutSeconds int
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		Port:                  envOrDefault("PORT", defaultPort),
		MongoURI:              os.Getenv("MONGODB_URI"),
		DBName:                envOrDefault("DB_NAME", defaultDBName),
		CollectionName:        envOrDefault("COLLECTION_NAME", defaultCollectionName),
		ShelterServiceURL:     envOrDefault("SHELTER_SERVICE_URL", defaultShelterServiceURL),
		MongoTimeoutSeconds:   intEnvOrDefault("MONGO_TIMEOUT_SECONDS", defaultMongoTimeout),
		ShelterTimeoutSeconds: intEnvOrDefault("SHELTER_TIMEOUT_SECONDS", defaultShelterTimeout),
	}

	if cfg.MongoURI == "" {
		return Config{}, fmt.Errorf("MONGODB_URI is required")
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func intEnvOrDefault(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}

	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return fallback
	}

	return n
}
