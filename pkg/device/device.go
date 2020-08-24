package device

import (
	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/google/uuid"
)

// Kind represents a kind of a device (i.e.: desktop/mobile/iot/etc...)
type Kind uint8

const ()

// Flags represent device flags
type Flags uint8

const (
	FEnabled Flags = 1 << iota
	FCompromised
	FLost
)

// Device represents
type Device struct {
	Name         string              `db:"name" json:"name"`
	ID           uuid.UUID           `db:"id" json:"id"`
	IMEI         string              `db:"imei" json:"imei"`
	MEID         string              `db:"meid" json:"meid"`
	SerialNumber string              `db:"esn" json:"esn"`
	RegisteredAt timestamp.Timestamp `db:"registered_at" json:"registered_at"`
	ExpireAt     timestamp.Timestamp `db:"expire_at" json:"expire_at"`
	Flags        Flags               `db:"flags" json:"flags"`
	_            struct{}
}
