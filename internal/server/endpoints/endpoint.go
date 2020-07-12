package endpoints

import (
	"bytes"
	"context"
	"database/sql/driver"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agubarev/hometown/internal/core"
	"github.com/davecgh/go-spew/spew"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type TName [32]byte

func NewName(s string) (name TName) {
	copy(name[:], strings.ToLower(strings.TrimSpace(s)))
	return name
}

func (name *TName) Scan(v interface{}) error {
	copy(name[:], v.([]byte))
	return nil
}

func (name TName) Value() (driver.Value, error) {
	if name[0] == 0 {
		return "", nil
	}

	// finding position of zero
	zeroPos := bytes.IndexByte(name[:], byte(0))
	if zeroPos == -1 {
		return name[:], nil
	}

	return name[0:zeroPos], nil
}

type ContextKey int

// context keys
const (
	CKUserID ContextKey = iota
)

type Endpoint struct {
	name    TName
	core    *core.Core
	ctx     context.Context
	handler Handler
}

// Handler represents a custom handler
type Handler func(ctx context.Context, c *core.Core, w http.ResponseWriter, r *http.Request) (result interface{}, code int, err error)

// Response is the main response wrapper
type Response struct {
	Error         error         `json:"error"`
	Result        interface{}   `json:"result,omitempty"`
	ExecutionTime time.Duration `json:"execution_time"`
}

func NewEndpoint(ctx context.Context, c *core.Core, h Handler, name TName) Endpoint {
	if c == nil {
		panic(core.ErrNilCore)
	}

	return Endpoint{
		ctx:     ctx,
		core:    c,
		name:    name,
		handler: h,
	}
}

func (e Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, err := e.core.UserManager().UserByID(
		r.Context(),
		r.Context().Value(CKUserID).(int64),
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	spew.Dump(u)

	start := time.Now()

	// executing handler
	result, code, err := e.handler(e.ctx, e.core, ac, w, r)
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
