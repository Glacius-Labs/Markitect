package desktop

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSettingsShowTheAgentsAndTheLogins(t *testing.T) {
	s, _ := testServer(t)
	if w := call(s, "GET", "/api/settings", ""); w.Code != http.StatusNotImplemented {
		t.Errorf("without settings: %d", w.Code)
	}
	expires := time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)
	s.settings = func(context.Context) AppSettings {
		return AppSettings{
			Agents: AgentSettings{Provider: "claude", Model: "claude-sonnet-5-5", Effort: "high", Network: "markitect-provider", Problems: []string{"no Manager image"}},
			Logins: []LoginState{{Provider: "claude", File: "~/.claude/.credentials.json", State: "ok", Expires: &expires, Command: "claude", Used: true}},
		}
	}
	w := call(s, "GET", "/api/settings", "")
	var got AppSettings
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if got.Agents.Model != "claude-sonnet-5-5" || len(got.Logins) != 1 || !got.Logins[0].Expires.Equal(expires) || got.Agents.Problems[0] != "no Manager image" {
		t.Errorf("settings %+v", got)
	}
}

func TestSettingsShowTheProjectsRequiredChecks(t *testing.T) {
	files := library(t)
	files[".markitect/execution.yaml"] = "checks:\n  - name: test\n    command: go test ./...\n    timeout: 10m\n    environment: golang:1.27\n"
	s, _, _, id := modelServer(t, files)
	var got projectSettings
	w := call(s, "GET", "/api/projects/"+id+"/settings", "")
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if !got.Initialized || len(got.Checks) != 1 || got.Checks[0].Command != "go test ./..." || got.Checks[0].Timeout != 10*time.Minute || got.Problem != "" {
		t.Errorf("settings %+v", got)
	}

	files[".markitect/execution.yaml"] = "checks:\n  - name: test\n"
	w = call(s, "GET", "/api/projects/"+id+"/settings", "")
	got = projectSettings{}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if !strings.Contains(got.Problem, "execution.yaml") {
		t.Errorf("a broken execution.yaml: %+v", got)
	}

	s2, _, _, id2 := modelServer(t, map[string]string{"README.md": "# x\n"})
	got = projectSettings{}
	_ = json.Unmarshal(call(s2, "GET", "/api/projects/"+id2+"/settings", "").Body.Bytes(), &got)
	if got.Initialized || len(got.Checks) != 0 {
		t.Errorf("a project without Markitect: %+v", got)
	}
}
