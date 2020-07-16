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

// EmailNewObject contains fields sufficient to create a new object
type NewEmailObject struct {
	EmailEssential
	UserID      uint32
	IsConfirmed bool
}

// EmailEssential represents an essential part of the primary object
type EmailEssential struct {
	Addr      string `db:"addr" json:"addr"`
	IsPrimary bool   `db:"is_primary" json:"is_primary"`
}

// EmailMetadata contains generic metadata of the primary object
type EmailMetadata struct {
	CreatedAt   dbr.NullTime `db:"created_at" json:"created_at"`
	ConfirmedAt dbr.NullTime `db:"confirmed_at" json:"confirmed_at"`
	UpdatedAt   dbr.NullTime `db:"updated_at" json:"updated_at"`

	keyHash uint64
}

// Email represents certain emails which are custom
// and are handled by the customer
type Email struct {
	UserID uint32 `db:"user_id" json:"user_id"`

	EmailEssential
	EmailMetadata
}

func (email *Email) hashKey() {
	// panic if ObjectID is zero or a name is empty
	if email.UserID == 0 || email.Addr == "" {
		panic(ErrInsufficientDataToHashKey)
	}

	// initializing a key buffer with and assuming the minimum key length
	key := bytes.NewBuffer(make([]byte, 0, len("email")+len(email.Addr)+8))

	// composing a key value
	key.WriteString("email")
	key.WriteString(email.Addr)

	// adding user ObjectID to the key
	if err := binary.Write(key, binary.LittleEndian, email.UserID); err != nil {
		panic(errors.Wrap(err, "failed to hash email key"))
	}

	// updating recalculated key
	email.keyHash = xxhash.Sum64(key.Bytes())
}

// SanitizeAndValidate validates the object
func (email Email) Validate() (err error) {
	_, err = govalidator.ValidateStruct(email)
	return nil
}

// Key returns a uint64 key hash to be used as a map/cache key
func (email Email) Key(rehash bool) uint64 {
	// returning a cached key if it's set
	if email.keyHash == 0 || rehash {
		email.hashKey()
	}

	return email.keyHash
}

// ApplyChangelog applies changes described by a diff.Diff()'s changelog
// NOTE: doing a manual update to avoid using reflection
func (email *Email) ApplyChangelog(changelog diff.Changelog) (err error) {
	// it's ok if there's nothing apply
	if len(changelog) == 0 {
		return nil
	}

	for _, change := range changelog {
		switch change.Path[0] {
		case "UserID":
			email.UserID = change.To.(uint32)
		case "Addr":
			email.Addr = change.To.(string)
		case "CreatedAt":
			email.CreatedAt = change.To.(dbr.NullTime)
		case "ConfirmedAt":
			email.ConfirmedAt = change.To.(dbr.NullTime)
		case "UpdatedAt":
			email.UpdatedAt = change.To.(dbr.NullTime)
		}
	}

	return nil
}
