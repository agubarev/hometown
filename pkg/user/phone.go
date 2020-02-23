package user

import (
	"bytes"
	"encoding/binary"

	"github.com/asaskevich/govalidator"
	"github.com/cespare/xxhash"
	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

// NewPhoneObject contains fields sufficient to create a new object
type NewPhoneObject struct {
	PhoneEssential
	IsConfirmed bool
}

// PhoneEssential represents an essential part of the primary object
type PhoneEssential struct {
	Number    TPhoneNumber `db:"number" json:"number"`
	IsPrimary bool         `db:"is_primary" json:"is_primary"`
}

// PhoneMetadata contains generic metadata of the primary object
type PhoneMetadata struct {
	CreatedAt   dbr.NullTime `db:"created_at" json:"created_at"`
	ConfirmedAt dbr.NullTime `db:"confirmed_at" json:"confirmed_at"`
	UpdatedAt   dbr.NullTime `db:"updated_at" json:"updated_at"`

	keyHash uint64
}

// Phone represents certain emails which are custom
// and are handled by the customer
type Phone struct {
	UserID int `db:"user_id" json:"user_id"`

	PhoneEssential
	PhoneMetadata
}

func (p *Phone) hashKey() {
	// panic if ObjectID is zero or a name is empty
	if p.UserID == 0 || p.Number[0] == 0 {
		panic(ErrInsufficientDataToHashKey)
	}

	// initializing a key buffer with and assuming the minimum key length
	key := bytes.NewBuffer(make([]byte, 0, len(p.Number[:])+13))

	// composing a key value
	key.WriteString("phone")
	key.Write(p.Number[:])

	// adding user ObjectID to the key
	if err := binary.Write(key, binary.LittleEndian, int64(p.UserID)); err != nil {
		panic(errors.Wrap(err, "failed to hash phone key"))
	}

	// updating recalculated key
	p.keyHash = xxhash.Sum64(key.Bytes())
}

// Validate validates the object
func (p Phone) Validate() (err error) {
	_, err = govalidator.ValidateStruct(p)
	return nil
}

// Key returns a uint64 key hash to be used as a map/cache key
func (p Phone) Key(rehash bool) uint64 {
	// returning a cached key if it's set
	if p.keyHash == 0 || rehash {
		p.hashKey()
	}

	return p.keyHash
}

// ApplyChangelog applies changes described by a diff.Diff()'s changelog
// NOTE: doing a manual update to avoid using reflection
func (p *Phone) ApplyChangelog(changelog diff.Changelog) (err error) {
	// it's ok if there's nothing apply
	if len(changelog) == 0 {
		return nil
	}

	for _, change := range changelog {
		switch change.Path[0] {
		case "UserID":
			p.UserID = change.To.(int)
		case "Number":
			p.Number = change.To.(TPhoneNumber)
		case "CreatedAt":
			p.CreatedAt = change.To.(dbr.NullTime)
		case "Confirmed_at":
			p.ConfirmedAt = change.To.(dbr.NullTime)
		case "UpdatedAt":
			p.UpdatedAt = change.To.(dbr.NullTime)
		}
	}

	return nil
}
