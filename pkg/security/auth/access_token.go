package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

func NewAccessToken(ctx context.Context, privateKey *rsa.PrivateKey, jti uuid.UUID, realm string, ident Identity, ttl timestamp.Timestamp) (signedToken string, err error) {
	if ident.ID == uuid.Nil {
		return "", ErrInvalidIdentityID
	}

	if privateKey == nil {
		return "", ErrNilPrivateKey
	}

	if err = privateKey.Validate(); err != nil {
		return "", errors.Wrap(err, "private key validation failed")
	}

	// generating and signing a new token
	atok := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{
		Identity: ident,
		StandardClaims: jwt.StandardClaims{
			Issuer:    realm,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: int64(timestamp.Now()+ttl) / 1e9,
			Id:        jti.String(),
		},
	})

	// signing access token
	signedToken, err = atok.SignedString(privateKey)
	if err != nil {
		return signedToken, fmt.Errorf("failed to obtain a signed token string: %s", err)
	}

	return signedToken, nil
}