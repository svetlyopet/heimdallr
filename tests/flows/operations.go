package flows

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// OperationsVerify asserts job and automation state via GET requests.
func OperationsVerify(t *testing.T, c *Client, state OperationsState) {
	t.Helper()

	resp, automationBody := c.Request(http.MethodGet, "/api/v1/automation/"+state.AutomationID, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, state.AutomationName, DataField(t, automationBody)["name"])

	successPath := "/api/v1/automation/" + state.AutomationID + "/job/" + state.JobIDSuccess
	resp, successBody := c.Request(http.MethodGet, successPath, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	successData := DataField(t, successBody)
	require.Equal(t, "success", successData["status"])
	require.NotEmpty(t, successData["output"])

	failurePath := "/api/v1/automation/" + state.AutomationID + "/job/" + state.JobIDFailure
	resp, failureBody := c.Request(http.MethodGet, failurePath, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "failed", DataField(t, failureBody)["status"])
}
