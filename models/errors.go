package models

import (
	"encoding/json"
	"net/http"
)

// StatusError carries an HTTP status alongside a message and serializes to JSON,
// so handlers can recover the status with ParseStatusError.
type StatusError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (err StatusError) Error() string {
	b, _ := json.Marshal(err)
	return string(b)
}

// ParseStatusError extracts the HTTP status + message from an error produced by
// a StatusError-backed type, defaulting to 500.
func ParseStatusError(err error) (int, string) {
	serr := StatusError{}
	if perr := json.Unmarshal([]byte(err.Error()), &serr); perr != nil {
		return http.StatusInternalServerError, err.Error()
	}
	if serr.Status == 0 {
		serr.Status = http.StatusInternalServerError
	}
	return serr.Status, serr.Message
}

// ErrPersonNotFound is returned when a person id has no matching row.
type ErrPersonNotFound struct {
	ID string
}

func (err ErrPersonNotFound) Error() string {
	msg := "person not found"
	if err.ID != "" {
		msg = "person not found: " + err.ID
	}
	return StatusError{Status: http.StatusNotFound, Message: msg}.Error()
}

// IsErrNotFound reports whether err carries a 404 status.
func IsErrNotFound(err error) bool {
	if err == nil {
		return false
	}
	status, _ := ParseStatusError(err)
	return status == http.StatusNotFound
}
