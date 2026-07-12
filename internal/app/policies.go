package app

import (
	"fmt"
	"reflect"

	"github.com/svetlyopet/heimdallr/internal/agent"
	agentapi "github.com/svetlyopet/heimdallr/internal/agent/api"
	"github.com/svetlyopet/heimdallr/internal/analytics"
	analyticsapi "github.com/svetlyopet/heimdallr/internal/analytics/api"
	"github.com/svetlyopet/heimdallr/internal/application"
	applicationapi "github.com/svetlyopet/heimdallr/internal/application/api"
	"github.com/svetlyopet/heimdallr/internal/auth"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/automation"
	automationapi "github.com/svetlyopet/heimdallr/internal/automation/api"
	"github.com/svetlyopet/heimdallr/internal/job"
	jobapi "github.com/svetlyopet/heimdallr/internal/job/api"
	"github.com/svetlyopet/heimdallr/internal/provider"
	providerapi "github.com/svetlyopet/heimdallr/internal/provider/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/release"
	releaseapi "github.com/svetlyopet/heimdallr/internal/release/api"
	"github.com/svetlyopet/heimdallr/internal/report"
	reportapi "github.com/svetlyopet/heimdallr/internal/report/api"
	"github.com/svetlyopet/heimdallr/internal/server"
	serverapi "github.com/svetlyopet/heimdallr/internal/server/api"
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
