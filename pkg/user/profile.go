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

// NewProfileObject contains fields sufficient to create a new object
type NewProfileObject struct {
	ProfileEssential
	UserID uint32 `db:"user_id" json:"user_id"`
}

// ProfileEssential represents an essential part of the primary object
type ProfileEssential struct {
	Firstname  string `db:"firstname" json:"firstname"`
	Lastname   string `db:"lastname" json:"lastname"`
	Middlename string `db:"middlename" json:"middlename"`
	Language   string `db:"language" json:"language"`
}

// ProfileMetadata contains generic metadata of the primary object
type ProfileMetadata struct {
	Checksum  uint64       `db:"checksum" json:"checksum"`
	CreatedAt dbr.NullTime `db:"created_at" json:"created_at"`
	UpdatedAt dbr.NullTime `db:"updated_at" json:"updated_at"`

	keyHash uint64
}

// Profile represents certain profiles which are custom
// and are handled by the customer
type Profile struct {
	UserID uint32 `db:"user_id" json:"-"`

	ProfileEssential
	ProfileMetadata
}

func (p *Profile) calculateChecksum() uint64 {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		[]byte(p.Firstname),
		[]byte(p.Lastname),
		[]byte(p.Middlename),
		[]byte(p.Language),
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

func (p *Profile) hashKey() {
	// panic if ObjectID is zero or a name is empty
	if p.UserID == 0 {
		panic(ErrInsufficientDataToHashKey)
	}

	// initializing a key buffer with and assuming the minimum key length
	key := bytes.NewBuffer(make([]byte, 0, len("profile")+8))

	// composing a key value
	key.WriteString("profile")

	// adding ObjectID to the key
	if err := binary.Write(key, binary.LittleEndian, p.UserID); err != nil {
		panic(errors.Wrap(err, "failed to hash profile key"))
	}

	// updating recalculated key
	p.keyHash = xxhash.Sum64(key.Bytes())
}

// SanitizeAndValidate validates the object
func (p Profile) Validate() (err error) {
	_, err = govalidator.ValidateStruct(p)
	return nil
}

// Key returns a uint64 key hash to be used as a map/cache key
func (p Profile) Key(rehash bool) uint64 {
	// returning a cached key if it's set
	if p.keyHash == 0 || rehash {
		p.hashKey()
	}

	return p.keyHash
}

// ApplyChangelog applies changes described by a diff.Diff()'s changelog
// NOTE: doing a manual update to avoid using reflection
func (p *Profile) ApplyChangelog(changelog diff.Changelog) (err error) {
	// it's ok if there's nothing apply
	if len(changelog) == 0 {
		return nil
	}

	for _, change := range changelog {
		switch change.Path[0] {
		case "Firstname":
			p.Firstname = change.To.(string)
		case "Middlename":
			p.Middlename = change.To.(string)
		case "Lastname":
			p.Lastname = change.To.(string)
		case "Language":
			p.Language = change.To.(string)
		case "Checksum":
			p.Checksum = change.To.(uint64)
		}
	}

	return nil
}
