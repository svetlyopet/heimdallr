//go:build e2e

package software_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/svetlyopet/heimdallr/tests/flows"
)

func TestSoftwareE2E(t *testing.T) {
	c := flows.NewLiveClient(t)
	runID := fmt.Sprintf("%d-%d", time.Now().Unix(), os.Getpid())

	var state flows.SoftwareState

	t.Run("Seed", func(t *testing.T) {
		state = flows.SoftwareSeed(t, c, runID)
	})

	t.Run("Run", func(t *testing.T) {
		flows.SoftwareRun(t, c, &state)
	})

	t.Run("Verify", func(t *testing.T) {
		flows.SoftwareVerify(t, c, state)
	})
}
