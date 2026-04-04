package tracing

import (
	"net/http"
)

// InitJaeger is a no-op stub for tracing initialization
func InitJaeger(serviceName string) (interface{}, error) {
	return nil, nil
}

// HTTPMiddleware is a no-op middleware that passes requests through unchanged
func HTTPMiddleware(next http.Handler) http.Handler {
	return next
}
