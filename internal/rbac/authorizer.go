package rbac

import (
	"slices"
	"strings"

	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
)

type Authorizer interface {
	HasScope(user authapi.AuthUser, scope string) bool
	HasRole(user authapi.AuthUser, role string) bool
	HasAnyRole(user authapi.AuthUser, roles ...string) bool
}

type authorizer struct{}

func NewAuthorizer() Authorizer {
	return authorizer{}
}

func (authorizer) rolesFromUser(user authapi.AuthUser) []string {
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, string(role))
	}

	return roles
}

func (a authorizer) HasScope(user authapi.AuthUser, scope string) bool {
	roles := a.rolesFromUser(user)

	if slices.Contains(roles, RoleAdmin) || slices.Contains(roles, ScopeAdmin) {
		return true
	}

	return slices.Contains(roles, scope)
}

func (a authorizer) HasRole(user authapi.AuthUser, role string) bool {
	return a.HasAnyRole(user, role)
}

func (a authorizer) HasAnyRole(user authapi.AuthUser, requiredRoles ...string) bool {
	if len(requiredRoles) == 0 {
		return false
	}

	userRoles := map[string]struct{}{}
	for _, role := range a.rolesFromUser(user) {
		userRoles[strings.ToLower(strings.TrimSpace(role))] = struct{}{}
	}

	for _, role := range requiredRoles {
		role = strings.ToLower(strings.TrimSpace(role))
		if _, ok := userRoles[role]; ok {
			return true
		}
	}

	return false
}

func ScopesToRoles(scopes []string) []string {
	if slices.Contains(scopes, ScopeAdmin) {
		return []string{RoleAdmin, RoleReader}
	}

	roles := []string{RoleReader}
	for _, scope := range scopes {
		if scope == ScopeApplicationWrite || scope == ScopeAutomationWrite || scope == ScopeRead {
			if !slices.Contains(roles, scope) {
				roles = append(roles, scope)
			}
		}
	}

	return roles
}

func LoginScopesForRoles(roles []string) []string {
	for _, role := range roles {
		if role == RoleAdmin {
			return []string{ScopeAdmin}
		}
	}

	return []string{ScopeRead}
}

func RolesFromLiveUser(roles []string) []string {
	return ScopesToRoles(LoginScopesForRoles(roles))
}
