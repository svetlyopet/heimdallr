//go:build e2e

package compliance_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/svetlyopet/heimdallr/tests/flows"
)

func TestComplianceE2E(t *testing.T) {
	c := flows.NewLiveClient(t)
	runID := fmt.Sprintf("%d-%d", time.Now().Unix(), os.Getpid())

	var state flows.ComplianceState

	t.Run("Seed", func(t *testing.T) {
		state = flows.ComplianceSeed(t, c, runID)
	})

	t.Run("Run", func(t *testing.T) {
		flows.ComplianceRun(t, c, &state)
	})

	t.Run("Verify", func(t *testing.T) {
		flows.ComplianceVerify(t, c, state)
	})
}
