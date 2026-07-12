package rbac

import "errors"

var (
	ErrInsufficientScope = errors.New("insufficient scope")
	ErrInsufficientRole  = errors.New("insufficient role")
	ErrUnauthorized      = errors.New("invalid credentials")
)

type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string {
	return e.Message
}

func unauthorizedError() error {
	return &HTTPError{Status: 401, Message: ErrUnauthorized.Error()}
}

func forbiddenScopeError() error {
	return &HTTPError{Status: 403, Message: ErrInsufficientScope.Error()}
}
