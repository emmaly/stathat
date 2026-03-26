package stathat

import (
	"errors"
	"fmt"
)

var (
	// ErrNoEZKey is returned when a posting method is called without an EZ key configured.
	ErrNoEZKey = errors.New("stathat: EZ key not configured")

	// ErrNoUserKey is returned when a classic posting method is called without a user key configured.
	ErrNoUserKey = errors.New("stathat: classic API user key not configured")

	// ErrNoAccessToken is returned when an export/management method is called without an access token.
	ErrNoAccessToken = errors.New("stathat: access token not configured")

	// ErrStatNotFound is returned when a stat lookup by name returns no result.
	ErrStatNotFound = errors.New("stathat: stat not found")

	// ErrAlertNotFound is returned when an alert lookup returns no result.
	ErrAlertNotFound = errors.New("stathat: alert not found")

	// ErrEmptyBatch is returned when PostBatch is called with no reports.
	ErrEmptyBatch = errors.New("stathat: empty batch")
)

// APIError represents a non-success HTTP response from the StatHat API.
type APIError struct {
	StatusCode int
	Message    string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("stathat: API error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("stathat: API error %d", e.StatusCode)
}
