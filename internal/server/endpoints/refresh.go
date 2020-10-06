package endpoints

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/agubarev/hometown/pkg/security/auth"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/pkg/errors"
)

// HandleRefreshToken returns a new accesspolicy token given the supplied
// refresh token is valid
func HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	rbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.WriteResponseErrorTo(w, "invalid_payload", err, http.StatusBadRequest)
		return
	}

	// request info
	ri := auth.NewRequestMetadata(r)

	// obtaining an authenticator
	a := r.Context().Value(auth.CKAuthenticator).(*auth.Authenticator)
	if a == nil {
		util.WriteResponseErrorTo(w, "internal", auth.ErrNilAuthenticator, http.StatusInternalServerError)
		return
	}

	// obtaining token manager
	tm := a.TokenManager()

	// obtaining the refresh token
	rtok, err := tm.Get(r.Context(), string(rbody))
	if err != nil {
		util.WriteResponseErrorTo(w, "refresh_failed", auth.ErrInvalidRefreshToken, http.StatusUnauthorized)
		return
	}

	// attempting authentication by a found refresh token
	u, err := a.AuthenticateUserByRefreshToken(r.Context(), rtok, ri)
	if err != nil {
		util.WriteResponseErrorTo(w, "refresh_failed", err, http.StatusUnauthorized)
		return
	}

	// logging changes
	u, _, err = a.UserManager().UpdateUser(r.Context(), u.ID, func(ctx context.Context, u user.User) (_ user.User, err error) {
		if err := u.LastLoginAt.Scan(time.Now()); err != nil {
			util.WriteResponseErrorTo(w, "internal", fmt.Errorf("failed to scan last login time"), http.StatusInternalServerError)
			return u, err
		}

		// updating IPAddr from where the user has just authenticated from
		u.LastLoginIP = ri.IP

		return u, nil
	})

	if err != nil {
		util.WriteResponseErrorTo(w, "internal", errors.Wrap(err, "failed to update user"), http.StatusInternalServerError)
		return
	}

	// generating new accesspolicy token
	atok, jti, err := a.GenerateAccessToken(r.Context(), u)
	if err != nil {
		util.WriteResponseErrorTo(w, "refresh_failed", err, http.StatusUnauthorized)
		return
	}

	// generating new session
	s, err := a.CreateSession(r.Context(), u, ri, jti, rtok)
	if err != nil {
		util.WriteResponseErrorTo(w, "refresh_failed", err, http.StatusUnauthorized)
		return
	}

	// constructing a new trinity by combining a new session and accesspolicy tokens,
	// along with an existing refresh token
	response, err := json.Marshal(auth.TokenPair{
		SessionToken: s.Token,
		AccessToken:  atok,
		RefreshToken: rtok.Hash,
	})

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
