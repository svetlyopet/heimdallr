package rbac

import "errors"

var (
	ErrInsufficientScope   = errors.New("insufficient scope")
	ErrInsufficientRole    = errors.New("insufficient role")
	ErrUnauthorized        = errors.New("invalid credentials")
	ErrPolicyNotConfigured = errors.New("authorization policy not configured")
)

type HTTPError struct {
	Status  int
	Message string
	Err     error
}

func (e *HTTPError) Error() string {
	return e.Message
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

func unauthorizedError() error {
	return &HTTPError{Status: 401, Message: ErrUnauthorized.Error()}
}

func forbiddenScopeError() error {
	return &HTTPError{Status: 403, Message: ErrInsufficientScope.Error()}
}
