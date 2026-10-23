package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestNormalLoginCopyClosureAndUncertainty(t *testing.T) {
	for _, kind := range []string{"observed exit", "uncertain stop", "cleanup failure", "partial installation"} {
		t.Run(kind, func(t *testing.T) {
			rt := newRuntime()
			switch kind {
			case "uncertain stop":
				rt.stopFail = true
			case "cleanup failure":
				rt.cleanupErr = errors.New("cleanup refused private-access")
			case "partial installation":
				rt.initializeErr = errors.New("setup refused private-access")
			}
			s, openErr := agentsFor(t, rt).Open(context.Background(), testSessionSpec(t))
			if s == nil {
				t.Fatal(openErr)
			}
			closeErr := s.Close(context.Background())
			if kind == "observed exit" && (openErr != nil || closeErr != nil) {
				t.Fatal(openErr, closeErr)
			}
			if kind != "observed exit" && openErr == nil && closeErr == nil {
				t.Fatal("uncertainty hidden")
			}
			for _, err := range []error{openErr, closeErr} {
				if err != nil && strings.Contains(err.Error(), "private-access") {
					t.Fatal("credential in error")
				}
			}
			var login runtimeLogin
			if json.Unmarshal(rt.installed, &login) != nil || login.Tokens.Refresh != "" || login.Tokens.Access != "private-access" || bytes.Contains(rt.installed, []byte("private-refresh")) {
				t.Fatal("bad login copy")
			}
			want := []string{"initialize", "start", "stop", "clear", "close"}
			if kind == "uncertain stop" {
				want = []string{"initialize", "start", "stop", "close"}
			}
			if kind == "partial installation" {
				want = []string{"initialize", "clear", "close"}
			}
			if !reflect.DeepEqual(rt.homeCalls, want) {
				t.Fatal(rt.homeCalls, want)
			}
			if rt.starts > 0 && (!reflect.DeepEqual(rt.start.Env, []string{"CODEX_HOME=/home/agent/.codex"}) || strings.Contains(strings.Join(rt.start.Argv, " "), "OPENAI_API_KEY")) {
				t.Fatal("login leaked to environment or API route retained")
			}
		})
	}
}
func TestLoginRefusesOtherModesMalformedOrMixedAuthority(t *testing.T) {
	for _, credential := range []RuntimeCredential{{}, {Mode: "api-key"}, {Mode: "desktop-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}, {Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(`{}`)}, {Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(strings.Replace(dedicatedLogin, `"auth_mode":"chatgpt"`, `"auth_mode":"apikey"`, 1))}, {Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(strings.Replace(dedicatedLogin, `"tokens":`, `"OPENAI_API_KEY":"key","tokens":`, 1))}, {Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(strings.Replace(dedicatedLogin, `"access_token":"private-access"`, `"access_token":""`, 1))}, {Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(strings.Replace(dedicatedLogin, `"account_id":"selected-account"`, `"account_id":""`, 1))}} {
		s := &nativeSession{}
		if _, err := s.credential(context.Background(), credential); err == nil {
			t.Fatalf("unsupported login admitted: %s", credential.Mode)
		}
	}
}
func TestLoginRedactsEveryTokenAndEscapedForm(t *testing.T) {
	s := &nativeSession{}
	if _, err := s.credential(context.Background(), RuntimeCredential{Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}); err != nil {
		t.Fatal(err)
	}
	if got := string(s.redact([]byte("private-id private-access private-refresh selected-account"))); got != "[REDACTED] [REDACTED] [REDACTED] selected-account" {
		t.Fatal(got)
	}
	s.secrets = append(s.secrets, []byte("with\"quote"))
	if got := string(s.redact([]byte(`with\"quote`))); got != "[REDACTED]" {
		t.Fatal(got)
	}
}
func TestNativeLaunchGuardsProjectLoadingAndNativeDelegation(t *testing.T) {
	argv := strings.Join(providerArgv(), " ")
	for _, required := range []string{"--disable multi_agent", "--disable multi_agent_v2", "agents.enabled=false", "project_doc_max_bytes=0", "project_doc_fallback_filenames=[]", `projects."/work".trust_level="untrusted"`, `model_provider="openai"`} {
		if !strings.Contains(argv, required) {
			t.Fatal("launch guard absent", required)
		}
	}
	if strings.Contains(argv, "env_key") || strings.Contains(argv, "api.openai.com") {
		t.Fatal("API-key route retained")
	}
}
