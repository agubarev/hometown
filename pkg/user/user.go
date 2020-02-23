package user

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/cespare/xxhash"
	"github.com/gocraft/dbr/v2"
	"github.com/oklog/ulid"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

// UserNewObject contains fields sufficient to create a new object
type NewUserObject struct {
	Essential
	ProfileEssential
	EmailAddr   TEmailAddr   `json:"email"`
	PhoneNumber TPhoneNumber `json:"phone"`
	Password    []byte       `json:"password"`
}

// Essential represents an essential part of the primary object
type Essential struct {
	Username    TUsername    `db:"username" json:"username"`
	DisplayName TDisplayName `db:"display_name" json:"display_name"`
}

type Metadata struct {
	Checksum uint64 `db:"checksum" json:"checksum"`

	// timestamps
	CreatedAt   dbr.NullTime `db:"created_at" json:"created_at"`
	CreatedByID int          `db:"created_by_id" json:"created_by_id"`
	UpdatedAt   dbr.NullTime `db:"updated_at" json:"updated_at"`
	UpdatedByID int          `db:"updated_by_id" json:"updated_by_id"`
	ConfirmedAt dbr.NullTime `db:"confirmed_at" json:"confirmed_at"`
	DeletedAt   dbr.NullTime `db:"deleted_at" json:"deleted_at"`
	DeletedByID int          `db:"deleted_by_id" json:"deleted_by_id"`

	// the most recent authentication information
	LastLoginAt       dbr.NullTime `db:"last_login_at" json:"last_login_at"`
	LastLoginIP       TIPAddr      `db:"last_login_ip" json:"last_login_ip"`
	LastLoginFailedAt dbr.NullTime `db:"last_login_failed_at" json:"last_login_failed_at"`
	LastLoginFailedIP TIPAddr      `db:"last_login_failed_ip" json:"last_login_failed_ip"`
	LastLoginAttempts uint8        `db:"last_login_attempts" json:"last_login_attempts"`

	// account suspension
	IsSuspended         bool         `db:"is_suspended" json:"is_suspended"`
	SuspendedAt         dbr.NullTime `db:"suspended_at" json:"suspended_at"`
	SuspensionExpiresAt dbr.NullTime `db:"suspension_expires_at" json:"suspension_expires_at"`
	SuspensionReason    [256]byte    `db:"suspension_reason" json:"suspension_reason"`
}

// User represents certain users which are custom
// and are handled by the customer
type User struct {
	ID   int       `db:"id" json:"id"`
	ULID ulid.ULID `db:"ulid" json:"ulid"`

	Essential
	Metadata

	keyHash uint64
}

func (u *User) calculateChecksum() uint64 {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		u.Username,
		u.DisplayName,
		u.IsSuspended,
		u.SuspendedAt,
		u.SuspensionExpiresAt,
		u.SuspensionReason,
	}

	for _, field := range fields {
		if err := binary.Write(buf, binary.LittleEndian, field); err != nil {
			panic(errors.Wrapf(err, "failed to write binary data [%v] to calculate checksum", field))
		}
	}

	// assigning a checksum calculated from a definite list of struct values
	u.Checksum = xxhash.Sum64(buf.Bytes())

	return u.Checksum
}

func (u *User) hashKey() {
	// panic if GroupMemberID is zero or a name is empty
	if u.ID == 0 || u.Username[0] == 0 {
		panic(ErrInsufficientDataToHashKey)
	}

	// initializing a key buffer with and assuming the minimum key length
	key := bytes.NewBuffer(make([]byte, 0, 4+len(u.Username)+8))

	// composing a key value
	key.WriteString("user")
	key.Write(u.Username[:])

	// adding GroupMemberID to the key
	if err := binary.Write(key, binary.LittleEndian, int64(u.ID)); err != nil {
		panic(errors.Wrap(err, "failed to hash user key"))
	}

	// updating recalculated key
	u.keyHash = xxhash.Sum64(key.Bytes())
}

// Key returns a uint64 key hash to be used as a map/cache key
func (u User) Key(rehash bool) uint64 {
	// returning a cached key if it's set
	if u.keyHash == 0 || rehash {
		u.hashKey()
	}

	return u.keyHash
}

// ApplyChangelog applies changes described by a diff.Diff()'s changelog
// NOTE: doing a manual update to avoid using reflection
func (u *User) ApplyChangelog(changelog diff.Changelog) (err error) {
	// it's ok if there's nothing apply
	if len(changelog) == 0 {
		return nil
	}

	for _, change := range changelog {
		switch change.Path[0] {
		case "Username":
			u.Username = change.To.(TUsername)
		case "DisplayName":
			u.DisplayName = change.To.(TDisplayName)
		case "Checksum":
			u.Checksum = change.To.(uint64)
		case "CreatedAt":
			u.CreatedAt = change.To.(dbr.NullTime)
		case "UpdatedAt":
			u.UpdatedAt = change.To.(dbr.NullTime)
		case "DeletedAt":
			u.DeletedAt = change.To.(dbr.NullTime)
		}
	}

	return nil
}

// StringID returns short info about the user
func (u *User) StringID() string {
	return fmt.Sprintf("user(%d:%s)", u.ID, u.Username)
}

// Validate user object
func (u *User) Validate() error {
	if u == nil {
		return ErrNilUser
	}

	if ok, err := govalidator.ValidateStruct(u); !ok {
		return errors.Wrapf(err, "user validation failed")
	}

	return nil
}
