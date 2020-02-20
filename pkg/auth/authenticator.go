package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/token"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"
)

// ContextKey is a named context key type for this package
type ContextKey uint8

// context keys
const (
	CKAuthenticator ContextKey = iota
	CKUserManager
	CKUser
)

// Claims holds required JWT claims
type Claims struct {
	UserID int64    `json:"uid"`
	Roles  []string `json:"rs,omitempty"`
	Groups []string `json:"gs,omitempty"`

	jwt.StandardClaims
}

// Session represents a user session
// NOTE: the session is used only to identify the session owner (user),
// verify the user's IP and UserAgent, and when to expire
// WARNING: session object must never be shared with the client,
// because it contains the refresh token
type Session struct {
	Token         string    `json:"t,omitempty"`
	UserID        int64     `json:"uid,omitempty"`
	IP            string    `json:"ip,omitempty"`
	UserAgent     string    `json:"ua,omitempty"`
	AccessTokenID string    `json:"jti,omitempty"`
	RefreshToken  string    `json:"rtok,omitempty"`
	ExpireAt      time.Time `json:"eat,omitempty"`
	CreatedAt     time.Time `json:"cat,omitempty"`
}

// Validate validates the session
func (s *Session) Validate() error {
	if s.Token == "" {
		return errors.New("empty token")
	}

	if s.UserID == 0 {
		return errors.New("user id not set")
	}

	if s.ExpireAt.IsZero() {
		return errors.New("expiration time not set")
	}

	if s.ExpireAt.Before(time.Now()) {
		return ErrTokenExpired
	}

	return nil
}

// RefreshTokenPayload represents the payload of a refresh token
type RefreshTokenPayload struct {
	UserID    int64  `json:"uid,omitempty"`
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"ua,omitempty"`
}

// TokenTrinity is what is returned upon a successful
// authentication by credentials, or by using a refresh token
// NOTE: typically, refresh token stays the same when obtained via refresh token
type TokenTrinity struct {
	SessionToken string `json:"session_token,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// ClientCredentials represents client authentication credentials
type ClientCredentials struct {
	ClientID string `json:"client_id"`
	Secret   string `json:"secret"`
}

// UserCredentials represents user authentication credentials
type UserCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Validate performs basic trimming and validations
func (c *UserCredentials) Validate() error {
	// trimming whitespace
	c.Username = strings.TrimSpace(c.Username)
	c.Password = strings.TrimSpace(c.Password)

	if c.Username == "" {
		return ErrEmptyUsername
	}

	if c.Password == "" {
		return ErrEmptyPassword
	}

	return nil
}

// RevokedAccessToken represents a blacklisted access token
type RevokedAccessToken struct {
	TokenID  string
	ExpireAt time.Time
}

// RequestMetadata holds request information
type RequestMetadata struct {
	IP        net.IP
	UserAgent string
}

// NewRequestInfo initializes RequestInfo from a given http.Request
func NewRequestInfo(r *http.Request) *RequestMetadata {
	// this is just a convenience for tests
	if r == nil {
		return &RequestMetadata{
			IP:        net.IPv4(0, 0, 0, 0),
			UserAgent: "",
		}
	}

	sip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil && err.Error() != "missing port in address" {
		log.Printf("NewRequestInfo: net.SplitHostPort failed: %s", err)
		return nil
	}

	return &RequestMetadata{
		IP:        net.ParseIP(sip),
		UserAgent: r.UserAgent(),
	}
}

// Validate validates revoked token
func (ri *RevokedAccessToken) Validate() error {
	// a little side-effect that won't hurt
	ri.TokenID = strings.TrimSpace(ri.TokenID)

	if ri.TokenID == "" {
		return ErrInvalidTokenID
	}

	if ri.ExpireAt.IsZero() {
		return ErrInvalidExpirationTime
	}

	return nil
}

// Authenticator represents an authenticator which is responsible
// for the user authentication and authorization
type Authenticator struct {
	UserManager     *user.UserManager
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	backend    Backend
	privateKey *rsa.PrivateKey
	logger     *zap.Logger
}

// NewAuthenticator initializes a new authenticator
// NOTE: if private key is nil, then using an autogenerated key
func NewAuthenticator(pk *rsa.PrivateKey, um *user.UserManager, b Backend) (*Authenticator, error) {
	if um == nil {
		return nil, core.ErrNilUserManager
	}

	// validating the supplied user manager
	if err := um.Validate(); err != nil {
		return nil, err
	}

	// if the private key is nil, then generating a temporary
	// key only for this instance
	if pk == nil {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %s", err)
		}

		pk = k
	}

	// using a default in-memory backend if nil is presented
	if b == nil {
		b = NewDefaultRegistryBackend()
	}

	// initializing new authenticator
	m := &Authenticator{
		UserManager:     um,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,

		backend:    b,
		privateKey: pk,
	}

	return m, nil
}

// SetLogger sets an own logger
func (a *Authenticator) SetLogger(logger *zap.Logger) error {
	if logger != nil {
		logger = logger.Named("[AUTHENTICATOR]")
	}

	a.logger = logger

	return nil
}

// Logger returns own logger
func (a *Authenticator) Logger() *zap.Logger {
	if a.logger == nil {
		l, err := zap.NewDevelopment()
		if err != nil {
			panic(fmt.Errorf("failed to initialize authenticator logger: %s", err))
		}

		a.logger = l
	}

	return a.logger
}

// PrivateKey returns a private key (RSA) used by this manager
func (a *Authenticator) PrivateKey() (*rsa.PrivateKey, error) {
	if a.privateKey == nil {
		return nil, ErrNilPrivateKey
	}

	return a.privateKey, nil
}

// Authenticate authenticates a user by a given username and password
func (a *Authenticator) Authenticate(username string, password string, ri *RequestMetadata) (*user.User, error) {
	user, err := a.UserManager.GetByKey("username", username)
	if err != nil {
		return nil, err
	}

	// obtaining logger
	l := a.Logger().With(
		zap.Int64("user_id", user.ID),
		zap.String("username", user.Username),
		zap.String("ip", ri.IP.String()),
		zap.String("user_agent", ri.UserAgent),
	)

	// before authentication, checking whether this user is suspended
	if user.IsSuspended {
		l.Info(
			"suspended user signin attempt",
			zap.Time("suspended_at", user.SuspendedAt.Time),
			zap.Time("suspension_expires_at", user.SuspensionExpiresAt.Time),
		)

		return nil, ErrUserSuspended
	}

	// obtaining password manager
	pm, err := a.UserManager.PasswordManager()
	if err != nil {
		l.Warn("failed to obtain password manager", zap.Error(err))
		return nil, err
	}

	// obtaining user's password
	userpass, err := pm.Get(user)
	if err != nil {
		l.Info("password not found", zap.Error(err))
		return nil, err
	}

	// comparing passwords
	if !userpass.Compare(password) {
		l.Info("wrong password signin attempt")
		return nil, ErrAuthenticationFailed
	}

	l.Info("authenticated by credentials")

	return user, nil
}

// AuthenticateByRefreshToken authenticates a user by a given refresh token
func (a *Authenticator) AuthenticateByRefreshToken(t *token.Token, ri *RequestMetadata) (*user.User, error) {
	tm, err := a.UserManager.TokenManager()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain a token manager: %s", err)
	}

	// validating refresh token
	err = t.Validate()
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// unmarshaling the payload
	payload := RefreshTokenPayload{}
	if err = json.Unmarshal(t.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %s", err)
	}

	// obtaining a user specified in the token's payload
	user, err := a.UserManager.GetByKey("id", payload.UserID)
	if err != nil {
		return nil, err
	}

	// obtaining logger
	l := a.Logger().With(
		zap.Int64("user_id", user.ID),
		zap.String("username", user.Username),
		zap.String("ip", ri.IP.String()),
		zap.String("user_agent", ri.UserAgent),
	)

	// comparing IPs
	// TODO implement a more flexible way instead of comparing just strings
	if payload.IP != ri.IP.String() {
		// IPs don't match, thus deleting this refresh token
		// to prevent any further use (safety first)
		if err = tm.Delete(t); err != nil {
			l.Warn("failed to delete refresh token", zap.Error(err), zap.String("token", t.Token))
			return nil, fmt.Errorf("failed to delete refresh token: %s", t.Token)
		}

		return nil, ErrWrongIP
	}

	// comparing User-Agent strings
	if payload.UserAgent != ri.UserAgent {
		// given user agent doesn't match to what's saved in the session
		// deleting session because it could've been exposed (safety first)
		if err = tm.Delete(t); err != nil {
			l.Warn("failed to delete refresh token", zap.Error(err), zap.String("token", t.Token))
			return nil, fmt.Errorf("failed to delete refresh token: %s", t.Token)
		}
	}

	// before authentication, checking whether this user is suspended
	if user.IsSuspended {
		l.Info(
			"suspended user signin attempt (via refresh token)",
			zap.Time("suspended_at", user.SuspendedAt.Time),
			zap.Time("suspension_expires_at", user.SuspensionExpiresAt.Time),
		)

		// since this user is suspended, then it's safe to assume
		// that this token is a liability and a possible threat,
		// and... is asking to be deleted
		err = tm.Delete(t)
		if err != nil {
			l.Warn("failed to delete refresh token", zap.Error(err), zap.String("token", t.Token))
			return nil, fmt.Errorf("failed to delete refresh token: %s", t.Token)
		}

		return nil, ErrUserSuspended
	}

	l.Info("authenticated by refresh token")

	return user, nil
}

// DestroySession destroys session by token, and as a given user
func (a *Authenticator) DestroySession(destroyedBy *user.User, stok string, ri *RequestMetadata) error {
	if destroyedBy == nil {
		return core.ErrNilUser
	}

	// obtaining token manager
	tm, err := a.UserManager.TokenManager()
	if err != nil {
		return err
	}

	// obtaining session from the backend to verify
	s, err := a.backend.GetSession(stok)
	if err != nil {
		return err
	}

	// verifying whether this session belongs to this revoker
	if s.UserID != destroyedBy.ID {
		return ErrWrongUser
	}

	if s.UserAgent != ri.UserAgent {
		return ErrWrongUserAgent
	}

	if s.IP != ri.IP.String() {
		return ErrWrongIP
	}

	// obtaining refresh token
	rtok, err := tm.Get(s.RefreshToken)
	if err != nil {
		return err
	}

	// deleting refresh token
	err = tm.Delete(rtok)
	if err != nil {
		return err
	}

	// verifying refresh token ownership

	// revoking a corresponding access token
	err = a.RevokeAccessToken(s.AccessTokenID, s.ExpireAt)
	if err != nil {
		return err
	}

	return a.backend.DeleteSession(s)
}

// GenerateAccessToken generates access token for a given user
func (a *Authenticator) GenerateAccessToken(user *user.User) (string, string, error) {
	if user == nil {
		return "", "", core.ErrNilUser
	}

	// slicing group names
	gs := make([]string, 0)
	rs := make([]string, 0)

	for _, g := range user.Groups(group.GKAll) {
		switch g.Kind {
		case group.GKRole:
			rs = append(rs, g.Key)
		case group.GKGroup:
			gs = append(gs, g.Key)
		}
	}

	// token id
	jti := util.NewULID().String()

	// generating and signing a new token
	atok := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{
		UserID: user.ID,
		Roles:  rs,
		Groups: gs,
		StandardClaims: jwt.StandardClaims{
			Issuer:    "hometown",
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(a.AccessTokenTTL).Unix(),
			Id:        jti,
		},
	})

	// obtaining private key
	pk, err := a.PrivateKey()
	if err != nil {
		return "", "", fmt.Errorf("failed to obtain private key: %s", err)
	}

	// creating an access token
	ss, err := atok.SignedString(pk)
	if err != nil {
		return "", "", fmt.Errorf("failed to obtain a signed token string: %s", err)
	}

	return ss, jti, nil
}

// GenerateRefreshToken generates a refresh token for a given user
func (a *Authenticator) GenerateRefreshToken(user *user.User, ri *RequestMetadata) (*token.Token, error) {
	if user == nil {
		return nil, core.ErrNilUser
	}

	// obtaining token manager
	tm, err := a.UserManager.TokenManager()
	if err != nil {
		return nil, err
	}

	return tm.Create(
		token.TkRefreshToken,
		RefreshTokenPayload{
			UserID:    user.ID,
			UserAgent: ri.UserAgent,
			IP:        ri.IP.String(),
		},
		a.RefreshTokenTTL,
		-1,
	)
}

// CreateSession generates a user session
// NOTE: the session uses AccessTokenTTL for its own expiry
func (a *Authenticator) CreateSession(user *user.User, ri *RequestMetadata, jti string, rtok *token.Token) (*Session, error) {
	if user == nil {
		return nil, core.ErrNilUser
	}

	// generating token
	buf, err := util.NewCSPRNG(24)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CSPRNG token: %s", err)
	}

	// initializing the actual session
	s := &Session{
		Token:         base64.URLEncoding.EncodeToString(buf),
		UserID:        user.ID,
		IP:            ri.IP.String(),
		UserAgent:     ri.UserAgent,
		AccessTokenID: jti,
		CreatedAt:     time.Now(),
		ExpireAt:      time.Now().Add(a.AccessTokenTTL),
		RefreshToken:  rtok.Token,
	}

	// adding session to the registry
	if err = a.RegisterSession(s); err != nil {
		return nil, fmt.Errorf("failed to add user session to the registry: %s", err)
	}

	return s, nil
}

// GenerateTokenTrinity generates a trinity of tokens: session, access and refresh
func (a *Authenticator) GenerateTokenTrinity(user *user.User, ri *RequestMetadata) (*TokenTrinity, error) {
	atok, jti, err := a.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	rtok, err := a.GenerateRefreshToken(user, ri)
	if err != nil {
		return nil, err
	}

	session, err := a.CreateSession(user, ri, jti, rtok)
	if err != nil {
		return nil, err
	}

	tt := &TokenTrinity{
		SessionToken: session.Token,
		AccessToken:  atok,
		RefreshToken: rtok.Token,
	}

	return tt, nil
}

// UserFromToken validates a JWT token and returns a corresponding user
func (a *Authenticator) UserFromToken(tok string) (*user.User, error) {
	// obtaining manager's private key
	pk, err := a.PrivateKey()
	if err != nil {
		return nil, ErrNilPrivateKey
	}

	// custom claims
	claims := new(Claims)

	// validating and parsing token
	token, err := jwt.ParseWithClaims(tok, claims, func(t *jwt.Token) (interface{}, error) {
		// making sure that proper signing method is used
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}

		return pk.Public(), nil
	})

	if err != nil {
		return nil, err
	}

	// validating a token on a time basis (i.e. expired, not eligible yet)
	if !token.Valid {
		return nil, ErrInvalidAccessToken
	}

	return a.UserManager.GetByKey("id", claims.UserID)
}

// RevokeAccessToken adds a given token ID to the blacklist for the duration
// of an access token expiration time + 1 minute
func (a *Authenticator) RevokeAccessToken(id string, eat time.Time) error {
	return a.backend.PutRevokedAccessToken(RevokedAccessToken{
		TokenID:  id,
		ExpireAt: eat,
	})
}

// IsRevoked checks whether a given token ID is among the blacklisted tokens
// NOTE: blacklisted token IDs are cleansed from the registry shortly
// after their respective access tokens expire
func (a *Authenticator) IsRevoked(tokenID string) bool {
	return a.backend.IsRevoked(tokenID)
}

// RegisterSession adds a given session
func (a *Authenticator) RegisterSession(s *Session) error {
	if s == nil {
		return ErrNilSession
	}

	return a.backend.PutSession(s)
}

// GetSession returns a session if it's found in the backend
// by its token string
func (a *Authenticator) GetSession(stok string) (*Session, error) {
	return a.backend.GetSession(stok)
}

// GetSessionByAccessToken returns a session by access token
func (a *Authenticator) GetSessionByAccessToken(jti string) (*Session, error) {
	return a.backend.GetSessionByAccessToken(jti)
}

// GetSessionBySessionToken returns a session by access token
func (a *Authenticator) GetSessionBySessionToken(rtok string) (*Session, error) {
	return a.backend.GetSessionByRefreshToken(rtok)
}
