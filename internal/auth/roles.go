package auth

import (
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
)

func rolesToAPI(roles []string) []api.AuthRole {
	result := make([]api.AuthRole, 0, len(roles))
	for _, role := range roles {
		result = append(result, api.AuthRole(role))
	}

	return result
}

func rolesFromAPI(roles *[]api.AuthRole) []string {
	if roles == nil {
		return nil
	}

	result := make([]string, 0, len(*roles))
	for _, role := range *roles {
		result = append(result, string(role))
	}

	return result
}

func rolesFromSlice(roles []api.AuthRole) []string {
	result := make([]string, 0, len(roles))
	for _, role := range roles {
		result = append(result, string(role))
	}

	return result
}

func emailToAPI(email string) openapi_types.Email {
	return openapi_types.Email(email)
}
