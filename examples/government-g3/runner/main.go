// Command runner supplies deterministic external actors for the public G3
// recursion fixture. Its receipts prove process mechanics, not agent quality.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const protocolVersion = "markitect.example.org/agent-execution/v1alpha1"

type artifact struct {
	Path    string `json:"path"`
	Mode    string `json:"mode"`
	Digest  string `json:"digest"`
	Content []byte `json:"content"`
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
type identity struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
}
type actorContext struct {
	Phase            string            `json:"phase"`
	RunID            string            `json:"runId"`
	Area             identity          `json:"area"`
	Attempt          int               `json:"attempt"`
	Repair           string            `json:"repair"`
	AllowedPaths     []string          `json:"allowedPaths"`
	InputPaths       []string          `json:"inputPaths"`
	Subjects         []identity        `json:"subjects"`
	InputDigest      string            `json:"inputDigest"`
	ChildReports     []json.RawMessage `json:"childReports"`
	Definitions      []json.RawMessage `json:"definitions"`
	Work             json.RawMessage   `json:"work"`
	DelegationDigest string            `json:"delegationDigest"`
	NodeDigest       string            `json:"nodeDigest"`
	Limits           json.RawMessage   `json:"limits"`
	Workspace        string            `json:"workspace"`
	Candidate        struct {
		ID             string `json:"id"`
		SnapshotDigest string `json:"snapshotDigest"`
	} `json:"candidate"`
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

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fatal(errors.New("usage: runner quantity|price|root|assent|assent-unaffected [parallel-token]"))
	}
	mode := os.Args[1]
	if mode != "quantity" && mode != "price" && mode != "outside-scope" && mode != "bridge" && mode != "root" && mode != "assent" && mode != "assent-unaffected" {
		fatal(fmt.Errorf("unsupported mode %q", mode))
	}
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 32<<20))
	decoder.DisallowUnknownFields()
	var inv invocation
	if err := decoder.Decode(&inv); err != nil {
		fatal(fmt.Errorf("decode invocation: %w", err))
	}
	if decoder.Decode(new(any)) != io.EOF {
		fatal(errors.New("invocation must contain one JSON value"))
	}
	if inv.APIVersion != protocolVersion || inv.RunID == "" || inv.Nonce == "" || inv.InputDigest == "" {
		fatal(errors.New("invocation binding is incomplete"))
	}
	var resp response
	var err error
	var ctx actorContext
	if mode == "assent" || mode == "assent-unaffected" {
		resp, err = vote(mode, inv)
	} else {
		ctx, err = decodeContext(inv.Request.Context)
		if err == nil {
			resp, err = execute(mode, inv, ctx)
		}
	}
	if err == nil && ctx.Phase == "execute" && (mode == "quantity" || mode == "price" || mode == "outside-scope") {
		if len(os.Args) != 3 {
			err = errors.New("child actor requires a unique parallel token")
		} else {
			side := mode
			if side == "outside-scope" {
				side = "quantity"
			}
			err = awaitSibling(mode, side, os.Args[2], inv.RunID, inv.InputDigest, ctx.InputDigest, ctx.Attempt)
		}
	}
	if err != nil {
		fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		fatal(err)
	}
}

// Both independent child processes publish a unique external start marker and
// wait for the other process. Successful receipts therefore prove overlapping
// native processes rather than infer concurrency from elapsed times.
func awaitSibling(mode, side, token, runID, inputDigest, snapshotDigest string, attempt int) error {
	if token == "" || strings.ContainsAny(token, `/\\:`) {
		return errors.New("parallel token must be a nonempty simple identifier")
	}
	if snapshotDigest == "" {
		return errors.New("child actor requires the shared parent candidate snapshot digest")
	}
	generation := safeToken(snapshotDigest) + "-a" + strconv.Itoa(attempt)
	correlation := token + "-" + generation
	markerDir := filepath.Join(os.TempDir(), "markitect-government-g3-parallel")
	if err := os.MkdirAll(markerDir, 0o700); err != nil {
		return err
	}
	own := filepath.Join(markerDir, correlation+"-"+side+".started")
	sibling := "quantity"
	if side == "quantity" {
		sibling = "price"
	}
	other := filepath.Join(markerDir, correlation+"-"+sibling+".started")
	started := time.Now().UTC()
	event := func(name string, at time.Time) error {
		return writeEvent(map[string]any{"event": name, "token": token, "generation": generation, "attempt": attempt, "side": side, "mode": mode, "pid": os.Getpid(), "runId": runID, "inputDigest": inputDigest, "markerPath": own, "atUtc": at.UTC().Format(time.RFC3339Nano)})
	}
	if err := event("process-start", started); err != nil {
		return err
	}
	f, err := os.OpenFile(own, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create unique child start marker: %w", err)
	}
	marker, err := json.Marshal(map[string]any{"mode": mode, "pid": os.Getpid(), "runId": runID, "startedUtc": started.Format(time.RFC3339Nano), "snapshotDigest": generation})
	if err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(marker); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(other); err == nil {
			observed := time.Now().UTC()
			if err := event("sibling-process-overlap", observed); err != nil {
				return err
			}
			time.Sleep(800 * time.Millisecond)
			finished := time.Now().UTC()
			if err := appendMarker(own, map[string]any{"siblingObservedUtc": observed.Format(time.RFC3339Nano), "finishedUtc": finished.Format(time.RFC3339Nano)}); err != nil {
				return err
			}
			return event("process-finish", finished)
		} else if !os.IsNotExist(err) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return errors.New("independent sibling process did not overlap within ten seconds")
}

func safeToken(value string) string {
	var out strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func writeEvent(event map[string]any) error {
	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	privateLog := os.Getenv("MARKITECT_AGENT_PRIVATE_LOG")
	if privateLog == "" {
		return errors.New("Host did not provide the bound private actor log path")
	}
	f, err := os.OpenFile(privateLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(encoded, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stderr, string(encoded))
	return err
}

func appendMarker(path string, event map[string]any) error {
	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append([]byte{'\n'}, encoded...)); err != nil {
		return err
	}
	return f.Sync()
}

func vote(mode string, inv invocation) (response, error) {
	if inv.Request.Role != "verifier" {
		return response{}, errors.New("Ressort actor requires verifier role")
	}
	var ctx voteContext
	if err := json.Unmarshal(inv.Request.Context, &ctx); err != nil {
		return response{}, fmt.Errorf("decode vote context: %w", err)
	}
	if ctx.Candidate.ID == "" || ctx.Evidence.ID == "" || ctx.Evidence.Round < 1 {
		return response{}, errors.New("vote must bind current candidate and evidence")
	}
	outcome := "assent"
	reason := "deterministic mechanics fixture explicitly evaluated the final root candidate"
	if mode == "assent-unaffected" {
		outcome = "assent-unaffected"
		reason = "deterministic fixture explicitly records this Ressort concern as unaffected by the final root candidate"
	}
	detail, err := json.Marshal(map[string]any{"outcome": outcome, "reason": reason, "materialCandidateId": ctx.Candidate.ID, "evidenceId": ctx.Evidence.ID, "round": ctx.Evidence.Round})
	if err != nil {
		return response{}, err
	}
	resp := response{APIVersion: protocolVersion, RunID: inv.RunID, Nonce: inv.Nonce, Role: inv.Request.Role,
		InputDigest: inv.InputDigest, Outcome: "passed", CandidateFiles: []candidateFile{}, EvidenceRefs: []string{},
		VerifierObservations: []observation{{Subject: "government-vote", Outcome: "passed", Detail: string(detail)}}, Uncertainty: []string{}}
	return resp, nil
}

func execute(mode string, inv invocation, ctx actorContext) (response, error) {
	resp := response{APIVersion: protocolVersion, RunID: inv.RunID, Nonce: inv.Nonce, Role: inv.Request.Role,
		InputDigest: inv.InputDigest, CandidateFiles: []candidateFile{}, EvidenceRefs: []string{},
		VerifierObservations: []observation{}, Uncertainty: []string{}}
	if ctx.Phase == "execute" {
		if inv.Request.Role != "executor" {
			return response{}, errors.New("execute context requires executor role")
		}
		resp.Outcome = "proposed"
		switch mode {
		case "bridge":
			if ctx.Area.Name != "bridge" {
				return response{}, errors.New("bridge actor received a different Area")
			}
			for _, path := range ctx.AllowedPaths {
				if path == "integration/invoice.txt" {
					total := 11
					if ctx.Attempt > 0 || strings.TrimSpace(ctx.Repair) != "" {
						quantity, err := readValue(inv.Request.Artifacts, "quantity/quantity.txt", "units=")
						if err != nil {
							return response{}, err
						}
						price, err := readValue(inv.Request.Artifacts, "price/price.txt", "unit=")
						if err != nil {
							return response{}, err
						}
						total = quantity * price
					}
					return propose(resp, ctx, path, fmt.Sprintf("total=%d\n", total))
				}
			}
			return resp, nil
		case "quantity":
			if ctx.Area.Name != "quantity" {
				return response{}, errors.New("quantity actor received a different Area")
			}
			time.Sleep(800 * time.Millisecond)
			return propose(resp, ctx, "quantity/quantity.txt", "units=3\n")
		case "price":
			if ctx.Area.Name != "price" {
				return response{}, errors.New("price actor received a different Area")
			}
			time.Sleep(800 * time.Millisecond)
			return propose(resp, ctx, "price/price.txt", "unit=4\n")
		case "outside-scope":
			if ctx.Area.Name != "quantity" {
				return response{}, errors.New("outside-scope actor is assigned only to the quantity Area")
			}
			resp.CandidateFiles = append(resp.CandidateFiles, candidateFile{Path: "integration/invoice.txt", Mode: "0644", Content: "total=99\n"})
			for _, item := range inv.Request.Artifacts {
				if item.Path == "quantity/quantity.txt" {
					resp.EvidenceRefs = append(resp.EvidenceRefs, item.Path)
				}
			}
			if len(resp.EvidenceRefs) == 0 {
				return response{}, errors.New("outside-scope negative requires the child's actual input artifact")
			}
			return resp, nil
		case "root":
			if ctx.Area.Name != "root" {
				return response{}, errors.New("root actor received a different Area")
			}
			// The initial root integration deliberately has a locally plausible but
			// compositionally wrong amount. A retry must use actual child artifact
			// bytes and repair feedback to derive the correct total.
			total := 11
			if ctx.Attempt > 0 || strings.TrimSpace(ctx.Repair) != "" {
				quantity, err := readValue(inv.Request.Artifacts, "quantity/quantity.txt", "units=")
				if err != nil {
					return response{}, err
				}
				price, err := readValue(inv.Request.Artifacts, "price/price.txt", "unit=")
				if err != nil {
					return response{}, err
				}
				total = quantity * price
			}
			return propose(resp, ctx, "integration/invoice.txt", fmt.Sprintf("total=%d\n", total))
		default:
			return response{}, fmt.Errorf("mode %q cannot execute an Area", mode)
		}
	}
	if ctx.Phase != "review" || inv.Request.Role != "verifier" {
		return response{}, errors.New("review context requires verifier role")
	}
	if mode == "assent" || mode == "assent-unaffected" {
		return response{}, errors.New("Ressort actor cannot serve as Area reviewer")
	}
	resp.Outcome = "passed"
	passed := true
	detail := "child-owned artifact satisfies its local acceptance condition"
	switch ctx.Area.Name {
	case "quantity":
		value, err := readValue(inv.Request.Artifacts, "quantity/quantity.txt", "units=")
		if err != nil {
			return response{}, err
		}
		passed = value == 3
	case "price":
		value, err := readValue(inv.Request.Artifacts, "price/price.txt", "unit=")
		if err != nil {
			return response{}, err
		}
		passed = value == 4
	case "root":
		quantity, err := readValue(inv.Request.Artifacts, "quantity/quantity.txt", "units=")
		if err != nil {
			return response{}, err
		}
		price, err := readValue(inv.Request.Artifacts, "price/price.txt", "unit=")
		if err != nil {
			return response{}, err
		}
		total, err := readValue(inv.Request.Artifacts, "integration/invoice.txt", "total=")
		if err != nil {
			return response{}, err
		}
		passed = total == quantity*price
		detail = fmt.Sprintf("parent composed-byte check: total=%d, quantity=%d, unit=%d; expected total=%d", total, quantity, price, quantity*price)
		if len(ctx.ChildReports) < 1 {
			return response{}, errors.New("root review requires actual child report lineage")
		}
	case "bridge":
		if len(ctx.ChildReports) == 0 {
			return response{}, errors.New("bridge review requires actual child lineage")
		}
		quantity, err := readValue(inv.Request.Artifacts, "quantity/quantity.txt", "units=")
		if err != nil {
			return response{}, err
		}
		writesInvoice := false
		for _, path := range ctx.AllowedPaths {
			if path == "integration/invoice.txt" {
				writesInvoice = true
			}
		}
		price, priceErr := readValue(inv.Request.Artifacts, "price/price.txt", "unit=")
		total, totalErr := readValue(inv.Request.Artifacts, "integration/invoice.txt", "total=")
		if writesInvoice {
			if len(ctx.ChildReports) < 2 {
				return response{}, errors.New("bridge invoice review requires both actual child reports")
			}
			if priceErr != nil || totalErr != nil {
				return response{}, errors.New("bridge invoice review requires actual quantity, price, and invoice bytes")
			}
			passed = total == quantity*price
			detail = fmt.Sprintf("bridge composed-byte check: total=%d, quantity=%d, unit=%d; expected total=%d", total, quantity, price, quantity*price)
		} else if priceErr == nil && totalErr == nil {
			passed = total == quantity*price
			detail = fmt.Sprintf("bridge composed-byte check: total=%d, quantity=%d, unit=%d; expected total=%d", total, quantity, price, quantity*price)
		} else {
			detail = fmt.Sprintf("structural bridge reviewed exact descendant quantity=%d bytes and its child report", quantity)
		}
	default:
		return response{}, fmt.Errorf("unknown Area %q", ctx.Area.Name)
	}
	if !passed {
		resp.Outcome = "failed"
	}
	for _, subject := range inv.Request.ScopeIDs {
		outcome := "passed"
		if !passed {
			outcome = "failed"
		}
		resp.VerifierObservations = append(resp.VerifierObservations, observation{Subject: subject, Outcome: outcome, Detail: detail})
	}
	if len(resp.VerifierObservations) == 0 {
		return response{}, errors.New("Area review has no frozen subjects")
	}
	return resp, nil
}

func propose(resp response, ctx actorContext, path, content string) (response, error) {
	allowed := false
	for _, candidate := range ctx.AllowedPaths {
		if candidate == path {
			allowed = true
		}
	}
	if !allowed {
		return response{}, fmt.Errorf("actor path %q is outside its exact local writer paths", path)
	}
	resp.CandidateFiles = append(resp.CandidateFiles, candidateFile{Path: path, Mode: "0644", Content: content})
	resp.EvidenceRefs = append(resp.EvidenceRefs, path)
	return resp, nil
}

func readValue(artifacts []artifact, path, prefix string) (int, error) {
	for _, item := range artifacts {
		if item.Path != path {
			continue
		}
		value := strings.TrimSpace(string(item.Content))
		if !strings.HasPrefix(value, prefix) {
			break
		}
		parsed, err := strconv.Atoi(strings.TrimPrefix(value, prefix))
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", path, err)
		}
		return parsed, nil
	}
	return 0, fmt.Errorf("actual invocation artifacts omit %s", path)
}

func decodeContext(raw json.RawMessage) (actorContext, error) {
	var ctx actorContext
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	if err := decoder.Decode(&ctx); err != nil {
		return ctx, fmt.Errorf("decode Area context: %w", err)
	}
	if ctx.Phase != "execute" && ctx.Phase != "review" {
		return ctx, fmt.Errorf("unknown phase %q", ctx.Phase)
	}
	if ctx.Area.Kind != "Area" || ctx.Area.Namespace == "" || ctx.Area.Name == "" || ctx.Attempt < 0 {
		return ctx, errors.New("Area context identity or attempt is invalid")
	}
	if ctx.Phase == "execute" && len(ctx.AllowedPaths) == 0 {
		return ctx, errors.New("execute context has no exact local writer paths")
	}
	return ctx, nil
}

func fatal(err error) {
	if err != nil {
		_ = writeEvent(map[string]any{"event": "runner-error", "error": err.Error(), "pid": os.Getpid(), "atUtc": time.Now().UTC().Format(time.RFC3339Nano)})
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)
}
