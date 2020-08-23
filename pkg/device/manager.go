package device

import "github.com/google/uuid"

type AssetKind uint8

const (
	AKClient AssetKind = iota
	AKIdentity
)

type Asset struct {
	ID   uuid.UUID `db:"id" json:"id"`
	Kind AssetKind `db:"kind" json:"kind"`
}

type Relation struct {
	DeviceID uuid.UUID `db:"device_id" json:"device_id"`
	Asset    `db:"asset" json:"asset"`
}

type Manager struct {
	devices   map[uuid.UUID]Device
	relations map[uuid.UUID][]Relation
	store     Store
}
