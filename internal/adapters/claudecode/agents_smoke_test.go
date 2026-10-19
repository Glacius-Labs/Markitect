package claudecode

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Runtime activation requires command-line flags. No environment variable can
// enable this smoke in ordinary CI.
var turnSmokeFlags = struct {
	enable                         *bool
	engine, image, profile, tokens *string
}{
	flag.Bool("markitect-claude-turn-smoke", false, "explicit authority to run one real Claude turn"),
	flag.String("markitect-claude-smoke-engine", "", "absolute engine executable"),
	flag.String("markitect-claude-smoke-image", "", "immutable sha256 image ID"),
	flag.String("markitect-claude-smoke-profile", "", "absolute closed SessionProfile JSON file"),
	flag.String("markitect-claude-smoke-token-env", "", "name of the environment variable holding a dedicated runtime Claude token"),
}

// TestClaudeTurnSmoke runs one real read-only turn through NewAgents: a volume
// holds one file, the agent reads it and reports its content through the Host
// tool. It runs only with its explicit flags and is not Apply or release proof.
func TestClaudeTurnSmoke(t *testing.T) {
	if !*turnSmokeFlags.enable {
		t.Skip("UNEXECUTED: explicit turn smoke flag required")
	}
	if !filepath.IsAbs(*turnSmokeFlags.engine) || !filepath.IsAbs(*turnSmokeFlags.profile) || !imageName.MatchString(*turnSmokeFlags.image) || *turnSmokeFlags.tokens == "" {
		t.Fatal("absolute engine and profile, an immutable image and a token variable name are required")
	}
	raw, err := os.ReadFile(*turnSmokeFlags.profile)
	if err != nil || len(raw) > 1<<20 {
		t.Fatal("bounded profile source unavailable")
	}
	var session SessionProfile
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&session) != nil || !slices.Contains(session.BuiltinTools, "Read") {
		t.Fatal("the profile must be a closed SessionProfile that offers Read")
	}
	token := os.Getenv(*turnSmokeFlags.tokens)
	if token == "" {
		t.Fatal("token variable is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	engine, err := container.NewEngine(*turnSmokeFlags.engine)
	if err != nil {
		t.Fatal(err)
	}
	if image, err := engine.ImageID(ctx, *turnSmokeFlags.image); err != nil || image != *turnSmokeFlags.image {
		t.Fatal("immutable image identity not observed")
	}
	var id [8]byte
	_, _ = rand.Read(id[:])
	name := "claude-turn-smoke-" + hex.EncodeToString(id[:])
	labels := map[string]string{"markitect.store": session.Store}
	volume := func(suffix string) string {
		v := name + "-" + suffix
		if _, err := engine.CreateVolumeExclusive(ctx, v, labels); err != nil {
			t.Fatal("exclusive smoke volume creation failed")
		}
		t.Cleanup(func() {
			clean, done := context.WithTimeout(context.Background(), 15*time.Second)
			defer done()
			if err := engine.RemoveVolume(clean, v); err != nil {
				t.Error("smoke volume cleanup incomplete")
			}
		})
		return v
	}
	workspace, home := volume("work"), volume("home")
	content := "smoke-content-" + hex.EncodeToString(id[:])
	seed := container.ContainerSpec{Name: name + "-seed", Image: *turnSmokeFlags.image, Labels: labels, User: "1000:1000", Workdir: "/work",
		Mounts: []container.Mount{{Volume: workspace, Target: "/work"}},
		CPUs:   session.CPUs, MemoryBytes: session.MemoryBytes, Pids: session.Pids, TmpBytes: session.TmpBytes, Network: session.Network}
	// Initial copy-up gives the volume the image's uid-1000 directory ownership.
	if err := engine.RunHelper(ctx, seed, []string{"/bin/sh", "-c", "test -z \"$(find /work -mindepth 1 -print -quit)\" && cat > /work/smoke.txt"}, strings.NewReader(content+"\n"), io.Discard); err != nil {
		t.Fatal("smoke file materialization failed")
	}

	credentialRef := "dedicated-runtime-token-env:" + *turnSmokeFlags.tokens
	agents, err := NewAgents(engine, map[string]Profile{"smoke": {ImageID: *turnSmokeFlags.image, CredentialRef: credentialRef, MaxWireBytes: 1 << 20, Session: session}},
		func(_ context.Context, ref string) (RuntimeCredential, error) {
			if ref != credentialRef {
				return RuntimeCredential{}, ErrSession
			}
			return RuntimeCredential{Token: token}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var reported []string
	turn := run.Turn{ID: name, Manager: "smoke", Kind: run.VerifyTurn, Profile: "smoke", Workspace: workspace, ReadOnly: true, Home: home,
		Prompt: "Read /work/smoke.txt with your file-reading tool. Do not change any file. Then call the report tool exactly once with the exact file contents as text.",
		Tools: []run.ToolSpec{{Name: "report", Description: "Report the exact contents of /work/smoke.txt to the Host.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"],"additionalProperties":false}`)}},
		Timeout: 4 * time.Minute, Transcript: filepath.Join(t.TempDir(), "turn.jsonl")}
	result, err := agents.Run(ctx, turn, func(_ context.Context, tool string, args json.RawMessage) run.ToolAnswer {
		mu.Lock()
		defer mu.Unlock()
		reported = append(reported, tool+" "+string(args))
		return run.ToolAnswer{Text: "recorded"}
	})
	if err != nil || !result.Stopped || result.Conversation == "" || result.Usage.OutputTokens == 0 {
		t.Fatalf("real turn incomplete: %v %+v", err, result)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(reported) == 0 || !strings.Contains(reported[0], content) {
		t.Fatalf("the agent did not report the file content: %q", reported)
	}
	if data, err := os.ReadFile(turn.Transcript); err != nil || bytes.Contains(data, []byte(token)) {
		t.Fatal("transcript missing or contains the token")
	}
	t.Logf("observed turn: conversation=%s usage=%+v reason=%s", result.Conversation, result.Usage, result.Reason)
}
