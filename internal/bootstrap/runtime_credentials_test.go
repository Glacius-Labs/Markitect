package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestRuntimeCredentialResolversDeferReadsAndFreezeMappings(t *testing.T) {
	path := filepath.Join(testkit.TempDir(t), "dedicated-key")
	sources := map[string]credentialSource{"selected": {Provider: "codex-app-server", Mode: "api-key", Path: path}}
	codex, claude, err := newCredentialResolvers(sources)
	if err != nil || codex == nil || claude == nil {
		t.Fatalf("construction must not read a missing credential: %v", err)
	}
	// Neither later map changes nor ambient credentials can supply a fallback.
	sources["selected"] = credentialSource{Provider: "claude-code", Mode: "dedicated-runtime-token", Path: filepath.Join(testkit.TempDir(t), "different")}
	t.Setenv("OPENAI_API_KEY", "fake-ambient-key")
	if got, err := codex(context.Background(), "selected"); !errors.Is(err, errCredentialRead) || got.APIKey != "" {
		t.Fatalf("missing mapped file must refuse: %v", err)
	}
	writeFakeCredential(t, path, []byte("fake-key-first"))
	got, err := codex(context.Background(), "selected")
	if err != nil || got.Mode != "api-key" || got.APIKey != "fake-key-first" || got.RuntimeLogin != nil {
		t.Fatalf("exact frozen mapping did not resolve: %v", err)
	}
	writeFakeCredential(t, path, []byte("fake-key-next"))
	got, err = codex(context.Background(), "selected")
	if err != nil || got.APIKey != "fake-key-next" {
		t.Fatalf("each invocation must read the selected source: %v", err)
	}
	for _, ref := range []string{"unknown", "selected ", "SELECTED"} {
		if got, err := codex(context.Background(), ref); !errors.Is(err, errRuntimeCredential) || got.APIKey != "" {
			t.Fatalf("inexact reference was admitted: %v", err)
		}
	}
	if got, err := claude(context.Background(), "selected"); !errors.Is(err, errRuntimeCredential) || got.Token != "" {
		t.Fatalf("cross-provider reference was admitted: %v", err)
	}
}

func TestRuntimeCredentialResolversPreserveLiteralLoginAndToken(t *testing.T) {
	dir := testkit.TempDir(t)
	loginPath, tokenPath := filepath.Join(dir, "login.json"), filepath.Join(dir, "claude-token")
	// This fake subscription login deliberately retains refresh: stripping the
	// session copy belongs to the Codex adapter, never the Orchestrator source resolver.
	login := []byte("{\n  \"auth_mode\":\"chatgpt\",\"OPENAI_API_KEY\":null,\"tokens\":{\"id_token\":\"fake-id\",\"access_token\":\"fake-access\",\"refresh_token\":\"fake-refresh\",\"account_id\":\"fake-account\"}\n}\n")
	writeFakeCredential(t, loginPath, login)
	writeFakeCredential(t, tokenPath, []byte("fake-dedicated-token"))
	codex, claude, err := newCredentialResolvers(map[string]credentialSource{
		"subscription": {Provider: "codex-app-server", Mode: "dedicated-runtime-login", Path: loginPath},
		"token":        {Provider: "claude-code", Mode: "dedicated-runtime-token", Path: tokenPath},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := codex(context.Background(), "subscription")
	if err != nil || got.Mode != "dedicated-runtime-login" || got.APIKey != "" || !bytes.Equal(got.RuntimeLogin, login) {
		t.Fatalf("subscription bytes changed: %v", err)
	}
	got.RuntimeLogin[0] = '!'
	again, err := codex(context.Background(), "subscription")
	if err != nil || !bytes.Equal(again.RuntimeLogin, login) {
		t.Fatalf("resolved login aliases another invocation or source: %v", err)
	}
	token, err := claude(context.Background(), "token")
	if err != nil || token.Token != "fake-dedicated-token" {
		t.Fatalf("literal token changed: %v", err)
	}
	stored, err := os.ReadFile(loginPath)
	if err != nil || !bytes.Equal(stored, login) {
		t.Fatalf("resolver changed the runtime login source: %v", err)
	}
}

func TestRuntimeCredentialMappingRefusesInvalidInputsWithoutIO(t *testing.T) {
	path := filepath.Join(testkit.TempDir(t), "never-created")
	for _, tc := range []struct {
		name, ref, provider, mode, path string
	}{
		{"empty reference", "", "codex-app-server", "api-key", path},
		{"reference whitespace", " key", "codex-app-server", "api-key", path},
		{"reference control", "key\n", "codex-app-server", "api-key", path},
		{"unknown provider", "key", "other", "api-key", path},
		{"ambient mode", "key", "codex-app-server", "environment", path},
		{"desktop mode", "key", "codex-app-server", "desktop-login", path},
		{"wrong Claude mode", "key", "claude-code", "api-key", path},
		{"wrong Codex mode", "key", "codex-app-server", "dedicated-runtime-token", path},
		{"relative path", "key", "codex-app-server", "api-key", "relative-key"},
		{"normalized path", "key", "codex-app-server", "api-key", filepath.Dir(path) + string(filepath.Separator) + "." + string(filepath.Separator) + "key"},
		{"alternate stream", "key", "codex-app-server", "api-key", path + ":stream"},
		{"path control", "key", "codex-app-server", "api-key", path + "\n"},
		{"path trailing dot", "key", "codex-app-server", "api-key", path + "."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codex, claude, err := newCredentialResolvers(map[string]credentialSource{tc.ref: {Provider: tc.provider, Mode: tc.mode, Path: tc.path}})
			if err == nil || codex != nil || claude != nil || strings.Contains(err.Error(), tc.path) {
				t.Fatal("invalid mapping produced a resolver or exposed its source")
			}
		})
	}
	if codex, claude, err := newCredentialResolvers(nil); err == nil || codex != nil || claude != nil {
		t.Fatal("missing explicit sources were admitted")
	}
}

func TestRuntimeCredentialReadBoundsTypesAndSafeErrors(t *testing.T) {
	dir := testkit.TempDir(t)
	for _, tc := range []struct {
		name string
		data []byte
		want error
	}{
		{"empty", nil, errCredentialLimit},
		{"oversized", bytes.Repeat([]byte{'x'}, maxRuntimeCredentialBytes+1), errCredentialLimit},
		{"newline", []byte("fake-secret\n"), errRuntimeCredential},
		{"nul", []byte("fake\x00secret"), errRuntimeCredential},
		{"invalid utf8", []byte{0xff}, errRuntimeCredential},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+"-credential")
			writeFakeCredential(t, path, tc.data)
			codex, claude, err := newCredentialResolvers(map[string]credentialSource{
				"key":   {Provider: "codex-app-server", Mode: "api-key", Path: path},
				"token": {Provider: "claude-code", Mode: "dedicated-runtime-token", Path: path},
			})
			if err != nil {
				t.Fatal(err)
			}
			key, keyErr := codex(context.Background(), "key")
			token, tokenErr := claude(context.Background(), "token")
			if !errors.Is(keyErr, tc.want) || !errors.Is(tokenErr, tc.want) || key.APIKey != "" || token.Token != "" {
				t.Fatalf("invalid file returned credential bytes: %v, %v", keyErr, tokenErr)
			}
			for _, err := range []error{keyErr, tokenErr} {
				if strings.Contains(err.Error(), path) || strings.Contains(err.Error(), "fake-secret") {
					t.Fatal("credential data escaped through an error")
				}
			}
		})
	}
	path := filepath.Join(dir, "exact-bound-login")
	data := bytes.Repeat([]byte{'x'}, maxRuntimeCredentialBytes)
	writeFakeCredential(t, path, data)
	codex, _, err := newCredentialResolvers(map[string]credentialSource{"login": {Provider: "codex-app-server", Mode: "dedicated-runtime-login", Path: path}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := codex(context.Background(), "login")
	if err != nil || !bytes.Equal(got.RuntimeLogin, data) {
		t.Fatalf("whole file at the exact bound was not preserved: %v", err)
	}
	// The adapter owns JSON login semantics; the Orchestrator only preserves bytes.
	writeFakeCredential(t, path, []byte("fake"))
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := codex(cancelled, "login"); !errors.Is(err, context.Canceled) || got.RuntimeLogin != nil {
		t.Fatalf("cancelled resolution returned bytes: %v", err)
	}
	if got, err := codex(nil, "login"); err == nil || got.RuntimeLogin != nil {
		t.Fatal("nil context resolution returned bytes")
	}
	directory, _, err := newCredentialResolvers(map[string]credentialSource{"dir": {Provider: "codex-app-server", Mode: "api-key", Path: dir}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := directory(context.Background(), "dir"); !errors.Is(err, errCredentialType) {
		t.Fatalf("directory was not refused as nonregular: %v", err)
	}
	missing := filepath.Join(dir, "fake-secret-path-must-not-leak")
	resolver, _, err := newCredentialResolvers(map[string]credentialSource{"missing": {Provider: "codex-app-server", Mode: "api-key", Path: missing}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver(context.Background(), "missing"); !errors.Is(err, errCredentialRead) || strings.Contains(err.Error(), "fake-secret-path") {
		t.Fatal("native file error was not safely classified")
	}
}

func writeFakeCredential(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
