package server

import (
	"context"
	"net/http"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/internal/server/endpoints"
	"github.com/go-chi/chi"
)

type Server struct {
	core *core.Core
}

type Response struct {
	StatusCode int         `json:"status_code"`
	Error      error       `json:"error"`
	Payload    interface{} `json:"payload,omitempty"`
}

func Run(ctx context.Context, c *core.Core, addr string) (err error) {
	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/user", func(r chi.Router) {
			//r.Method(http.MethodPost, "/", endpoints.NewEndpoint(ctx, c, true, endpoints.UserPost, "post_user"))
			r.Method(http.MethodGet, "/{id}", endpoints.NewEndpoint(ctx, c, true, endpoints.UserGet, "get_user"))
			//r.Method(http.MethodPatch, "/{id}", endpoints.NewEndpoint(ctx, c, true, endpoints.UserPatch, "patch_user"))
			//r.Method(http.MethodDelete, "/{id}", endpoints.NewEndpoint(ctx, c, true, endpoints.UserDelete, "delete_user"))
		})
	})

	return nil
}
