//go:build !windows

package government

import "path/filepath"

func samePromotionPathSpelling(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func promotionExtendedPathForTest(path string) string { return path }

func promotionShortPathForTest(path string) (string, bool) { return path, false }
