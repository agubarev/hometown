package user

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/agubarev/hometown/pkg/util"
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/asaskevich/govalidator"
	"github.com/cespare/xxhash"
	"github.com/google/uuid"
	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

// IPAddr is a 16 byte is an amortized size to accommodate both IPv4 and IPv6
type IPAddr [16]byte

func (addr IPAddr) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) (newBuf []byte, err error) {
	if addr[0] == 0 {
		return nil, nil
	}

	zpos := bytes.IndexByte(addr[:], byte(0))
	if zpos == -1 {
		return append(buf, addr[:]...), nil
	}

	return append(buf, addr[0:zpos]...), nil
}

func (addr *IPAddr) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	copy(addr[:], src)
	return nil
}

func (addr IPAddr) StringIPv4() string {
	return fmt.Sprintf(
		"%x.%x.%x.%x",
		addr[0],
		addr[1],
		addr[2],
		addr[3],
	)
}

// UserNewObject contains fields sufficient to create a new object
type NewUserObject struct {
	Essential
	ProfileEssential
	EmailAddr   bytearray.ByteString256 `json:"email_addr"`
	PhoneNumber bytearray.ByteString16  `json:"phone_number"`
	Password    []byte                  `json:"password"`
}

// Essential represents an essential part of the primary object
type Essential struct {
	Username    bytearray.ByteString32 `db:"username" json:"username"`
	DisplayName bytearray.ByteString32 `db:"display_name" json:"display_name"`
}

// TODO: solve net.IPAddr field problem
type Metadata struct {
	Checksum uint64 `db:"checksum" json:"checksum"`

	// timestamps
	CreatedAt   util.Timestamp `db:"created_at" json:"created_at"`
	CreatedByID uuid.UUID      `db:"created_by_id" json:"created_by_id"`
	UpdatedAt   util.Timestamp `db:"updated_at" json:"updated_at"`
	UpdatedByID uuid.UUID      `db:"updated_by_id" json:"updated_by_id"`
	ConfirmedAt util.Timestamp `db:"confirmed_at" json:"confirmed_at"`
	DeletedAt   util.Timestamp `db:"deleted_at" json:"deleted_at"`
	DeletedByID uuid.UUID      `db:"deleted_by_id" json:"deleted_by_id"`

	// the most recent authentication information
	LastLoginAt       util.Timestamp `db:"last_login_at" json:"last_login_at"`
	LastLoginIP       IPAddr         `db:"last_login_ip" json:"last_login_ip"`
	LastLoginFailedAt util.Timestamp `db:"last_login_failed_at" json:"last_login_failed_at"`
	LastLoginFailedIP IPAddr         `db:"last_login_failed_ip" json:"last_login_failed_ip"`
	LastLoginAttempts uint8          `db:"last_login_attempts" json:"last_login_attempts"`

	// account suspension
	IsSuspended         bool                    `db:"is_suspended" json:"is_suspended"`
	SuspendedAt         util.Timestamp          `db:"suspended_at" json:"suspended_at"`
	SuspendedByID       uuid.UUID               `db:"suspended_by_id" json:"suspended_by_id"`
	SuspensionExpiresAt util.Timestamp          `db:"suspension_expires_at" json:"suspension_expires_at"`
	SuspensionReason    bytearray.ByteString128 `db:"suspension_reason" json:"suspension_reason"`
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
			u.Username = change.To.(bytearray.ByteString32)
		case "DisplayName":
			u.DisplayName = change.To.(bytearray.ByteString32)
		case "LastLoginAt":
			u.LastLoginAt = change.To.(util.Timestamp)
		case "LastLoginIP":
			u.LastLoginIP = change.To.(IPAddr)
		case "Checksum":
			u.Checksum = change.To.(uint64)
		case "CreatedAt":
			u.CreatedAt = change.To.(util.Timestamp)
		case "UpdatedAt":
			u.UpdatedAt = change.To.(util.Timestamp)
		case "DeletedAt":
			u.DeletedAt = change.To.(util.Timestamp)
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
