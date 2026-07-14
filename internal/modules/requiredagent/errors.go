package requiredagent

import (
	"errors"
	"fmt"
)

const (
	ErrMsgRequiredAgentNotFound      = "required agent not found"
	ErrMsgRequiredAgentAlreadyExists = "required agent already exists"
	ErrMsgInvalidRequiredAgentID     = "invalid required agent id"
	ErrMsgCreateRequiredAgent        = "failed to create required agent"
	ErrMsgGetRequiredAgent           = "failed to get required agent"
	ErrMsgListRequiredAgents         = "failed to list required agents"
	ErrMsgUpdateRequiredAgent        = "failed to update required agent"
	ErrMsgDeleteRequiredAgent        = "failed to delete required agent"
)

var (
	ErrRequiredAgentNotFound      = errors.New(ErrMsgRequiredAgentNotFound)
	ErrRequiredAgentAlreadyExists = errors.New(ErrMsgRequiredAgentAlreadyExists)
	ErrInvalidRequiredAgentID     = errors.New(ErrMsgInvalidRequiredAgentID)
	ErrCreateRequiredAgent        = errors.New(ErrMsgCreateRequiredAgent)
	ErrGetRequiredAgent           = errors.New(ErrMsgGetRequiredAgent)
	ErrListRequiredAgents         = errors.New(ErrMsgListRequiredAgents)
	ErrUpdateRequiredAgent        = errors.New(ErrMsgUpdateRequiredAgent)
	ErrDeleteRequiredAgent        = errors.New(ErrMsgDeleteRequiredAgent)
)

type RequiredAgentError struct {
	Message string
	Err     error
}

func (e RequiredAgentError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e RequiredAgentError) Unwrap() error {
	return e.Err
}

func NewRequiredAgentError(message string, err error) error {
	return RequiredAgentError{
		Message: message,
		Err:     err,
	}
}

func requiredAgentErrorMessage(err error, fallback string) string {
	if requiredAgentErr, ok := errors.AsType[RequiredAgentError](err); ok {
		return requiredAgentErr.Message
	}

	return fallback
}
