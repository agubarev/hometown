package endpoints

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/agubarev/hometown/pkg/auth"
	"github.com/agubarev/hometown/pkg/util"
)

// HandleRefreshToken returns a new access token given the supplied
// refresh token is valid
func HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	rbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.WriteResponseErrorTo(w, "invalid_payload", err, http.StatusBadRequest)
		return
	}

	// request info
	ri := auth.NewRequestInfo(r)

	// obtaining an authenticator
	a := r.Context().Value(auth.CKAuthenticator).(*auth.Authenticator)
	if a == nil {
		util.WriteResponseErrorTo(w, "internal", auth.ErrNilAuthenticator, http.StatusInternalServerError)
		return
	}

	// obtaining token manager
	tm, err := a.UserManager.TokenManager()
	if err != nil {
		util.WriteResponseErrorTo(w, "internal", err, http.StatusInternalServerError)
	}

	// obtaining the refresh token
	rtok, err := tm.Get(string(rbody))
	if err != nil {
		util.WriteResponseErrorTo(w, "refresh_failed", auth.ErrInvalidRefreshToken, http.StatusUnauthorized)
		return
	}

	// attempting authentication by a found refresh token
	user, err := a.AuthenticateByRefreshToken(rtok, ri)
	if err != nil {
		util.WriteResponseErrorTo(w, "refresh_failed", err, http.StatusUnauthorized)
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

	// generating new access token
	atok, jti, err := a.GenerateAccessToken(user)
	if err != nil {
		util.WriteResponseErrorTo(w, "refresh_failed", err, http.StatusUnauthorized)
		return
	}

	// generating new session
	s, err := a.CreateSession(user, ri, jti)
	if err != nil {
		util.WriteResponseErrorTo(w, "refresh_failed", err, http.StatusUnauthorized)
		return
	}

	// constructing a new trinity by combining a new session and access tokens,
	// along with an existing refresh token
	response, err := json.Marshal(auth.TokenTrinity{
		SessionToken: s.Token,
		AccessToken:  atok,
		RefreshToken: rtok.Token,
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
	w.Write([]byte(response))
}
