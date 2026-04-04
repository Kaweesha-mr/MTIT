package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gateway/internal/cache"

	"golang.org/x/time/rate"
	"github.com/golang-jwt/jwt/v5"
)

var cacheStore *cache.Store
var authEnabled bool
var authUser string
var authPass string
var jwtSecret string
var jwtTTL time.Duration
var requestLogger *log.Logger
var logFileMutex sync.Mutex

// serviceTarget defines an upstream target and optional path prefix to strip.
type serviceTarget struct {
	prefix  string
	target  *url.URL
	proxy   *httputil.ReverseProxy
	limiter *rate.Limiter
}

type captureResponseWriter struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (w *captureResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *captureResponseWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func newTarget(prefix, rawURL string, limiter *rate.Limiter) *serviceTarget {
	u, err := url.Parse(rawURL)
	if err != nil {
		log.Fatalf("invalid upstream url %s: %v", rawURL, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	// Tighten timeouts via Transport.
	proxy.Transport = &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}
	return &serviceTarget{prefix: prefix, target: u, proxy: proxy, limiter: limiter}
}

func buildCacheKey(prefix string, r *http.Request) string {
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	return fmt.Sprintf("%s:%s:%s", r.Method, prefix, path)
}

func newLimiterFromEnv(envKey string, fallbackRPS float64) *rate.Limiter {
	value := strings.TrimSpace(os.Getenv(envKey))
	if value == "" && fallbackRPS <= 0 {
		return nil
	}

	rps := fallbackRPS
	if value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || parsed <= 0 {
			log.Printf("invalid rate for %s=%s, disabling limiter", envKey, value)
			return nil
		}
		rps = parsed
	}

	burst := int(math.Ceil(rps * 2))
	if burst < 1 {
		burst = 1
	}
	return rate.NewLimiter(rate.Limit(rps), burst)
}

func copyHeaders(h http.Header) map[string]string {
	result := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}

func writeCachedResponse(w http.ResponseWriter, entry *cache.Entry) {
	for k, v := range entry.Header {
		w.Header().Set(k, v)
	}
	status := entry.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write(entry.Body)
}

func (t *serviceTarget) handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()
	useCache := r.Method == http.MethodGet && cacheStore != nil
	cacheKey := ""

	isAuthEndpoint := t.prefix == "/login" || t.prefix == "/auth"
	if authEnabled && !isAuthEndpoint && !checkAuth(w, r) {
		logRequest(r, t.prefix, http.StatusUnauthorized, time.Since(startTime), false, false)
		return
	}

	if t.limiter != nil && !t.limiter.Allow() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
		logRequest(r, t.prefix, http.StatusTooManyRequests, 0, false, true)
		return
	}

	if useCache {
		cacheKey = buildCacheKey(t.prefix, r)
		if entry, hit, err := cacheStore.Get(ctx, cacheKey); err == nil && hit {
			writeCachedResponse(w, entry)
			logRequest(r, t.prefix, entry.Status, time.Since(startTime), true, false)
			return
		} else if err != nil {
			log.Printf("cache get failed for %s: %v", cacheKey, err)
		}
	}

	r.Host = t.target.Host

	// Ensure authorization is passed through for internal service usage
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		r.Header.Set("Authorization", authHeader)
	}

	recorder := &captureResponseWriter{ResponseWriter: w, status: http.StatusOK}
	t.proxy.ServeHTTP(recorder, r)

	logRequest(r, t.prefix, recorder.status, time.Since(startTime), false, false)

	// Cache GET responses
	if useCache && recorder.status >= http.StatusOK && recorder.status < http.StatusBadRequest {
		entry := &cache.Entry{
			Status: recorder.status,
			Header: copyHeaders(recorder.Header()),
			Body:   recorder.buf.Bytes(),
		}
		if err := cacheStore.Set(ctx, cacheKey, entry); err != nil {
			log.Printf("cache set failed for %s: %v", cacheKey, err)
		}
	}

	// Invalidate cache on successful mutations (POST, PUT, DELETE)
	isMutation := r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete
	isSuccessful := recorder.status >= http.StatusOK && recorder.status < http.StatusBadRequest
	if cacheStore != nil && isMutation && isSuccessful {
		// Invalidate all GET cache entries for this service prefix
		// Example: POST /shelters/1 invalidates all GET:/shelters:* cache entries
		invalidatePattern := fmt.Sprintf("GET:%s:*", t.prefix)
		if err := cacheStore.InvalidateByPrefix(ctx, invalidatePattern); err != nil {
			log.Printf("cache invalidation failed for pattern %s: %v", invalidatePattern, err)
		} else {
			log.Printf("cache invalidated for pattern %s after %s %s", invalidatePattern, r.Method, r.URL.Path)
		}
	}
}

func initLogger(logPath string) error {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	requestLogger = log.New(logFile, "", 0)
	log.Printf("gateway request logger initialized: %s", logPath)
	return nil
}

func logRequest(r *http.Request, service string, status int, duration time.Duration, isCacheHit bool, isRateLimited bool) {
	if requestLogger == nil {
		return
	}

	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}

	entry := map[string]interface{}{
		"timestamp":      time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
		"method":         r.Method,
		"path":           r.URL.Path,
		"query":          r.URL.RawQuery,
		"service":        service,
		"status":         status,
		"duration_ms":    duration.Milliseconds(),
		"client_ip":      clientIP,
		"cache_hit":      isCacheHit,
		"rate_limited":   isRateLimited,
	}

	if data, err := json.Marshal(entry); err == nil {
		requestLogger.Println(string(data))
	}
}

func main() {

	addr := ":8080"
	mux := http.NewServeMux()

	// Initialize request logger
	logPath := getenv("GATEWAY_LOG_FILE", "/var/log/gateway/requests.log")
	if err := initLogger(logPath); err != nil {
		log.Printf("warning: failed to initialize request logger: %v", err)
	}

	authUser = strings.TrimSpace(os.Getenv("GATEWAY_AUTH_USER"))
	authPass = strings.TrimSpace(os.Getenv("GATEWAY_AUTH_PASS"))
	authEnabled = authUser != "" && authPass != ""
	if !authEnabled {
		log.Printf("gateway auth disabled; set GATEWAY_AUTH_USER and GATEWAY_AUTH_PASS to enable")
	}

	jwtSecret = strings.TrimSpace(os.Getenv("GATEWAY_JWT_SECRET"))
	if jwtSecret == "" {
		jwtSecret = "dev-change-me"
		log.Printf("warning: using default JWT secret; set GATEWAY_JWT_SECRET")
	}
	jwtTTL = durationFromEnvSeconds("GATEWAY_JWT_TTL_SECONDS", time.Hour)

	cacheTTL := durationFromEnvSeconds("CACHE_TTL_SECONDS", 120*time.Second)
	cacheEnabled := boolFromEnv("CACHE_ENABLED", true) && cacheTTL > 0
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	cacheStore = cache.NewStore(redisAddr, redisPassword, cacheTTL, cacheEnabled)
	if !cacheEnabled {
		log.Printf("response cache disabled (CACHE_ENABLED=%v, CACHE_TTL_SECONDS=%s)", cacheEnabled, cacheTTL)
		cacheStore = nil
	}
	if cacheStore != nil && cacheEnabled {
		defer func() {
			if err := cacheStore.Close(); err != nil {
				log.Printf("cache close failed: %v", err)
			}
		}()
	}

	incidentURL := getenv("INCIDENT_URL", "http://localhost:8081")
	volunteerURL := getenv("VOLUNTEER_URL", "http://localhost:8082")
	resourceURL := getenv("RESOURCE_URL", "http://localhost:8083")
	shelterURL := getenv("SHELTER_URL", "http://localhost:8084")
	alertURL := getenv("ALERT_URL", "http://localhost:8085")
	fleetURL := getenv("FLEET_URL", "http://localhost:8086")
	authServiceURL := getenv("AUTH_SERVICE_URL", "http://localhost:8090")

	incidentLimiter := newLimiterFromEnv("INCIDENT_RATE_LIMIT_RPS", 20)
	volunteerLimiter := newLimiterFromEnv("VOLUNTEER_RATE_LIMIT_RPS", 20)
	resourceLimiter := newLimiterFromEnv("RESOURCE_RATE_LIMIT_RPS", 20)
	shelterLimiter := newLimiterFromEnv("SHELTER_RATE_LIMIT_RPS", 20)
	alertLimiter := newLimiterFromEnv("ALERT_RATE_LIMIT_RPS", 20)
	fleetLimiter := newLimiterFromEnv("FLEET_RATE_LIMIT_RPS", 20)
	authLimiter := newLimiterFromEnv("AUTH_RATE_LIMIT_RPS", 30)

	targets := []*serviceTarget{
		newTarget("/incidents", incidentURL, incidentLimiter),
		newTarget("/volunteers", volunteerURL, volunteerLimiter),
		newTarget("/resources", resourceURL, resourceLimiter),
		newTarget("/shelters", shelterURL, shelterLimiter),
		newTarget("/alerts", alertURL, alertLimiter),
		newTarget("/trips", fleetURL, fleetLimiter),
		newTarget("/vehicles", fleetURL, fleetLimiter),
		newTarget("/login", authServiceURL, authLimiter),
		newTarget("/auth", authServiceURL, authLimiter),
	}

	serviceHealthChecks := []serviceHealth{
		{name: "incident", url: incidentURL},
		{name: "volunteer", url: volunteerURL},
		{name: "resource", url: resourceURL},
		{name: "shelter", url: shelterURL},
		{name: "alert", url: alertURL},
		{name: "fleet", url: fleetURL},
		{name: "auth", url: authServiceURL},
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/status", statusHandler(serviceHealthChecks))

	for _, t := range targets {
		mux.HandleFunc(t.prefix, t.handler)
		mux.HandleFunc(t.prefix+"/", t.handler)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("gateway listening on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func boolFromEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("invalid boolean for %s=%s, using default %v", key, value, fallback)
		return fallback
	}
	return parsed
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

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func checkAuth(w http.ResponseWriter, r *http.Request) bool {
	if !authEnabled {
		return true
	}

	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		unauthorized(w)
		return false
	}
	tokenStr := strings.TrimPrefix(header, prefix)

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		unauthorized(w)
		return false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["sub"] != authUser {
		unauthorized(w)
		return false
	}

	return true
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}

type serviceHealth struct {
	name string
	url  string
}

func statusHandler(services []serviceHealth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authEnabled && !checkAuth(w, r) {
			return
		}

		client := &http.Client{Timeout: 3 * time.Second}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		results := make(map[string]string, len(services))
		for _, svc := range services {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/health", svc.url), nil)
			if err != nil {
				results[svc.name] = "error"
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				results[svc.name] = "down"
				continue
			}
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				results[svc.name] = "up"
			} else {
				results[svc.name] = fmt.Sprintf("down (%d)", resp.StatusCode)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(results)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !authEnabled {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"auth disabled"}`))
		return
	}

	user, pass, ok := r.BasicAuth()
	if !ok || user != authUser || pass != authPass {
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
	fmt.Fprintf(w, `{"token":"%s","expiresIn":%d}`, signed, int(jwtTTL.Seconds()))
}
