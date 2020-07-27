package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/agubarev/hometown/pkg/token"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Context holds metadata which describes authenticated session
type Context struct {
	UserID uint32
	Domain
}

// Domain represents an accesspolicy domain
type Domain uint8

const (
	DGeneric Domain = 0
	DAdmin   Domain = 1 << (iota - Domain(1))
)

func (d Domain) String() string {
	switch d {
	case DGeneric:
		return "generic domain"
	case DAdmin:
		return "administrative domain"
	default:
		return "unrecognized domain"
	}
}

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
	UserID uuid.UUID    `json:"uid"`
	Roles  []group.TKey `json:"rs,omitempty"`
	Groups []group.TKey `json:"gs,omitempty"`

	jwt.StandardClaims
}

// Session represents a user session
// NOTE: the session is used only to identify the session owner (user),
// verify the user's IPAddr and UserAgent, and when to expire
// WARNING: session object must never be shared with the client,
// because it contains the refresh token
type Session struct {
	Token         string    `json:"t,omitempty"`
	UserID        uint32    `json:"uid,omitempty"`
	IP            string    `json:"ip,omitempty"`
	UserAgent     string    `json:"ua,omitempty"`
	AccessTokenID string    `json:"jti,omitempty"`
	RefreshToken  string    `json:"rtok,omitempty"`
	ExpireAt      time.Time `json:"eat,omitempty"`
	CreatedAt     time.Time `json:"cat,omitempty"`
}

// SanitizeAndValidate validates the session
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
	UserID    uint32 `json:"uid,omitempty"`
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
	Password []byte `json:"password"`
}

type Config struct {
	CompareIP        bool
	CompareUserAgent bool
}

func NewDefaultConfig() Config {
	return Config{
		CompareIP:        false,
		CompareUserAgent: true,
	}
}

// SanitizeAndValidate performs basic trimming and validations
// TODO: consider preserving original username but still change case to lower
// NOTE: trimming passwords whitespace to prevent problems when people copy+paste
func (c *UserCredentials) SanitizeAndValidate() error {
	c.Username = strings.ToLower(strings.TrimSpace(c.Username))
	c.Password = bytes.TrimSpace(c.Password)

	if c.Username == "" {
		return ErrEmptyUsername
	}

	if len(c.Password) == 0 {
		return ErrEmptyPassword
	}

	return nil
}

// RevokedAccessToken represents a blacklisted accesspolicy token
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

// SanitizeAndValidate validates revoked token
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
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	config     Config
	users      *user.Manager
	backend    Backend
	privateKey *rsa.PrivateKey
	logger     *zap.Logger
}

// NewAuthenticator initializes a new authenticator
// NOTE: if private key is nil, then using an autogenerated key
func NewAuthenticator(pk *rsa.PrivateKey, um *user.Manager, b Backend, cfg Config) (*Authenticator, error) {
	if um == nil {
		return nil, ErrNilUserManager
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
		users:           um,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,

		config:     cfg,
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

func (a *Authenticator) UserManager() *user.Manager {
	if a.users == nil {
		panic(ErrNilUserManager)
	}

	return a.users
}

func (a *Authenticator) GroupManager() *group.Manager {
	return a.UserManager().GroupManager()
}

func (a *Authenticator) PasswordManager() password.Manager {
	return a.UserManager().PasswordManager()
}

func (a *Authenticator) TokenManager() *token.Manager {
	return a.UserManager().TokenManager()
}

// PrivateKey returns a private key (RSA) used by this manager
func (a *Authenticator) PrivateKey() (*rsa.PrivateKey, error) {
	if a.privateKey == nil {
		return nil, ErrNilPrivateKey
	}

	return a.privateKey, nil
}

// Authenticate authenticates a user by a given username and password
func (a *Authenticator) Authenticate(ctx context.Context, username string, rawpass []byte, ri *RequestMetadata) (u user.User, err error) {
	u, err = a.UserManager().UserByUsername(ctx, username)
	if err != nil {
		return u, err
	}

	// obtaining logger
	l := a.Logger().With(
		zap.String("user_id", u.ID.String()),
		zap.String("username", u.Username.String()),
		zap.String("ip", ri.IP.String()),
		zap.String("user_agent", ri.UserAgent),
	)

	// before authentication, checking whether this user is suspended
	if u.IsSuspended {
		l.Info(
			"suspended user signin attempt",
			zap.Time("suspended_at", util.TimeFromU32Unix(u.SuspendedAt)),
			zap.Time("suspension_expires_at", util.TimeFromU32Unix(u.SuspensionExpiresAt)),
		)

		return u, ErrUserSuspended
	}

	// obtaining password manager
	pm := a.PasswordManager()

	// obtaining user's password
	userpass, err := pm.Get(ctx, password.KUser, u.ID)
	if err != nil {
		if err == password.ErrPasswordNotFound {
			l.Info("password not found", zap.Error(err))
		} else {
			l.Warn("failed to obtain password", zap.Error(err))
		}

		return u, err
	}

	// comparing passwords
	if !userpass.Compare(rawpass) {
		l.Info("wrong password signin attempt")
		return u, ErrAuthenticationFailed
	}

	l.Info("authenticated by credentials")

	return u, nil
}

// AuthenticateByRefreshToken authenticates a user by a given refresh token
func (a *Authenticator) AuthenticateByRefreshToken(ctx context.Context, t *token.Token, ri *RequestMetadata) (u user.User, err error) {
	tm := a.TokenManager()

	// validating refresh token
	err = t.Validate()
	if err != nil {
		return u, ErrInvalidRefreshToken
	}

	// unmarshaling the payload
	payload := RefreshTokenPayload{}
	if err = json.Unmarshal(t.Payload, &payload); err != nil {
		return u, fmt.Errorf("failed to unmarshal payload: %s", err)
	}

	// obtaining a user specified in the token's payload
	u, err = a.users.UserByID(ctx, payload.UserID)
	if err != nil {
		return u, err
	}

	// obtaining logger
	l := a.Logger().With(
		zap.Uint32("user_id", u.ID),
		zap.String("username", u.Username),
		zap.String("ip", ri.IP.String()),
		zap.String("user_agent", ri.UserAgent),
	)

	// comparing IPs
	if a.config.CompareIP {
		// TODO implement a more flexible way instead of comparing just strings
		if payload.IP != ri.IP.String() {
			// IPs don't match, thus deleting this refresh token
			// to prevent any further use (safety first)
			if err = tm.Delete(ctx, t); err != nil {
				l.Warn("IPAddr MISMATCH: failed to delete refresh token", zap.Error(err), zap.String("token", t.Hash))
				return u, fmt.Errorf("failed to delete refresh token: %s", t.Hash)
			}

			return u, ErrWrongIP
		}
	}

	// comparing User-Agent strings
	if a.config.CompareUserAgent {
		if payload.UserAgent != ri.UserAgent {
			// given user agent doesn't match to what's saved in the session
			// deleting session because it could've been exposed (safety first)
			if err = tm.Delete(ctx, t); err != nil {
				l.Warn("USER-AGENT MISMATCH: failed to delete refresh token", zap.Error(err), zap.String("token", t.Hash))
				return u, fmt.Errorf("failed to delete refresh token: %s", t.Hash)
			}
		}
	}

	// before authentication, checking whether this user is suspended
	if u.IsSuspended {
		l.Info(
			"suspended user signin attempt (via refresh token)",
			zap.Time("suspended_at", u.SuspendedAt.Time),
			zap.Time("suspension_expires_at", u.SuspensionExpiresAt.Time),
		)

		// since this user is suspended, then it's safe to assume
		// that this token is a liability and a possible threat,
		// and... is asking to be deleted
		if err = tm.Delete(ctx, t); err != nil {
			l.Warn("USER SUSPENDED: failed to delete refresh token", zap.Error(err), zap.String("token", t.Hash))
			return u, fmt.Errorf("failed to delete refresh token: %s", t.Hash)
		}

		return u, ErrUserSuspended
	}

	l.Info("authenticated by refresh token")

	return u, nil
}

// DestroySession destroys session by token, and as a given user
func (a *Authenticator) DestroySession(ctx context.Context, destroyedByID uint32, stok string, ri *RequestMetadata) error {
	if destroyedByID == 0 {
		return user.ErrZeroUserID
	}

	// obtaining token manager
	tm := a.TokenManager()

	// obtaining session from the backend to verify
	s, err := a.backend.GetSession(stok)
	if err != nil {
		return err
	}

	// verifying whether this session belongs to this revoker
	if s.UserID != destroyedByID {
		return ErrWrongUser
	}

	if s.UserAgent != ri.UserAgent {
		return ErrWrongUserAgent
	}

	if s.IP != ri.IP.String() {
		return ErrWrongIP
	}

	// obtaining refresh token
	rtok, err := tm.Get(ctx, s.RefreshToken)
	if err != nil {
		return err
	}

	// deleting refresh token
	err = tm.Delete(ctx, rtok)
	if err != nil {
		return err
	}

	// verifying refresh token ownership

	// revoking a corresponding accesspolicy token
	err = a.RevokeAccessToken(s.AccessTokenID, s.ExpireAt)
	if err != nil {
		return err
	}

	return a.backend.DeleteSession(s)
}

// GenerateAccessToken generates accesspolicy token for a given user
// TODO: add dynamic token realm
func (a *Authenticator) GenerateAccessToken(ctx context.Context, u user.User) (string, string, error) {
	if u.ID == uuid.Nil {
		return "", "", user.ErrNilUser
	}

	gm := a.GroupManager()

	// slicing group names
	gs := make([]group.TKey, 0)
	rs := make([]group.TKey, 0)

	for _, g := range gm.GroupsByAssetID(ctx, group.FAllGroups, u.ID) {
		switch g.Flags {
		case group.FRole:
			rs = append(rs, g.Key)
		case group.FGroup:
			gs = append(gs, g.Key)
		}
	}

	// token id
	jti := util.NewULID().String()

	// generating and signing a new token
	atok := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{
		UserID: u.ID,
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

	// creating an accesspolicy token
	ss, err := atok.SignedString(pk)
	if err != nil {
		return "", "", fmt.Errorf("failed to obtain a signed token string: %s", err)
	}

	return ss, jti, nil
}

// GenerateRefreshToken generates a refresh token for a given user
func (a *Authenticator) GenerateRefreshToken(ctx context.Context, u user.User, ri *RequestMetadata) (*token.Token, error) {
	if u.ID == uuid.Nil {
		return nil, user.ErrZeroUserID
	}

	// obtaining token manager
	tm := a.TokenManager()

	return tm.Create(
		ctx,
		token.TRefreshToken,
		RefreshTokenPayload{
			UserID:    u.ID,
			UserAgent: ri.UserAgent,
			IP:        ri.IP.String(),
		},
		a.RefreshTokenTTL,
		-1,
	)
}

// CreateSession generates a user session
// NOTE: the session uses AccessTokenTTL for its own expiry
func (a *Authenticator) CreateSession(ctx context.Context, u user.User, ri *RequestMetadata, jti string, rtok *token.Token) (sess Session, err error) {
	if u.ID == 0 {
		return sess, user.ErrZeroUserID
	}

	// generating token
	buf, err := util.NewCSPRNG(24)
	if err != nil {
		return sess, errors.Wrap(err, "failed to generate CSPRNG token")
	}

	// initializing the actual session
	sess = Session{
		Token:         base64.URLEncoding.EncodeToString(buf),
		UserID:        u.ID,
		IP:            ri.IP.String(),
		UserAgent:     ri.UserAgent,
		AccessTokenID: jti,
		CreatedAt:     time.Now(),
		ExpireAt:      time.Now().Add(a.AccessTokenTTL),
		RefreshToken:  rtok.Hash,
	}

	// adding session to the registry
	if err = a.RegisterSession(sess); err != nil {
		return sess, fmt.Errorf("failed to add user session to the registry: %s", err)
	}

	return sess, nil
}

// GenerateTokenTrinity generates a trinity of tokens: session, accesspolicy and refresh
func (a *Authenticator) GenerateTokenTrinity(ctx context.Context, user user.User, ri *RequestMetadata) (*TokenTrinity, error) {
	atok, jti, err := a.GenerateAccessToken(ctx, user)
	if err != nil {
		return nil, err
	}

	rtok, err := a.GenerateRefreshToken(ctx, user, ri)
	if err != nil {
		return nil, err
	}

	session, err := a.CreateSession(ctx, user, ri, jti, rtok)
	if err != nil {
		return nil, err
	}

	tt := &TokenTrinity{
		SessionToken: session.Token,
		AccessToken:  atok,
		RefreshToken: rtok.Hash,
	}

	return tt, nil
}

// ParseToken validates and parses a JWT token and returns its claims
func (a *Authenticator) claimsFromToken(tok string) (claims Claims, err error) {
	// obtaining manager's private key
	pk, err := a.PrivateKey()
	if err != nil {
		return claims, ErrNilPrivateKey
	}

	// validating and parsing token
	t, err := jwt.ParseWithClaims(tok, &claims, func(t *jwt.Token) (interface{}, error) {
		// making sure that proper signing method is used
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}

		return pk.Public(), nil
	})

	if err != nil {
		return claims, err
	}

	// validating a token on a time basis (i.e. expired, not eligible yet)
	if !t.Valid {
		return claims, ErrInvalidAccessToken
	}

	return claims, nil
}

// UserIDFromToken parses accesspolicy token and returns user ActorID
func (a *Authenticator) UserIDFromToken(tok string) (_ uint32, err error) {
	claims, err := a.claimsFromToken(tok)
	if err != nil {
		return 0, errors.Wrap(err, "failed to obtain user id from accesspolicy token")
	}

	return claims.UserID, nil
}

func (a *Authenticator) UserFromToken(ctx context.Context, tok string) (u user.User, err error) {
	userID, err := a.UserIDFromToken(tok)
	if err != nil {
		return u, err
	}

	u, err = a.users.UserByID(ctx, userID)
	if err != nil {
		return u, errors.Wrap(err, "failed to obtain user from accesspolicy token")
	}

	return u, nil
}

// RevokeAccessToken adds a given token ObjectID to the blacklist for the duration
// of an accesspolicy token expiration time + 1 minute
func (a *Authenticator) RevokeAccessToken(id string, eat time.Time) error {
	return a.backend.PutRevokedAccessToken(RevokedAccessToken{
		TokenID:  id,
		ExpireAt: eat,
	})
}

// IsRevoked checks whether a given token ObjectID is among the blacklisted tokens
// NOTE: blacklisted token IDs are cleansed from the registry shortly
// after their respective accesspolicy tokens expire
func (a *Authenticator) IsRevoked(tokenID string) bool {
	return a.backend.IsRevoked(tokenID)
}

// RegisterSession adds a given session
func (a *Authenticator) RegisterSession(sess Session) (err error) {
	if err = sess.Validate(); err != nil {
		return err
	}

	return a.backend.PutSession(sess)
}

// GetSession returns a session if it's found in the backend
// by its token string
func (a *Authenticator) GetSession(stok string) (Session, error) {
	return a.backend.GetSession(stok)
}

// GetSessionByAccessToken returns a session by accesspolicy token
func (a *Authenticator) GetSessionByAccessToken(jti string) (Session, error) {
	return a.backend.GetSessionByAccessToken(jti)
}

// GetSessionBySessionToken returns a session by accesspolicy token
func (a *Authenticator) GetSessionBySessionToken(rtok string) (Session, error) {
	return a.backend.GetSessionByRefreshToken(rtok)
}
