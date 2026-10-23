package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

const dedicatedLogin = `{"auth_mode":"chatgpt","OPENAI_API_KEY":null,"tokens":{"id_token":"dedicated-id-token","access_token":"dedicated-access-token","refresh_token":"dedicated-refresh-token","account_id":"runtime-account"},"last_refresh":"2026-10-17T12:00:00Z"}`

func TestDedicatedRuntimeLoginCopyCleanupAndUncertainty(t *testing.T) {
	for _, mode := range []string{"observed-exit", "uncertain-stop", "cleanup-failure", "install-failure", "container-start-failure"} {
		t.Run(mode, func(t *testing.T) {
			rt := newRuntime()
			rt.stopFail = mode == "uncertain-stop"
			if mode == "cleanup-failure" {
				rt.cleanupErr = errors.New("cleanup refused dedicated-access-token")
			}
			if mode == "install-failure" {
				rt.initializeErr = errors.New("partial setup dedicated-access-token")
			}
			started := 0
			a, err := NewAgentsWithRuntime(func(context.Context, container.ContainerSpec) (RuntimeSession, error) {
				started++
				if mode == "container-start-failure" {
					return nil, errors.New("container refused")
				}
				return rt, nil
			}, map[string]Profile{"codex": testProfile()}, func(_ context.Context, ref string) (RuntimeCredential, error) {
				if ref != "exact-credential" {
					t.Fatal("credential reference changed")
				}
				return RuntimeCredential{Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			a.stopGrace, a.stopTimeout = 1e9, 1e9
			t.Cleanup(func() { _ = rt.Close() })
			res, err := a.Run(context.Background(), testTurn(), okHandler(new([]string)))
			if started != 1 {
				t.Fatal("more than one Manager container started")
			}
			if err != nil && strings.Contains(err.Error(), "dedicated-access-token") {
				t.Fatal("secret escaped in error")
			}
			if mode == "container-start-failure" {
				if err == nil || len(rt.homeCalls) != 0 {
					t.Fatalf("setup ran without a container: %v %v", rt.homeCalls, err)
				}
				return
			}
			var copy runtimeLogin
			if bytes.Contains(rt.installed, []byte("dedicated-refresh-token")) || json.Unmarshal(rt.installed, &copy) != nil || copy.Tokens.Refresh != "" || copy.Tokens.Access != "dedicated-access-token" {
				t.Fatal("bad runtime login copy")
			}
			wantCalls := []string{"initialize", "start", "stop", "clear", "close"}
			switch mode {
			case "observed-exit":
				if err != nil || !res.Stopped || len(rt.start.Env) != 1 || rt.start.Env[0] != "CODEX_HOME=/home/agent/.codex" || strings.Contains(strings.Join(rt.start.Argv, " "), "env_key") {
					t.Fatalf("%+v %v start=%+v", res, err, rt.start)
				}
			case "uncertain-stop":
				wantCalls = []string{"initialize", "start", "stop", "close"}
				if err == nil || res.Stopped {
					t.Fatalf("uncertain stop claimed success: %+v %v", res, err)
				}
			case "cleanup-failure":
				if err == nil || !res.Stopped || !strings.Contains(res.Reason, "cleanup incomplete") {
					t.Fatalf("cleanup failure hidden: %+v %v", res, err)
				}
			case "install-failure":
				wantCalls = []string{"initialize", "clear", "close"}
				if err == nil || rt.starts != 0 || !res.Stopped {
					t.Fatalf("partial setup not closed: %+v %v", res, err)
				}
			}
			if !reflect.DeepEqual(rt.homeCalls, wantCalls) {
				t.Fatalf("lifecycle = %v, want %v", rt.homeCalls, wantCalls)
			}
		})
	}
}
func TestCredentialModesFailClosedWithoutFallback(t *testing.T) {
	for _, c := range []RuntimeCredential{{APIKey: "ambient-mode-not-admitted"}, {Mode: "api-key", APIKey: "key", RuntimeLogin: json.RawMessage(dedicatedLogin)}, {Mode: "desktop-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}} {
		turn := &turnRun{agents: &Agents{}}
		if _, err := turn.credential(context.Background(), c); err == nil {
			t.Fatalf("unsupported credential accepted %+v", c.Mode)
		}
	}
}

func TestRedactionCoversEveryLoginSecret(t *testing.T) {
	turn := &turnRun{agents: &Agents{}, turn: run.Turn{Home: "home", Workspace: "work"}, profile: testProfile()}
	if _, err := turn.credential(context.Background(), RuntimeCredential{Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}); err != nil {
		t.Fatal(err)
	}
	text := turn.redact([]byte(`dedicated-id-token dedicated-access-token dedicated-refresh-token runtime-account`))
	if string(text) != `[REDACTED] [REDACTED] [REDACTED] runtime-account` {
		t.Fatalf("%s", text)
	}
}
