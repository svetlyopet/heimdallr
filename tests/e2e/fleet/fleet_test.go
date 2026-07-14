//go:build e2e

package fleet_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/svetlyopet/heimdallr/tests/flows"
)

func TestFleetE2E(t *testing.T) {
	c := flows.NewLiveClient(t)
	runID := fmt.Sprintf("%d-%d", time.Now().Unix(), os.Getpid())

	t.Run("Seed", func(t *testing.T) {
		flows.FleetComplianceSeed(t, c, runID)
	})

	t.Run("Verify", func(t *testing.T) {
		flows.FleetComplianceVerify(t, c)
	})
}
