package endpoints

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/security/auth"
	"github.com/agubarev/hometown/pkg/util/report"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Endpoint struct {
	ctx     context.Context
	core    *core.Core
	name    string
	handler Handler
}

// Handler represents a custom handler
type Handler func(ctx context.Context, c *core.Core, w http.ResponseWriter, r *http.Request) (result interface{}, aux interface{}, code int, rep *report.Report)

type Response struct {
	RequestID     uuid.UUID      `json:"request_id"`
	Result        interface{}    `json:"result"`
	Auxiliary     interface{}    `json:"aux"`
	Report        *report.Report `json:"report"`
	ExecutionTime time.Duration  `json:"exec_time"`
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

// NOTE: using vanilla string context keys atm
func (e Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// injecting report into the context
	_, ctx := report.NewWithContext(e.ctx, nil)

	// generating request ID
	requestID := uuid.New()

	// injecting request ID into the context
	ctx = context.WithValue(ctx, "request_id", requestID)

	//---------------------------------------------------------------------------
	// handling access domain
	//---------------------------------------------------------------------------
	/*
		if domain, ok := ctx.Value(auth.CKDomain).(auth.Domain); ok && domain.IsProtected {
			// authenticating request
			s, err := e.core.Authenticator().AuthenticateByRequest(ctx, r)
			if err != nil {
				http.Error(
					w,
					"authentication failed",
					http.StatusUnauthorized,
				)

				return
			}

			// adding session to the context
			ctx = context.WithValue(ctx, "session", s)
		}
	*/

	//---------------------------------------------------------------------------
	// processing request
	//---------------------------------------------------------------------------
	start := time.Now()

	// executing handler
	result, aux, code, rep := e.handler(ctx, e.core, w, r)

	// initializing response
	response := Response{
		RequestID:     requestID,
		Result:        result,
		Auxiliary:     aux,
		ExecutionTime: time.Since(start),
	}

	// adding report to the response only if report contains an error
	if rep != nil && rep.HasError() {
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
