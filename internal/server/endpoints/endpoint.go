package endpoints

import (
	"net/http"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/davecgh/go-spew/spew"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type contextKey int

// context keys
const (
	keyUserID contextKey = iota
)

type Endpoint struct {
	ap      *user.AccessPolicy
	core    *core.Manager
	handler http.HandlerFunc
}

func (e Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, err := e.user.UserManager().GetUserByID(r.Context(), r.Context().Value(keyUserID).(int))
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// if access policy is set, then checking whether
	// this client has access to this endpoint
	if e.ap != nil {
	}

	spew.Dump(u)

	/*
		if err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
	*/

	e.handler.ServeHTTP(w, r)
}
