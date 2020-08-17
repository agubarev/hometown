package auth

import "github.com/pkg/errors"

// errors
var (
	ErrNilUserManager        = errors.New("user manager is nil")
	ErrNilPrivateKey         = errors.New("private key is nil")
	ErrAuthenticationFailed  = errors.New("authentication failed")
	ErrNilAuthenticator      = errors.New("authenticator is nil")
	ErrEmptyUsername         = errors.New("username is empty")
	ErrEmptyPassword         = errors.New("password is empty")
	ErrUserSuspended         = errors.New("user is suspended")
	ErrInvalidAccessToken    = errors.New("invalid accesspolicy token")
	ErrInvalidRefreshToken   = errors.New("invalid refresh token")
	ErrRefreshTokenExpired   = errors.New("refresh token has expired")
	ErrIPAddrMismatch        = errors.New("ip address mismatch")
	ErrIdentityMismatch      = errors.New("identity mismatch")
	ErrUserAgentMismatch     = errors.New("user agent mismatch")
	ErrNilSession            = errors.New("session is nil")
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionAlreadyRevoked = errors.New("session is already revoked")
	ErrInvalidRevocationFlag = errors.New("invalid revocation flag")
	ErrInvalidIdentityID     = errors.New("invalid identity id")
	ErrRefreshTokenNotFound  = errors.New("refresh token not found")
	ErrInvalidJTI            = errors.New("invalid access token id")
	ErrInvalidSessionID      = errors.New("invalid session id")
	ErrInvalidExpirationTime = errors.New("invalid expiration time")
	ErrZeroExpiration        = errors.New("expiration time is zero")
	ErrRefreshTokenIsEmpty   = errors.New("refresh token is empty")
	ErrNilPasswordManager    = errors.New("password manager is nil")
)
