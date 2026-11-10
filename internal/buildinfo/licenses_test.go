package buildinfo

import (
	"strings"
	"testing"
)

func TestNoticesAreEmbedded(t *testing.T) {
	if strings.TrimSpace(Notices) == "" {
		t.Fatal("the embedded license notices are empty")
	}
}
