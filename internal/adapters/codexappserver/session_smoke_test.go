package codexappserver

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Runtime activation requires command-line flags. No environment variable can
// enable this smoke in ordinary CI. Source preparation is not execution proof.
var sessionSmokeFlags = struct {
	enable                                                            *bool
	engine, image, profile, profileDigest, metadataDigest, credential *string
}{
	flag.Bool("markitect-codex-session-smoke", false, "explicit finite authority to run three real adapter steps"),
	flag.String("markitect-codex-smoke-engine", "", "absolute engine executable"),
	flag.String("markitect-codex-smoke-image", "", "immutable sha256 image ID"),
	flag.String("markitect-codex-smoke-profile", "", "absolute closed SessionProfile JSON file"),
	flag.String("markitect-codex-smoke-profile-sha256", "", "expected whole profile SHA256, lowercase hex"),
	flag.String("markitect-codex-smoke-metadata-sha256", "", "expected whole runtime.env SHA256, lowercase hex"),
	flag.String("markitect-codex-smoke-login-volume", "", "explicit preexisting dedicated runtime login volume; never desktop login"),
}

// controlledSessionIO is smoke scaffolding. Receipts acknowledge this in-memory
// observer only: no durable journal, Coordinator admission, Core integration,
// task acceptance, replay authority or publication is established.
type controlledSessionIO struct {
	step                         run.StepID
	phase                        run.Phase
	artifact, continuity, answer string
	next                         uint64
	bound                        run.ProviderSessionID
	preparations                 map[run.MessageID]run.TransportPreparation
	observed                     map[run.MessageID]bool
	answered                     map[run.ProviderCallID]bool
	terminal, stopped, outcome   bool
}

func (c *controlledSessionIO) ack() run.AppendReceipt {
	c.next++
	return run.AppendReceipt{BatchID: fmt.Sprintf("smoke-scaffolding-%d", c.next), Revision: c.next}
}
func (c *controlledSessionIO) Ready(_ context.Context, o run.SessionReadiness) (run.AppendReceipt, error) {
	if !o.Wire.Journal.Valid() {
		return run.AppendReceipt{}, run.ErrInvalid
	}
	return c.ack(), nil
}
func (c *controlledSessionIO) Bind(_ context.Context, o run.SessionObservation) (run.AppendReceipt, error) {
	if !o.Valid() || o.Ready || o.Uncertain || c.bound != "" {
		return run.AppendReceipt{}, run.ErrInvalid
	}
	c.bound = o.ProviderSession
	return c.ack(), nil
}
func (c *controlledSessionIO) PrepareSend(_ context.Context, p run.TransportPreparation) (run.SendPermit, error) {
	if p.Message == "" || !p.Wire.Journal.Valid() || p.Wire.Redacted ||
		p.Wire.Identity.ContentID != p.Wire.Journal.Ref().ContentID ||
		p.Wire.Identity.Size != p.Wire.Journal.Ref().Size {
		return run.SendPermit{}, run.ErrInvalid
	}
	if _, exists := c.preparations[p.Message]; exists {
		return run.SendPermit{}, run.ErrConflict
	}
	if p.Step != "" && p.Step != c.step || p.Call != "" && !c.answered[p.Call] {
		return run.SendPermit{}, run.ErrInvalid
	}
	c.preparations[p.Message] = p
	return run.SendPermit{Receipt: c.ack(), Identity: p.Wire.Identity, Dispatch: true}, nil
}
func (c *controlledSessionIO) ObserveSend(_ context.Context, o run.TransportObservation) (run.AppendReceipt, error) {
	if _, exists := c.preparations[o.Message]; !exists || c.observed[o.Message] || o.Outcome != run.Sent {
		return run.AppendReceipt{}, run.ErrObservationLimit
	}
	c.observed[o.Message] = true
	return c.ack(), nil
}
func (c *controlledSessionIO) Call(_ context.Context, call run.ToolCall) (run.ToolResult, error) {
	if c.bound == "" || call.Step != c.step || call.Call == "" || c.answered[call.Call] ||
		!run.PhaseTool(c.phase, call.Tool) || !call.Input.Journal.Valid() {
		return run.ToolResult{}, run.ErrInvalid
	}
	outcome := run.ToolOutcome{Status: run.ToolRefused, Reason: "This bounded smoke supports only its requested outcome tool."}
	expected := run.ToolSubmit
	if c.phase == run.PhaseReviewerCandidate {
		expected = run.ToolSubmitReview
	}
	if call.Tool == expected {
		wireCall, err := frame.DecodeToolCall(call.Input.Journal.Bytes())
		if err != nil || wireCall.Name != call.Tool {
			return run.ToolResult{}, run.ErrInvalid
		}
		var summary string
		if c.phase == run.PhaseReviewerCandidate {
			var review struct {
				Verdict     string            `json:"verdict"`
				Summary     string            `json:"summary"`
				Findings    []json.RawMessage `json:"findings"`
				Resolutions []json.RawMessage `json:"resolutions"`
				Evidence    []json.RawMessage `json:"evidence"`
				Cases       []json.RawMessage `json:"cases"`
			}
			if strictValue(wireCall.Arguments, &review) != nil || review.Verdict != "pass" ||
				review.Findings == nil || len(review.Findings) != 0 || review.Resolutions == nil || len(review.Resolutions) != 0 ||
				review.Evidence == nil || len(review.Evidence) != 0 || review.Cases == nil || len(review.Cases) != 0 {
				return run.ToolResult{}, run.ErrInvalid
			}
			summary = review.Summary
		} else {
			var submission struct {
				Purpose      string            `json:"purpose"`
				Scope        string            `json:"scope"`
				Summary      string            `json:"summary"`
				Assessments  []json.RawMessage `json:"assessments"`
				Dependencies []string          `json:"dependencies"`
			}
			if strictValue(wireCall.Arguments, &submission) != nil || submission.Purpose != "local" || submission.Scope != "complete" ||
				submission.Assessments == nil || len(submission.Assessments) != 0 || submission.Dependencies == nil || len(submission.Dependencies) != 0 {
				return run.ToolResult{}, run.ErrInvalid
			}
			summary = submission.Summary
		}
		if c.outcome || !strings.Contains(summary, c.artifact) ||
			c.continuity != "" && !strings.Contains(summary, c.continuity) {
			return run.ToolResult{}, run.ErrInvalid
		}
		c.outcome = true
		data, _ := json.Marshal(map[string]string{"smokeConversationToken": c.answer})
		outcome = run.ToolOutcome{Status: run.ToolOK, Result: data}
	}
	data, err := json.Marshal(outcome)
	if err != nil {
		return run.ToolResult{}, err
	}
	output, err := run.NewPayload("application/vnd.markitect.tool-outcome+json", "utf-8", data)
	if err != nil {
		return run.ToolResult{}, err
	}
	c.answered[call.Call] = true
	return run.ToolResult{Receipt: c.ack(), Outcome: outcome, Output: output}, nil
}
func (c *controlledSessionIO) ObserveStep(_ context.Context, o run.StepObservation) (run.AppendReceipt, error) {
	if o.Step != c.step || o.ProviderStep == "" || o.Uncertain || !o.Wire.Journal.Valid() || o.Proposed.Valid() {
		return run.AppendReceipt{}, run.ErrObservationLimit
	}
	if o.Terminal {
		if !c.outcome || len(c.preparations) != len(c.observed) {
			return run.AppendReceipt{}, run.ErrObservationLimit
		}
		c.terminal = true
	}
	return c.ack(), nil
}
func (c *controlledSessionIO) ObserveStop(_ context.Context, o run.StopObservation) (run.AppendReceipt, error) {
	if !o.Terminated || c.stopped {
		return run.AppendReceipt{}, run.ErrObservationLimit
	}
	c.stopped = true
	return c.ack(), nil
}

func TestSessionSmokeObserverDecodesActualToolEnvelope(t *testing.T) {
	for _, tc := range []struct {
		phase           run.Phase
		tool, arguments string
	}{
		{run.PhaseWork, run.ToolSubmit, `{"purpose":"local","scope":"complete","summary":"artifact-token conversation-token","assessments":[],"dependencies":[]}`},
		{run.PhaseReviewerCandidate, run.ToolSubmitReview, `{"verdict":"pass","summary":"artifact-token conversation-token","findings":[],"resolutions":[],"evidence":[],"cases":[]}`},
	} {
		t.Run(string(tc.phase), func(t *testing.T) {
			channel := &controlledSessionIO{step: "step", phase: tc.phase, artifact: "artifact-token", continuity: "conversation-token",
				bound: "observed-provider", answered: map[run.ProviderCallID]bool{}}
			call := run.ToolCall{Step: "step", Call: "provider-call", Tool: tc.tool}
			// A bare submission, including submission-shaped final text, is no observed tool
			// envelope and must never satisfy this smoke.
			call.Input.Journal = sessionSmokePayload(t, "application/vnd.markitect.agentd-tool-call", []byte(tc.arguments))
			if _, err := channel.Call(context.Background(), call); err == nil || channel.outcome {
				t.Fatal("bare outcome bytes accepted without an actual tool envelope")
			}
			raw, err := frame.EncodeToolCall(frame.ToolCall{ID: 1, Name: tc.tool, Arguments: json.RawMessage(tc.arguments)})
			if err != nil {
				t.Fatal(err)
			}
			call.Input.Journal = sessionSmokePayload(t, "application/vnd.markitect.agentd-tool-call", raw)
			got, err := channel.Call(context.Background(), call)
			if err != nil || got.Receipt.BatchID == "" || got.Outcome.Status != run.ToolOK || !channel.outcome {
				t.Fatal("actual admitted tool envelope did not produce acknowledged smoke outcome")
			}
		})
	}
}

func sessionSmokeToken(t *testing.T) string {
	t.Helper()
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatal("smoke random identity unavailable")
	}
	return hex.EncodeToString(raw[:])
}
func sessionSmokePayload(t *testing.T, media string, data []byte) run.Payload {
	t.Helper()
	p, err := run.NewPayload(media, "utf-8", data)
	if err != nil {
		t.Fatal("invalid smoke payload")
	}
	return p
}

// TestAdmittedSessionAdapterSmoke runs only with its explicit flag and explicit
// source bindings. It uses NewSessions, never NewSessionsWithRuntime or a codec
// substitute. It is three steps, one per container, under one five-minute bound.
func TestAdmittedSessionAdapterSmoke(t *testing.T) {
	if !*sessionSmokeFlags.enable {
		t.Skip("UNEXECUTED: explicit adapter smoke flag required")
	}
	if !filepath.IsAbs(*sessionSmokeFlags.engine) || !filepath.IsAbs(*sessionSmokeFlags.profile) ||
		!imageName.MatchString(*sessionSmokeFlags.image) || *sessionSmokeFlags.credential == "" {
		t.Fatal("explicit absolute engine/profile, immutable image and credential reference required")
	}
	raw, err := os.ReadFile(*sessionSmokeFlags.profile)
	if err != nil || len(raw) > 1<<20 {
		t.Fatal("bounded profile source unavailable")
	}
	profileSum := sha256.Sum256(raw)
	if hex.EncodeToString(profileSum[:]) != *sessionSmokeFlags.profileDigest {
		t.Fatal("profile source digest mismatch")
	}
	var profile SessionProfile
	if strictValue(raw, &profile) != nil {
		t.Fatal("invalid closed SessionProfile source")
	}
	if profile.Model != "gpt-6-luna" || profile.Effort != "high" {
		t.Fatal("smoke requires explicitly selected Codex Luna/high profile")
	}
	profilePayload := sessionSmokePayload(t, "application/vnd.markitect.codex-session-profile+json", raw)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	engine, err := container.NewEngine(*sessionSmokeFlags.engine)
	if err != nil {
		t.Fatal("explicit engine unavailable")
	}
	image, err := engine.ImageID(ctx, *sessionSmokeFlags.image)
	if err != nil || image != *sessionSmokeFlags.image {
		t.Fatal("immutable image identity not observed")
	}
	metadata, err := engine.ReadImageFile(ctx, image, "/opt/markitect/runtime.env")
	if err != nil {
		t.Fatal("image metadata unavailable")
	}
	metadataSum := sha256.Sum256(metadata)
	if hex.EncodeToString(metadataSum[:]) != *sessionSmokeFlags.metadataDigest ||
		!bytes.Contains(metadata, []byte("codex=codex-cli 0.162.0\n")) {
		t.Fatal("image metadata binding or pinned provider version mismatch")
	}
	name := "admitted-codex-" + sessionSmokeToken(t)
	labels := map[string]string{"markitect.store": profile.Store}
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
	workspace, managerHome, reviewerHome := volume("work"), volume("manager"), volume("reviewer")
	helperSpec := container.ContainerSpec{Name: name + "-seed", Image: image, Labels: labels, User: "1000:1000",
		Workdir: "/work", Mounts: []container.Mount{{Volume: workspace, Target: "/work", NoCopy: true}},
		CPUs: profile.CPUs, MemoryBytes: profile.MemoryBytes, Pids: profile.Pids, TmpBytes: profile.TmpBytes, Network: profile.Network}
	artifact := "artifact-" + sessionSmokeToken(t)
	// Initial copy-up supplies the image's uid-1000 directory ownership. Refuse
	// any image-populated entry before writing the exact one-file candidate.
	helperSpec.Mounts[0].NoCopy = false
	if err := engine.RunHelper(ctx, helperSpec, []string{"/bin/sh", "-c", "test -z \"$(find /work -mindepth 1 -print -quit)\" && cat > /work/smoke.txt"}, strings.NewReader(artifact+"\n"), io.Discard); err != nil {
		t.Fatal("smoke artifact materialization failed")
	}
	helperSpec.Mounts[0].NoCopy = true
	path, _ := snapshot.NewPath("smoke.txt")
	entry, _ := snapshot.NewEntry(path, snapshot.Regular, []byte(artifact+"\n"))
	tree, _ := snapshot.NewTree([]snapshot.Entry{entry})
	readback := func() {
		var out bytes.Buffer
		spec := helperSpec
		spec.Name = name + "-readback"
		spec.Mounts = []container.Mount{{Volume: workspace, Target: "/work", ReadOnly: true, NoCopy: true}}
		if err := engine.RunHelper(ctx, spec, []string{"/bin/sh", "-c", "test \"$(find /work -mindepth 1 | wc -l)\" -eq 1 && test -f /work/smoke.txt && test ! -L /work/smoke.txt && cat /work/smoke.txt"}, nil, &out); err != nil || out.String() != artifact+"\n" {
			t.Fatal("actual artifact readback differs from bound candidate")
		}
	}
	readback()
	credentialRef := "dedicated-runtime-volume:" + *sessionSmokeFlags.credential
	if !volumeName.MatchString(*sessionSmokeFlags.credential) {
		t.Fatal("invalid dedicated runtime login volume reference")
	}
	resolver := func(resolveCtx context.Context, ref string) (RuntimeCredential, error) {
		if ref != credentialRef {
			return RuntimeCredential{}, ErrSession
		}
		spec := helperSpec
		spec.Name = name + "-read-login"
		spec.Workdir = "/auth"
		spec.Mounts = []container.Mount{{Volume: *sessionSmokeFlags.credential, Target: "/auth", ReadOnly: true, NoCopy: true}}
		var login bytes.Buffer
		if err := engine.RunHelper(resolveCtx, spec, []string{"/usr/bin/head", "-c", "1048577", "/auth/auth.json"}, nil, &login); err != nil || login.Len() > 1<<20 {
			return RuntimeCredential{}, ErrSession
		}
		return RuntimeCredential{Mode: "dedicated-runtime-login", RuntimeLogin: login.Bytes()}, nil
	}
	sessions, err := NewSessions(engine, resolver)
	if err != nil {
		t.Fatal("actual adapter construction failed")
	}
	conversation := "conversation-" + sessionSmokeToken(t)
	oneStep := func(suffix, role, home string, phase run.Phase, resume run.ProviderSessionID, recall bool) run.ProviderSessionID {
		tools, err := run.ToolCatalog(phase)
		if err != nil {
			t.Fatal("actual admitted tool catalogue unavailable")
		}
		contract, err := run.NewStepContract(phase)
		if err != nil {
			t.Fatal("actual StepContract unavailable")
		}
		plan := run.SessionPlan{ID: run.SessionID(name + "-" + suffix), Role: role, Workspace: workspace,
			ProviderHome: home, ImageID: image, CredentialRef: credentialRef, Resume: resume,
			Candidate: tree.Digest(), ReadOnly: true, MaxWireBytes: 1 << 20, Profile: profilePayload.Ref(), Tools: tools.Ref()}
		channel := &controlledSessionIO{step: run.StepID(suffix), phase: phase, artifact: artifact, answer: conversation,
			preparations: map[run.MessageID]run.TransportPreparation{}, observed: map[run.MessageID]bool{}, answered: map[run.ProviderCallID]bool{}}
		prompt := "Read /work/smoke.txt with your native file-reading tool. Do not change any file. "
		if role == "reviewer" {
			prompt += "Independently inspect this one-file candidate. Call submit_review exactly once with verdict pass, summary containing the exact file contents, and empty cases, findings, resolutions and evidence arrays."
		} else {
			prompt += "Call submit exactly once with purpose local, scope complete, a summary that contains the exact file contents, and empty assessments and dependencies arrays."
		}
		if recall {
			channel.continuity = conversation
			prompt += " Also include in summary the exact smokeConversationToken returned by the previous step's Host answer in this conversation. It is deliberately absent from this input and the workspace."
		}
		input := sessionSmokePayload(t, "text/plain", []byte(prompt))
		// These explicit admissions acknowledge scaffolding only, not a real Core
		// authorization or a Coordinator-issued durable admission.
		admitted := run.SessionAdmission{Plan: plan, Profile: profilePayload, Tools: tools, Receipt: channel.ack(), Invocation: "smoke-open"}
		opened, openErr := sessions.Open(ctx, admitted, channel)
		stopped, stopAttempted := false, false
		stop := func() {
			if stopAttempted {
				return
			}
			stopAttempted = true
			clean, done := context.WithTimeout(context.Background(), 20*time.Second)
			defer done()
			result, stopErr := sessions.Stop(clean, run.StopRequest{Session: plan.ID, ProviderSession: channel.bound, Timeout: 15 * time.Second, Reason: "bounded adapter smoke finished"}, channel)
			stopped = stopErr == nil && result.Terminated && channel.stopped
			if !stopped {
				t.Error("actual provider Stop observation incomplete; preserve uncertainty")
			}
		}
		defer stop()
		if openErr != nil || opened.Uncertain || !opened.Valid() {
			t.Fatal("actual adapter Open failed; no replay")
		}
		stepPlan := run.StepPlan{Session: plan.ID, ID: channel.step, SessionBinding: plan.Binding(), Input: input.Ref(), OutputContract: contract.Ref()}
		observed, stepErr := sessions.Step(ctx, run.StepAdmission{Plan: stepPlan, Session: plan, Input: input, OutputContract: contract, Receipt: channel.ack(), Invocation: "smoke-step"}, channel)
		if stepErr != nil || observed.Uncertain || !observed.Terminal || !channel.terminal || !channel.outcome || channel.bound == "" {
			t.Fatal("actual admitted step, Host outcome or exact acknowledged sends incomplete; no replay")
		}
		if resume != "" && channel.bound != resume {
			t.Fatal("new-container resume changed actual provider identity")
		}
		stop()
		if !stopped {
			t.FailNow()
		}
		readback()
		t.Logf("observed %s: image=%s profile=%x metadata=%x candidate=%s; controlled in-memory receipts only", suffix, image, profileSum, metadataSum, tree.Digest())
		return channel.bound
	}
	first := oneStep("manager-first", "manager", managerHome, run.PhaseWork, "", false)
	resumed := oneStep("manager-resume", "manager", managerHome, run.PhaseWork, first, true)
	reviewer := oneStep("reviewer-fresh", "reviewer", reviewerHome, run.PhaseReviewerCandidate, "", false)
	if resumed != first || reviewer == first {
		t.Fatal("manager continuity or fresh reviewer identity not distinguished")
	}
}
