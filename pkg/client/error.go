package client

import "github.com/pkg/errors"

var (
	ErrInvalidClientID       = errors.New("client id is invalid")
	ErrInvalidDeviceID       = errors.New("device id is invalid")
	ErrClientNotFound        = errors.New("client is not found")
	ErrDeviceNotFound        = errors.New("device is not found")
	ErrRelationNotFound      = errors.New("relation is not found")
	ErrRelationAlreadyExists = errors.New("relation already exists")
	ErrNilDatabase           = errors.New("database is nil")
	ErrNilStore              = errors.New("store is nil")
	ErrNoName                = errors.New("name cannot be empty")
	ErrEmptyEntropy          = errors.New("entropy is empty")
	ErrNilPasswordManager    = errors.New("password is nil")
)
