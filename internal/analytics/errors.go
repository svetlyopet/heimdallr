package analytics

import (
	"errors"
	"fmt"
)

const (
	ErrMsgGetAutomationAnalytics = "failed to get automation analytics"
	ErrMsgAutomationNotFound     = "automation not found"
)

var (
	ErrGetAutomationAnalytics = errors.New(ErrMsgGetAutomationAnalytics)
	ErrAutomationNotFound     = errors.New(ErrMsgAutomationNotFound)
)

type AnalyticsError struct {
	Message string
	Err     error
}

func (e AnalyticsError) Error() string {
	if e.Err == nil {
		return e.Message
	}

	if e.Message == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e AnalyticsError) Unwrap() error {
	return e.Err
}

func NewAnalyticsError(message string, err error) error {
	return AnalyticsError{
		Message: message,
		Err:     err,
	}
}

func NewGetAutomationAnalyticsError(err error) error {
	return NewAnalyticsError(ErrGetAutomationAnalytics.Error(), err)
}
