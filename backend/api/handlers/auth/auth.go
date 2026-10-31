package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrWeakPassword       = errors.New("password must be 12–128 chars with letters and digits")
	ErrEmailTaken         = errors.New("email already registered")
	ErrAccountInactive    = errors.New("account is not active")
	ErrSessionNotFound    = errors.New("session not found or expired")
)
