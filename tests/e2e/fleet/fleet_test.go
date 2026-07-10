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
	var state flows.FleetState

	t.Run("Seed", func(t *testing.T) {
		state = flows.FleetSeed(t, c, runID)
	})

	t.Run("VerifyAfterSeed", func(t *testing.T) {
		flows.FleetVerifyAfterSeed(t, c, state)
	})

	t.Run("Run", func(t *testing.T) {
		flows.FleetRun(t, c, &state)
	})

	t.Run("VerifyAfterRun", func(t *testing.T) {
		flows.FleetVerifyAfterRun(t, c, state)
	})

	t.Run("Detach", func(t *testing.T) {
		flows.FleetDetach(t, c, state)
	})

	t.Run("VerifyFinal", func(t *testing.T) {
		flows.FleetVerifyFinal(t, c, state)
	})
}
