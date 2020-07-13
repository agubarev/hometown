package server

import (
	"context"
	"net/http"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/internal/server/endpoints"
	epauth "github.com/agubarev/hometown/internal/server/endpoints/auth"
	epgroup "github.com/agubarev/hometown/internal/server/endpoints/group"
	"github.com/go-chi/chi"
)

type ContextKey uint8

const (
	CKCore ContextKey = 1
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

	//---------------------------------------------------------------------------
	// API ROUTING (V1)
	//---------------------------------------------------------------------------
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(MiddlewareBase(ctx))
		//r.Use(epauth.MiddlewareAuth)

		r.Use(func(handler http.Handler) http.Handler {
			return endpoints.NewEndpoint(endpoints.NewName("authentication middleware"))
		})

		r.Route("/group", func(r chi.Router) {
			r.Method(http.MethodPost, "/", endpoints.NewEndpoint(endpoints.NewName("post_group"), c, epgroup.Post))
			r.Method(http.MethodGet, "/{id}", endpoints.NewEndpoint(endpoints.NewName("get_group"), c, epgroup.Get))
			r.Method(http.MethodPatch, "/{id}", endpoints.NewEndpoint(endpoints.NewName("patch_group"), c, epgroup.Patch))
			r.Method(http.MethodDelete, "/{id}", endpoints.NewEndpoint(endpoints.NewName("delete_group"), c, epgroup.Delete))
		})

		r.Route("/user", func(r chi.Router) {
			//r.Method(http.MethodPost, "/", endpoints.NewEndpoint(ctx, c, true, endpoints.UserPost, "post_user"))
			//r.Method(http.MethodGet, "/{id}", endpoints.NewEndpoint(ctx, c, user.UserGet, endpoints.NewName("get_user")))
			//r.Method(http.MethodPatch, "/{id}", endpoints.NewEndpoint(ctx, c, true, endpoints.UserPatch, "patch_user"))
			//r.Method(http.MethodDelete, "/{id}", endpoints.NewEndpoint(ctx, c, true, endpoints.UserDelete, "delete_user"))
		})
	})

	return http.ListenAndServe(addr, r)
}
