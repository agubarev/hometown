package secret

import "github.com/agubarev/hometown/pkg/util/bytearray"

type Secret struct {
	Payload map[bytearray.ByteString64]interface{}
}
