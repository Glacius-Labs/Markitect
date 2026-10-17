//go:build !linux && !windows

package codexappserver

import (
	"fmt"
	"os"
)

func openMetadataVolume(string) (*os.File, error) {
	return nil, fmt.Errorf("safe native metadata acquisition is unsupported on this platform")
}
func openMetadataChild(*os.File, string, bool) (*os.File, error) {
	return nil, fmt.Errorf("safe native metadata acquisition is unsupported on this platform")
}
