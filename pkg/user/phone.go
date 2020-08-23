package user

import (
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/asaskevich/govalidator"
	"github.com/google/uuid"
	"github.com/r3labs/diff"
)

// NewPhoneObject contains fields sufficient to create a new object
type NewPhoneObject struct {
	PhoneEssential
	UserID      uuid.UUID
	IsConfirmed bool
}

// PhoneEssential represents an essential part of the primary object
type PhoneEssential struct {
	Number    string `db:"number" json:"number"`
	IsPrimary bool   `db:"is_primary" json:"is_primary"`
}

// PhoneMetadata contains generic metadata of the primary object
type PhoneMetadata struct {
	CreatedAt   timestamp.Timestamp `db:"created_at" json:"created_at"`
	ConfirmedAt timestamp.Timestamp `db:"confirmed_at" json:"confirmed_at"`
	UpdatedAt   timestamp.Timestamp `db:"updated_at" json:"updated_at"`

	keyHash uint64
}

// Phone represents certain emails which are custom
// and are handled by the customer
type Phone struct {
	UserID uuid.UUID `db:"user_id" json:"user_id"`

	PhoneEssential
	PhoneMetadata
}

// SanitizeAndValidate validates the object
func (p Phone) Validate() (err error) {
	_, err = govalidator.ValidateStruct(p)
	return nil
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
			p.UserID = change.To.(uuid.UUID)
		case "Number":
			p.Number = change.To.(string)
		case "CreatedAt":
			p.CreatedAt = change.To.(timestamp.Timestamp)
		case "Confirmed_at":
			p.ConfirmedAt = change.To.(timestamp.Timestamp)
		case "UpdatedAt":
			p.UpdatedAt = change.To.(timestamp.Timestamp)
		}
	}

	return nil
}
