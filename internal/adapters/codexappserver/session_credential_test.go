package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

const dedicatedLogin = `{"auth_mode":"chatgpt","OPENAI_API_KEY":null,"tokens":{"id_token":"dedicated-id-token","access_token":"dedicated-access-token","refresh_token":"dedicated-refresh-token","account_id":"runtime-account"},"last_refresh":"2026-10-17T12:00:00Z"}`

func TestOpenCredentialFailureBeforeRuntimeIsCertainlyStopped(t *testing.T) {
	for _, mode := range []string{"resolver-error", "invalid-credential", "container-start-error"} {
		t.Run(mode, func(t *testing.T) {
			channel := &fakeChannel{}
			starts, helpers := 0, 0
			s, err := NewSessionsWithRuntime(func(context.Context, container.ContainerSpec) (RuntimeSession, error) {
				starts++
				return nil, errors.New("container creation/start result unknown")
			}, func(context.Context, container.ContainerSpec, []string, io.Reader, io.Writer) error {
				helpers++
				return errors.New("unexpected helper")
			}, func(context.Context, string) (RuntimeCredential, error) {
				if mode == "resolver-error" {
					return RuntimeCredential{}, errors.New("credential resolver failed")
				}
				if mode == "invalid-credential" {
					return RuntimeCredential{Mode: "api-key"}, nil
				}
				return RuntimeCredential{Mode: "api-key", APIKey: "synthetic-key"}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			a, _ := sessionAdmissions(t)
			opened, openErr := s.Open(context.Background(), a, channel)
			if openErr == nil || opened.ProviderSession != "" || opened.Ready || opened.Wire.Identity != (run.WireIdentity{}) || opened.Wire.Journal.Valid() || helpers != 0 || len(channel.sends) != 0 || len(channel.preparations) != 0 {
				t.Fatalf("unexpected start evidence: %+v %v starts=%d helpers=%d", opened, openErr, starts, helpers)
			}
			uncertain := mode == "container-start-error"
			if opened.Uncertain != uncertain || uncertain && (starts != 1 || len(channel.bindings) != 1 || !channel.bindings[0].Valid()) || !uncertain && (starts != 0 || len(channel.bindings) != 0) {
				t.Fatalf("wrong certainty/binding: %+v starts=%d bindings=%+v", opened, starts, channel.bindings)
			}
			if _, retryErr := s.Open(context.Background(), a, channel); retryErr == nil {
				t.Fatal("failed Open could be retried")
			}
			stopped, stopErr := s.Stop(context.Background(), run.StopRequest{Session: a.Plan.ID, Timeout: time.Second}, channel)
			if stopped.Terminated == uncertain || (stopErr != nil) != uncertain || len(channel.stops) != 1 || channel.stops[0] != stopped {
				t.Fatalf("wrong stop observation: %+v %v callbacks=%+v", stopped, stopErr, channel.stops)
			}
			if len(channel.steps) != 0 || len(channel.calls) != 0 || helpers != 0 {
				t.Fatal("failed Open dispatched work or credential helper")
			}
		})
	}
}

func TestDedicatedRuntimeLoginCopyCleanupAndUncertainty(t *testing.T) {
	for _, mode := range []string{"observed-exit", "uncertain-stop", "cleanup-failure", "install-failure"} {
		t.Run(mode, func(t *testing.T) {
			channel := &fakeChannel{}
			runtime := peer(channel)
			runtime.stopFail = mode == "uncertain-stop"
			var installed []byte
			cleaned := 0
			helper := func(_ context.Context, spec container.ContainerSpec, argv []string, in io.Reader, _ io.Writer) error {
				if len(spec.Mounts) != 1 || spec.Mounts[0].Volume != "provider-home" || spec.Mounts[0].Target != "/home/agent" || spec.Mounts[0].NoCopy || spec.Image != "sha256:"+strings.Repeat("a", 64) {
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
				if bytes.Contains(installed, []byte("dedicated-refresh-token")) {
					t.Fatal("refresh token installed")
				}
				var copy runtimeLogin
				if json.Unmarshal(installed, &copy) != nil || copy.Tokens.Refresh != "" || copy.Tokens.Access != "dedicated-access-token" {
					t.Fatal("bad runtime login copy")
				}
				if mode == "install-failure" {
					return errors.New("partial install")
				}
				return nil
			}
			s, err := NewSessionsWithRuntime(func(context.Context, container.ContainerSpec) (RuntimeSession, error) { return runtime, nil }, helper, func(_ context.Context, ref string) (RuntimeCredential, error) {
				if ref != "exact-credential" {
					t.Fatal("credential reference changed")
				}
				return RuntimeCredential{Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = runtime.Close() })
			a, _ := sessionAdmissions(t)
			opened, openErr := s.Open(context.Background(), a, channel)
			if mode == "install-failure" {
				if openErr == nil || !opened.Uncertain || len(runtime.writes) > 0 {
					t.Fatal("install failure started provider")
				}
			} else if openErr != nil {
				t.Fatal(openErr)
			}
			if mode != "install-failure" && (len(runtime.start.Env) != 1 || strings.Contains(strings.Join(runtime.start.Argv, " "), "env_key")) {
				t.Fatalf("subscription received API-key route %+v", runtime.start)
			}
			e, _ := s.entry(a.Plan.ID)
			wire := e.wire([]byte(dedicatedLogin))
			for _, secret := range []string{"dedicated-id-token", "dedicated-access-token", "dedicated-refresh-token"} {
				if bytes.Contains(wire.Journal.Bytes(), []byte(secret)) {
					t.Fatal("credential entered journal")
				}
			}
			stopped, stopErr := s.Stop(context.Background(), run.StopRequest{Session: a.Plan.ID, ProviderSession: opened.ProviderSession, Timeout: time.Second}, channel)
			switch mode {
			case "observed-exit":
				if stopErr != nil || !stopped.Terminated || cleaned != 1 || e.loginInstalled {
					t.Fatalf("cleanup result %+v %v cleaned=%d", stopped, stopErr, cleaned)
				}
			case "uncertain-stop":
				if stopErr == nil || stopped.Terminated || cleaned != 0 || !e.loginInstalled {
					t.Fatal("uncertain stop claimed credential removal")
				}
			case "cleanup-failure":
				if stopErr == nil || !stopped.Terminated || !e.loginInstalled || !strings.Contains(stopped.Reason, "cleanup incomplete") {
					t.Fatal("cleanup failure hidden")
				}
			case "install-failure":
				if cleaned != 1 || e.loginInstalled || stopErr == nil || stopped.Terminated {
					t.Fatal("partial installation not cleaned with uncertainty retained")
				}
			}
		})
	}
}

func TestSessionCredentialModesFailClosedWithoutFallback(t *testing.T) {
	for _, c := range []RuntimeCredential{{APIKey: "ambient-mode-not-admitted"}, {Mode: "api-key", APIKey: "key", RuntimeLogin: json.RawMessage(dedicatedLogin)}, {Mode: "desktop-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}, {Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}} {
		e := &sessionEntry{}
		if _, err := e.credential(context.Background(), c, nil); err == nil {
			t.Fatalf("unsupported credential accepted %+v", c.Mode)
		}
	}
}
