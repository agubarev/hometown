package usermanager

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/agubarev/hometown/util"
	"github.com/asaskevich/govalidator"
	"github.com/go-sql-driver/mysql"
	"github.com/oklog/ulid"
)

// User represents a user account, a unique entity
// TODO: workout the length restrictions
type User struct {
	ID   int64     `json:"id" db:"id"`
	ULID ulid.ULID `json:"uid" db:"uid"`

	// Username and Email are the primary IDs associated with the user account
	Username string `json:"username" valid:"required,alphanum" db:"username"`
	Email    string `json:"email" valid:"required,email" db:"email"`

	// project-specific fields
	UserReference             string        `json:"user_reference" db:"user_ref"`
	IsPasswordChangeRequested bool          `json:"is_pass_change_req" db:"is_pass_change_req"`
	ReadingUnitID             int64         `json:"reading_unit_id" db:"ru_id"`
	ReadingUnit               interface{}   `json:"reading_unit" db:"-"`
	ReadingCenters            []interface{} `json:"reading_centers" db:"-"`
	Devices                   []interface{} `json:"devices" db:"-"`

	// the name, birthdate, country
	Firstname  string `json:"firstname" valid:"optional,utfletter" db:"firstname"`
	Lastname   string `json:"lastname" valid:"optional,utfletter" db:"lastname"`
	Middlename string `json:"middlename,omitempty" valid:"optional,utfletter" db:"middlename"`
	Phone      string `json:"phone" valid:"optional,dialstring" db:"phone"`
	Language   string `json:"language" valid:"optional,utfletter" db:"language"`

	// account confirmation
	EmailConfirmedAt mysql.NullTime `json:"email_confirmed_at,omitempty" db:"-"`
	PhoneConfirmedAt mysql.NullTime `json:"phone_confirmed_at,omitempty" db:"-"`

	// timestamps
	CreatedAt   time.Time      `json:"t_cr" db:"created_at"`
	UpdatedAt   mysql.NullTime `json:"t_up,omitempty" db:"updated_at"`
	ConfirmedAt mysql.NullTime `json:"t_co,omitempty" db:"confirmed_at"`

	// metadata
	CreatedByID int64 `json:"created_by_id" db:"created_by_id"`
	UpdatedByID int64 `json:"updated_by_id" db:"updated_by_id"`

	// the most recent authentication information
	LastLoginAt       mysql.NullTime `json:"last_login_at,omitempty" db:"last_login_at"`
	LastLoginIP       string         `json:"last_login_ip,omitempty" db:"last_login_ip"`
	LastLoginFailedAt mysql.NullTime `json:"last_login_failed_at,omitempty" db:"last_login_failed_at"`
	LastLoginFailedIP string         `json:"last_login_failed_ip,omitempty" db:"last_login_failed_ip"`
	LastLoginAttempts uint8          `json:"last_login_attempts,omitempty" db:"last_login_attempts"`

	// account suspension
	IsSuspended         bool           `json:"is_suspended,omitempty" db:"is_suspended"`
	SuspendedAt         mysql.NullTime `json:"suspended_at,omitempty" db:"suspended_at"`
	SuspensionExpiresAt mysql.NullTime `json:"suspension_expires_at,omitempty" db:"suspension_expires_at"`
	SuspensionReason    string         `json:"suspension_reason,omitempty" db:"suspension_reason"`

	// corresponding container for easier backtracking
	container *UserContainer

	// tracking all group kinds in one slice
	groups []*Group
}

// StringInfo returns short info about the user
func (u *User) StringInfo() string {
	return fmt.Sprintf("user(%d:%s)", u.ID, u.Username)
}

// Container returns the corresponding user container to
// which this user belongs
func (u *User) Container() (*UserContainer, error) {
	if u.container == nil {
		return nil, ErrNilUserContainer
	}

	return u.container, nil
}

// Fullname returns full name of a user
func (u User) Fullname(withMiddlename bool) string {
	if withMiddlename {
		return fmt.Sprintf("%s %s %s", u.Firstname, u.Middlename, u.Lastname)
	}

	return fmt.Sprintf("%s %s", u.Firstname, u.Lastname)
}

// NewUser initializing a new User
// TODO: consider changing userinfo value type to interface{} to allow variable data
func NewUser(username string, email string, userinfo map[string]string) (*User, error) {
	u := &User{
		ID:        0,
		ULID:      util.NewULID(),
		Username:  username,
		Email:     email,
		CreatedAt: time.Now(),

		groups: make([]*Group, 0),
	}

	// processing given userinfo
	for k, v := range userinfo {
		k = strings.ToLower(k)
		v = strings.TrimSpace(v)

		// whitelist
		switch k {
		case "firstname":
			u.Firstname = v
		case "lastname":
			u.Lastname = v
		case "middlename":
			u.Middlename = v
		case "language":
			u.Language = v
		default:
			return nil, fmt.Errorf("unrecognized user info field: %s", k)
		}
	}

	// setting defaults if they're not specified
	if u.Language == "" {
		u.Language = "en"
	}

	// final pre-initialization validation
	if err := u.Validate(); err != nil {
		return nil, err
	}

	return u, nil
}

// Validate user object
func (u *User) Validate() error {
	if u == nil {
		return ErrNilUser
	}

	ok, err := govalidator.ValidateStruct(u)
	if err != nil {
		return fmt.Errorf("%s validation failed: %s", u.StringInfo(), err)
	}

	if !ok {
		// haven't figured why govalidator sometimes returns
		// false and no error
	}

	return nil
}

// HasPassword tests whether this user has password
func (u *User) HasPassword() bool {
	userContainer, err := u.Container()
	if err != nil {
		return false
	}

	userManager, err := userContainer.Manager()
	if err != nil {
		return false
	}

	passwordManager, err := userManager.PasswordManager()
	if err != nil {
		return false
	}

	if _, err = passwordManager.Get(u); err != nil {
		return false
	}

	return true
}

// Save saves current user to the container's store
func (u *User) Save() error {
	if u == nil {
		return ErrNilUser
	}

	// first, obtaining the container in which this user resides
	c, err := u.Container()
	if err != nil {
		return err
	}

	// second, obtaining the user manager to which this container belongs
	m, err := c.Manager()
	if err != nil {
		return err
	}

	// saving through the user manager
	err = m.Save(u)
	if err != nil {
		return err
	}

	return nil
}

// LinkGroup tracking which groups this user is a member of
func (u *User) LinkGroup(g *Group) error {
	if g == nil {
		return ErrNilGroup
	}

	// safeguard in case this slice is not initialized
	if u.groups == nil {
		u.groups = make([]*Group, 0)
	}

	// appending group to slice for easier runtime access
	u.groups = append(u.groups, g)

	return nil
}

// UnlinkGroup removing group from the tracklist
func (u *User) UnlinkGroup(g *Group) error {
	if u.groups == nil {
		// initializing just in case
		u.groups = make([]*Group, 0)

		return nil
	}

	// removing group from the tracklist
	for i, ug := range u.groups {
		if ug.ID == g.ID {
			u.groups = append(u.groups[0:i], u.groups[i+1:]...)
			break
		}
	}

	return ErrGroupNotFound
}

// Groups to which the user belongs
func (u *User) Groups(mask GroupKind) []*Group {
	if u.groups == nil {
		u.groups = make([]*Group, 0)
	}

	groups := make([]*Group, 0)
	for _, g := range u.groups {
		if (g.Kind | mask) == mask {
			groups = append(groups, g)
		}
	}

	return groups
}

// LinkContainer links user to container for easier backtracking
func (u *User) LinkContainer(c *UserContainer) error {
	u.container = c

	return nil
}

// IsRegisteredAndStored returns true if the user is both:
// 1. registered within a user container
// 2. persisted to the store
func (u *User) IsRegisteredAndStored() (bool, error) {
	c, err := u.Container()
	if err != nil {
		// for convenience, the user is allowed to not be inside a container
		if err == ErrNilUserContainer {
			log.Printf("IsRegisteredAndStored(%d:%s): %s", u.ID, u.Username, err)
			return false, nil
		}

		return false, fmt.Errorf("IsRegisteredAndStored(): failed to obtain user container: %s", err)
	}

	// obtaining the user manager
	m, err := c.Manager()
	if err != nil {
		return false, err
	}

	return m.IsRegisteredAndStored(u)
}
