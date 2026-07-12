package server

import (
	"errors"
	"fmt"
)

const (
	ErrMsgServerNotFound           = "server not found"
	ErrMsgServerAlreadyExists      = "server already exists"
	ErrMsgInvalidServerID          = "invalid server id"
	ErrMsgCreateServer             = "failed to create server"
	ErrMsgGetServer                = "failed to get server"
	ErrMsgListServers              = "failed to list servers"
	ErrMsgJobNotFound              = "job not found"
	ErrMsgJobAlreadyAssociated     = "job already associated with server"
	ErrMsgAssociateJob             = "failed to associate job"
	ErrMsgDissociateJob            = "failed to dissociate job"
	ErrMsgListJobs                 = "failed to list jobs"
	ErrMsgReleaseNotFound          = "release not found"
	ErrMsgReleaseAlreadyAssociated = "release already associated with server"
	ErrMsgAssociateRelease         = "failed to associate release"
	ErrMsgDissociateRelease        = "failed to dissociate release"
	ErrMsgListReleases             = "failed to list releases"
	ErrMsgAttachAgents             = "failed to attach agents to server"
	ErrMsgAgentAlreadyLinked       = "agent is already linked to this server"
	ErrMsgAgentAlreadyExists       = "agent already exists"
	ErrMsgDuplicateAgentIDs        = "duplicate agent ids in request"
	ErrMsgDuplicateAgentNames      = "duplicate agent names in request"
	ErrMsgUpdateServer             = "failed to update server"
	ErrMsgCorruptMetadata          = "corrupt stored metadata"
)

var (
	ErrServerNotFound           = errors.New(ErrMsgServerNotFound)
	ErrServerAlreadyExists      = errors.New(ErrMsgServerAlreadyExists)
	ErrInvalidServerID          = errors.New(ErrMsgInvalidServerID)
	ErrCreateServer             = errors.New(ErrMsgCreateServer)
	ErrGetServer                = errors.New(ErrMsgGetServer)
	ErrListServers              = errors.New(ErrMsgListServers)
	ErrJobNotFound              = errors.New(ErrMsgJobNotFound)
	ErrJobAlreadyAssociated     = errors.New(ErrMsgJobAlreadyAssociated)
	ErrAssociateJob             = errors.New(ErrMsgAssociateJob)
	ErrDissociateJob            = errors.New(ErrMsgDissociateJob)
	ErrListJobs                 = errors.New(ErrMsgListJobs)
	ErrReleaseNotFound          = errors.New(ErrMsgReleaseNotFound)
	ErrReleaseAlreadyAssociated = errors.New(ErrMsgReleaseAlreadyAssociated)
	ErrAssociateRelease         = errors.New(ErrMsgAssociateRelease)
	ErrDissociateRelease        = errors.New(ErrMsgDissociateRelease)
	ErrListReleases             = errors.New(ErrMsgListReleases)
	ErrAttachAgents             = errors.New(ErrMsgAttachAgents)
	ErrAgentAlreadyLinked       = errors.New(ErrMsgAgentAlreadyLinked)
	ErrAgentAlreadyExists       = errors.New(ErrMsgAgentAlreadyExists)
	ErrDuplicateAgentIDs        = errors.New(ErrMsgDuplicateAgentIDs)
	ErrDuplicateAgentNames      = errors.New(ErrMsgDuplicateAgentNames)
	ErrUpdateServer             = errors.New(ErrMsgUpdateServer)
	ErrCorruptMetadata          = errors.New(ErrMsgCorruptMetadata)
)

type ServerError struct {
	Message string
	Err     error
}

func (e ServerError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e ServerError) Unwrap() error {
	return e.Err
}

func NewServerError(message string, err error) error {
	return ServerError{
		Message: message,
		Err:     err,
	}
}

func NewServerNotFoundError(err error) error {
	return NewServerError(ErrServerNotFound.Error(), err)
}

func NewServerAlreadyExistsError(err error) error {
	return NewServerError(ErrServerAlreadyExists.Error(), err)
}

func NewInvalidServerIDError(err error) error {
	return NewServerError(ErrInvalidServerID.Error(), err)
}

func NewCreateServerError(err error) error {
	return NewServerError(ErrCreateServer.Error(), err)
}

func NewGetServerError(err error) error {
	return NewServerError(ErrGetServer.Error(), err)
}

func NewGetServersError(err error) error {
	return NewServerError(ErrListServers.Error(), err)
}

func NewJobNotFoundError(err error) error {
	return NewServerError(ErrJobNotFound.Error(), err)
}

func NewJobAlreadyAssociatedError(err error) error {
	return NewServerError(ErrJobAlreadyAssociated.Error(), err)
}

func NewAssociateJobError(err error) error {
	return NewServerError(ErrAssociateJob.Error(), err)
}

func NewDissociateJobError(err error) error {
	return NewServerError(ErrDissociateJob.Error(), err)
}

func NewListJobsError(err error) error {
	return NewServerError(ErrListJobs.Error(), err)
}

func NewReleaseNotFoundError(err error) error {
	return NewServerError(ErrReleaseNotFound.Error(), err)
}

func NewReleaseAlreadyAssociatedError(err error) error {
	return NewServerError(ErrReleaseAlreadyAssociated.Error(), err)
}

func NewAssociateReleaseError(err error) error {
	return NewServerError(ErrAssociateRelease.Error(), err)
}

func NewDissociateReleaseError(err error) error {
	return NewServerError(ErrDissociateRelease.Error(), err)
}

func NewListReleasesError(err error) error {
	return NewServerError(ErrListReleases.Error(), err)
}

func NewAttachAgentsError(err error) error {
	return NewServerError(ErrAttachAgents.Error(), err)
}

func NewAgentAlreadyLinkedError(err error) error {
	return NewServerError(ErrAgentAlreadyLinked.Error(), err)
}

func NewAgentAlreadyExistsError(err error) error {
	return NewServerError(ErrAgentAlreadyExists.Error(), err)
}

func NewUpdateServerError(err error) error {
	return NewServerError(ErrUpdateServer.Error(), err)
}
