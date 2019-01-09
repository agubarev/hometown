package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"gitlab.com/agubarev/hometown/usermanager"
)

var (
	contextKeyUser = contextKey("user")
)

// MiddlewareAuth validates the authorization header and adds
// a corresponding user to the context
func MiddlewareAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		var authType, authToken string
		_, err := fmt.Sscanf(authHeader, "%s %s", &authType, &authToken)
		if err != nil && err != io.EOF {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// just a friendly piece of code for a longer shot (if things are to move onward)
		switch authType {
		case "Bearer":
			// TODO: implement
		default:
			http.Error(w, "unsupported authorization type", http.StatusUnauthorized)
			return
		}

		// passing just the test user for now
		user, err := usermanager.NewUser("testauthuser", "testme@example.com")
		if err != nil {
			http.Error(w, "failed to initialize test user", http.StatusInternalServerError)
			return
		}

		// extending request context
		ctx := context.WithValue(r.Context(), contextKeyUser, user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
