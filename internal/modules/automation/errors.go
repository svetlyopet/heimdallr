package automation

import (
	"errors"
	"fmt"
)

const (
	ErrMsgAutomationNotFound      = "automation not found"
	ErrMsgAutomationAlreadyExists = "automation already exists"
	ErrMsgInvalidAutomationID     = "invalid automation id"
	ErrMsgCreateAutomation        = "failed to create automation"
	ErrMsgUpdateAutomation        = "failed to update automation"
	ErrMsgDeleteAutomation        = "failed to delete automation"
	ErrMsgGetAutomation           = "failed to get automation"
	ErrMsgListAutomations         = "failed to list automations"
)

var (
	ErrAutomationNotFound      = errors.New(ErrMsgAutomationNotFound)
	ErrAutomationAlreadyExists = errors.New(ErrMsgAutomationAlreadyExists)
	ErrInvalidAutomationID     = errors.New(ErrMsgInvalidAutomationID)
	ErrCreateAutomation        = errors.New(ErrMsgCreateAutomation)
	ErrUpdateAutomation        = errors.New(ErrMsgUpdateAutomation)
	ErrDeleteAutomation        = errors.New(ErrMsgDeleteAutomation)
	ErrGetAutomation           = errors.New(ErrMsgGetAutomation)
	ErrListAutomations         = errors.New(ErrMsgListAutomations)
)

type AutomationError struct {
	Message string
	Err     error
}

func (e AutomationError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e AutomationError) Unwrap() error {
	return e.Err
}

func NewAutomationError(message string, err error) error {
	return AutomationError{
		Message: message,
		Err:     err,
	}
}

func NewAutomationNotFoundError(err error) error {
	return NewAutomationError(ErrAutomationNotFound.Error(), err)
}

func NewAutomationAlreadyExistsError(err error) error {
	return NewAutomationError(ErrAutomationAlreadyExists.Error(), err)
}

func NewInvalidAutomationIDError(err error) error {
	return NewAutomationError(ErrInvalidAutomationID.Error(), err)
}

func NewCreateAutomationError(err error) error {
	return NewAutomationError(ErrCreateAutomation.Error(), err)
}

func NewUpdateAutomationError(err error) error {
	return NewAutomationError(ErrUpdateAutomation.Error(), err)
}

func NewDeleteAutomationError(err error) error {
	return NewAutomationError(ErrDeleteAutomation.Error(), err)
}

func NewGetAutomationError(err error) error {
	return NewAutomationError(ErrGetAutomation.Error(), err)
}

func NewGetAutomationsError(err error) error {
	return NewAutomationError(ErrListAutomations.Error(), err)
}
