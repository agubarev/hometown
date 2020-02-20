package endpoints

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/auth"
	"github.com/agubarev/hometown/pkg/util"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// HandleSignin handles user authentication
func HandleSignin(w http.ResponseWriter, r *http.Request) {
	// unmarshaling credentials
	creds := auth.UserCredentials{}
	ri := auth.NewRequestInfo(r)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.WriteResponseErrorTo(w, "invalid_payload", err, http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &creds)
	if err != nil {
		util.WriteResponseErrorTo(w, "invalid_payload", err, http.StatusBadRequest)
		return
	}

	// performing basic validation of credentials
	err = creds.Validate()
	if err != nil {
		util.WriteResponseErrorTo(w, "invalid_payload", err, http.StatusBadRequest)
		return
	}

	// obtaining an authenticator
	a := r.Context().Value(auth.CKAuthenticator).(*auth.Authenticator)
	if a == nil {
		util.WriteResponseErrorTo(w, "internal", auth.ErrNilAuthenticator, http.StatusInternalServerError)
		return
	}

	// authenticating
	user, err := a.Authenticate(creds.Username, creds.Password, ri)
	if err != nil {
		switch err {
		case core.ErrUserNotFound:
			util.WriteResponseErrorTo(w, "auth_failed", auth.ErrAuthenticationFailed, http.StatusUnauthorized)
		case auth.ErrAuthenticationFailed:
			util.WriteResponseErrorTo(w, "auth_failed", err, http.StatusUnauthorized)
		case auth.ErrUserSuspended:
			util.WriteResponseErrorTo(w, "user_suspended", err, http.StatusUnauthorized)
		}

		return
	}

	// generating a token trinity
	tokenTrinity, err := a.GenerateTokenTrinity(user, ri)
	if err != nil {
		util.WriteResponseErrorTo(w, "internal", err, http.StatusInternalServerError)
		return
	}

	// logging time
	if err := user.LastLoginAt.Scan(time.Now()); err != nil {
		util.WriteResponseErrorTo(w, "internal", fmt.Errorf("failed to scan time"), http.StatusInternalServerError)
		return
	}

	// updating IP from where the user has just authenticated from
	user.LastLoginIP = ri.IP.String()

	// logging user
	if err := user.Save(ctx); err != nil {
		util.WriteResponseErrorTo(w, "internal", fmt.Errorf("failed to save user"), http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(tokenTrinity)
	if err != nil {
		util.WriteResponseErrorTo(
			w,
			"internal",
			fmt.Errorf("failed to marshal token trinity: %s", err),
			http.StatusInternalServerError,
		)

		return
	}

	w.Header().Set("Content-Type", "application/text")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}
