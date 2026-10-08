package projectcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestDecodeResolutionChoicesIsClosedAndExplicit(t *testing.T) {
	valid := []byte(`{"actor":"user","authorityClaim":"Owner supplied this decision","decisionReference":"decision-1","questions":[],"scopes":[{"scopeId":"orders","status":"defer","reason":"Not in scope"}]}`)
	choices, err := decodeResolutionChoices(valid)
	if err != nil || choices.Actor != "user" || len(choices.Questions) != 0 || len(choices.Scopes) != 1 {
		t.Fatalf("decode choices = %#v, err=%v", choices, err)
	}
	for _, invalid := range [][]byte{
		[]byte(`{"actor":"user","authorityClaim":"x","decisionReference":"d","authenticated":false,"questions":[],"scopes":[]}`),
		[]byte(`{"actor":"user","Actor":"admin","authorityClaim":"x","decisionReference":"d","questions":[],"scopes":[]}`),
		[]byte(`{"actor":"user","actor":"admin","authorityClaim":"x","decisionReference":"d","questions":[],"scopes":[]}`),
		[]byte(`{"actor":"user","authorityClaim":"x","decisionReference":"d","questions":null,"scopes":[]}`),
	} {
		if _, err := decodeResolutionChoices(invalid); err == nil {
			t.Errorf("accepted non-closed or incomplete choices %s", invalid)
		}
	}
}

func TestResolveCLIHostBindsExactTargetAndWritesOnlyResolutionRecord(t *testing.T) {
	repo := copyProjectWorld(t)
	commit := gitOutput(t, repo, "rev-parse", "HEAD")
	target, err := projectwork.Load(repo, commit)
	if err != nil {
		t.Fatal(err)
	}
	targetContext, err := projectadoption.TargetContextForProject(target)
	if err != nil {
		t.Fatal(err)
	}
	beforeDigest := target.Digest
	modelPath := filepath.Join(repo, filepath.FromSlash(".markitect/model/commerce/sales/orders/cancellation.yaml"))
	request := projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion, ID: "shop-discovery", Purpose: "Resolve selected cancellation evidence",
		Review: "review-1", Commit: commit, ScopeRoots: []string{"docs"},
		Selected:   []projectadoption.SelectedPath{{ID: "intent", Path: "docs/cancellation.md", Reason: "Read the cancellation contract", Basis: "documentation"}},
		Exclusions: []projectadoption.PathReason{}, Unselected: []projectadoption.PathReason{},
	}
	discovery, err := projectadoption.Discover(repo, request)
	if err != nil {
		t.Fatal(err)
	}
	schemaDigest, _, err := projectadoption.CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	evidence := discovery.Evidence[0]
	lines := strings.Split(strings.ReplaceAll(evidence.Content, "\r\n", "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("fixture evidence unexpectedly short: %q", evidence.Content)
	}
	report := projectadoption.Distillation{
		APIVersion: projectadoption.DistillationVersion, DiscoveryDigest: discovery.Digest, Method: "human-review", SchemaDigest: schemaDigest,
		TargetBasis: target.Digest, TargetRevision: target.Revision, TargetContextDigest: targetContext.Digest,
		Claims: []projectadoption.Claim{{
			ID: "documented-cancellation", ScopeID: "orders", Kind: "documented-intent", Method: "documentation",
			Statement:   "The selected documentation defines when cancellation is allowed.",
			Evidence:    []projectadoption.EvidenceRef{{EvidenceID: evidence.ID, StartLine: 3, EndLine: 3, Excerpt: strings.TrimSuffix(lines[2], "\r")}},
			Uncertainty: []string{},
		}},
		Terms: []projectadoption.Term{}, Contradictions: []projectadoption.Contradiction{}, Questions: []projectadoption.Question{},
		Scopes: []projectadoption.ScopeProposal{{ID: "orders", Name: "Order cancellation", ClaimIDs: []string{"documented-cancellation"}}},
		Proposal: projectadoption.ModelProposal{Goal: "Represent selected order cancellation scope", Files: []projectadoption.ProposedFile{{
			ScopeID: "orders", Path: ".markitect/model/commerce/sales/orders/cancellation.yaml",
			Content: "apiVersion: project.markitect.example.org/v1alpha1\nkind: Statement\nmetadata:\n  name: cancellation\n  namespace: commerce.sales.orders\nspec:\n  category: rule\n  description: Cancel only confirmed orders before shipment.\n",
		}}},
	}
	projectadoption.SealDistillation(&report)
	discoveryBytes, err := projectadoption.EncodeDiscovery(discovery)
	if err != nil {
		t.Fatal(err)
	}
	reportBytes, err := projectadoption.EncodeDistillation(report)
	if err != nil {
		t.Fatal(err)
	}
	choicesBytes := []byte(`{"actor":"user","authorityClaim":"Shop owner selected the order scope","decisionReference":"shop-decision-17","questions":[],"scopes":[{"scopeId":"orders","status":"adopt","reason":"The owner selected this scope for review"}]}`)
	drafts := filepath.Join(repo, ".markitect", "drafts")
	if err := os.MkdirAll(drafts, 0755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"discovery.json": discoveryBytes, "report.json": reportBytes, "choices.json": choicesBytes} {
		if err := os.WriteFile(filepath.Join(drafts, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	output := new(bytes.Buffer)
	errout := new(bytes.Buffer)
	args := []string{"resolve", "--repo", repo, "--source-repo", repo, "--revision", commit, "--discovery", ".markitect/drafts/discovery.json", "--report", ".markitect/drafts/report.json", "--input", ".markitect/drafts/choices.json", "--output", ".markitect/drafts/resolution.json"}
	if code := Run(args, output, errout); code != 0 {
		t.Fatalf("resolve CLI code=%d err=%s out=%s", code, errout.String(), output.String())
	}
	resolutionBytes, err := readRecord(repo, ".markitect/drafts/resolution.json")
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := projectadoption.DecodeResolution(resolutionBytes, discovery, report)
	if err != nil {
		t.Fatalf("decode generated resolution: %v", err)
	}
	activeSchema, activeBuild, err := projectadoption.CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	if resolution.TargetBasis != target.Digest || resolution.SchemaDigest != activeSchema || resolution.BuildDigest != activeBuild || resolution.ProposalDigest != projectadoption.ProposalDigest(report.Proposal) || resolution.Authenticated == nil || *resolution.Authenticated {
		t.Fatalf("resolution bindings do not match active Host values: %+v", resolution)
	}
	if _, err := os.Stat(modelPath); !os.IsNotExist(err) {
		t.Fatalf("resolve wrote a model proposal to disk: %v", err)
	}
	after, err := projectwork.Load(repo, commit)
	if err != nil || after.Digest != beforeDigest {
		t.Fatalf("resolve changed fixed project model: digest=%q err=%v", after.Digest, err)
	}
	badChoices := []byte(`{"actor":"user","authorityClaim":"x","decisionReference":"d","targetBasis":"caller-controlled","questions":[],"scopes":[{"scopeId":"orders","status":"defer","reason":"Not selected"}]}`)
	if err := os.WriteFile(filepath.Join(drafts, "bad-choices.json"), badChoices, 0644); err != nil {
		t.Fatal(err)
	}
	badArgs := append([]string(nil), args...)
	for index := range badArgs {
		if badArgs[index] == ".markitect/drafts/choices.json" {
			badArgs[index] = ".markitect/drafts/bad-choices.json"
		}
		if badArgs[index] == ".markitect/drafts/resolution.json" {
			badArgs[index] = ".markitect/drafts/should-not-exist.json"
		}
	}
	if code := Run(badArgs, new(bytes.Buffer), new(bytes.Buffer)); code == 0 {
		t.Fatal("resolve accepted caller-controlled binding fields")
	}
	if _, err := os.Stat(filepath.Join(drafts, "should-not-exist.json")); !os.IsNotExist(err) {
		t.Fatalf("invalid choices emitted a resolution record: %v", err)
	}

	wrongTargetReport := report
	wrongTargetReport.TargetBasis = strings.Repeat("0", 64)
	projectadoption.SealDistillation(&wrongTargetReport)
	wrongTargetBytes, err := projectadoption.EncodeDistillation(wrongTargetReport)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(drafts, "wrong-target-report.json"), wrongTargetBytes, 0644); err != nil {
		t.Fatal(err)
	}
	wrongTargetArgs := append([]string(nil), args...)
	for index := range wrongTargetArgs {
		if wrongTargetArgs[index] == ".markitect/drafts/report.json" {
			wrongTargetArgs[index] = ".markitect/drafts/wrong-target-report.json"
		}
		if wrongTargetArgs[index] == ".markitect/drafts/resolution.json" {
			wrongTargetArgs[index] = ".markitect/drafts/wrong-target-resolution.json"
		}
	}
	wrongTargetErr := new(bytes.Buffer)
	if code := Run(wrongTargetArgs, new(bytes.Buffer), wrongTargetErr); code == 0 || !strings.Contains(wrongTargetErr.String(), "distillation target binding does not match") {
		t.Fatalf("resolve accepted a report grounded in another target: code=%d stderr=%s", code, wrongTargetErr.String())
	}
	if _, err := os.Stat(filepath.Join(drafts, "wrong-target-resolution.json")); !os.IsNotExist(err) {
		t.Fatalf("mismatched target report emitted a resolution: %v", err)
	}
	var outputRecord map[string]string
	if err := json.Unmarshal(output.Bytes(), &outputRecord); err != nil || outputRecord["path"] != ".markitect/drafts/resolution.json" {
		t.Fatalf("resolve output record = %#v, err=%v", outputRecord, err)
	}
}
