package oauth2

import (
	"github.com/google/uuid"
)

type AuthorizationRequest struct {
	ResponseType        string    `json:"response_type"`
	ClientID            uuid.UUID `json:"client_id"`
	RedirectURI         string    `json:"redirect_uri"`
	Scope               []string  `json:"scope"`
	State               string    `json:"state"`
	CodeChallenge       string    `json:"code_challenge"`
	CodeChallengeMethod string    `json:"code_challenge_method"`
}

type TokenRequest struct {
	GrantType         string    `json:"grant_type"`
	RedirectURI       string    `json:"redirect_uri"`
	ClientID          uuid.UUID `json:"client_id"`
	AuthorizationCode string    `json:"authorization_code"`
	CodeVerifier      string    `json:"code_verifier"`
}
