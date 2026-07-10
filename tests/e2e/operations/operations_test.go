//go:build e2e

package operations_test

import (
	"os"
	"testing"

	"github.com/svetlyopet/heimdallr/tests/flows"
)

func TestOperationsVerifyE2E(t *testing.T) {
	automationID := os.Getenv("AUTOMATION_ID")
	if automationID == "" {
		t.Fatal("AUTOMATION_ID is required")
	}

	jobIDSuccess := os.Getenv("JOB_ID_SUCCESS")
	if jobIDSuccess == "" {
		jobIDSuccess = "1000"
	}

	jobIDFailure := os.Getenv("JOB_ID_FAILURE")
	if jobIDFailure == "" {
		jobIDFailure = "1001"
	}

	c := flows.NewLiveClient(t)
	flows.OperationsVerify(t, c, flows.OperationsState{
		AutomationID:   automationID,
		JobIDSuccess:   jobIDSuccess,
		JobIDFailure:   jobIDFailure,
		AutomationName: "e2e-deploy",
	})
}
