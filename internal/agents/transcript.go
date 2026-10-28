package agents

import (
	"encoding/json"
	"time"
)

// The transcript holds the provider's output lines as they came, so tools
// that read a provider's stream read it too, and the host's own records as
// lines of type "markitect". Every secret is redacted before it is written.

func (a *Agent) write(line []byte) {
	if a.spec.Transcript == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	out := append(a.redact(append([]byte(nil), line...)), '\n')
	_, _ = a.spec.Transcript.Write(out)
}

func (a *Agent) record(kind string, value any) {
	if a.spec.Transcript == nil {
		return
	}
	now := time.Now
	if a.spec.Now != nil {
		now = a.spec.Now
	}
	raw, err := json.Marshal(map[string]any{"type": "markitect", "kind": kind, "time": now().UTC().Format(time.RFC3339Nano), "value": value})
	if err == nil {
		a.write(raw)
	}
}
