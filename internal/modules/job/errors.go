package job

import (
	"errors"
	"fmt"
)

const (
	ErrMsgJobNotFound         = "job not found"
	ErrMsgInvalidJobID        = "invalid job id"
	ErrMsgInvalidAutomationID = "invalid automation id"
	ErrMsgInvalidMetadata     = "invalid metadata structure"
	ErrMsgInvalidOutput       = "invalid output encoding"
	ErrMsgCreateJob           = "failed to create job"
	ErrMsgGetJob              = "failed to get job"
	ErrMsgListJobs            = "failed to list jobs"
)

var (
	ErrJobNotFound         = errors.New(ErrMsgJobNotFound)
	ErrInvalidJobID        = errors.New(ErrMsgInvalidJobID)
	ErrInvalidAutomationID = errors.New(ErrMsgInvalidAutomationID)
	ErrInvalidInput        = errors.New("invalid input")
	ErrInvalidMetadata     = errors.New(ErrMsgInvalidMetadata)
	ErrInvalidOutput       = errors.New(ErrMsgInvalidOutput)
	ErrCreateJob           = errors.New(ErrMsgCreateJob)
	ErrGetJob              = errors.New(ErrMsgGetJob)
	ErrListJobs            = errors.New(ErrMsgListJobs)
)

type JobError struct {
	Message string
	Err     error
}

func (e JobError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e JobError) Unwrap() error {
	return e.Err
}

func NewJobError(message string, err error) error {
	return JobError{
		Message: message,
		Err:     err,
	}
}

func NewJobNotFoundError(err error) error {
	return NewJobError(ErrJobNotFound.Error(), err)
}

func NewInvalidJobIDError(err error) error {
	return NewJobError(ErrInvalidJobID.Error(), err)
}

func NewInvalidAutomationIDError(err error) error {
	return NewJobError(ErrInvalidAutomationID.Error(), err)
}

func NewInvalidMetadataError(err error) error { return NewJobError(ErrInvalidMetadata.Error(), err) }

func NewInvalidOutputError(err error) error { return NewJobError(ErrInvalidOutput.Error(), err) }

func NewCreateJobError(err error) error {
	return NewJobError(ErrCreateJob.Error(), err)
}

func NewGetJobError(err error) error {
	return NewJobError(ErrGetJob.Error(), err)
}

func NewGetJobsError(err error) error {
	return NewJobError(ErrListJobs.Error(), err)
}
