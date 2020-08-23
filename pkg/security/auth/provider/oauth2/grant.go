package oauth2

import (
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/google/uuid"
)

type GrantAuthCode struct {
	Type        string    `json:"type"`
	ClientID    uuid.UUID `json:"client_id"`
	RedirectURI string    `json:"redirect_uri"`
	Scope       []string  `json:"scope"`
	State       string    `json:"state"`
}
