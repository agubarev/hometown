package server

import (
	"context"
	"net/http"
)

// MiddlewareContext is used to inject context into the middleware
// NOTE: for example use this with the first middleware in the chain
func MiddlewareContext(ctx context.Context) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
