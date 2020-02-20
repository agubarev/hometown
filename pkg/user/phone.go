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

// errors
var (
	ErrNilPhone       = errors.New("phone is nil")
	ErrDuplicatePhone = errors.New("duplicate phone")
	ErrPhoneNotFound  = errors.New("phone not found")
)

// PhoneNewObject contains fields sufficient to create a new object
type NewPhoneObject struct {
	PhoneEssential
}

// PhoneEssential represents an essential part of the primary object
type PhoneEssential struct {
	ExternalID  int    `db:"external_id" json:"external_id"`
	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
}

// PhoneMetadata contains generic metadata of the primary object
type PhoneMetadata struct {
	Checksum  uint64       `db:"checksum" json:"checksum"`
	CreatedAt dbr.NullTime `db:"created_at" json:"created_at"`
	UpdatedAt dbr.NullTime `db:"updated_at" json:"updated_at"`
	DeletedAt dbr.NullTime `db:"deleted_at" json:"deleted_at"`

	keyHash uint64
}

// Phone represents certain phones which are custom
// and are handled by the customer
type Phone struct {
	ID   int       `db:"id" json:"id"`
	ULID ulid.ULID `db:"ulid" json:"ulid"`

	PhoneEssential
	PhoneMetadata
}

func (p *Phone) calculateChecksum() uint64 {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		int64(p.ID),
		int64(p.ExternalID),
		[]byte(p.Name),
	}

	for _, field := range fields {
		if err := binary.Write(buf, binary.LittleEndian, field); err != nil {
			panic(errors.Wrapf(err, "failed to write binary data [%v] to calculate checksum", field))
		}
	}

	// assigning a checksum calculated from a definite list of struct values
	p.Checksum = xxhash.Sum64(buf.Bytes())

	return p.Checksum
}

func (p *Phone) hashKey() {
	// panic if ID is zero or a name is empty
	if p.ID == 0 || len(p.Name) == 0 {
		panic(ErrInsufficientDataToHashKey)
	}

	// initializing a key buffer with and assuming the minimum key length
	key := bytes.NewBuffer(make([]byte, 0, len("phone")+len(p.Name)+8))

	// composing a key value
	key.WriteString("phone")
	key.WriteString(p.Name)

	// adding ID to the key
	if err := binary.Write(key, binary.LittleEndian, int64(p.ID)); err != nil {
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
			p.ExternalID = change.To.(int)
		case "Name":
			p.Name = change.To.(string)
		case "Checksum":
			p.Checksum = change.To.(uint64)
		case "CreatedAt":
			p.CreatedAt = change.To.(dbr.NullTime)
		case "UpdatedAt":
			p.UpdatedAt = change.To.(dbr.NullTime)
		case "DeletedAt":
			p.DeletedAt = change.To.(dbr.NullTime)
		}
	}

	return nil
}
