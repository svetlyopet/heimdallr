package auth

import (
	"errors"
	"fmt"
)

const (
	ErrMsgUserNotFound         = "user not found"
	ErrMsgUserAlreadyExists    = "user already exists"
	ErrMsgRootDeleteForbidden  = "root user cannot be deleted"
	ErrMsgRootRoleForbidden    = "root user role cannot be changed"
	ErrMsgInvalidUserID        = "invalid user id"
	ErrMsgInvalidCredentials   = "invalid credentials"
	ErrMsgInvalidRole          = "invalid role"
	ErrMsgInsufficientRole     = "insufficient role"
	ErrMsgListUsers            = "failed to list users"
	ErrMsgCreateUser           = "failed to create user"
	ErrMsgUpdateUser           = "failed to update user"
	ErrMsgDeleteUser           = "failed to delete user"
	ErrMsgRootBootstrap        = "failed to bootstrap root user"
	ErrMsgInvalidAuthHeader    = "missing authentication headers"
	ErrMsgInvalidPasswordValue = "invalid password"
	ErrMsgConcurrentUserUpdate = "concurrent user update"
)

var (
	ErrUserNotFound         = errors.New(ErrMsgUserNotFound)
	ErrUserAlreadyExists    = errors.New(ErrMsgUserAlreadyExists)
	ErrRootDeleteForbidden  = errors.New(ErrMsgRootDeleteForbidden)
	ErrRootRoleForbidden    = errors.New(ErrMsgRootRoleForbidden)
	ErrInvalidUserID        = errors.New(ErrMsgInvalidUserID)
	ErrInvalidCredentials   = errors.New(ErrMsgInvalidCredentials)
	ErrInvalidRole          = errors.New(ErrMsgInvalidRole)
	ErrInsufficientRole     = errors.New(ErrMsgInsufficientRole)
	ErrListUsers            = errors.New(ErrMsgListUsers)
	ErrCreateUser           = errors.New(ErrMsgCreateUser)
	ErrUpdateUser           = errors.New(ErrMsgUpdateUser)
	ErrDeleteUser           = errors.New(ErrMsgDeleteUser)
	ErrRootBootstrap        = errors.New(ErrMsgRootBootstrap)
	ErrInvalidAuthHeader    = errors.New(ErrMsgInvalidAuthHeader)
	ErrInvalidPasswordValue = errors.New(ErrMsgInvalidPasswordValue)
	ErrConcurrentUserUpdate = errors.New(ErrMsgConcurrentUserUpdate)
)

type AuthError struct {
	Message string
	Err     error
}

func (e AuthError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e AuthError) Unwrap() error {
	return e.Err
}

func NewAuthError(message string, err error) error {
	return AuthError{Message: message, Err: err}
}

func NewCreateUserError(err error) error {
	return NewAuthError(ErrCreateUser.Error(), err)
}

func NewListUsersError(err error) error {
	return NewAuthError(ErrListUsers.Error(), err)
}

func NewUpdateUserError(err error) error {
	return NewAuthError(ErrUpdateUser.Error(), err)
}

func NewDeleteUserError(err error) error {
	return NewAuthError(ErrDeleteUser.Error(), err)
}

func NewInvalidUserIDError(err error) error {
	return NewAuthError(ErrInvalidUserID.Error(), err)
}
