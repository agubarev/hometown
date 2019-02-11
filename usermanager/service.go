package usermanager

/*
// Service represents a User manager contract interface
type Service interface {
	CreateUser(ctx context.Context, d *Domain, data *NewUserData) (*User, error)
	DeleteUser(ctx context.Context, d *Domain, id ulid.ULID) error
	SetUsername(ctx context.Context, d *Domain, id ulid.ULID, username string) error
	GetUser(ctx context.Context, d *Domain, id ulid.ULID) (*User, error)
	GetUserByUsername(ctx context.Context, d *Domain, username string) (*User, error)
	GetUserByEmail(ctx context.Context, d *Domain, email string) (*User, error)
}

// NewUserData holds data necessary to create a new user
type NewUserData struct {
	// origin an object that contains identity provider
	// info along with user-specific data
	// TODO: change origin type
	Origin   string `json:"origin"`
	Username string `json:"username" valid:""`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// NewUserService is a default user manager implementation
func NewUserService(m *UserManager) (Service, error) {
	if m == nil {
		return nil, ErrNilUserManager
	}

	return &service{m}, nil
}

type service struct {
	m *UserManager
}

// CreateUser new user
func (s *service) CreateUser(ctx context.Context, d *Domain, data *NewUserData) (*User, error) {
	if d == nil {
		return nil, ErrNilDomain
	}

	if data == nil {
		return nil, fmt.Errorf("new user request is nil")
	}

	err := d.Users.Create(data.Username, data.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to create new user: %s", err)
	}

	// TODO: send verification email

	return newUser, nil
}

// Get existing user
func (s *service) GetUser(ctx context.Context, d *Domain, userID ulid.ULID) (*User, error) {
	return d.Users.Get(userID)
}

// GetUserByUsername returns a user by username
func (s *service) GetUserByUsername(ctx context.Context, d *Domain, username string) (*User, error) {
	return d.Users.GetByUsername(username)
}

// GetUserByEmail returns a user by email
func (s *service) GetUserByEmail(ctx context.Context, d *Domain, email string) (*User, error) {
	return d.Users.GetByEmail(email)
}

// SetUsername update username for an existing user
// TODO: username validation
func (s *service) SetUsername(ctx context.Context, d *Domain, userID ulid.ULID, newUsername string) error {
	u, err := d.Users.Get(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// lookup existing user by that username
	eu, err := d.Users.GetByUsername(newUsername)
	if eu != nil {
		return ErrUsernameTaken
	}

	// setting new username and updating
	u.Username = newUsername

	if err = u.Save(); err != nil {
		return err
	}

	return nil
}

// DeleteUser unregisters user
func (s *service) DeleteUser(ctx context.Context, d *Domain, userID ulid.ULID) error {
	return d.Users.DeleteUser(userID)
}
*/
