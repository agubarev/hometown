package middleware

import (
	"net/http"

	"github.com/agubarev/hometown/pkg/security/accesspolicy"
	"github.com/agubarev/hometown/pkg/security/auth"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func Policy(policy accesspolicy.Policy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// obtaining request logger
			logger, ok := r.Context().Value("rlog").(*zap.Logger)
			if !ok || logger == nil {
				panic(errors.New("request logger is nil"))
			}

			// user manager
			userManager, ok := r.Context().Value(user.CKUserManager).(*user.Manager)
			if !ok || userManager == nil {
				panic(auth.ErrNilUserManager)
			}

			// obtaining user from the context
			usr, ok := r.Context().Value(user.CKUser).(user.User)
			if !ok {
				panic(user.ErrNilUser)
			}

			// checking whether current user has viewing rights on this policy
			ok = userManager.AccessPolicyManager().HasRights(
				r.Context(),
				policy.ID,
				accesspolicy.NewActor(accesspolicy.AKUser, usr.ID),
				accesspolicy.APView,
			)

			// aborting middleware chain if the user has no access rights
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("access denied"))
				return
			}

			logger.Debug("request access granted",
				zap.String("user_id", usr.ID.String()),
				zap.String("username", usr.Username),
			)

			next.ServeHTTP(w, r)
		})
	}
}
