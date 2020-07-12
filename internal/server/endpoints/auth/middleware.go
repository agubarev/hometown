package auth

import (
	"net/http"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

// MiddlewareAuth validates the authorization header and adds
// a corresponding user to the context
func MiddlewareAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if len(parts) != 2 {
			http.Error(w, "invalid authorization header", http.StatusBadRequest)
			return
		}

		// just a friendly piece of code for a longer shot (if things are to move onward)
		switch parts[0] {
		case "Bearer":
			spew.Dump(parts[1])
		default:
			http.Error(w, "unsupported authorization type", http.StatusUnauthorized)
			return
		}

		// obtaining user from the manager

		// extending request context
		// ctx := context.WithValue(r.Context(), contextKeyUser, user)

		// next.ServeHTTP(w, r.WithContext(ctx))
	})
}
