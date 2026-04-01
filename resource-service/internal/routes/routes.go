package routes

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"resource-service/internal/handlers"
)

func NewRouter(handler *handlers.ResourceHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/resources", handler.Resources)
	mux.HandleFunc("/resources/", handler.ResourceByID)
	mux.HandleFunc("/openapi.yaml", serveOpenAPI)
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/openapi.yaml", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/docs", swaggerDocs)
	mux.HandleFunc("/docs/", swaggerDocs)

	return corsMiddleware(mux)
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
  <title>Resource Service API Docs</title>
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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
