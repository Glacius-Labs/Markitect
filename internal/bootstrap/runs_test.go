package bootstrap

import "testing"

// The host reads the decision log while hooks append to it: a push still
// being written is a last line without its newline, and is left for later.
func TestTheDecisionLogSkipsAPushStillBeingWritten(t *testing.T) {
	whole := `{"user":"a","updates":[],"accepted":true,"kind":"work"}` + "\n"
	pushes, err := parseLog([]byte(whole + `{"user":"b","upda`))
	if err != nil || len(pushes) != 1 || pushes[0].User != "a" || !pushes[0].Decision.Accepted {
		t.Fatalf("%+v %v", pushes, err)
	}
	if pushes, err := parseLog(nil); err != nil || len(pushes) != 0 {
		t.Fatalf("an empty log: %+v %v", pushes, err)
	}
	if _, err := parseLog([]byte("not json\n")); err == nil {
		t.Fatal("a broken complete line is no error")
	}
}
