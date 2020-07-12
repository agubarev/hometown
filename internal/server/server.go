package server

import (
	"context"
	"net/http"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/internal/server/endpoints"
	epgroup "github.com/agubarev/hometown/internal/server/endpoints/group"
	"github.com/agubarev/hometown/internal/server/endpoints/user"
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

	//---------------------------------------------------------------------------
	// API ROUTING (V1)
	//---------------------------------------------------------------------------
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(epauth.AuthMiddleware)

		r.Route("/group", func(r chi.Router) {
			r.Use(midd)
			r.Method(http.MethodPost, "/", endpoints.NewEndpoint(ctx, c, epgroup.Post, endpoints.NewName("post_group")))
			r.Method(http.MethodGet, "/{id}", endpoints.NewEndpoint(ctx, c, epgroup.Get, endpoints.NewName("get_group")))
			r.Method(http.MethodPatch, "/{id}", endpoints.NewEndpoint(ctx, c, epgroup.Patch, endpoints.NewName("patch_group")))
			r.Method(http.MethodDelete, "/{id}", endpoints.NewEndpoint(ctx, c, epgroup.Delete, endpoints.NewName("delete_group")))
		})

		r.Route("/user", func(r chi.Router) {
			//r.Method(http.MethodPost, "/", endpoints.NewEndpoint(ctx, c, true, endpoints.UserPost, "post_user"))
			r.Method(http.MethodGet, "/{id}", endpoints.NewEndpoint(ctx, c, user.UserGet, endpoints.NewName("get_user")))
			//r.Method(http.MethodPatch, "/{id}", endpoints.NewEndpoint(ctx, c, true, endpoints.UserPatch, "patch_user"))
			//r.Method(http.MethodDelete, "/{id}", endpoints.NewEndpoint(ctx, c, true, endpoints.UserDelete, "delete_user"))
		})
	})

	return http.ListenAndServe(addr, r)
}
