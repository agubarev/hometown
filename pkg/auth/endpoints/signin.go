package endpoints

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/agubarev/hometown/pkg/auth"
	"github.com/agubarev/hometown/pkg/user"
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

	if err = json.Unmarshal(body, &creds); err != nil {
		util.WriteResponseErrorTo(w, "invalid_payload", err, http.StatusBadRequest)
		return
	}

	// performing basic validation of credentials

	if err = creds.SanitizeAndValidate(); err != nil {
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
	u, err := a.Authenticate(r.Context(), creds.Username, creds.Password, ri)
	if err != nil {
		switch err {
		case user.ErrUserNotFound:
			util.WriteResponseErrorTo(w, "auth_failed", auth.ErrAuthenticationFailed, http.StatusUnauthorized)
		case auth.ErrAuthenticationFailed:
			util.WriteResponseErrorTo(w, "auth_failed", err, http.StatusUnauthorized)
		case auth.ErrUserSuspended:
			util.WriteResponseErrorTo(w, "user_suspended", err, http.StatusUnauthorized)
		}

		return
	}

	// generating a token trinity
	tokenTrinity, err := a.GenerateTokenTrinity(r.Context(), u, ri)
	if err != nil {
		util.WriteResponseErrorTo(w, "internal", err, http.StatusInternalServerError)
		return
	}

	// logging time
	if err = u.LastLoginAt.Scan(time.Now()); err != nil {
		util.WriteResponseErrorTo(w, "internal", fmt.Errorf("failed to scan time"), http.StatusInternalServerError)
		return
	}

	// updating IP from where the user has just authenticated from
	u.LastLoginIP = ri.IP.String()

	/*
		// logging user
		if err := u.Update(r.Context()); err != nil {
			util.WriteResponseErrorTo(w, "internal", fmt.Errorf("failed to save u"), http.StatusInternalServerError)
			return
		}
	*/

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
	w.Write(response)
}
