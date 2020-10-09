package client

import (
	"crypto/rand"
	"net/url"
	"strings"
	"sync"
	"time"

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
	Name         string    `db:"name" json:"name"`
	ID           uuid.UUID `db:"id" json:"id"`
	RegisteredAt time.Time `db:"registered_at" json:"registered_at"`
	ExpireAt     time.Time `db:"expire_at" json:"expire_at"`
	Flags        Flags     `db:"flags" json:"flags"`
	ReturnURLs   []url.URL `db:"urls" json:"urls"`
	entropy      []byte
	sync.RWMutex
	_ struct{}
}

func (c *Client) IsEnabled() bool      { return c.Flags&FEnabled == FEnabled }
func (c *Client) IsConfidential() bool { return c.Flags&FConfidential == FConfidential }
func (c *Client) IsExpired() bool      { return c.ExpireAt.After(time.Now()) }

func New(name string, flags Flags) (c *Client, err error) {
	e := make([]byte, 16)
	if _, err = rand.Read(e); err != nil {
		return nil, errors.Wrap(err, "failed to generate entropy")
	}

	c = &Client{
		Name:         strings.TrimSpace(name),
		ID:           uuid.New(),
		RegisteredAt: time.Now(),
		Flags:        flags,
		ReturnURLs:   make([]url.URL, 0),
		entropy:      e,
	}

	return c, errors.Wrap(c.Validate(), "new client validation failed")
}

func (c *Client) Validate() error {
	if c.ID == uuid.Nil {
		return ErrInvalidClientID
	}

	if c.RegisteredAt.IsZero() {
		return errors.New("registration timestamp is not set")
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

	if c.ReturnURLs == nil {
		c.ReturnURLs = []url.URL{uri}
	} else {
		c.ReturnURLs = append(c.ReturnURLs, uri)
	}

	return nil
}

func (c *Client) HasURL(uri url.URL) bool {
	for _, existing := range c.ReturnURLs {
		if uri == existing {
			return true
		}
	}

	return false
}
