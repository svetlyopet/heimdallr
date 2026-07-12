package testutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaNameFromQualifiedDistinctAcrossPackages(t *testing.T) {
	t.Parallel()

	const testName = "TestServiceGetByIdReturnsNotFound"
	provider := schemaNameFromQualified("github.com/svetlyopet/heimdallr/modules/provider." + testName)
	agent := schemaNameFromQualified("github.com/svetlyopet/heimdallr/modules/agent." + testName)

	require.NotEqual(t, provider, agent)
	require.True(t, strings.HasPrefix(provider, "t_"))
	require.True(t, strings.HasPrefix(agent, "t_"))
}

func TestTestSchemaNameUsesQualifiedCaller(t *testing.T) {
	schema := postgresSchemaName(t)
	require.NotEmpty(t, schema)
	require.True(t, strings.HasPrefix(schema, "t_"))

	qualified := schemaNameFromQualified(t.Name())
	require.NotEqual(t, schema, qualified, "schema name should include package, not bare test name")
}

func TestTestSchemaNameDistinctAcrossHelperDepths(t *testing.T) {
	shallow := postgresSchemaName(t)
	deep := postgresSchemaNameViaInjector(t)
	require.Equal(t, shallow, deep, "schema name should resolve to the test function regardless of helper depth")
}

func postgresSchemaName(t *testing.T) string {
	t.Helper()
	return testSchemaName(t)
}

func postgresSchemaNameViaInjector(t *testing.T) string {
	t.Helper()
	return postgresDatabaseURLSchemaName(t)
}

func postgresDatabaseURLSchemaName(t *testing.T) string {
	t.Helper()
	return testSchemaName(t)
}
