package application

import (
	"errors"
	"fmt"
)

const (
	ErrMsgApplicationNotFound      = "application not found"
	ErrMsgApplicationAlreadyExists = "application already exists"
	ErrMsgInvalidApplicationID     = "invalid application id"
	ErrMsgCreateApplication        = "failed to create application"
	ErrMsgGetApplication           = "failed to get application"
	ErrMsgListApplications         = "failed to list applications"
)

var (
	ErrApplicationNotFound      = errors.New(ErrMsgApplicationNotFound)
	ErrApplicationAlreadyExists = errors.New(ErrMsgApplicationAlreadyExists)
	ErrInvalidApplicationID     = errors.New(ErrMsgInvalidApplicationID)
	ErrCreateApplication        = errors.New(ErrMsgCreateApplication)
	ErrGetApplication           = errors.New(ErrMsgGetApplication)
	ErrListApplications         = errors.New(ErrMsgListApplications)
)

type ApplicationError struct {
	Message string
	Err     error
}

func (e ApplicationError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e ApplicationError) Unwrap() error {
	return e.Err
}

func NewApplicationError(message string, err error) error {
	return ApplicationError{
		Message: message,
		Err:     err,
	}
}

func NewApplicationNotFoundError(err error) error {
	return NewApplicationError(ErrApplicationNotFound.Error(), err)
}

func NewApplicationAlreadyExistsError(err error) error {
	return NewApplicationError(ErrApplicationAlreadyExists.Error(), err)
}

func NewInvalidApplicationIDError(err error) error {
	return NewApplicationError(ErrInvalidApplicationID.Error(), err)
}

func NewCreateApplicationError(err error) error {
	return NewApplicationError(ErrCreateApplication.Error(), err)
}

func NewGetApplicationError(err error) error {
	return NewApplicationError(ErrGetApplication.Error(), err)
}

func NewGetApplicationsError(err error) error {
	return NewApplicationError(ErrListApplications.Error(), err)
}
