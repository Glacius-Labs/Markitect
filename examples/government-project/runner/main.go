// Command runner is a fixed deterministic protocol fixture. It is not a model provider.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const protocolVersion = "markitect.example.org/agent-execution/v1alpha1"

type artifact struct {
	Path    string `json:"path"`
	Mode    string `json:"mode"`
	Digest  string `json:"digest"`
	Content []byte `json:"content"`
}
type identity struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
}
type request struct {
	Role           string          `json:"role"`
	SourceRevision string          `json:"sourceRevision"`
	ModelDigest    string          `json:"modelDigest"`
	ModulePin      string          `json:"modulePin"`
	ProjectionID   string          `json:"projectionId"`
	ScopeIDs       []string        `json:"scopeIds"`
	PolicyIDs      []string        `json:"policyIds"`
	Context        json.RawMessage `json:"context"`
	Artifacts      []artifact      `json:"artifacts"`
}
type invocation struct {
	APIVersion  string  `json:"apiVersion"`
	RunID       string  `json:"runId"`
	Nonce       string  `json:"nonce"`
	InputDigest string  `json:"inputDigest"`
	Request     request `json:"request"`
}
type candidateFile struct {
	Path    string `json:"path"`
	Mode    string `json:"mode"`
	Content string `json:"content"`
}
type observation struct {
	Subject string `json:"subject"`
	Outcome string `json:"outcome"`
	Detail  string `json:"detail"`
}
type response struct {
	APIVersion           string          `json:"apiVersion"`
	RunID                string          `json:"runId"`
	Nonce                string          `json:"nonce"`
	Role                 string          `json:"role"`
	InputDigest          string          `json:"inputDigest"`
	Outcome              string          `json:"outcome"`
	CandidateFiles       []candidateFile `json:"candidateFiles"`
	EvidenceRefs         []string        `json:"evidenceRefs"`
	VerifierObservations []observation   `json:"verifierObservations"`
	Uncertainty          []string        `json:"uncertainty"`
}
type actorContext struct {
	Phase        string   `json:"phase"`
	Area         identity `json:"area"`
	AllowedPaths []string `json:"allowedPaths"`
}
type voteContext struct {
	Candidate struct {
		ID string `json:"id"`
	} `json:"candidate"`
	Evidence struct {
		ID    string `json:"id"`
		Round int    `json:"round"`
	} `json:"evidence"`
}
type source struct {
	APIVersion   string           `json:"apiVersion"`
	Kind         string           `json:"kind"`
	Constitution map[string]any   `json:"constitution"`
	Schemas      []map[string]any `json:"schemas"`
	Definitions  []map[string]any `json:"definitions"`
	Observation  map[string]any   `json:"observation"`
}

func main() {
	if len(os.Args) != 2 {
		fatal(errors.New("usage: runner independent|propose|review|review-independent|assent|assent-unaffected"))
	}
	inv, err := decodeInvocation(os.Stdin)
	if err != nil {
		fatal(fmt.Errorf("decode invocation: %w", err))
	}
	mode := os.Args[1]
	var out response
	if mode == "assent-unaffected" || mode == "assent" {
		out, err = assent(mode, inv)
	} else {
		out, err = runRole(mode, inv)
	}
	if err != nil {
		fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fatal(err)
	}
}

func decodeInvocation(input io.Reader) (invocation, error) {
	dec := json.NewDecoder(io.LimitReader(input, 32<<20))
	dec.DisallowUnknownFields()
	var inv invocation
	if err := dec.Decode(&inv); err != nil {
		return invocation{}, err
	}
	if dec.Decode(new(any)) != io.EOF || inv.APIVersion != protocolVersion || inv.RunID == "" || inv.Nonce == "" || inv.InputDigest == "" {
		return invocation{}, errors.New("invocation must be one complete, bound protocol record")
	}
	return inv, nil
}

func runRole(mode string, inv invocation) (response, error) {
	var ctx actorContext
	if err := json.Unmarshal(inv.Request.Context, &ctx); err != nil {
		return response{}, fmt.Errorf("decode role context: %w", err)
	}
	out := response{APIVersion: protocolVersion, RunID: inv.RunID, Nonce: inv.Nonce, Role: inv.Request.Role, InputDigest: inv.InputDigest,
		Outcome: "proposed", CandidateFiles: []candidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []observation{}, Uncertainty: []string{}}
	if ctx.Phase == "execute" && inv.Request.Role == "executor" {
		switch mode {
		case "independent":
			return proposal(out, ctx, []candidateFile{{Path: "independent/participant.txt", Mode: "0644", Content: "channel=web-v2\n"}})
		case "propose":
			if err := allowed(ctx, "government.yaml", "studio/studio.go", "studio/studio_test.go"); err != nil {
				return response{}, err
			}
			model, err := sourceArtifact(inv.Request.Artifacts)
			if err != nil {
				return response{}, err
			}
			setRequirement(model, "booking-rule", "The studio booking total equals seat count multiplied by seat price, plus exactly one fixed booking fee of 2 euros per booking.", "Three seats at 4 euros plus one 2-euro fee total 14; one seat at 5 euros plus one fee totals 7.")
			setOpenQuestions(model, "booking-rule", "Refund handling remains undecided and outside this bounded change.")
			setRealization(model, "booking-implementation", "Adds the fixed booking fee once after multiplying seats by seat price.")
			setRealization(model, "booking-acceptance", "Checks both one-seat and three-seat totals against the fixed 2-euro fee.")
			encoded, err := json.MarshalIndent(model, "", "  ")
			if err != nil {
				return response{}, err
			}
			files := []candidateFile{
				{Path: "government.yaml", Mode: "0644", Content: string(append(encoded, '\n'))},
				{Path: "studio/studio.go", Mode: "0644", Content: studioImplementation},
				{Path: "studio/studio_test.go", Mode: "0644", Content: candidateTest},
			}
			return proposal(out, ctx, files)
		default:
			return response{}, fmt.Errorf("unsupported execution mode %q", mode)
		}
	}
	if ctx.Phase != "review" || inv.Request.Role != "verifier" {
		return response{}, errors.New("review mode requires a verifier review context")
	}
	passed := false
	detail := ""
	switch mode {
	case "review-independent":
		passed = string(findArtifact(inv.Request.Artifacts, "independent/participant.txt")) == "channel=web-v2\n"
		detail = "checked the exact independent confirmation-label bytes"
	case "review":
		model, err := sourceArtifact(inv.Request.Artifacts)
		if err != nil {
			return response{}, err
		}
		statement := requirementStatement(model, "booking-rule")
		fixed := requirementStatement(model, "published-booking")
		code := string(findArtifact(inv.Request.Artifacts, "studio/studio.go"))
		test := string(findArtifact(inv.Request.Artifacts, "studio/studio_test.go"))
		rate := string(findArtifact(inv.Request.Artifacts, "booking/rate.txt"))
		passed = strings.Contains(statement, "exactly one fixed booking fee of 2 euros per booking") &&
			strings.Contains(fixed, "exactly 2 euros") && strings.Contains(code, "seats*seatPrice + bookingFee") &&
			strings.Contains(test, "BookingTotal(1, 5, fee)") && strings.Contains(test, "BookingTotal(3, 4, fee)") && strings.Contains(rate, "fee=2")
		detail = "checked the candidate rule, unchanged fixed fee, one-time fee implementation, two-seat-count acceptance cases, and published fee bytes"
	default:
		return response{}, fmt.Errorf("unsupported review mode %q", mode)
	}
	out.Outcome = "passed"
	if !passed {
		out.Outcome = "failed"
	}
	for _, scope := range inv.Request.ScopeIDs {
		out.VerifierObservations = append(out.VerifierObservations, observation{Subject: scope, Outcome: out.Outcome, Detail: detail})
	}
	if len(out.VerifierObservations) == 0 {
		return response{}, errors.New("review has no frozen subjects")
	}
	return out, nil
}

func assent(mode string, inv invocation) (response, error) {
	if inv.Request.Role != "verifier" {
		return response{}, errors.New("Ressort vote requires verifier role")
	}
	var ctx voteContext
	if err := json.Unmarshal(inv.Request.Context, &ctx); err != nil {
		return response{}, fmt.Errorf("decode vote binding: %w", err)
	}
	if ctx.Candidate.ID == "" || ctx.Evidence.ID == "" || ctx.Evidence.Round < 1 {
		return response{}, errors.New("vote requires current candidate and evidence bindings")
	}
	detail, err := json.Marshal(map[string]any{"outcome": mode, "reason": "fixed fixture records this Ressort's explicit final assent to the current evidence", "materialCandidateId": ctx.Candidate.ID, "evidenceId": ctx.Evidence.ID, "round": ctx.Evidence.Round})
	if err != nil {
		return response{}, err
	}
	return response{APIVersion: protocolVersion, RunID: inv.RunID, Nonce: inv.Nonce, Role: inv.Request.Role, InputDigest: inv.InputDigest,
		Outcome: "passed", CandidateFiles: []candidateFile{}, EvidenceRefs: []string{},
		VerifierObservations: []observation{{Subject: "government-vote", Outcome: "passed", Detail: string(detail)}}, Uncertainty: []string{}}, nil
}

func allowed(ctx actorContext, paths ...string) error {
	set := map[string]bool{}
	for _, p := range ctx.AllowedPaths {
		set[p] = true
	}
	for _, p := range paths {
		if !set[p] {
			return fmt.Errorf("required file %q is outside the frozen Writer assignment", p)
		}
	}
	return nil
}

func proposal(out response, ctx actorContext, files []candidateFile) (response, error) {
	set := map[string]bool{}
	for _, p := range ctx.AllowedPaths {
		set[p] = true
	}
	for _, file := range files {
		if !set[file.Path] {
			return response{}, fmt.Errorf("proposed file %q is outside the frozen Writer assignment", file.Path)
		}
		out.CandidateFiles = append(out.CandidateFiles, file)
		out.EvidenceRefs = append(out.EvidenceRefs, file.Path)
	}
	return out, nil
}

func sourceArtifact(artifacts []artifact) (source, error) {
	data := findArtifact(artifacts, "government.yaml")
	if len(data) == 0 {
		return source{}, errors.New("GovernmentSource bytes are absent")
	}
	var model source
	if err := json.Unmarshal(data, &model); err != nil {
		return source{}, fmt.Errorf("decode GovernmentSource: %w", err)
	}
	return model, nil
}

func findArtifact(artifacts []artifact, path string) []byte {
	for _, file := range artifacts {
		if file.Path == path {
			return file.Content
		}
	}
	return nil
}

func setRequirement(model source, name, statement, acceptance string) {
	for _, d := range model.Definitions {
		if d["kind"] == "Requirement" && metadataName(d) == name {
			spec, _ := d["spec"].(map[string]any)
			spec["statement"], spec["acceptance"] = statement, acceptance
			return
		}
	}
}

func setOpenQuestions(model source, name, questions string) {
	for _, d := range model.Definitions {
		if d["kind"] == "Requirement" && metadataName(d) == name {
			spec, _ := d["spec"].(map[string]any)
			spec["openQuestions"] = questions
			return
		}
	}
}

func setRealization(model source, name, role string) {
	for _, d := range model.Definitions {
		if d["kind"] == "Realization" && metadataName(d) == name {
			spec, _ := d["spec"].(map[string]any)
			spec["role"] = role
			return
		}
	}
}

func metadataName(d map[string]any) string {
	metadata, _ := d["metadata"].(map[string]any)
	name, _ := metadata["name"].(string)
	return name
}

func requirementStatement(model source, name string) string {
	for _, d := range model.Definitions {
		if d["kind"] == "Requirement" && metadataName(d) == name {
			spec, _ := d["spec"].(map[string]any)
			statement, _ := spec["statement"].(string)
			return statement
		}
	}
	return ""
}

const studioImplementation = "package studio\n\nfunc BookingTotal(seats, seatPrice, bookingFee int) int {\n\treturn seats*seatPrice + bookingFee\n}\n"

const candidateTest = `package studio

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestBookingRuleAndRealizationAgree(t *testing.T) {
	modelBytes, err := os.ReadFile("../government.yaml")
	if err != nil { t.Fatal(err) }
	var model struct { Definitions []struct { Kind string ` + "`json:\"kind\"`" + `; Metadata struct { Name string ` + "`json:\"name\"`" + ` } ` + "`json:\"metadata\"`" + `; Spec struct { Statement string ` + "`json:\"statement\"`" + ` } ` + "`json:\"spec\"`" + ` } ` + "`json:\"definitions\"`" + ` }
	if err := json.Unmarshal(modelBytes, &model); err != nil { t.Fatal(err) }
	statement := ""
	for _, d := range model.Definitions { if d.Kind == "Requirement" && d.Metadata.Name == "booking-rule" { statement = d.Spec.Statement } }
	rateBytes, err := os.ReadFile("../booking/rate.txt")
	if err != nil { t.Fatal(err) }
	var fee int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(rateBytes)), "fee=%d", &fee); err != nil { t.Fatal(err) }
	if fee != 2 { t.Fatalf("published booking fee = %d, want 2", fee) }
	if strings.Contains(statement, "exactly one fixed booking fee") {
		if got, want := BookingTotal(3, 4, fee), 14; got != want { t.Fatalf("three-seat booking total = %d, want %d", got, want) }
		if got, want := BookingTotal(1, 5, fee), 7; got != want { t.Fatalf("one-seat booking total = %d, want %d", got, want) }
		return
	}
	if got, want := BookingTotal(3, 4, fee), 12; got != want { t.Fatalf("prior three-seat total = %d, want %d", got, want) }
	if got, want := BookingTotal(1, 5, fee), 5; got != want { t.Fatalf("prior one-seat total = %d, want %d", got, want) }
}
`

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
