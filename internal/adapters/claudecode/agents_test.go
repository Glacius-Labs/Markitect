package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"strings"
	"testing"
)

func TestUnqualifiedClaudeSessionStartsNothingAndFabricatesNoObservation(t *testing.T) {
	a := &Agents{}
	session, err := a.Open(context.Background(), run.SessionSpec{})
	if session != nil || !errors.Is(err, run.ErrUnavailable) {
		t.Fatal(session, err)
	}
	if _, ok := any(a).(run.ObservedSession); ok {
		t.Fatal("unconstructed native identity fabricated")
	}
}
func TestTokenRedactionRetainsLiteralEscapedAndErrorChain(t *testing.T) {
	r := redactor{token: "private\"token"}
	encoded, _ := json.Marshal(r.token)
	if got := r.text(r.token + " " + string(encoded[1:len(encoded)-1])); got != "[REDACTED] [REDACTED]" {
		t.Fatal(got)
	}
	cause := errors.New("failure private\"token")
	err := r.err(cause)
	if !errors.Is(err, cause) || strings.Contains(err.Error(), "private") {
		t.Fatal("redaction lost cause or secret", err)
	}
}
