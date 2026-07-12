package release

import (
	"errors"
	"fmt"
)

const (
	ErrMsgReleaseNotFound      = "release not found"
	ErrMsgReleaseAlreadyExists = "release already exists"
	ErrMsgInvalidReleaseID     = "invalid release id"
	ErrMsgInvalidApplicationID = "invalid application id"
	ErrMsgCreateRelease        = "failed to create release"
	ErrMsgGetRelease           = "failed to get release"
	ErrMsgListReleases         = "failed to list releases"
)

var (
	ErrReleaseNotFound      = errors.New(ErrMsgReleaseNotFound)
	ErrReleaseAlreadyExists = errors.New(ErrMsgReleaseAlreadyExists)
	ErrInvalidReleaseID     = errors.New(ErrMsgInvalidReleaseID)
	ErrInvalidApplicationID = errors.New(ErrMsgInvalidApplicationID)
	ErrCreateRelease        = errors.New(ErrMsgCreateRelease)
	ErrGetRelease           = errors.New(ErrMsgGetRelease)
	ErrListReleases         = errors.New(ErrMsgListReleases)
)

type ReleaseError struct {
	Message string
	Err     error
}

func (e ReleaseError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e ReleaseError) Unwrap() error {
	return e.Err
}

func NewReleaseError(message string, err error) error {
	return ReleaseError{Message: message, Err: err}
}

func NewReleaseNotFoundError(err error) error {
	return NewReleaseError(ErrReleaseNotFound.Error(), err)
}

func NewReleaseAlreadyExistsError(err error) error {
	return NewReleaseError(ErrReleaseAlreadyExists.Error(), err)
}

func NewInvalidReleaseIDError(err error) error {
	return NewReleaseError(ErrInvalidReleaseID.Error(), err)
}

func NewInvalidApplicationIDError(err error) error {
	return NewReleaseError(ErrInvalidApplicationID.Error(), err)
}

func NewCreateReleaseError(err error) error {
	return NewReleaseError(ErrCreateRelease.Error(), err)
}

func NewGetReleaseError(err error) error {
	return NewReleaseError(ErrGetRelease.Error(), err)
}

func NewGetReleasesError(err error) error {
	return NewReleaseError(ErrListReleases.Error(), err)
}
