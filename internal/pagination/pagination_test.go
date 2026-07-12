package pagination_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/pagination"
)

func TestSafeTotalsAvoidsOverflow(t *testing.T) {
	t.Parallel()

	total, pages := pagination.SafeTotals(math.MaxInt64, 100)
	require.Equal(t, math.MaxInt, total)
	expectedPages := (int64(math.MaxInt64)-1)/100 + 1
	if expectedPages > int64(math.MaxInt) {
		expectedPages = int64(math.MaxInt)
	}
	require.Equal(t, int(expectedPages), pages)
}
