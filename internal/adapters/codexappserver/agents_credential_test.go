package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

const dedicatedLogin = `{"auth_mode":"chatgpt","OPENAI_API_KEY":null,"tokens":{"id_token":"dedicated-id-token","access_token":"dedicated-access-token","refresh_token":"dedicated-refresh-token","account_id":"runtime-account"},"last_refresh":"2026-10-17T12:00:00Z"}`

func TestDedicatedRuntimeLoginCopyCleanupAndUncertainty(t *testing.T) {
	for _, mode := range []string{"observed-exit", "uncertain-stop", "cleanup-failure", "install-failure"} {
		t.Run(mode, func(t *testing.T) {
			rt := newRuntime()
			rt.stopFail = mode == "uncertain-stop"
			var installed []byte
			cleaned, started := 0, 0
			helper := func(_ context.Context, spec container.ContainerSpec, argv []string, in io.Reader, _ io.Writer) error {
				if len(spec.Mounts) != 1 || spec.Mounts[0].Volume != "provider-home" || spec.Mounts[0].Target != "/home/agent" || spec.Mounts[0].NoCopy || spec.Image != testProfile().ImageID {
					t.Fatalf("credential helper resolved other inputs %+v", spec)
				}
				if argv[0] == "/bin/rm" {
					cleaned++
					if mode == "cleanup-failure" {
						return errors.New("cleanup refused")
					}
					return nil
				}
				installed, _ = io.ReadAll(in)
				var copy runtimeLogin
				if bytes.Contains(installed, []byte("dedicated-refresh-token")) || json.Unmarshal(installed, &copy) != nil || copy.Tokens.Refresh != "" || copy.Tokens.Access != "dedicated-access-token" {
					t.Fatal("bad runtime login copy")
				}
				if mode == "install-failure" {
					return errors.New("partial install with dedicated-access-token")
				}
				return nil
			}
			a, err := NewAgentsWithRuntime(func(context.Context, container.ContainerSpec) (RuntimeSession, error) {
				started++
				return rt, nil
			}, helper, map[string]Profile{"codex": testProfile()}, func(_ context.Context, ref string) (RuntimeCredential, error) {
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
			switch mode {
			case "observed-exit":
				if err != nil || !res.Stopped || cleaned != 1 || len(rt.start.Env) != 1 || rt.start.Env[0] != "CODEX_HOME=/home/agent/.codex" || strings.Contains(strings.Join(rt.start.Argv, " "), "env_key") {
					t.Fatalf("%+v %v cleaned=%d start=%+v", res, err, cleaned, rt.start)
				}
			case "uncertain-stop":
				if err == nil || res.Stopped || cleaned != 0 {
					t.Fatalf("uncertain stop claimed credential removal: %+v %v cleaned=%d", res, err, cleaned)
				}
			case "cleanup-failure":
				if err == nil || !res.Stopped || !strings.Contains(res.Reason, "cleanup incomplete") {
					t.Fatalf("cleanup failure hidden: %+v %v", res, err)
				}
			case "install-failure":
				if err == nil || strings.Contains(err.Error(), "dedicated-access-token") || started != 0 || cleaned != 1 || !res.Stopped {
					t.Fatalf("partial installation not cleaned: %+v %v started=%d cleaned=%d", res, err, started, cleaned)
				}
			}
		})
	}
}

func TestCredentialModesFailClosedWithoutFallback(t *testing.T) {
	for _, c := range []RuntimeCredential{{APIKey: "ambient-mode-not-admitted"}, {Mode: "api-key", APIKey: "key", RuntimeLogin: json.RawMessage(dedicatedLogin)}, {Mode: "desktop-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}, {Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}} {
		turn := &turnRun{agents: &Agents{}}
		if _, err := turn.credential(context.Background(), c); err == nil {
			t.Fatalf("unsupported credential accepted %+v", c.Mode)
		}
	}
}

func TestRedactionCoversEveryLoginSecret(t *testing.T) {
	turn := &turnRun{agents: &Agents{helper: func(context.Context, container.ContainerSpec, []string, io.Reader, io.Writer) error { return nil }}, turn: run.Turn{Home: "home", Workspace: "work"}, profile: testProfile()}
	if _, err := turn.credential(context.Background(), RuntimeCredential{Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}); err != nil {
		t.Fatal(err)
	}
	text := turn.redact([]byte(`dedicated-id-token dedicated-access-token dedicated-refresh-token runtime-account`))
	if string(text) != `[REDACTED] [REDACTED] [REDACTED] runtime-account` {
		t.Fatalf("%s", text)
	}
}
