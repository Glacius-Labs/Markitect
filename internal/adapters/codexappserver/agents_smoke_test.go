package codexappserver

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Runtime activation requires command-line flags. No environment variable can
// enable this smoke in ordinary CI.
var agentsSmokeFlags = struct {
	enable                                                            *bool
	engine, image, profile, profileDigest, metadataDigest, credential *string
}{
	flag.Bool("markitect-codex-session-smoke", false, "explicit finite authority to run three real adapter turns"),
	flag.String("markitect-codex-smoke-engine", "", "absolute engine executable"),
	flag.String("markitect-codex-smoke-image", "", "immutable sha256 image ID"),
	flag.String("markitect-codex-smoke-profile", "", "absolute closed SessionProfile JSON file"),
	flag.String("markitect-codex-smoke-profile-sha256", "", "expected whole profile SHA256, lowercase hex"),
	flag.String("markitect-codex-smoke-metadata-sha256", "", "expected whole runtime.env SHA256, lowercase hex"),
	flag.String("markitect-codex-smoke-login-volume", "", "explicit preexisting dedicated runtime login volume; never desktop login"),
}

func smokeToken(t *testing.T) string {
	t.Helper()
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatal("smoke random identity unavailable")
	}
	return hex.EncodeToString(raw[:])
}

// TestAgentsSmoke runs only with its explicit flag and source bindings, through
// the real NewAgents. It runs three read-only turns, one container each, under
// one five-minute bound: an Execute turn, an Execute resume in a new container
// with the same provider home, and a fresh Verify turn with another home.
func TestAgentsSmoke(t *testing.T) {
	if !*agentsSmokeFlags.enable {
		t.Skip("UNEXECUTED: explicit adapter smoke flag required")
	}
	if !filepath.IsAbs(*agentsSmokeFlags.engine) || !filepath.IsAbs(*agentsSmokeFlags.profile) ||
		!imageName.MatchString(*agentsSmokeFlags.image) || !volumeName.MatchString(*agentsSmokeFlags.credential) {
		t.Fatal("explicit absolute engine/profile, immutable image and login volume required")
	}
	raw, err := os.ReadFile(*agentsSmokeFlags.profile)
	if err != nil || len(raw) > 1<<20 {
		t.Fatal("bounded profile source unavailable")
	}
	profileSum := sha256.Sum256(raw)
	if hex.EncodeToString(profileSum[:]) != *agentsSmokeFlags.profileDigest {
		t.Fatal("profile source digest mismatch")
	}
	var session SessionProfile
	if strictValue(raw, &session) != nil || validateSessionProfile(session) != nil {
		t.Fatal("invalid closed SessionProfile source")
	}
	if session.Model != "gpt-6-luna" || session.Effort != "high" {
		t.Fatal("smoke requires explicitly selected Codex Luna/high profile")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	engine, err := container.NewEngine(*agentsSmokeFlags.engine)
	if err != nil {
		t.Fatal("explicit engine unavailable")
	}
	image, err := engine.ImageID(ctx, *agentsSmokeFlags.image)
	if err != nil || image != *agentsSmokeFlags.image {
		t.Fatal("immutable image identity not observed")
	}
	metadata, err := engine.ReadImageFile(ctx, image, "/opt/markitect/runtime.env")
	if err != nil {
		t.Fatal("image metadata unavailable")
	}
	metadataSum := sha256.Sum256(metadata)
	if hex.EncodeToString(metadataSum[:]) != *agentsSmokeFlags.metadataDigest || !bytes.Contains(metadata, []byte("codex=codex-cli 0.162.0\n")) {
		t.Fatal("image metadata binding or pinned provider version mismatch")
	}
	name := "codex-smoke-" + smokeToken(t)
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
	workspace, managerHome, verifierHome := volume("work"), volume("manager"), volume("verifier")
	helperSpec := container.ContainerSpec{Name: name + "-seed", Image: image, Labels: labels, User: "1000:1000",
		Workdir: "/work", Mounts: []container.Mount{{Volume: workspace, Target: "/work"}},
		CPUs: session.CPUs, MemoryBytes: session.MemoryBytes, Pids: session.Pids, TmpBytes: session.TmpBytes, Network: session.Network}
	artifact := "artifact-" + smokeToken(t)
	// Initial copy-up supplies the image's uid-1000 directory ownership. Refuse
	// any image-populated entry before writing the exact one-file workspace.
	if err := engine.RunHelper(ctx, helperSpec, []string{"/bin/sh", "-c", "test -z \"$(find /work -mindepth 1 -print -quit)\" && cat > /work/smoke.txt"}, strings.NewReader(artifact+"\n"), io.Discard); err != nil {
		t.Fatal("smoke artifact materialization failed")
	}
	readback := func() {
		var out bytes.Buffer
		spec := helperSpec
		spec.Name = name + "-readback"
		spec.Mounts = []container.Mount{{Volume: workspace, Target: "/work", ReadOnly: true, NoCopy: true}}
		if err := engine.RunHelper(ctx, spec, []string{"/bin/sh", "-c", "test \"$(find /work -mindepth 1 | wc -l)\" -eq 1 && test -f /work/smoke.txt && test ! -L /work/smoke.txt && cat /work/smoke.txt"}, nil, &out); err != nil || out.String() != artifact+"\n" {
			t.Fatal("actual workspace readback differs from the seeded file")
		}
	}
	readback()
	credentialRef := "dedicated-runtime-volume:" + *agentsSmokeFlags.credential
	resolver := func(resolveCtx context.Context, ref string) (RuntimeCredential, error) {
		if ref != credentialRef {
			return RuntimeCredential{}, ErrSession
		}
		spec := helperSpec
		spec.Name = name + "-read-login"
		spec.Workdir = "/auth"
		spec.Mounts = []container.Mount{{Volume: *agentsSmokeFlags.credential, Target: "/auth", ReadOnly: true, NoCopy: true}}
		var login bytes.Buffer
		if err := engine.RunHelper(resolveCtx, spec, []string{"/usr/bin/head", "-c", "1048577", "/auth/auth.json"}, nil, &login); err != nil || login.Len() > 1<<20 {
			return RuntimeCredential{}, ErrSession
		}
		return RuntimeCredential{Mode: "dedicated-runtime-login", RuntimeLogin: login.Bytes()}, nil
	}
	agents, err := NewAgents(engine, map[string]Profile{"smoke": {ImageID: image, CredentialRef: credentialRef, MaxWireBytes: 1 << 20, Session: session}}, resolver)
	if err != nil {
		t.Fatal("actual adapter construction failed")
	}
	tools := []run.ToolSpec{{Name: "report", Description: "Report the turn's outcome. Call it exactly once.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"],"additionalProperties":false}`)}}
	conversation := "conversation-" + smokeToken(t)
	oneTurn := func(id string, kind run.TurnKind, home, resume string, recall bool) string {
		var mu sync.Mutex
		var summaries []string
		handler := func(_ context.Context, tool string, args json.RawMessage) run.ToolAnswer {
			var a struct {
				Summary string `json:"summary"`
			}
			if tool != "report" || json.Unmarshal(args, &a) != nil {
				return run.ToolAnswer{Text: "unexpected call", Refused: true}
			}
			mu.Lock()
			summaries = append(summaries, a.Summary)
			mu.Unlock()
			return run.ToolAnswer{Text: `{"smokeConversationToken":"` + conversation + `"}`}
		}
		prompt := "Read /work/smoke.txt with your native file-reading tool. Do not change any file. Call report exactly once with a summary that contains the exact file contents."
		if recall {
			prompt += " Also include in the summary the exact smokeConversationToken returned by the previous turn's report answer in this conversation. It is deliberately absent from this prompt and the workspace."
		}
		res, err := agents.Run(ctx, run.Turn{ID: name + "-" + id, Manager: "smoke", Kind: kind, Profile: "smoke", Workspace: workspace, ReadOnly: true,
			Home: home, Resume: resume, Prompt: prompt, Tools: tools, Timeout: 90 * time.Second}, handler)
		if err != nil || !res.Stopped || res.Conversation == "" {
			t.Fatalf("actual turn %s failed or did not observe the end of its container: %v stopped=%v", id, err, res.Stopped)
		}
		if len(summaries) != 1 || !strings.Contains(summaries[0], artifact) || recall && !strings.Contains(summaries[0], conversation) {
			t.Fatalf("turn %s did not report the expected file contents or recalled token through the Orchestrator tool", id)
		}
		if resume != "" && res.Conversation != resume {
			t.Fatal("new-container resume changed the actual conversation identity")
		}
		readback()
		t.Logf("observed %s: image=%s profile=%x metadata=%x usage=%+v", id, image, profileSum, metadataSum, res.Usage)
		return res.Conversation
	}
	first := oneTurn("manager-first", run.ExecuteTurn, managerHome, "", false)
	resumed := oneTurn("manager-resume", run.ExecuteTurn, managerHome, first, true)
	verifier := oneTurn("verifier-fresh", run.VerifyTurn, verifierHome, "", false)
	if resumed != first || verifier == first {
		t.Fatal("Execute continuity or fresh Verify identity not distinguished")
	}
}
