package middleware

import (
	"context"
	"net/http"

	"github.com/agubarev/hometown/pkg/security/auth"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func Authenticator(extractor func(r *http.Request) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// obtaining authenticator from request context
			authenticator, ok := r.Context().Value(auth.CKAuthenticator).(*auth.Authenticator)
			if !ok || authenticator == nil {
				panic(auth.ErrNilAuthenticator)
			}

			// user manager
			userManager, ok := r.Context().Value(user.CKUserManager).(*user.Manager)
			if !ok || userManager == nil {
				panic(auth.ErrNilUserManager)
			}

			logger := authenticator.Logger()

			// using supplied function to extract request credentials
			signedToken, err := extractor(r)
			if err != nil {
				logger.Warn(
					"failed to extract access token",
					zap.Error(err),
				)

				return
			}

			// obtaining a session associated with this access token
			session, err := authenticator.SessionByAccessToken(r.Context(), signedToken)
			if err != nil {
				if errors.Cause(err) != auth.ErrSessionNotFound {
					logger.Error(
						"failed to obtain session by access token",
						zap.Error(errors.Cause(err)),
					)

					w.WriteHeader(http.StatusInternalServerError)

					return
				}

				w.WriteHeader(http.StatusUnauthorized)

				return
			}

			// validating session
			if session.IsRevoked() {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// injecting session into context
			ctx := context.WithValue(r.Context(), auth.CKSession, session)

			switch session.Identity.Kind {
			case auth.IKUser:
				usr, err := userManager.UserByID(ctx, session.Identity.ID)
				if err != nil {
					if errors.Cause(err) != user.ErrUserNotFound {
						logger.Error(
							"failed to obtain user",
							zap.String("atok", signedToken),
							zap.String("user_id", session.Identity.ID.String()),
							zap.Error(errors.Cause(err)),
						)
					}

					w.WriteHeader(http.StatusInternalServerError)

					return
				}

				// checking user status
				if usr.IsSuspended {
					logger.Info(
						"authentication attempt by a suspended user (access token)",
						zap.Time("suspended_at", usr.SuspendedAt),
						zap.Time("suspension_expires_at", usr.SuspensionExpiresAt),
					)

					w.WriteHeader(http.StatusUnauthorized)

					return
				}

				logger.Debug(
					"user authenticated (access token)",
					zap.String("id", usr.ID.String()),
					zap.String("username", usr.Username),
					zap.String("ip", session.IP.String()),
				)

				ctx = context.WithValue(ctx, user.CKUser, usr)
			case auth.IKApplication:
				// TODO: inject client
			default:
				logger.Debug(
					"unrecognized identity kind",
					zap.Int("identity_kind", int(session.Identity.Kind)),
					zap.String("ip", session.IP.String()),
				)

				w.WriteHeader(http.StatusUnauthorized)

				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
