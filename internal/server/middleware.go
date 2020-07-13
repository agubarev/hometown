package server

import (
	"context"
	"net/http"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/internal/server/endpoints"
	epgroup "github.com/agubarev/hometown/internal/server/endpoints/group"
)

// MiddlewareBase is used to inject context into the middleware
// NOTE: for example use this with the first middleware in the chain
func MiddlewareBase(ctx context.Context) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func MiddlewareBaseTest(ctx context.Context, c *core.Core) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return endpoints.NewEndpoint(endpoints.NewName("post_group"), c, epgroup.Post)
		//return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//	next.ServeHTTP(w, r.WithContext(ctx))
		//})
	}
}
