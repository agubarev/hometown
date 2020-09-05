package client

import (
	"crypto/rand"
	"net/url"
	"strings"

	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const PasswordLength = 32

// Flags represent client flags
type Flags uint8

const (
	FEnabled Flags = 1 << iota
	FConfidential
	FPassword
	FPKI
	FMutualTLS
	FDynamic
)

// Client represents any external client that interfaces with this API
type Client struct {
	Name         string              `db:"name" json:"name"`
	ID           uuid.UUID           `db:"id" json:"id"`
	RegisteredAt timestamp.Timestamp `db:"registered_at" json:"registered_at"`
	ExpireAt     timestamp.Timestamp `db:"expire_at" json:"expire_at"`
	Flags        Flags               `db:"flags" json:"flags"`
	URLs         []url.URL           `db:"urls" json:"urls"`
	entropy      []byte
	_            struct{}
}

func (c *Client) IsEnabled() bool      { return c.Flags&FEnabled == FEnabled }
func (c *Client) IsConfidential() bool { return c.Flags&FConfidential == FConfidential }
func (c *Client) IsExpired() bool      { return c.ExpireAt < timestamp.Now() }

func New(name string, flags Flags) (c *Client, err error) {
	e := make([]byte, 16)
	if _, err = rand.Read(e); err != nil {
		return nil, errors.Wrap(err, "failed to generate entropy")
	}

	c = &Client{
		Name:         strings.TrimSpace(name),
		ID:           uuid.New(),
		RegisteredAt: timestamp.Now(),
		ExpireAt:     0,
		Flags:        flags,
		URLs:         make([]url.URL, 0),
		entropy:      e,
	}

	return c, errors.Wrap(c.Validate(), "new client validation failed")
}

func (c *Client) Validate() error {
	if c.ID == uuid.Nil {
		return ErrInvalidClientID
	}

	if c.RegisteredAt == 0 {
		return errors.New("registration timestamp is empty")
	}

	if len(c.entropy) == 0 {
		return ErrEmptyEntropy
	}

	if c.Name == "" {
		return ErrNoName
	}

	return nil
}

func (c *Client) AddURL(uri url.URL) (err error) {
	if c.HasURL(uri) {
		return ErrDuplicateURL
	}

	if c.URLs == nil {
		c.URLs = []url.URL{uri}
	} else {
		c.URLs = append(c.URLs, uri)
	}

	return nil
}

func (c *Client) HasURL(uri url.URL) bool {
	for _, existing := range c.URLs {
		if uri == existing {
			return true
		}
	}

	return false
}
