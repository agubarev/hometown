package endpoints

import (
	"net/http"
	"strings"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/davecgh/go-spew/spew"
)

func Post(c *core.Core, w http.ResponseWriter, r *http.Request) (result interface{}, code int, err error) {
	newUser := user.NewUserObject{
		Essential:        user.Essential{},
		ProfileEssential: user.ProfileEssential{},
		EmailAddr:        "",
		PhoneNumber:      "",
		Password:         nil,
	}

	/*
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	*/

	return newUser, http.StatusOK, nil
}

// MiddlewareAuth validates the authorization header and adds
// a corresponding user to the context
func MiddlewareAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if len(parts) != 2 {
			http.Error(w, "invalid authorization header", http.StatusBadRequest)
			return
		}

		// just a friendly piece of code for a longer shot (if things are to move onward)
		switch parts[0] {
		case "Bearer":
			spew.Dump(parts[1])
		default:
			http.Error(w, "unsupported authorization type", http.StatusUnauthorized)
			return
		}

		// obtaining user from the manager

		// extending request context
		//ctx := context.WithValue(r.Context(), endpoints.CKUserID, userID)

		//next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MWAuthentication(c *core.Core) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return NewEndpoint(NewName("authentication"), c, func(c *core.Core, w http.ResponseWriter, r *http.Request) (result interface{}, code int, err error) {
			spew.Dump("AUTHENTICATION MIDDLEWARE")

			return result, http.StatusOK, nil
		})
	}
}
