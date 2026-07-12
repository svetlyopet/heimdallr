package agent

import (
	"errors"
	"fmt"
)

const (
	ErrMsgAgentNotFound      = "agent not found"
	ErrMsgInvalidAgentID     = "invalid agent id"
	ErrMsgInvalidServerID    = "invalid server id"
	ErrMsgCreateAgent        = "failed to create agent"
	ErrMsgGetAgent           = "failed to get agent"
	ErrMsgListAgents         = "failed to list agents"
	ErrMsgDeleteAgent        = "failed to delete agent"
	ErrMsgAgentAlreadyLinked = "agent is already linked to this server"
	ErrMsgAgentAlreadyExists = "agent already exists"
	ErrMsgCorruptMetadata    = "corrupt stored metadata"
)

var (
	ErrAgentNotFound      = errors.New(ErrMsgAgentNotFound)
	ErrInvalidAgentID     = errors.New(ErrMsgInvalidAgentID)
	ErrInvalidServerID    = errors.New(ErrMsgInvalidServerID)
	ErrCreateAgent        = errors.New(ErrMsgCreateAgent)
	ErrGetAgent           = errors.New(ErrMsgGetAgent)
	ErrListAgents         = errors.New(ErrMsgListAgents)
	ErrDeleteAgent        = errors.New(ErrMsgDeleteAgent)
	ErrAgentAlreadyLinked = errors.New(ErrMsgAgentAlreadyLinked)
	ErrAgentAlreadyExists = errors.New(ErrMsgAgentAlreadyExists)
	ErrCorruptMetadata    = errors.New(ErrMsgCorruptMetadata)
)

type AgentError struct {
	Message string
	Err     error
}

func (e AgentError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e AgentError) Unwrap() error {
	return e.Err
}

func NewAgentError(message string, err error) error {
	return AgentError{
		Message: message,
		Err:     err,
	}
}

func NewAgentNotFoundError(err error) error {
	return NewAgentError(ErrAgentNotFound.Error(), err)
}

func NewInvalidAgentIDError(err error) error {
	return NewAgentError(ErrInvalidAgentID.Error(), err)
}

func NewInvalidServerIDError(err error) error {
	return NewAgentError(ErrInvalidServerID.Error(), err)
}

func NewCreateAgentError(err error) error {
	return NewAgentError(ErrCreateAgent.Error(), err)
}

func NewGetAgentError(err error) error {
	return NewAgentError(ErrGetAgent.Error(), err)
}

func NewListAgentsError(err error) error {
	return NewAgentError(ErrListAgents.Error(), err)
}

func NewDeleteAgentError(err error) error {
	return NewAgentError(ErrDeleteAgent.Error(), err)
}

func NewAgentAlreadyExistsError(err error) error {
	return NewAgentError(ErrAgentAlreadyExists.Error(), err)
}
