package server

import (
	"context"
	"net/http"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/internal/server/endpoints"
	"github.com/go-chi/chi"
)

func Run(ctx context.Context, c *core.Core, addr string) (err error) {
	r := chi.NewRouter()

	// handling CORS
	r.Use(cors.New(cors.Options{
		AllowedOrigins: []string{"http://balticbits.eu", "http://balticbytes.eu"},
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			return true
		},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
		},
		AllowCredentials: true,
	}).Handler)

	// main routes
	r.Route("/rpc/v1", func(r chi.Router) {
		r.Route("/group", func(r chi.Router) {
			r.Method(http.MethodPost, "/", endpoints.NewEndpoint(ctx, c, endpoints.GroupPost, "group_post"))
			r.Method(http.MethodPatch, "/{id}", endpoints.NewEndpoint(ctx, c, endpoints.GroupPatch, "group_patch"))
			r.Method(http.MethodGet, "/", endpoints.NewEndpoint(ctx, c, endpoints.GroupList, "group_list"))
			r.Method(http.MethodGet, "/{id}", endpoints.NewEndpoint(ctx, c, endpoints.GroupGet, "group_get"))
			r.Method(http.MethodDelete, "/{id}", endpoints.NewEndpoint(ctx, c, endpoints.GroupDelete, "group_delete"))
		})
	})

	return http.ListenAndServe(addr, r)
}
