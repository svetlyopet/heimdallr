package pagination

import "math"

func SafeTotals(total int64, limit int) (int, int) {
	if total <= 0 || limit <= 0 {
		return 0, 0
	}

	totalPages := (total-1)/int64(limit) + 1
	return clampToInt(total), clampToInt(totalPages)
}

func clampToInt(value int64) int {
	if value > int64(math.MaxInt) {
		return math.MaxInt
	}

	return int(value)
}
