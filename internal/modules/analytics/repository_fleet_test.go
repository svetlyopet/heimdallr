package analytics

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/modules/agent"
	"github.com/svetlyopet/heimdallr/internal/modules/requiredagent"
	"github.com/svetlyopet/heimdallr/internal/modules/server"
	"gorm.io/datatypes"
)

func TestRepositoryGetFleetComplianceOverviewAggregatesData(t *testing.T) {
	db := newAnalyticsTestDB(t)
	repo := NewRepository(db)

	require.NoError(t, db.Create(&requiredagent.RequiredAgent{
		ID:        uuid.New(),
		AgentName: "req-alpha",
		AgentType: "security",
	}).Error)
	require.NoError(t, db.Create(&requiredagent.RequiredAgent{
		ID:        uuid.New(),
		AgentName: "req-beta",
		AgentType: "monitoring",
	}).Error)

	alphaID := uuid.New()
	betaID := uuid.New()
	require.NoError(t, db.Create(&agent.Agent{
		ID: alphaID, Name: "req-alpha", Type: "security", Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)
	require.NoError(t, db.Create(&agent.Agent{
		ID: betaID, Name: "req-beta", Type: "monitoring", Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	compliantServerID := uuid.New()
	partialServerID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID: compliantServerID, Hostname: "compliant.example.com", Metadata: datatypes.JSON([]byte(`{}`)),
		OperatingSystem: "linux", Hypervisor: "kvm", Location: "dc1",
	}).Error)
	require.NoError(t, db.Create(&server.Server{
		ID: partialServerID, Hostname: "partial.example.com", Metadata: datatypes.JSON([]byte(`{}`)),
		OperatingSystem: "linux", Hypervisor: "kvm", Location: "dc2",
	}).Error)

	require.NoError(t, db.Create(&agent.ServerAgent{ServerID: compliantServerID, AgentID: alphaID}).Error)
	require.NoError(t, db.Create(&agent.ServerAgent{ServerID: compliantServerID, AgentID: betaID}).Error)
	require.NoError(t, db.Create(&agent.ServerAgent{ServerID: partialServerID, AgentID: alphaID}).Error)

	response, err := repo.GetFleetComplianceOverview(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, response.TotalServers)
	require.Equal(t, 1, response.CompliantServers)
	require.Equal(t, 1, response.NonCompliantServers)
	require.Equal(t, 50.0, response.ComplianceRate)
	require.Equal(t, 2, response.TotalRequiredAgents)
	require.Len(t, response.RequiredAgentCoverage, 2)
}

func TestRepositoryListNonCompliantServersPaginatesResults(t *testing.T) {
	db := newAnalyticsTestDB(t)
	repo := NewRepository(db)

	require.NoError(t, db.Create(&requiredagent.RequiredAgent{
		ID:        uuid.New(),
		AgentName: "req-alpha",
		AgentType: "security",
	}).Error)
	require.NoError(t, db.Create(&requiredagent.RequiredAgent{
		ID:        uuid.New(),
		AgentName: "req-beta",
		AgentType: "monitoring",
	}).Error)

	alphaID := uuid.New()
	betaID := uuid.New()
	require.NoError(t, db.Create(&agent.Agent{
		ID: alphaID, Name: "req-alpha", Type: "security", Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)
	require.NoError(t, db.Create(&agent.Agent{
		ID: betaID, Name: "req-beta", Type: "monitoring", Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	compliantServerID := uuid.New()
	partialServerID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID: compliantServerID, Hostname: "compliant.example.com", Metadata: datatypes.JSON([]byte(`{}`)),
		OperatingSystem: "linux", Hypervisor: "kvm", Location: "dc1",
	}).Error)
	require.NoError(t, db.Create(&server.Server{
		ID: partialServerID, Hostname: "partial.example.com", Metadata: datatypes.JSON([]byte(`{}`)),
		OperatingSystem: "linux", Hypervisor: "kvm", Location: "dc2",
	}).Error)

	require.NoError(t, db.Create(&agent.ServerAgent{ServerID: compliantServerID, AgentID: alphaID}).Error)
	require.NoError(t, db.Create(&agent.ServerAgent{ServerID: compliantServerID, AgentID: betaID}).Error)
	require.NoError(t, db.Create(&agent.ServerAgent{ServerID: partialServerID, AgentID: alphaID}).Error)

	details, total, err := repo.ListNonCompliantServers(context.Background(), 1, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, details, 1)
	require.Equal(t, "partial.example.com", details[0].Hostname)
	require.Equal(t, []string{"req-beta"}, details[0].MissingAgents)
}
