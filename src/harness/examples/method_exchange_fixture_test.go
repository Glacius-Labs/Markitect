package examples

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const (
	methodExchangeHappy             = "happy"
	methodExchangeForgottenFile     = "forgotten-file"
	methodExchangeMissingFile       = "missing-required-file"
	methodExchangeOutOfScope        = "out-of-scope"
	methodExchangeIntegrationDefect = "integration-defect"
	methodExchangeHoldFirst         = "hold-first-request"

	methodExchangeRootID      = `["project.markitect.example.org/v1alpha1","Manager","","project-owner"]`
	methodExchangeOrdersID    = `["project.markitect.example.org/v1alpha1","Manager","orders","orders"]`
	methodExchangeInventoryID = `["project.markitect.example.org/v1alpha1","Manager","inventory","inventory"]`
	methodExchangeTaskKind    = "projectrun-task/v1"
	methodExchangeReviewKind  = "projectrun-review/v1"
)

// methodExchangeFixture is a small committed three-Manager project served by
// the real CLI and exchange executor. All mutable exchange state and literal
// comparison files live beside, rather than inside, the governed repository.
type methodExchangeFixture struct {
	root        string
	exchangeDir string
	ledgerPath  string
	expectedDir string
	baseline    map[string]string
	expected    map[string]string
	mode        string
	held        chan methodExchangeHeld

	mu       sync.Mutex
	entries  []methodExchangeLedgerEntry
	errs     []error
	seen     map[string]bool
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	stopOnce sync.Once
}

// methodExchangeHeld is sent when hold-first-request has published a request
// but deliberately has not answered it. The parent test owns and kills the
// CLI process; the exchange adapter's exact on-disk request is included here.
type methodExchangeHeld struct {
	RunID        string    `json:"runId"`
	Nonce        string    `json:"nonce"`
	Role         string    `json:"role"`
	Kind         string    `json:"kind,omitempty"`
	Phase        string    `json:"phase,omitempty"`
	ManagerID    string    `json:"managerId,omitempty"`
	RepairRound  int       `json:"repairRound,omitempty"`
	RawRequest   []byte    `json:"rawRequest"`
	ExchangeDir  string    `json:"exchangeDir"`
	RequestPath  string    `json:"requestPath"`
	ResponsePath string    `json:"responsePath"`
	ReceivedAt   time.Time `json:"receivedAt"`
}

// methodExchangeLedgerEntry is one raw exchange request plus its decoded
// routing facts and any adapter rejection explanation observed afterward.
type methodExchangeLedgerEntry struct {
	RunID         string    `json:"runId"`
	Nonce         string    `json:"nonce"`
	Role          string    `json:"role"`
	Kind          string    `json:"kind,omitempty"`
	Phase         string    `json:"phase,omitempty"`
	ManagerID     string    `json:"managerId,omitempty"`
	RepairRound   int       `json:"repairRound,omitempty"`
	RawRequest    []byte    `json:"rawRequest"`
	RequestPath   string    `json:"requestPath"`
	ResponseError string    `json:"responseError,omitempty"`
	ReceivedAt    time.Time `json:"receivedAt"`
}

type methodExchangeInvocation struct {
	APIVersion string `json:"apiVersion"`
	RunID      string `json:"runId"`
	Nonce      string `json:"nonce"`
	Request    struct {
		Role      string          `json:"role"`
		Context   json.RawMessage `json:"context"`
		Artifacts []struct {
			Path    string `json:"path"`
			Content []byte `json:"content"`
		} `json:"artifacts"`
	} `json:"request"`
}

type methodExchangeTaskContext struct {
	Kind             string `json:"kind"`
	Phase            string `json:"phase"`
	RepairRound      int    `json:"repairRound"`
	RepairDiagnostic string `json:"repairDiagnostic"`
	ManagerID        string `json:"managerId"`
	Manager          struct {
		Manager struct {
			ID string `json:"id"`
		} `json:"manager"`
	} `json:"manager"`
	DirectChildren       []string `json:"directChildren"`
	RequiredSubjects     []string `json:"requiredSubjects"`
	RequiredEvidenceRefs []string `json:"requiredEvidenceRefs"`
	CandidateDigest      string   `json:"candidateDigest"`
	CheckResults         []struct {
		ID, CandidateID, Outcome string
		ExitCode                 int
	} `json:"checkResults"`
	RepairChecks []struct {
		ID       string `json:"id"`
		ExitCode int    `json:"exitCode"`
	} `json:"repairChecks"`
}

func decodeMethodExchangeContext(invocation methodExchangeInvocation) (methodExchangeTaskContext, error) {
	var context methodExchangeTaskContext
	if err := json.Unmarshal(invocation.Request.Context, &context); err != nil {
		return methodExchangeTaskContext{}, fmt.Errorf("decode invocation context: %w", err)
	}
	return context, nil
}

// newMethodExchangeFixture creates the source baseline and out-of-repository
// expected-byte inputs. The caller configures it through the built CLI and
// starts the responder only when ready to execute a plan.
func newMethodExchangeFixture(t *testing.T, workDir, mode string) *methodExchangeFixture {
	t.Helper()
	if mode != methodExchangeHappy && mode != methodExchangeForgottenFile && mode != methodExchangeMissingFile && mode != methodExchangeOutOfScope && mode != methodExchangeIntegrationDefect && mode != methodExchangeHoldFirst {
		t.Fatalf("unknown method exchange scenario %q", mode)
	}
	repo := testkit.NewRepo(t)
	root := repo.Dir
	exchangeDir := filepath.Join(workDir, "exchange")
	expectedDir := filepath.Join(workDir, "expected")
	for _, dir := range []string{root, exchangeDir, expectedDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("create method exchange directory %s: %v", dir, err)
		}
	}
	fixture := &methodExchangeFixture{
		root: root, exchangeDir: exchangeDir, ledgerPath: filepath.Join(workDir, "exchange-ledger.jsonl"), expectedDir: expectedDir,
		baseline: map[string]string{},
		expected: map[string]string{
			"src/orders/implementation.txt":    "orders implementation v2\n",
			"src/inventory/implementation.txt": "inventory implementation v2\n",
			"src/project-integration.txt":      "project integration: orders=v2; inventory=v2\n",
		},
		mode: mode, held: make(chan methodExchangeHeld, 1), seen: map[string]bool{},
	}
	for path, content := range fixture.expected {
		writeMethodExchangeFile(t, filepath.Join(expectedDir, filepath.FromSlash(path)), content)
	}
	repo.Git("switch", "-c", "dev/method-exchange-fixture")
	files := methodExchangeBaseline(expectedDir)
	// An omitted replacement retains a required path, so Verify reaches the
	// declared process check. A wholly absent required artifact is rejected
	// structurally before checks and is not eligible for explicit Repair.
	if mode != methodExchangeMissingFile {
		files["src/orders/implementation.txt"] = "orders implementation v1\n"
	}
	for path, content := range files {
		repo.Write(path, content)
		fixture.baseline[path] = content
	}
	repo.Commit("Freeze method exchange fixture")
	return fixture
}

func methodExchangeBaseline(expectedDir string) map[string]string {
	project := `apiVersion: project.markitect.example.org/v1alpha1
name: method-exchange
coverageMode: selected
documentPath: docs/markitect/project.md
modelFiles:
  - .markitect/model/manager.yaml
  - .markitect/model/root/statement.yaml
  - .markitect/model/root/artifact.yaml
  - .markitect/model/root/check.yaml
  - .markitect/model/orders/manager.yaml
  - .markitect/model/orders/statement.yaml
  - .markitect/model/orders/artifact.yaml
  - .markitect/model/orders/check.yaml
  - .markitect/model/inventory/manager.yaml
  - .markitect/model/inventory/statement.yaml
  - .markitect/model/inventory/artifact.yaml
  - .markitect/model/inventory/check.yaml
inventoryRoots:
  - src
exclusions: []
`
	return map[string]string{
		"README.md":               "Three-Manager method exchange fixture.\n",
		".markitect/project.yaml": project,
		".markitect/model/manager.yaml": `apiVersion: project.markitect.example.org/v1alpha1
kind: Manager
metadata:
  name: project-owner
  namespace: ""
purpose: Owns project integration and delegates bounded implementation work.
spec:
  owns: [.]
`,
		".markitect/model/root/statement.yaml": `apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  name: integration-output
  namespace: root
purpose: Project integration summarizes the two owned implementation outputs.
spec:
  category: concept
  description: The integration output names the expected Orders and Inventory implementations.
`,
		".markitect/model/root/artifact.yaml": `apiVersion: project.markitect.example.org/v1alpha1
kind: Artifact
metadata:
  name: integration-output
  namespace: root
purpose: Records the project owner's composed implementation summary.
spec:
  role: implementation
  realizes:
    - apiVersion: project.markitect.example.org/v1alpha1
      kind: Statement
      namespace: root
      name: integration-output
  paths: [src/project-integration.txt]
  checks:
    - apiVersion: project.markitect.example.org/v1alpha1
      kind: Check
      namespace: root
      name: integration-check
  required: true
`,
		".markitect/model/root/check.yaml":          methodExchangeCheck(expectedDir, "src/project-integration.txt", "integration-check", "integration-output", "root"),
		".markitect/model/orders/manager.yaml":      methodExchangeManager("orders", "Orders owns its implementation output."),
		".markitect/model/orders/statement.yaml":    methodExchangeStatement("orders", "orders-output", "Orders implementation output is materialized exactly."),
		".markitect/model/orders/artifact.yaml":     methodExchangeArtifact("orders", "orders-output", "orders-output", "orders-check", "src/orders/implementation.txt"),
		".markitect/model/orders/check.yaml":        methodExchangeCheck(expectedDir, "src/orders/implementation.txt", "orders-check", "orders-output", "orders"),
		".markitect/model/inventory/manager.yaml":   methodExchangeManager("inventory", "Inventory owns its implementation output."),
		".markitect/model/inventory/statement.yaml": methodExchangeStatement("inventory", "inventory-output", "Inventory implementation output is materialized exactly."),
		".markitect/model/inventory/artifact.yaml":  methodExchangeArtifact("inventory", "inventory-output", "inventory-output", "inventory-check", "src/inventory/implementation.txt"),
		".markitect/model/inventory/check.yaml":     methodExchangeCheck(expectedDir, "src/inventory/implementation.txt", "inventory-check", "inventory-output", "inventory"),
	}
}

func methodExchangeManager(name, purpose string) string {
	return fmt.Sprintf(`apiVersion: project.markitect.example.org/v1alpha1
kind: Manager
metadata:
  name: %s
  namespace: %s
purpose: %s
spec:
  parent:
    apiVersion: project.markitect.example.org/v1alpha1
    kind: Manager
    namespace: ""
    name: project-owner
  owns: [src/%s/]
`, name, name, purpose, name)
}

func methodExchangeStatement(namespace, name, description string) string {
	return fmt.Sprintf(`apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  name: %s
  namespace: %s
purpose: %s
spec:
  category: concept
  description: %s
`, name, namespace, description, description)
}

func methodExchangeArtifact(namespace, name, statement, check, path string) string {
	return fmt.Sprintf(`apiVersion: project.markitect.example.org/v1alpha1
kind: Artifact
metadata:
  name: %s
  namespace: %s
purpose: Materializes the independently checked implementation bytes.
spec:
  role: implementation
  realizes:
    - apiVersion: project.markitect.example.org/v1alpha1
      kind: Statement
      namespace: %s
      name: %s
  paths: [%s]
  checks:
    - apiVersion: project.markitect.example.org/v1alpha1
      kind: Check
      namespace: %s
      name: %s
  required: true
`, name, namespace, namespace, statement, path, namespace, check)
}

func methodExchangeCheck(expectedDir, sourcePath, checkName, statementName, namespace string) string {
	expectedPath := filepath.ToSlash(filepath.Join(expectedDir, filepath.FromSlash(sourcePath)))
	if sourcePath == "src/project-integration.txt" {
		expectedPath = filepath.ToSlash(filepath.Join(expectedDir, "src/project-integration.txt"))
	}
	return fmt.Sprintf(`apiVersion: project.markitect.example.org/v1alpha1
kind: Check
metadata:
  name: %s
  namespace: %q
purpose: Compares exact candidate bytes with a test-owned literal expectation.
spec:
  command: [git, diff, --no-index, --exit-code, --, %s, %s]
  uses:
    - apiVersion: project.markitect.example.org/v1alpha1
      kind: Statement
      namespace: %q
      name: %s
  limitation: This check compares only the exact expected file bytes for this fixture.
`, checkName, namespace, strconv.Quote(expectedPath), strconv.Quote(sourcePath), namespace, statementName)
}

func writeMethodExchangeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create fixture parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture file %s: %v", path, err)
	}
}

// start begins the asynchronous responder. All errors are collected for the
// test goroutine and returned by stop; the goroutine never calls Fatal.
func (f *methodExchangeFixture) start(t *testing.T) {
	t.Helper()
	f.mu.Lock()
	if f.done != nil {
		f.mu.Unlock()
		t.Fatal("method exchange responder was started more than once")
	}
	f.ctx, f.cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	f.done = make(chan struct{})
	f.mu.Unlock()
	go func() {
		defer close(f.done)
		f.serve()
	}()
	t.Cleanup(func() {
		_, err := f.stop()
		if err != nil {
			t.Errorf("stop method exchange responder: %v", err)
		}
	})
}

// stop cancels the responder and returns a copy of the durable request ledger
// plus any I/O or protocol errors observed by its goroutine.
func (f *methodExchangeFixture) stop() ([]methodExchangeLedgerEntry, error) {
	f.mu.Lock()
	started, cancel, done := f.done != nil, f.cancel, f.done
	f.mu.Unlock()
	if !started {
		return nil, nil
	}
	f.stopOnce.Do(cancel)
	<-done
	f.mu.Lock()
	defer f.mu.Unlock()
	entries := make([]methodExchangeLedgerEntry, len(f.entries))
	copy(entries, f.entries)
	for i := range entries {
		entries[i].RawRequest = bytes.Clone(entries[i].RawRequest)
	}
	return entries, errors.Join(f.errs...)
}

func (f *methodExchangeFixture) serve() {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		f.poll()
		select {
		case <-f.ctx.Done():
			f.poll()
			return
		case <-ticker.C:
		}
	}
}

func (f *methodExchangeFixture) poll() {
	requests, err := filepath.Glob(filepath.Join(f.exchangeDir, "*", "request.json"))
	if err != nil {
		f.addError(fmt.Errorf("list exchange requests: %w", err))
		return
	}
	for _, requestPath := range requests {
		f.mu.Lock()
		seen := f.seen[requestPath]
		f.mu.Unlock()
		if seen {
			f.observeResponseError(requestPath)
			continue
		}
		raw, err := os.ReadFile(requestPath)
		if err != nil {
			f.addError(fmt.Errorf("read exchange request %s: %w", requestPath, err))
			continue
		}
		var invocation methodExchangeInvocation
		if err := json.Unmarshal(raw, &invocation); err != nil {
			f.addError(fmt.Errorf("decode exchange request %s: %w", requestPath, err))
			continue
		}
		invocationContext, err := decodeMethodExchangeContext(invocation)
		if err != nil {
			f.addError(err)
			continue
		}
		record := methodExchangeLedgerEntry{
			RunID: invocation.RunID, Nonce: invocation.Nonce, Role: invocation.Request.Role, Kind: invocationContext.Kind,
			Phase: invocationContext.Phase, ManagerID: invocationContext.ManagerID,
			RepairRound: invocationContext.RepairRound, RawRequest: bytes.Clone(raw), RequestPath: requestPath,
			ReceivedAt: time.Now().UTC(),
		}
		if record.ManagerID == "" {
			record.ManagerID = invocationContext.Manager.Manager.ID
		}
		f.mu.Lock()
		f.seen[requestPath] = true
		f.entries = append(f.entries, record)
		f.mu.Unlock()
		if err := f.appendLedger(record); err != nil {
			f.addError(err)
		}
		exchangePath := filepath.Dir(requestPath)
		responsePath := filepath.Join(exchangePath, "response.json")
		if f.mode == methodExchangeHoldFirst && len(f.snapshotEntries()) == 1 {
			held := methodExchangeHeld{RunID: record.RunID, Nonce: record.Nonce, Role: record.Role, Kind: record.Kind,
				Phase: record.Phase, ManagerID: record.ManagerID, RepairRound: record.RepairRound, RawRequest: bytes.Clone(raw),
				ExchangeDir: exchangePath, RequestPath: requestPath, ResponsePath: responsePath, ReceivedAt: record.ReceivedAt}
			select {
			case f.held <- held:
			default:
				f.addError(errors.New("held exchange request channel was already occupied"))
			}
			continue
		}
		response, err := f.response(invocation, invocationContext)
		if err != nil {
			f.addError(err)
			continue
		}
		if err := writeMethodExchangeResponse(exchangePath, response); err != nil {
			f.addError(fmt.Errorf("answer exchange request %s: %w", requestPath, err))
		}
	}
}

func (f *methodExchangeFixture) response(invocation methodExchangeInvocation, context methodExchangeTaskContext) ([]byte, error) {
	response := map[string]any{
		"candidateFiles": []any{}, "evidenceRefs": []any{}, "verifierObservations": []any{}, "uncertainty": []any{},
	}
	if invocation.Request.Role == "verifier" {
		subjects, refs, err := methodExchangeVerifierExpectation(context)
		if err != nil {
			return nil, err
		}
		observations := make([]map[string]string, 0, len(subjects))
		for _, subject := range subjects {
			observations = append(observations, map[string]string{"subject": subject, "outcome": "passed", "detail": "The three fixture-owned literal checks passed for this candidate."})
		}
		response["outcome"] = "passed"
		response["evidenceRefs"] = refs
		response["verifierObservations"] = observations
		return json.Marshal(response)
	}
	if context.Kind == methodExchangeReviewKind {
		if invocation.Request.Role != "executor" {
			return nil, fmt.Errorf("review request has unexpected role %q", invocation.Request.Role)
		}
		report, err := f.reviewReport(invocation)
		if err != nil {
			return nil, err
		}
		response["outcome"] = "proposed"
		response["reportJson"] = report
		return json.Marshal(response)
	}
	if invocation.Request.Role != "executor" || context.Kind != methodExchangeTaskKind {
		return nil, fmt.Errorf("unexpected exchange request role=%q kind=%q", invocation.Request.Role, context.Kind)
	}
	managerID := context.Manager.Manager.ID
	if managerID == "" {
		return nil, errors.New("task request omitted manager.manager.id")
	}
	files := []map[string]string{}
	delegations := []map[string]string{}
	integrated := context.Phase == "integrate"
	switch context.Phase {
	case "work":
		switch managerID {
		case methodExchangeRootID:
			for _, child := range context.DirectChildren {
				delegations = append(delegations, map[string]string{"managerId": child, "goal": "Produce the exact bytes for the owned implementation artifact."})
			}
		case methodExchangeOrdersID:
			if f.mode == methodExchangeOutOfScope {
				files = append(files, map[string]string{"path": "src/inventory/foreign.txt", "mode": "0644", "content": "unauthorized foreign write\n"})
			} else if f.mode != methodExchangeMissingFile && !(f.mode == methodExchangeForgottenFile && !methodExchangeHasFailedCheck(context, "orders-check") && context.RepairRound == 0 && context.RepairDiagnostic == "") {
				files = append(files, methodExchangeCandidate("src/orders/implementation.txt", f.expected["src/orders/implementation.txt"]))
			}
		case methodExchangeInventoryID:
			files = append(files, methodExchangeCandidate("src/inventory/implementation.txt", f.expected["src/inventory/implementation.txt"]))
		default:
			return nil, fmt.Errorf("unexpected work Manager %s", managerID)
		}
	case "integrate":
		if managerID != methodExchangeRootID {
			return nil, fmt.Errorf("unexpected integration Manager %s", managerID)
		}
		content := f.expected["src/project-integration.txt"]
		if f.mode == methodExchangeIntegrationDefect && context.RepairRound == 0 && context.RepairDiagnostic == "" {
			content = "project integration: orders=v1; inventory=v2\n"
		}
		files = append(files, methodExchangeCandidate("src/project-integration.txt", content))
	default:
		return nil, fmt.Errorf("unexpected task phase %q", context.Phase)
	}
	report := map[string]any{
		"status": "complete", "summary": fmt.Sprintf("Scripted %s result for %s.", context.Phase, managerID),
		"delegations": delegations, "reworkRequests": []any{}, "integrated": integrated,
		"questions": []any{}, "risks": []any{}, "resolvedQuestions": []any{}, "resolvedRisks": []any{}, "escalateTo": "",
	}
	response["outcome"] = "proposed"
	response["candidateFiles"] = files
	response["reportJson"] = report
	return json.Marshal(response)
}

func methodExchangeVerifierExpectation(context methodExchangeTaskContext) ([]string, []string, error) {
	// The expected subject set is owned by this fixture, not copied from a
	// request that could omit a required artifact or invent a passed subject.
	artifactIDs := []string{
		`["project.markitect.example.org/v1alpha1","Artifact","root","integration-output"]`,
		`["project.markitect.example.org/v1alpha1","Artifact","orders","orders-output"]`,
		`["project.markitect.example.org/v1alpha1","Artifact","inventory","inventory-output"]`,
	}
	checkIDs := []string{
		`["project.markitect.example.org/v1alpha1","Check","root","integration-check"]`,
		`["project.markitect.example.org/v1alpha1","Check","orders","orders-check"]`,
		`["project.markitect.example.org/v1alpha1","Check","inventory","inventory-check"]`,
	}
	subjects := append(append([]string{}, artifactIDs...), checkIDs...)
	refs := []string{}
	for _, id := range artifactIDs {
		refs = append(refs, "artifact:"+id)
	}
	checks := map[string]bool{}
	for _, id := range checkIDs {
		refs = append(refs, "check:"+id)
		checks[id] = false
	}
	sort.Strings(subjects)
	sort.Strings(refs)
	if !reflect.DeepEqual(context.RequiredSubjects, subjects) || !reflect.DeepEqual(context.RequiredEvidenceRefs, refs) {
		return nil, nil, errors.New("verifier request differs from the fixture-owned six subjects/evidence references")
	}
	if context.CandidateDigest == "" || len(context.CheckResults) != 3 {
		return nil, nil, errors.New("verifier request lacks exact candidate/three-check evidence")
	}
	candidate := ""
	for _, check := range context.CheckResults {
		seen, known := checks[check.ID]
		if !known || seen || check.Outcome != "passed" || check.ExitCode != 0 || check.CandidateID == "" || (candidate != "" && candidate != check.CandidateID) {
			return nil, nil, fmt.Errorf("verifier check evidence is not a complete known passing candidate: %s", check.ID)
		}
		candidate, checks[check.ID] = check.CandidateID, true
	}
	return subjects, refs, nil
}

func methodExchangeHasFailedCheck(context methodExchangeTaskContext, id string) bool {
	for _, check := range context.RepairChecks {
		if check.ID == id && check.ExitCode != 0 {
			return true
		}
	}
	return false
}

func methodExchangeCandidate(path, content string) map[string]string {
	return map[string]string{"path": path, "mode": "0644", "content": content}
}

func (f *methodExchangeFixture) reviewReport(invocation methodExchangeInvocation) (map[string]any, error) {
	var context struct {
		ManagerID      string   `json:"managerId"`
		Phase          string   `json:"phase"`
		ChangedPaths   []string `json:"changedPaths"`
		CandidateFiles []struct {
			Path      string   `json:"path"`
			Grounding []string `json:"grounding"`
		} `json:"candidateFiles"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &context); err != nil {
		return nil, fmt.Errorf("decode review context: %w", err)
	}
	grounding := map[string]string{}
	for _, file := range context.CandidateFiles {
		if len(file.Grounding) > 0 {
			grounding[file.Path] = file.Grounding[0]
		}
	}
	actual := map[string][]byte{}
	for _, artifact := range invocation.Request.Artifacts {
		actual[artifact.Path] = bytes.Clone(artifact.Content)
	}
	findings := []map[string]string{}
	changed := map[string]bool{}
	for _, path := range context.ChangedPaths {
		changed[path] = true
	}
	for path, want := range f.expected {
		got, supplied := actual[path]
		// This scripted change reviewer checks delivered changes. The separate
		// declared checks assess completeness, including an omitted replacement.
		if !changed[path] || !supplied || bytes.Equal(got, []byte(want)) {
			continue
		}
		basis := grounding[path]
		if basis == "" {
			return nil, fmt.Errorf("review candidate %s has no model grounding", path)
		}
		findings = append(findings, map[string]string{"path": path, "expectation": "Match the exact scripted fixture bytes for this owned artifact.", "grounding": basis})
	}
	status, summary := "pass", "The supplied changed bytes match literal expectations; completeness is assessed by the declared checks."
	if len(findings) > 0 {
		status, summary = "fail", "The supplied candidate contains bytes that differ from the independent fixture expectations."
	}
	return map[string]any{"status": status, "summary": summary, "findings": findings}, nil
}

func (f *methodExchangeFixture) snapshotEntries() []methodExchangeLedgerEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]methodExchangeLedgerEntry(nil), f.entries...)
}

func (f *methodExchangeFixture) addError(err error) {
	if err == nil {
		return
	}
	f.mu.Lock()
	f.errs = append(f.errs, err)
	f.mu.Unlock()
}

func (f *methodExchangeFixture) appendLedger(entry methodExchangeLedgerEntry) error {
	encoded, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(f.ledgerPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open raw exchange ledger: %w", err)
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		_ = file.Close()
		return fmt.Errorf("append raw exchange ledger: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close raw exchange ledger: %w", err)
	}
	return nil
}

func (f *methodExchangeFixture) observeResponseError(requestPath string) {
	path := filepath.Join(filepath.Dir(requestPath), "response-error.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	message := strings.TrimSpace(string(data))
	f.mu.Lock()
	for i := range f.entries {
		if f.entries[i].RequestPath == requestPath && f.entries[i].ResponseError != message {
			f.entries[i].ResponseError = message
			entry := f.entries[i]
			f.mu.Unlock()
			if ledgerErr := f.appendLedger(entry); ledgerErr != nil {
				f.addError(ledgerErr)
			}
			return
		}
	}
	f.mu.Unlock()
}

func writeMethodExchangeResponse(exchangeDir string, response []byte) error {
	staged := filepath.Join(exchangeDir, "response.staged")
	if err := os.WriteFile(staged, response, 0o600); err != nil {
		return err
	}
	if err := os.Rename(staged, filepath.Join(exchangeDir, "response.json")); err != nil {
		return err
	}
	return nil
}

// methodExchangeCommandLine quotes the path-sensitive executor arguments for
// use by the CLI config command owned by the parent E2E test.
func (f *methodExchangeFixture) commandArgs(executable string) []string {
	return []string{"--provider-executable", executable, "--provider-arg", "--dir", "--provider-arg", f.exchangeDir,
		"--provider-version", "exchange/test-fixture", "--cost-mode", "unmetered", "--max-cost-micros", "1000000"}
}

func (r methodExchangeLedgerEntry) String() string {
	return fmt.Sprintf("%s/%s/%s/%s repairRound=%d", r.RunID, r.Role, r.Phase, r.ManagerID, r.RepairRound)
}
