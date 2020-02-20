package user

import (
	"bytes"
	"encoding/binary"

	"github.com/asaskevich/govalidator"
	"github.com/cespare/xxhash"
	"github.com/gocraft/dbr/v2"
	"github.com/oklog/ulid"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

// EmailNewObject contains fields sufficient to create a new object
type NewEmailObject struct {
	EmailEssential
}

// EmailEssential represents an essential part of the primary object
type EmailEssential struct {
	ExternalID  int    `db:"external_id" json:"external_id"`
	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
}

// EmailMetadata contains generic metadata of the primary object
type EmailMetadata struct {
	Checksum  uint64       `db:"checksum" json:"checksum"`
	CreatedAt dbr.NullTime `db:"created_at" json:"created_at"`
	UpdatedAt dbr.NullTime `db:"updated_at" json:"updated_at"`
	DeletedAt dbr.NullTime `db:"deleted_at" json:"deleted_at"`

	keyHash uint64
}

// Email represents certain emails which are custom
// and are handled by the customer
type Email struct {
	ID   int       `db:"id" json:"id"`
	ULID ulid.ULID `db:"ulid" json:"ulid"`

	EmailEssential
	EmailMetadata
}

func (email *Email) calculateChecksum() uint64 {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		int64(email.ID),
		int64(email.ExternalID),
		[]byte(email.Name),
	}

	for _, field := range fields {
		if err := binary.Write(buf, binary.LittleEndian, field); err != nil {
			panic(errors.Wrapf(err, "failed to write binary data [%v] to calculate checksum", field))
		}
	}

	// assigning a checksum calculated from a definite list of struct values
	email.Checksum = xxhash.Sum64(buf.Bytes())

	return email.Checksum
}

func (email *Email) hashKey() {
	// panic if ID is zero or a name is empty
	if email.ID == 0 || len(email.Name) == 0 {
		panic(ErrInsufficientDataToHashKey)
	}

	// initializing a key buffer with and assuming the minimum key length
	key := bytes.NewBuffer(make([]byte, 0, len("email")+len(email.Name)+8))

	// composing a key value
	key.WriteString("email")
	key.WriteString(email.Name)

	// adding ID to the key
	if err := binary.Write(key, binary.LittleEndian, int64(email.ID)); err != nil {
		panic(errors.Wrap(err, "failed to hash email key"))
	}

	// updating recalculated key
	email.keyHash = xxhash.Sum64(key.Bytes())
}

// Validate validates the object
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
			email.ExternalID = change.To.(int)
		case "Name":
			email.Name = change.To.(string)
		case "Checksum":
			email.Checksum = change.To.(uint64)
		case "CreatedAt":
			email.CreatedAt = change.To.(dbr.NullTime)
		case "UpdatedAt":
			email.UpdatedAt = change.To.(dbr.NullTime)
		case "DeletedAt":
			email.DeletedAt = change.To.(dbr.NullTime)
		}
	}

	return nil
}
