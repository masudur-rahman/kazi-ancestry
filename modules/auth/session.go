// Package auth provides Google OAuth login and stateless signed-cookie sessions.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// SessionCookie is the cookie name carrying the signed session.
const SessionCookie = "kazi_session"

// sessionTTL is how long a login stays valid.
const sessionTTL = 30 * 24 * time.Hour

// Session is the authenticated identity carried in the cookie.
type Session struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	Exp   int64  `json:"exp"`
}

var errInvalidSession = errors.New("invalid session")

// NewSession builds a session with the default TTL.
func NewSession(email, name, role string) Session {
	return Session{Email: email, Name: name, Role: role, Exp: time.Now().Add(sessionTTL).Unix()}
}

// Sign serializes and HMAC-signs the session into a cookie value: payload.sig
// (both base64url). The signature covers the payload with the given secret.
func Sign(s Session, secret string) string {
	payload, _ := json.Marshal(s)
	enc := base64.RawURLEncoding.EncodeToString(payload)
	return enc + "." + sign(enc, secret)
}

// Verify checks the signature and expiry, returning the session.
func Verify(value, secret string) (*Session, error) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return nil, errInvalidSession
	}
	if !hmac.Equal([]byte(parts[1]), []byte(sign(parts[0], secret))) {
		return nil, errInvalidSession
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errInvalidSession
	}
	var s Session
	if err := json.Unmarshal(payload, &s); err != nil {
		return nil, errInvalidSession
	}
	if time.Now().Unix() > s.Exp {
		return nil, errInvalidSession
	}
	return &s, nil
}

func sign(msg, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// MaxAgeSeconds is the cookie Max-Age matching the session TTL.
func MaxAgeSeconds() int { return int(sessionTTL / time.Second) }
