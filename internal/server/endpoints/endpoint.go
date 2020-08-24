package endpoints

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/util/report"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type contextKey uint

// context keys
const (
	CKAuthenticator contextKey = iota
	CKCore
)

type Endpoint struct {
	ctx     context.Context
	core    *core.Core
	name    string
	handler Handler
}

// Handler represents a custom handler
type Handler func(ctx context.Context, c *core.Core, w http.ResponseWriter, r *http.Request) (result interface{}, aux interface{}, code int, rep *report.Report)

type Response struct {
	Result        interface{}    `json:"result"`
	Auxiliary     interface{}    `json:"aux"`
	Report        *report.Report `json:"report"`
	ExecutionTime time.Duration  `json:"exec_time"`
}

// HTTPError represents a common error wrapper to be used
// as an HTTP error response
// WARNING: this is used only as a complete, and not an embedded error response
type HTTPError struct {
	Scope   string `json:"scope"`
	Key     string `json:"key"`
	Message string `json:"msg"`
	Code    int    `json:"code"`
}

func NewEndpoint(ctx context.Context, c *core.Core, h Handler, name string) (e Endpoint) {
	if c == nil {
		panic(core.ErrNilCore)
	}

	// basic validation
	name = strings.ToLower(strings.TrimSpace(name))

	if name == "" {
		panic(errors.New("empty endpoint name"))
	}

	e = Endpoint{
		ctx:     ctx,
		core:    c,
		name:    name,
		handler: h,
	}

	return e
}

func (e Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: preliminary checks (i.e.: authentication, etc...)
	// TODO: ...

	// injecting report into the context
	_, e.ctx = report.NewWithContext(e.ctx, nil)

	//---------------------------------------------------------------------------
	// processing request
	//---------------------------------------------------------------------------
	start := time.Now()

	// executing handler
	result, aux, code, rep := e.handler(e.ctx, e.core, w, r)

	// initializing response
	response := Response{
		Result:        result,
		Auxiliary:     aux,
		ExecutionTime: time.Since(start),
	}

	// adding report to the response only if report contains an error
	if rep.HasError() {
		response.Report = rep
	}

	// marshaling handler's result
	payload, err := json.Marshal(response)
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
