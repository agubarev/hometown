package oauth2

import (
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/google/uuid"
)

type GrantAuthCode struct {
	Type        bytearray.ByteString16   `json:"type"`
	ClientID    uuid.UUID                `json:"client_id"`
	RedirectURI bytearray.ByteString256  `json:"redirect_uri"`
	Scope       []bytearray.ByteString16 `json:"scope"`
	State       bytearray.ByteString64   `json:"state"`
}
