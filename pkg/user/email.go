package user

import (
	"github.com/agubarev/hometown/pkg/util"
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/asaskevich/govalidator"
	"github.com/google/uuid"
	"github.com/r3labs/diff"
)

// EmailNewObject contains fields sufficient to create a new object
type NewEmailObject struct {
	EmailEssential
	UserID      uuid.UUID
	IsConfirmed bool
}

// EmailEssential represents an essential part of the primary object
type EmailEssential struct {
	Addr      bytearray.ByteString256 `db:"addr" json:"addr"`
	IsPrimary bool                    `db:"is_primary" json:"is_primary"`
}

// EmailMetadata contains generic metadata of the primary object
type EmailMetadata struct {
	CreatedAt   util.Timestamp `db:"created_at" json:"created_at"`
	ConfirmedAt util.Timestamp `db:"confirmed_at" json:"confirmed_at"`
	UpdatedAt   util.Timestamp `db:"updated_at" json:"updated_at"`

	keyHash uint64
}

// Email represents certain emails which are custom
// and are handled by the customer
type Email struct {
	UserID uuid.UUID `db:"user_id" json:"user_id"`

	EmailEssential
	EmailMetadata
}

// SanitizeAndValidate validates the object
func (email Email) Validate() (err error) {
	_, err = govalidator.ValidateStruct(email)
	return nil
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
			email.UserID = change.To.(uuid.UUID)
		case "Addr":
			email.Addr = change.To.(bytearray.ByteString256)
		case "CreatedAt":
			email.CreatedAt = change.To.(util.Timestamp)
		case "ConfirmedAt":
			email.ConfirmedAt = change.To.(util.Timestamp)
		case "UpdatedAt":
			email.UpdatedAt = change.To.(util.Timestamp)
		}
	}

	return nil
}
