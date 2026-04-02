package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var authUser string
var authPass string
var jwtSecret string
var jwtTTL time.Duration

func main() {

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8090"
	}

	authUser = strings.TrimSpace(os.Getenv("AUTH_USER"))
	authPass = strings.TrimSpace(os.Getenv("AUTH_PASS"))
	if authUser == "" || authPass == "" {
		log.Fatal("AUTH_USER and AUTH_PASS are required")
	}

	jwtSecret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		jwtSecret = "dev-change-me"
		log.Printf("warning: using default JWT secret; set JWT_SECRET")
	}

	jwtTTL = durationFromEnvSeconds("JWT_TTL_SECONDS", time.Hour)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/openapi.yaml", openAPIHandler)
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/openapi.yaml", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/docs", docsHandler)
	mux.HandleFunc("/docs/", docsHandler)
	mux.HandleFunc("/swagger", docsHandler)
	mux.HandleFunc("/swagger/", docsHandler)

	handler := mux

	addr := ":" + port
	log.Printf("auth-service listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok","db":{"status":"n/a","source":"none"}}`))
}

func openAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	b, err := os.ReadFile("api/openapi.yaml")
	if err != nil {
		// Keep local auth usable even if the spec file is missing.
		_, _ = w.Write([]byte("openapi: 3.0.3\ninfo:\n  title: auth-service\n  version: 1.0.0\npaths: {}\n"))
		return
	}
	_, _ = w.Write(b)
}

func docsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
<head>
	<meta charset="UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1.0" />
	<title>Auth Service API Docs</title>
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
</html>`))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, pass, ok := r.BasicAuth()
	if !ok {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		user = body.Username
		pass = body.Password
	}

	if strings.TrimSpace(user) != authUser || strings.TrimSpace(pass) != authPass {
		unauthorized(w)
		return
	}

	expiresAt := time.Now().Add(jwtTTL)
	claims := jwt.MapClaims{
		"sub": authUser,
		"exp": expiresAt.Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		log.Printf("failed to sign token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"token":     signed,
		"expiresIn": int(jwtTTL.Seconds()),
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Basic")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}

func durationFromEnvSeconds(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		log.Printf("invalid duration seconds for %s=%s, using default %s", key, value, fallback)
		return fallback
	}
	if seconds == 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
