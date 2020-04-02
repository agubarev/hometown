package endpoints

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/davecgh/go-spew/spew"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type contextKey int

// context keys
const (
	keyUserID contextKey = iota
)

type Endpoint struct {
	ctx         context.Context
	ap          *user.AccessPolicy
	isProtected bool
	core        *core.Core
	name        string
	handler     Handler
}

// Handler represents a custom handler
type Handler func(ctx context.Context, c *core.Core, w http.ResponseWriter, r *http.Request) (result interface{}, code int, err error)

type Response struct {
	Error         error         `json:"error"`
	Result        interface{}   `json:"result,omitempty"`
	ExecutionTime time.Duration `json:"execution_time"`
}

func NewEndpoint(ctx context.Context, c *core.Core, isProtected bool, h Handler, name string) (e Endpoint) {
	if c == nil {
		panic(core.ErrNilCore)
	}

	e = Endpoint{
		ctx:         ctx,
		isProtected: isProtected,
		core:        c,
		name:        name,
		handler:     h,
	}

	return e
}

func (e Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, err := e.core.UserManager().UserByID(
		r.Context(),
		r.Context().Value(keyUserID).(int64),
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// if access policy is set, then checking whether
	// this client has access to this Endpoint
	if e.ap != nil {
		http.Error(w, "this endpoint is protected by access policy", http.StatusForbidden)
		return
	}

	spew.Dump(u)

	start := time.Now()

	// executing handler
	result, code, err := e.handler(e.ctx, e.core, w, r)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	// marshaling handler's result
	payload, err := json.Marshal(Response{
		Error:         err,
		Result:        result,
		ExecutionTime: time.Since(start),
	})

	if err != nil {
		http.Error(
			w,
			errors.Wrap(err, "failed to marshal server response").Error(),
			http.StatusInternalServerError,
		)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.WriteHeader(code)
	w.Write(payload)
}
