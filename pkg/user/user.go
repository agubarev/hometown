package user

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/cespare/xxhash"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

// UserNewObject contains fields sufficient to create a new object
type NewUserObject struct {
	Essential
	ProfileEssential
	EmailAddr   string `json:"email_addr"`
	PhoneNumber string `json:"phone_number"`
	Password    []byte `json:"password"`
}

// Essential represents an essential part of the primary object
type Essential struct {
	Username    string `db:"username" json:"username"`
	DisplayName string `db:"display_name" json:"display_name"`
}

// TODO: solve net.IPAddr field problem
type Metadata struct {
	Checksum uint64 `db:"checksum" json:"checksum"`

	// timestamps
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	ConfirmedAt time.Time `db:"confirmed_at" json:"confirmed_at"`
	DeletedAt   time.Time `db:"deleted_at" json:"deleted_at"`

	// the most recent authentication information
	LastLoginAt       time.Time `db:"last_login_at" json:"last_login_at"`
	LastLoginIP       net.IP    `db:"last_login_ip" json:"last_login_ip"`
	LastLoginFailedAt time.Time `db:"last_login_failed_at" json:"last_login_failed_at"`
	LastLoginFailedIP net.IP    `db:"last_login_failed_ip" json:"last_login_failed_ip"`
	LastLoginAttempts uint8     `db:"last_login_attempts" json:"last_login_attempts"`

	// account suspension
	IsSuspended         bool      `db:"is_suspended" json:"is_suspended"`
	SuspendedAt         time.Time `db:"suspended_at" json:"suspended_at"`
	SuspensionExpiresAt time.Time `db:"suspension_expires_at" json:"suspension_expires_at"`
	SuspensionReason    string    `db:"suspension_reason" json:"suspension_reason"`
}

// User represents certain users which are custom
// and are handled by the customer
type User struct {
	ID uuid.UUID `db:"id" json:"id"`

	Essential
	Metadata

	keyHash uint64
}

func (u *User) calculateChecksum() uint64 {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		[]byte(u.Username),
		[]byte(u.DisplayName),
		u.IsSuspended,
		u.SuspendedAt.UnixNano(),
		u.SuspensionExpiresAt.UnixNano(),
		[]byte(u.SuspensionReason),
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
			u.Username = change.To.(string)
		case "DisplayName":
			u.DisplayName = change.To.(string)
		case "LastLoginAt":
			u.LastLoginAt = change.To.(time.Time)
		case "LastLoginIP":
			u.LastLoginIP = change.To.(net.IP)
		case "Checksum":
			u.Checksum = change.To.(uint64)
		case "CreatedAt":
			u.CreatedAt = change.To.(time.Time)
		case "UpdatedAt":
			u.UpdatedAt = change.To.(time.Time)
		case "DeletedAt":
			u.DeletedAt = change.To.(time.Time)
		}
	}

	return nil
}

// SanitizeAndValidate user object
func (u User) Validate() error {
	if ok, err := govalidator.ValidateStruct(u); !ok {
		return errors.Wrapf(err, "user validation failed")
	}

	return nil
}
