package server

import (
	"context"

	"github.com/agubarev/hometown/internal/core"
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

func New(ctx context.Context, c *core.Core, addr string) (s *Server, err error) {
	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/user", func(r chi.Router) {
			r.Get("/{id}")
		})
	})

	return s, nil
}
