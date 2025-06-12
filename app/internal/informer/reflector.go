package informer

import (
	"errors"
	"io"
)

var errReflectorClosed = errors.New("reflector closed")

// Reflector will reflect the keys.
type Reflector interface {
	// Watch will watch the keys.
	// No need for thead-safe.
	// If the reflector is closed, it will return an errReflectorClosed error.
	Watch() ([]string, error)
	// Get will get the value of the key.
	// Need for thead-safe.
	Get(key string) (interface{}, bool)
	io.Closer
}

// NewReflectorClosedError returns an errReflectorClosed error.
func NewReflectorClosedError() error {
	return errReflectorClosed
}

// IsReflectorClosedError checks if the error is an errReflectorClosed error.
func IsReflectorClosedError(err error) bool {
	return errors.Is(err, errReflectorClosed)
}
