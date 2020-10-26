package auth

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

func NewAccessToken(
	privateKey *rsa.PrivateKey,
	jti uuid.UUID,
	ident Identity,
	expireAt time.Time,
) (signedToken string, err error) {
	// validating identity
	if err = ident.Validate(); err != nil {
		return "", errors.Wrap(err, "invalid identity")
	}

	if privateKey == nil {
		return "", ErrNilPrivateKey
	}

	if err = privateKey.Validate(); err != nil {
		return "", errors.Wrap(err, "private key validation failed")
	}

	// generating and signing a new access token
	atok := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{
		Identity: ident,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: expireAt.Unix(),
			Id:        jti.String(),
		},
	})

	// signing access token
	signedToken, err = atok.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to obtain a signed token string: %s", err)
	}

	return signedToken, nil
}
