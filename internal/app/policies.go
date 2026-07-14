package app

import (
	"fmt"
	"reflect"

	"github.com/svetlyopet/heimdallr/internal/auth"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/modules/agent"
	agentapi "github.com/svetlyopet/heimdallr/internal/modules/agent/api"
	"github.com/svetlyopet/heimdallr/internal/modules/analytics"
	analyticsapi "github.com/svetlyopet/heimdallr/internal/modules/analytics/api"
	"github.com/svetlyopet/heimdallr/internal/modules/application"
	applicationapi "github.com/svetlyopet/heimdallr/internal/modules/application/api"
	"github.com/svetlyopet/heimdallr/internal/modules/automation"
	automationapi "github.com/svetlyopet/heimdallr/internal/modules/automation/api"
	"github.com/svetlyopet/heimdallr/internal/modules/job"
	jobapi "github.com/svetlyopet/heimdallr/internal/modules/job/api"
	"github.com/svetlyopet/heimdallr/internal/modules/provider"
	providerapi "github.com/svetlyopet/heimdallr/internal/modules/provider/api"
	"github.com/svetlyopet/heimdallr/internal/modules/release"
	releaseapi "github.com/svetlyopet/heimdallr/internal/modules/release/api"
	"github.com/svetlyopet/heimdallr/internal/modules/report"
	reportapi "github.com/svetlyopet/heimdallr/internal/modules/report/api"
	"github.com/svetlyopet/heimdallr/internal/modules/requiredagent"
	requiredagentapi "github.com/svetlyopet/heimdallr/internal/modules/requiredagent/api"
	"github.com/svetlyopet/heimdallr/internal/modules/server"
	serverapi "github.com/svetlyopet/heimdallr/internal/modules/server/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/token"
	tokenapi "github.com/svetlyopet/heimdallr/internal/token/api"
)

func ValidatePolicies() error {
	checks := []struct {
		domain           string
		strictServerType reflect.Type
		policies         map[string]string
		publicOperations []string
	}{
		{"agent", reflect.TypeFor[agentapi.StrictServerInterface](), agent.Policies, nil},
		{"analytics", reflect.TypeFor[analyticsapi.StrictServerInterface](), analytics.Policies, nil},
		{"application", reflect.TypeFor[applicationapi.StrictServerInterface](), application.Policies, nil},
		{"auth", reflect.TypeFor[authapi.StrictServerInterface](), auth.Policies, []string{"Login", "Logout"}},
		{"automation", reflect.TypeFor[automationapi.StrictServerInterface](), automation.Policies, nil},
		{"job", reflect.TypeFor[jobapi.StrictServerInterface](), job.Policies, nil},
		{"provider", reflect.TypeFor[providerapi.StrictServerInterface](), provider.Policies, nil},
		{"release", reflect.TypeFor[releaseapi.StrictServerInterface](), release.Policies, nil},
		{"report", reflect.TypeFor[reportapi.StrictServerInterface](), report.Policies, nil},
		{"requiredagent", reflect.TypeFor[requiredagentapi.StrictServerInterface](), requiredagent.Policies, nil},
		{"server", reflect.TypeFor[serverapi.StrictServerInterface](), server.Policies, nil},
		{"token", reflect.TypeFor[tokenapi.StrictServerInterface](), token.Policies, nil},
	}

	for _, check := range checks {
		if err := rbac.ValidatePolicyCompleteness(
			check.strictServerType,
			check.policies,
			check.publicOperations...,
		); err != nil {
			return fmt.Errorf("%s policies: %w", check.domain, err)
		}
	}

	return nil
}
