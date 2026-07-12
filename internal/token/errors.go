package token

import (
	"errors"
	"fmt"
)

var (
	ErrTokenNotFound = errors.New("api token not found")
	ErrInvalidToken  = errors.New("invalid api token")
	ErrCreateToken   = errors.New("failed to create api token")
	ErrListTokens    = errors.New("failed to list api tokens")
	ErrDeleteToken   = errors.New("failed to delete api token")
	ErrInvalidScopes = errors.New("invalid token scopes")
	ErrInvalidTTL    = errors.New("invalid token ttl")
)

type TokenError struct {
	Message string
	Err     error
}

func (e TokenError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e TokenError) Unwrap() error {
	return e.Err
}

func NewTokenError(message string, err error) error {
	return TokenError{Message: message, Err: err}
}
