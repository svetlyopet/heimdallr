package provider

import (
	"errors"
	"fmt"
)

const (
	ErrMsgProviderNotFound      = "provider not found"
	ErrMsgProviderAlreadyExists = "provider already exists"
	ErrMsgInvalidProviderID     = "invalid provider id"
	ErrMsgCreateProvider        = "failed to create provider"
	ErrMsgGetProvider           = "failed to get provider"
	ErrMsgListProviders         = "failed to list providers"
)

var (
	ErrProviderNotFound      = errors.New(ErrMsgProviderNotFound)
	ErrProviderAlreadyExists = errors.New(ErrMsgProviderAlreadyExists)
	ErrInvalidProviderID     = errors.New(ErrMsgInvalidProviderID)
	ErrCreateProvider        = errors.New(ErrMsgCreateProvider)
	ErrGetProvider           = errors.New(ErrMsgGetProvider)
	ErrListProviders         = errors.New(ErrMsgListProviders)
)

type ProviderError struct {
	Message string
	Err     error
}

func (e ProviderError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e ProviderError) Unwrap() error {
	return e.Err
}

func NewProviderError(message string, err error) error {
	return ProviderError{
		Message: message,
		Err:     err,
	}
}

func NewProviderNotFoundError(err error) error {
	return NewProviderError(ErrProviderNotFound.Error(), err)
}

func NewProviderAlreadyExistsError(err error) error {
	return NewProviderError(ErrProviderAlreadyExists.Error(), err)
}

func NewInvalidProviderIDError(err error) error {
	return NewProviderError(ErrInvalidProviderID.Error(), err)
}

func NewCreateProviderError(err error) error {
	return NewProviderError(ErrCreateProvider.Error(), err)
}

func NewGetProviderError(err error) error {
	return NewProviderError(ErrGetProvider.Error(), err)
}

func NewGetProvidersError(err error) error {
	return NewProviderError(ErrListProviders.Error(), err)
}
