package codexappserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// These flags can only be supplied by an explicit finite native execution
// grant. Default tests never read stdin, contact an engine or start a provider.
var privateNative = struct {
	enable                                                            *bool
	engine, image, profile, profileSHA, metadataSHA, agentd, evidence *string
}{
	flag.Bool("markitect-private-native", false, "explicit finite authority for one Codex private-init session"),
	flag.String("markitect-private-engine", "", "absolute engine executable"),
	flag.String("markitect-private-image", "", "immutable runtime image ID"),
	flag.String("markitect-private-profile", "", "absolute nonsecret closed session JSON"),
	flag.String("markitect-private-profile-sha256", "", "whole session JSON SHA256"),
	flag.String("markitect-private-metadata-sha256", "", "whole runtime.env SHA256"),
	flag.String("markitect-private-agentd", "", "exact source-bound agentd build version"),
	flag.String("markitect-private-evidence", "", "new absolute private evidence directory"),
}

type privateNativePeer struct {
	*container.Session
	stages *[]string
}

func (p privateNativePeer) Initialize(ctx context.Context, req frame.Initialize) error {
	if err := p.Session.Initialize(ctx, req); err != nil {
		return err
	}
	*p.stages = append(*p.stages, "private-initialized")
	return nil
}
func (p privateNativePeer) StartProvider(ctx context.Context, req frame.Start) (frame.Started, error) {
	started, err := p.Session.StartProvider(ctx, req)
	if err == nil {
		*p.stages = append(*p.stages, "provider-started")
	}
	return started, err
}
func (p privateNativePeer) ClearPrivateHome(ctx context.Context) error {
	if err := p.Session.ClearPrivateHome(ctx); err != nil {
		return err
	}
	*p.stages = append(*p.stages, "private-cleared")
	return nil
}
func (p privateNativePeer) Close() error {
	if err := p.Session.Close(); err != nil {
		return err
	}
	*p.stages = append(*p.stages, "container-closed")
	return nil
}

// TestPrivateHomeNativeProbe tests one technical provider session, not a
// product Run. Its two messages test a native instruction canary, MCP and
// same-conversation recall. It cannot qualify precedence, refresh or V1 Apply.
func TestPrivateHomeNativeProbe(t *testing.T) {
	if !*privateNative.enable {
		t.Skip("UNEXECUTED: separate finite native grant required")
	}
	runPrivateNativeProbe(t, false)
}

// Both scenarios use the existing private initialization, image checks and
// resource cleanup. Neither scenario defines a product Run or inbox port.
func runPrivateNativeProbe(t *testing.T, inboxMode bool) {
	if *privateNative.enable && *inboxNative {
		t.Fatal("exactly one native probe scenario may be enabled")
	}
	if !filepath.IsAbs(*privateNative.engine) || !filepath.IsAbs(*privateNative.profile) || !filepath.IsAbs(*privateNative.evidence) || !imageName.MatchString(*privateNative.image) || *privateNative.agentd == "" {
		t.Fatal("explicit source/image/profile/evidence bindings required")
	}
	readBounded := func(path string) []byte {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal("nonsecret profile unavailable")
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, 1<<20+1))
		if err != nil || len(data) > 1<<20 {
			t.Fatal("nonsecret profile exceeds bound")
		}
		return data
	}
	raw := readBounded(*privateNative.profile)
	profileSum := sha256.Sum256(raw)
	if hex.EncodeToString(profileSum[:]) != *privateNative.profileSHA {
		t.Fatal("profile digest mismatch")
	}
	var profile SessionProfile
	if strictValue(raw, &profile) != nil || validateSessionProfile(profile) != nil || profile.CPUs > 1 || profile.MemoryBytes > 2<<30 || profile.Pids > 192 || profile.TmpBytes > 256<<20 {
		t.Fatal("invalid profile or probe resource ceiling exceeded")
	}
	if err := os.Mkdir(*privateNative.evidence, 0700); err != nil {
		t.Fatal("evidence directory must be new and private")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	// Private login bytes enter only stdin, never argv, environment or evidence.
	input := make(chan []byte, 1)
	go func() {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, frame.MaxPrivateAuthBytes+1))
		if err != nil {
			clear(data)
			data = nil
		}
		input <- data
	}()
	var login []byte
	select {
	case login = <-input:
	case <-ctx.Done():
		_ = os.Stdin.Close()
		t.Fatal("private stdin exceeded the execution deadline")
	}
	defer clear(login)
	credential := RuntimeCredential{Mode: "dedicated-runtime-login", RuntimeLogin: login}
	if _, err := (&turnRun{}).credential(ctx, credential); err != nil {
		t.Fatal("bounded account login refused")
	}
	engine, err := container.NewEngine(*privateNative.engine)
	if err != nil {
		t.Fatal("engine binding unavailable")
	}
	name := "private-native-" + smokeToken(t)
	profile.Store = name
	canary, recall := "instruction-"+smokeToken(t), "recall-"+smokeToken(t)
	profile.BaseInstructions = "This is a bounded technical transport probe. Do not modify the workspace, run shell commands, or read authentication files."
	profile.DeveloperInstructions = "On each message call native_receipt exactly once with instruction equal to " + canary + ". On a later message include the recall value from the preceding tool response."
	var inbox *nativeInboxProbe
	if inboxMode {
		inbox = newNativeInboxProbe(smokeToken(t), smokeToken(t))
		profile.DeveloperInstructions = inbox.instructions()
	}
	effective, _ := json.Marshal(profile)
	effectiveSum := sha256.Sum256(effective)
	stages := []string{}
	volumes := []container.VolumeObservation{}
	attemptedVolumes := []string{}
	var session run.AssignmentSession
	passed := false
	started := time.Now().UTC()
	defer func() {
		clean, done := context.WithTimeout(context.Background(), 90*time.Second)
		defer done()
		closed := false
		if session != nil {
			res, err := session.Close(clean)
			closed = err == nil && res.Stopped
			if !closed {
				t.Error("native session closure incomplete")
			}
		}
		containers, err := engine.Containers(clean, "markitect.store", name)
		if err != nil || len(containers) != 0 {
			passed = false
			t.Error("native container absence not established")
		}
		for _, v := range volumes {
			current, err := engine.InspectVolume(clean, v.Name)
			if err != nil || !current.Exists || !maps.Equal(current.Labels, v.Labels) {
				passed = false
				t.Error("native volume ownership not re-established")
				continue
			}
			mounts, err := engine.VolumeMounts(clean, v.Name)
			if err != nil || len(mounts) != 0 || engine.RemoveVolume(clean, v.Name) != nil {
				passed = false
				t.Error("native volume cleanup incomplete")
				continue
			}
			after, err := engine.InspectVolume(clean, v.Name)
			if err != nil || after.Exists {
				passed = false
				t.Error("native volume absence not established")
			}
		}
		status := "incomplete"
		var inboxEvidence *nativeInboxEvidence
		if inbox != nil {
			if passed && inbox.verifyTranscript(filepath.Join(*privateNative.evidence, "transcript.jsonl")) != nil {
				passed = false
				t.Error("native inbox transcript correlation failed")
			}
			inboxEvidence = inbox.evidence()
		}
		if !slices.Equal(stages, []string{"agentd-hello", "private-initialized", "provider-started", "private-cleared", "container-closed"}) {
			passed = false
		}
		if passed && closed && !t.Failed() {
			status = "passed"
		}
		record, _ := json.MarshalIndent(struct {
			Status, ProbeID, Image, Agentd, Provider, Model, Effort, ProfileSHA, EffectiveProfileSHA, MetadataSHA, Started, Ended string
			Protocol                                                                                                              int
			Stages                                                                                                                []string
			AttemptedVolumes                                                                                                      []string
			Volumes                                                                                                               []container.VolumeObservation
			Inbox                                                                                                                 *nativeInboxEvidence `json:",omitempty"`
		}{status, name, *privateNative.image, *privateNative.agentd, supportedCodexVersion, profile.Model, profile.Effort,
			*privateNative.profileSHA, hex.EncodeToString(effectiveSum[:]), *privateNative.metadataSHA,
			started.Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), frame.ProtocolVersion, stages, attemptedVolumes, volumes, inboxEvidence}, "", "  ")
		if err := os.WriteFile(filepath.Join(*privateNative.evidence, "probe.json"), record, 0600); err != nil {
			t.Error("native evidence write failed")
		}
	}()
	image, err := engine.ImageID(ctx, *privateNative.image)
	if err != nil || image != *privateNative.image {
		t.Fatal("immutable image binding differs")
	}
	metadata, err := engine.ReadImageFile(ctx, image, runtimeMetadataPath)
	metadataSum := sha256.Sum256(metadata)
	if err != nil || hex.EncodeToString(metadataSum[:]) != *privateNative.metadataSHA || !strings.Contains(string(metadata), "codex="+supportedCodexVersion+"\n") || !strings.Contains(string(metadata), "agentd="+*privateNative.agentd+"\n") {
		t.Fatal("runtime metadata differs from frozen build")
	}
	makeVolume := func(suffix string) string {
		attemptedVolumes = append(attemptedVolumes, name+suffix)
		v, err := engine.CreateVolumeExclusive(ctx, name+suffix, map[string]string{"markitect.store": name})
		if err != nil {
			t.Fatal("private volume creation failed; preserve any uncertain effects")
		}
		volumes = append(volumes, v)
		return v.Name
	}
	workspace, home := makeVolume("-work"), makeVolume("-home")
	starts := 0
	agents, err := NewAgentsWithRuntime(func(ctx context.Context, spec container.ContainerSpec) (RuntimeSession, error) {
		starts++
		if starts != 1 {
			return nil, ErrSession
		}
		peer, err := engine.Start(ctx, spec)
		if err != nil {
			return nil, err
		}
		if peer.Hello().Version != *privateNative.agentd {
			_ = peer.Close()
			return nil, ErrSession
		}
		stages = append(stages, "agentd-hello")
		return privateNativePeer{Session: peer, stages: &stages}, nil
	}, map[string]Profile{"probe": {ImageID: image, CredentialRef: "stdin-account", MaxWireBytes: maxProtocolMessage, Session: profile}},
		func(context.Context, string) (RuntimeCredential, error) { return credential, nil })
	if err != nil {
		t.Fatal("native adapter construction failed")
	}
	var mu sync.Mutex
	calls := 0
	valid := true
	tool := run.ToolSpec{Name: "native_receipt", Description: "Report the native instruction canary and previous tool recall value.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"instruction":{"type":"string"},"recall":{"type":"string"}},"required":["instruction"],"additionalProperties":false}`)}
	tools := []run.ToolSpec{tool}
	handler := func(_ context.Context, name string, args json.RawMessage) run.ToolAnswer {
		mu.Lock()
		defer mu.Unlock()
		calls++
		var receipt struct{ Instruction, Recall string }
		if calls > 2 || name != tool.Name || json.Unmarshal(args, &receipt) != nil || receipt.Instruction != canary || calls == 2 && receipt.Recall != recall {
			valid = false
			cancel()
			return run.ToolAnswer{Refused: true, Text: "probe aborted"}
		}
		return run.ToolAnswer{Text: `{"recall":"` + recall + `"}`}
	}
	if inbox != nil {
		tools = inbox.tools()
		handler = func(ctx context.Context, name string, args json.RawMessage) run.ToolAnswer {
			answer := inbox.handle(ctx, name, args)
			if answer.Refused {
				cancel()
			}
			return answer
		}
	}
	session, err = agents.OpenAssignment(ctx, run.AssignmentSpec{ID: name, Profile: "probe", Workspace: workspace, Home: home, Tools: tools, Transcript: filepath.Join(*privateNative.evidence, "transcript.jsonl")}, handler)
	if err != nil {
		t.Fatal("native private initialization or provider startup failed")
	}
	if inbox != nil {
		if err := inbox.exercise(ctx, session); err != nil {
			t.Fatal(err)
		}
		passed = true
		return
	}
	conversation := ""
	for i, message := range []string{"Follow the native probe instructions and report once.", "Report once again and recall the value from the previous tool response."} {
		result, err := session.Message(ctx, message, 45*time.Second)
		mu.Lock()
		count, ok := calls, valid
		mu.Unlock()
		if err != nil || !result.Completed || result.Conversation == "" || count != i+1 || !ok || i == 1 && result.Conversation != conversation {
			t.Fatal("native instruction/tool/continuity probe failed")
		}
		conversation = result.Conversation
	}
	passed = true
}
