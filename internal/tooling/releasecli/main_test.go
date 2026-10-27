package releasecli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunUsesProvidedOutputStreamsAndPreservesExitCodes(t *testing.T) {
	var out, errout bytes.Buffer
	if code := Run([]string{"--help"}, &out, &errout); code != 0 || out.Len() != 0 || !strings.Contains(errout.String(), "Usage of markitect-release") {
		t.Fatalf("help invocation: code=%d stdout=%q stderr=%q", code, out.String(), errout.String())
	}
	for _, args := range [][]string{{"--unknown"}, {}, {"distribution"}, {"--tag", "v1.2.3"}} {
		out.Reset()
		errout.Reset()
		if code := Run(args, &out, &errout); code != 2 || out.Len() != 0 || errout.Len() == 0 {
			t.Fatalf("invalid invocation %v: code=%d stdout=%q stderr=%q", args, code, out.String(), errout.String())
		}
	}
}
