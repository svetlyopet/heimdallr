package rbac

const (
	ScopeApplicationWrite = "application:write"
	ScopeAutomationWrite  = "automation:write"
	ScopeRead             = "read"
	ScopeAdmin            = "admin"
)

var allowedScopes = map[string]struct{}{
	ScopeApplicationWrite: {},
	ScopeAutomationWrite:  {},
	ScopeRead:             {},
	ScopeAdmin:            {},
}

func NormalizeScopes(scopes []string) []string {
	normalized := make([]string, 0, len(scopes))
	seen := map[string]struct{}{}

	for _, scope := range scopes {
		if _, ok := allowedScopes[scope]; !ok {
			continue
		}

		if _, ok := seen[scope]; ok {
			continue
		}

		seen[scope] = struct{}{}
		normalized = append(normalized, scope)
	}

	return normalized
}
