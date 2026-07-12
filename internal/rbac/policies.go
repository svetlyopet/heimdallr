package rbac

import (
	"fmt"
	"reflect"
)

func ValidatePolicyCompleteness(
	strictServerInterface reflect.Type,
	policies map[string]string,
	publicOperations ...string,
) error {
	if strictServerInterface.Kind() != reflect.Interface {
		return fmt.Errorf("strict server type must be an interface")
	}

	public := make(map[string]struct{}, len(publicOperations))
	for _, operationID := range publicOperations {
		public[operationID] = struct{}{}
	}

	operations := make(map[string]struct{}, strictServerInterface.NumMethod())
	for index := range strictServerInterface.NumMethod() {
		operationID := strictServerInterface.Method(index).Name
		operations[operationID] = struct{}{}
		if _, isPublic := public[operationID]; isPublic {
			continue
		}

		if scope, ok := policies[operationID]; !ok || scope == "" {
			return fmt.Errorf("%w: %s", ErrPolicyNotConfigured, operationID)
		}
	}

	for operationID := range policies {
		if _, ok := operations[operationID]; !ok {
			return fmt.Errorf("stale authorization policy: %s", operationID)
		}
		if _, isPublic := public[operationID]; isPublic {
			return fmt.Errorf("public operation must not have an authorization policy: %s", operationID)
		}
	}

	return nil
}
