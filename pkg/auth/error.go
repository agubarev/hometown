package auth

import "errors"

// errors
var (
	ErrNilPrivateKey              = errors.New("private key is nil")
	ErrAuthenticationFailed       = errors.New("authentication failed")
	ErrNilAuthenticator           = errors.New("authenticator is nil")
	ErrEmptyUsername              = errors.New("username is empty")
	ErrEmptyPassword              = errors.New("password is empty")
	ErrUserSuspended              = errors.New("user is suspended")
	ErrInvalidAccessToken         = errors.New("invalid access token")
	ErrInvalidRefreshToken        = errors.New("invalid refresh token")
	ErrInvalidExpirationTime      = errors.New("invalid expiration time")
	ErrTokenExpired               = errors.New("token has expired")
	ErrWrongIP                    = errors.New("wrong ip")
	ErrWrongUser                  = errors.New("wrong user")
	ErrWrongUserAgent             = errors.New("wrong user agent")
	ErrInvalidRefreshTokenPayload = errors.New("invalid refresh token payload")
	ErrInvalidTokenID             = errors.New("invalid token id")
	ErrNilSession                 = errors.New("session is nil")
	ErrSessionNotFound            = errors.New("session not found")
)
