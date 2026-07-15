package report

import (
	"errors"
	"fmt"
)

const (
	ErrMsgReportNotFound       = "report not found"
	ErrMsgInvalidReportID      = "invalid report id"
	ErrMsgInvalidReleaseID     = "invalid release id"
	ErrMsgInvalidApplicationID = "invalid application id"
	ErrMsgInvalidMetadata      = "invalid metadata structure"
	ErrMsgInvalidOutput        = "invalid output encoding"
	ErrMsgInvalidReportType    = "invalid report type"
	ErrMsgCreateReport         = "failed to create report"
	ErrMsgGetReport            = "failed to get report"
	ErrMsgListReports          = "failed to list reports"
)

var (
	ErrReportNotFound       = errors.New(ErrMsgReportNotFound)
	ErrInvalidReportID      = errors.New(ErrMsgInvalidReportID)
	ErrInvalidReleaseID     = errors.New(ErrMsgInvalidReleaseID)
	ErrInvalidApplicationID = errors.New(ErrMsgInvalidApplicationID)
	ErrInvalidMetadata      = errors.New(ErrMsgInvalidMetadata)
	ErrInvalidOutput        = errors.New(ErrMsgInvalidOutput)
	ErrInvalidReportType    = errors.New(ErrMsgInvalidReportType)
	ErrCreateReport         = errors.New(ErrMsgCreateReport)
	ErrGetReport            = errors.New(ErrMsgGetReport)
	ErrListReports          = errors.New(ErrMsgListReports)
)

type ReportError struct {
	Message string
	Err     error
}

func (e ReportError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e ReportError) Unwrap() error {
	return e.Err
}

func NewReportError(message string, err error) error {
	return ReportError{Message: message, Err: err}
}

func NewReportNotFoundError(err error) error {
	return NewReportError(ErrReportNotFound.Error(), err)
}

func NewInvalidReportIDError(err error) error {
	return NewReportError(ErrInvalidReportID.Error(), err)
}

func NewInvalidReleaseIDError(err error) error {
	return NewReportError(ErrInvalidReleaseID.Error(), err)
}

func NewInvalidApplicationIDError(err error) error {
	return NewReportError(ErrInvalidApplicationID.Error(), err)
}

func NewInvalidMetadataError(err error) error {
	return NewReportError(ErrInvalidMetadata.Error(), err)
}

func NewInvalidOutputError(err error) error {
	return NewReportError(ErrInvalidOutput.Error(), err)
}

func NewCreateReportError(err error) error {
	return NewReportError(ErrCreateReport.Error(), err)
}

func NewGetReportError(err error) error {
	return NewReportError(ErrGetReport.Error(), err)
}

func NewGetReportsError(err error) error {
	return NewReportError(ErrListReports.Error(), err)
}
