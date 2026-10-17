package realization

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func recordLimits() CandidateRecordLimits {
	return CandidateRecordLimits{EncodedBytes: 1 << 20, FileBytes: 1 << 20, ManifestEntries: 100, MetadataRows: 1000}
}
func recordPath(t *testing.T, s string) snapshot.Path {
	t.Helper()
	p, e := snapshot.NewPath(s)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func recordTree(t *testing.T, content string, mode snapshot.Mode) snapshot.Tree {
	t.Helper()
	entries := []snapshot.Entry{}
	for _, row := range []struct {
		path, content string
		mode          snapshot.Mode
	}{{"out/item.go", content, mode}, {"inputs/model.yaml", "fixed input", snapshot.Regular}, {"old/unexplained", "old", snapshot.Regular}} {
		e, err := snapshot.NewEntry(recordPath(t, row.path), row.mode, []byte(row.content))
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, e)
	}
	tree, err := snapshot.NewTree(entries)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}
func recordRegistry(t *testing.T, who Owner, why string) Registry {
	t.Helper()
	a, err := NewArtifact(recordPath(t, "out/item.go"), Code, []model.Identity{{Package: "lending", ID: "Item"}, {Package: "lending", ID: "borrow"}}, why)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRegistration(who, a)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewRegistry([]Registration{r})
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

// Compile the repository's normative Core as a real imported model and a small
// producer-local lending fixture. No RUN import or hand-made Passed result.
func recordCompiled(t *testing.T) model.Result {
	t.Helper()
	core := model.SourceModel{Origin: "record-core", ManifestPath: "model.yaml", Entries: []model.Entry{
		{Path: "model.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nid: record-core\nversion: '1.0'\nroot: model\ndependencies: {}\n")},
		{Path: "model-lock.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nmodels: {}\n")},
	}}
	coreDirectory := filepath.Join("..", "..", ".markitect", "model", "markitect", "core")
	files, err := os.ReadDir(coreDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
			t.Fatalf("unexpected Core fixture entry: %s", f.Name())
		}
		content, err := os.ReadFile(filepath.Join(coreDirectory, f.Name()))
		if err != nil {
			t.Fatal(err)
		}
		core.Entries = append(core.Entries, model.Entry{Path: "model/markitect/core/" + f.Name(), Kind: model.Regular, Bytes: content})
	}
	passed := model.Compile(model.Input{Root: core})
	if passed.Status != model.Passed {
		t.Fatalf("Core compilation: %+v", passed.Diagnostics)
	}
	local := model.SourceModel{Origin: "record-local", ManifestPath: "model.yaml", Entries: []model.Entry{
		{Path: "model.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nid: record-local\nroot: model\ndependencies: {record-core: '1.0'}\n")},
		{Path: "model-lock.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nmodels:\n  record-core:\n    version: '1.0'\n    digest: " + passed.Models[0].Digest + "\n")},
		{Path: "model/lending/item.yaml", Kind: model.Regular, Bytes: []byte("imports: [{package: markitect/core, as: Core}]\nid: Item\nkind: Core.Kind\nbody: An item can be borrowed.\nspec: {}\n")},
		{Path: "model/lending/borrow.yaml", Kind: model.Regular, Bytes: []byte("id: borrow\nkind: Item\nbody: Borrow an available item.\nspec: {}\n")},
		{Path: "model/empty", Kind: model.Directory},
	}}
	compiled := model.Compile(model.Input{Root: local, Dependencies: []model.SourceModel{core}})
	if compiled.Status != model.Passed {
		t.Fatalf("local compilation: %+v", compiled.Diagnostics)
	}
	return compiled
}

type recordFixture struct {
	compiled  model.Result
	base      CandidateBase
	candidate Candidate
	classes   []Classification
	rows      []DefinitionAssessment
}

func recordSetup(t *testing.T, changed bool) recordFixture {
	t.Helper()
	compiled := recordCompiled(t)
	who, err := PackageOwner("lending")
	if err != nil {
		t.Fatal(err)
	}
	baseTree := recordTree(t, "before", snapshot.Executable)
	registry := recordRegistry(t, who, "Owns lending implementation.")
	dir, err := snapshot.DirectoryAt(recordPath(t, "out"))
	if err != nil {
		t.Fatal(err)
	}
	area, err := NewCreationArea(who, dir)
	if err != nil {
		t.Fatal(err)
	}
	protection, err := ProtectFile(recordPath(t, "inputs/model.yaml"), "Canonical captured input.")
	if err != nil {
		t.Fatal(err)
	}
	protectedDir, err := snapshot.DirectoryAt(recordPath(t, "secrets"))
	if err != nil {
		t.Fatal(err)
	}
	dirProtection, err := ProtectDirectory(protectedDir, "Host protected boundary.")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := NewPathFacts(compiled, registry, baseTree, []CreationArea{area}, []Protection{protection, dirProtection})
	if err != nil {
		t.Fatal(err)
	}
	base, err := NewCandidateBase(compiled, facts)
	if err != nil {
		t.Fatal(err)
	}
	tree := baseTree
	if changed {
		tree = recordTree(t, "after!", snapshot.Regular)
	}
	capture := CaptureCandidate(base, who, PathClaims{}, tree, registry)
	candidate, ok := capture.Candidate()
	if !ok {
		t.Fatal(capture.Findings())
	}
	class, err := NewClassification(recordPath(t, "inputs/model.yaml"), CanonicalInput, "Exact canonical input.")
	if err != nil {
		t.Fatal(err)
	}
	rows := []DefinitionAssessment{}
	for _, id := range []string{"Item", "borrow"} {
		rows = append(rows, DefinitionAssessment{Definition: model.Identity{Package: "lending", ID: id}, Disposition: Realized, Artifacts: []snapshot.Path{recordPath(t, "out/item.go")}, Uses: []model.Identity{}, Reason: "Explains local realization.", Evidence: []EvidenceUse{{Ref: "host:check", Relevance: "Checks actual item behavior."}}})
	}
	return recordFixture{compiled, base, candidate, []Classification{class}, rows}
}
func (f recordFixture) record(t *testing.T) CandidateRecord {
	t.Helper()
	r, e := NewCandidateRecord(f.compiled, f.candidate, f.classes, f.rows, recordLimits())
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func recordWireFrom(t *testing.T, r CandidateRecord) candidateRecordWire {
	t.Helper()
	var w candidateRecordWire
	if err := json.Unmarshal(r.Bytes(), &w); err != nil {
		t.Fatal(err)
	}
	return w
}
func recordEncode(t *testing.T, w candidateRecordWire) []byte {
	t.Helper()
	b, e := json.Marshal(w)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func refuseRecord(t *testing.T, data []byte, limits CandidateRecordLimits, want error) {
	t.Helper()
	r, e := DecodeCandidateRecord(data, limits)
	if e == nil || want != nil && !errors.Is(e, want) || r.Valid() || r.Observation().Candidate().Valid() || r.Observation().Census().Basis().Valid() {
		t.Fatalf("partial/successful refusal: %v %+v", e, r)
	}
}

func TestCandidateRecordRoundTripActualCompiledNoChangeAndEdit(t *testing.T) {
	if (snapshot.Entry{}).ByteLength() != 0 || (snapshot.Tree{}).EntryCount() != 0 {
		t.Fatal("zero snapshot size/count is nonzero")
	}

	for _, changed := range []bool{false, true} {
		t.Run(map[bool]string{false: "no change", true: "existing edit"}[changed], func(t *testing.T) {
			f := recordSetup(t, changed)
			record := f.record(t)
			decoded, err := DecodeCandidateRecord(record.Bytes(), recordLimits())
			if err != nil {
				t.Fatal(err)
			}

			got := decoded.Observation().Candidate()
			if got.Tree().EntryCount() != len(got.Tree().Entries()) {
				t.Fatal("snapshot entry count differs from manifest")
			}
			for _, e := range got.Tree().Entries() {
				if e.ByteLength() != int64(len(e.Bytes())) {
					t.Fatal("snapshot byte length differs from content")
				}
			}
			if !got.Base().Equal(f.base) || got.Owner() != f.candidate.Owner() || !got.Tree().Equal(f.candidate.Tree()) || !reflect.DeepEqual(got.Registry(), f.candidate.Registry()) || !reflect.DeepEqual(got.Changes(), f.candidate.Changes()) || !reflect.DeepEqual(got.RegistryChanges(), f.candidate.RegistryChanges()) || !slices.Equal(got.TouchedPaths(), f.candidate.TouchedPaths()) || len(got.Claims()) != 0 {
				t.Fatal("exact candidate subject lost")
			}
			live := CheckCandidateCensus(f.compiled, f.candidate, f.classes, f.rows)
			if !reflect.DeepEqual(decoded.Observation().Census(), live) {
				t.Fatalf("historical census differs from live: %+v %+v", decoded.Observation().Census(), live)
			}
			if live.Status() != CensusStructurallyComplete || live.Accounting().Status() != AccountingInvalid {
				t.Fatal("old unrelated defect became global census gate")
			}
			w := recordWireFrom(t, record)
			want := 3
			if changed {
				want = 4
			}
			if len(w.Data) != want {
				t.Fatalf("unchanged bytes not deduplicated: %d", len(w.Data))
			}
			if record.Digest() != decoded.Digest() || !bytes.Equal(record.Bytes(), decoded.Bytes()) {
				t.Fatal("identity/encoding changed on decode")
			}
			exposed := decoded.Bytes()
			exposed[0] = 'x'
			census := decoded.Observation().Census()
			rows := census.Assessments()
			rows[0].Evidence[0].Relevance = "mutated"
			rows[0].Artifacts[0] = recordPath(t, "other")
			candidateBytes := got.Tree().Entries()[0].Bytes()
			if len(candidateBytes) > 0 {
				candidateBytes[0] = 'x'
			}
			if !bytes.Equal(record.Bytes(), decoded.Bytes()) || !reflect.DeepEqual(census.Assessments(), live.Assessments()) {
				t.Fatal("output alias mutated record")
			}
		})
	}
}

func TestCandidateRecordWholeSubjectIdentityAndProposalStatuses(t *testing.T) {
	f := recordSetup(t, false)
	original := f.record(t)
	registry := recordRegistry(t, f.candidate.Owner(), "Changed registry explanation only.")
	capture := CaptureCandidate(f.base, f.candidate.Owner(), PathClaims{}, f.candidate.Tree(), registry)
	candidate, ok := capture.Candidate()
	if !ok {
		t.Fatal(capture.Findings())
	}
	f.candidate = candidate
	why := f.record(t)
	if !why.Observation().Candidate().Tree().Equal(original.Observation().Candidate().Tree()) || why.Digest() == original.Digest() || len(why.Observation().Candidate().RegistryChanges()) != 1 {
		t.Fatal("Why-only difference not bound")
	}
	f.rows[0].Reason = "Changed assessment only."
	assessment := f.record(t)
	if assessment.Digest() == why.Digest() {
		t.Fatal("assessment omitted from whole payload identity")
	}
	slices.Reverse(f.rows)
	if f.record(t).Digest() == assessment.Digest() {
		t.Fatal("proposal order lost")
	}
	slices.Reverse(f.rows)
	f.rows = f.rows[:1]
	missing := f.record(t).Observation().Census()
	if missing.Status() != CensusIncomplete || len(missing.Missing()) != 1 {
		t.Fatal("missing own row was not retained incomplete")
	}
	f.rows[0].Uses = []model.Identity{{Package: "markitect/core", ID: "Kind"}}
	invalid := f.record(t).Observation().Census()
	if invalid.Status() != CensusInvalid || !slices.ContainsFunc(invalid.Findings(), func(f Finding) bool { return f.Code == "census.invalid_use" }) {
		t.Fatal("imported use not recomputed invalid")
	}
}

func TestCandidateRecordClosedCanonicalCorruptionAndConsistency(t *testing.T) {
	f := recordSetup(t, false)
	record := f.record(t)
	for name, change := range map[string]func(*candidateRecordWire){
		"hash corruption": func(w *candidateRecordWire) { w.Data[0].Digest = strings.Repeat("0", 64) },
		"byte corruption": func(w *candidateRecordWire) { w.Data[0].Bytes[0] ^= 1 },
		"bad mode":        func(w *candidateRecordWire) { w.ResultTree[0].Mode = 99 },
		"bad length":      func(w *candidateRecordWire) { w.ResultTree[0].Length++ },
		"missing data":    func(w *candidateRecordWire) { w.Data = w.Data[:len(w.Data)-1] },
		"unused data": func(w *candidateRecordWire) {
			w.Data = append(w.Data, recordData{Digest: fmt.Sprintf("%x", sha256.Sum256([]byte("extra"))), Bytes: []byte("extra")})
		},
		"duplicate data":           func(w *candidateRecordWire) { w.Data = append(w.Data, w.Data[0]) },
		"duplicate manifest path":  func(w *candidateRecordWire) { w.BaseTree = append(w.BaseTree, w.BaseTree[0]) },
		"duplicate registration":   func(w *candidateRecordWire) { w.ResultRegistry = append(w.ResultRegistry, w.ResultRegistry[0]) },
		"foreign definition model": func(w *candidateRecordWire) { w.Models[0].Definitions[0].ModelID = "wrong" },
		"absent package":           func(w *candidateRecordWire) { w.Models[0].Packages = []string{} },
		"duplicate package": func(w *candidateRecordWire) {
			w.Models[0].Packages = append(w.Models[0].Packages, w.Models[0].Packages[0])
		},
		"duplicate definition": func(w *candidateRecordWire) {
			w.Models[0].Definitions = append(w.Models[0].Definitions, w.Models[0].Definitions[0])
		},
		"no local": func(w *candidateRecordWire) {
			for i := range w.Models {
				w.Models[i].Local = false
			}
		},
		"null required list":       func(w *candidateRecordWire) { w.Claims = nil },
		"unused claim":             func(w *candidateRecordWire) { w.Claims = []recordClaim{{"made-up-grant", "out/unused.go", Code}} },
		"duplicate assessment":     func(w *candidateRecordWire) { w.Assessments = append(w.Assessments, w.Assessments[0]) },
		"duplicate classification": func(w *candidateRecordWire) { w.Classifications = append(w.Classifications, w.Classifications[0]) },
		"duplicate protection":     func(w *candidateRecordWire) { w.Protections = append(w.Protections, w.Protections[0]) },
		"unsorted models":          func(w *candidateRecordWire) { slices.Reverse(w.Models) },
		"unsupported version":      func(w *candidateRecordWire) { w.Version++ },
	} {
		t.Run(name, func(t *testing.T) {
			w := recordWireFrom(t, record)
			change(&w)
			refuseRecord(t, recordEncode(t, w), recordLimits(), ErrInvalidCandidateRecord)
		})
	}
	for name, data := range map[string][]byte{
		"unknown field":      bytes.Replace(record.Bytes(), []byte(`{"version":1,`), []byte(`{"unknown":1,"version":1,`), 1),
		"duplicate field":    bytes.Replace(record.Bytes(), []byte(`{"version":1,`), []byte(`{"version":1,"version":1,`), 1),
		"missing field":      bytes.Replace(record.Bytes(), []byte(`"claims":[],`), nil, 1),
		"whitespace variant": append(record.Bytes(), '\n'),
		"truncated":          record.Bytes()[:len(record.Bytes())-1],
	} {
		t.Run(name, func(t *testing.T) { refuseRecord(t, data, recordLimits(), ErrInvalidCandidateRecord) })
	}
}

func TestHistoricalCandidateRecordReusesCaptureRefusals(t *testing.T) {
	f := recordSetup(t, true)
	record := f.record(t)
	for name, change := range map[string]func(*candidateRecordWire){
		"protected existing edit": func(w *candidateRecordWire) {
			w.Protections = append(w.Protections, recordProtection{File: "out/item.go", Why: "Captured protection."})
		},
		"foreign existing owner": func(w *candidateRecordWire) { w.BaseRegistry[0].Owner = recordOwner{Root: true} },
		"new unclaimed path": func(w *candidateRecordWire) {
			w.ResultTree = append(w.ResultTree, w.ResultTree[len(w.ResultTree)-1])
			w.ResultTree[len(w.ResultTree)-1].Path = "out/new.go"
			w.ResultRegistry = append(w.ResultRegistry, w.ResultRegistry[0])
			w.ResultRegistry[len(w.ResultRegistry)-1].Path = "out/new.go"
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := recordWireFrom(t, record)
			change(&w)
			encoded := recordEncode(t, w)
			refuseRecord(t, encoded, recordLimits(), ErrInvalidCandidateRecord)
			_, err := DecodeCandidateRecord(encoded, recordLimits())
			if !strings.Contains(err.Error(), "candidate capture refused") {
				t.Fatalf("did not exercise shared capture: %v", err)
			}
		})
	}
	// Public live entrypoint still requires a real passed matching result.
	failed := f.compiled
	failed.Status = model.Failed
	r, e := NewCandidateRecord(failed, f.candidate, f.classes, f.rows, recordLimits())
	if e == nil || r.Valid() {
		t.Fatal("failed compilation recorded")
	}
	mismatch := f.compiled
	mismatch.Models = append([]model.ModelResult{}, f.compiled.Models...)
	mismatch.Models[0].Digest = "sha256:" + strings.Repeat("0", 64)
	r, e = NewCandidateRecord(mismatch, f.candidate, f.classes, f.rows, recordLimits())
	if e == nil || r.Valid() {
		t.Fatal("mismatched result recorded")
	}
}

func TestCandidateRecordLimitsRefuseWithoutPartialObservation(t *testing.T) {
	f := recordSetup(t, false)
	record := f.record(t)
	for name, limits := range map[string]CandidateRecordLimits{
		"zero":       {},
		"ceiling":    {EncodedBytes: 65 << 20, FileBytes: 1 << 20, ManifestEntries: 100, MetadataRows: 1000},
		"encoded":    {EncodedBytes: int64(len(record.Bytes()) - 1), FileBytes: 1 << 20, ManifestEntries: 100, MetadataRows: 1000},
		"file bytes": {EncodedBytes: 1 << 20, FileBytes: 1, ManifestEntries: 100, MetadataRows: 1000},
		"manifest":   {EncodedBytes: 1 << 20, FileBytes: 1 << 20, ManifestEntries: 5, MetadataRows: 1000},
		"metadata":   {EncodedBytes: 1 << 20, FileBytes: 1 << 20, ManifestEntries: 100, MetadataRows: 1},
	} {
		t.Run(name, func(t *testing.T) {
			refuseRecord(t, record.Bytes(), limits, ErrCandidateRecordLimit)
			r, e := NewCandidateRecord(f.compiled, f.candidate, f.classes, f.rows, limits)
			if !errors.Is(e, ErrCandidateRecordLimit) || r.Valid() {
				t.Fatalf("live limit: %v", e)
			}
		})
	}
	// A huge declared length is rejected by the token pass before file allocation.
	w := recordWireFrom(t, record)
	w.BaseTree[0].Length = 1 << 62
	refuseRecord(t, recordEncode(t, w), recordLimits(), ErrCandidateRecordLimit)
}

func TestCandidateRecordLiveLimitsPrecedeContentAllocation(t *testing.T) {
	f := recordSetup(t, false)
	large := recordTree(t, strings.Repeat("x", 8<<20), snapshot.Regular)
	facts, err := NewPathFacts(f.compiled, f.base.Registry(), large, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	base, err := NewCandidateBase(f.compiled, facts)
	if err != nil {
		t.Fatal(err)
	}
	capture := CaptureCandidate(base, f.candidate.Owner(), PathClaims{}, large, f.base.Registry())
	candidate, ok := capture.Candidate()
	if !ok {
		t.Fatal(capture.Findings())
	}
	for _, limits := range []CandidateRecordLimits{
		{EncodedBytes: 64 << 10, FileBytes: 64 << 20, ManifestEntries: 100, MetadataRows: 1000},
		{EncodedBytes: 1 << 20, FileBytes: 1, ManifestEntries: 100, MetadataRows: 1000},
	} {
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		r, err := NewCandidateRecord(f.compiled, candidate, f.classes, f.rows, limits)
		runtime.ReadMemStats(&after)
		if !errors.Is(err, ErrCandidateRecordLimit) || r.Valid() {
			t.Fatalf("limit refusal: %v", err)
		}
		// A generous allocation ceiling distinguishes a structural refusal from the
		// previous full 8 MiB content copy/serialization. No timing claim is made.
		if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 1<<20 {
			t.Fatalf("allocated %d bytes before budget refusal", allocated)
		}
	}
}

func TestCandidateRecordJSONBudgetMatchesCanonicalSize(t *testing.T) {
	f := recordSetup(t, false)
	f.rows[0].Reason = "Quotes \" and slash \\ and controls \x01 \b \f \n \r \t and HTML <>& and Unicode 日本語 \u2028 \u2029"
	record := f.record(t)
	wire := recordWireFrom(t, record)
	exact := int64(len(record.Bytes()))
	if err := boundRecordJSON(wire, exact); err != nil {
		t.Fatalf("exact JSON size refused: %v", err)
	}
	if err := boundRecordJSON(wire, exact-1); !errors.Is(err, ErrCandidateRecordLimit) {
		t.Fatalf("short JSON budget accepted: %v", err)
	}
	limits := recordLimits()
	limits.EncodedBytes = exact
	if _, err := NewCandidateRecord(f.compiled, f.candidate, f.classes, f.rows, limits); err != nil {
		t.Fatalf("exact live byte budget refused: %v", err)
	}
}

func TestCandidateRecordKeepsClaims(t *testing.T) {
	f := recordSetup(t, false)
	path := recordPath(t, "out/new.go")
	request, err := NewPathRequest("host:new-file", f.candidate.Owner(), path, Code)
	if err != nil {
		t.Fatal(err)
	}
	claims, decision := ReserveNewPath(f.base.value.facts, PathClaims{}, request)
	if decision.Status() != PathAdmissible {
		t.Fatal(decision.Findings())
	}
	e, err := snapshot.NewEntry(path, snapshot.Regular, []byte("new file"))
	if err != nil {
		t.Fatal(err)
	}
	tree, err := snapshot.NewTree(append(f.base.Tree().Entries(), e))
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := NewArtifact(path, Code, []model.Identity{{Package: "lending", ID: "Item"}}, "New candidate artifact.")
	if err != nil {
		t.Fatal(err)
	}
	registration, err := NewRegistration(f.candidate.Owner(), artifact)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewRegistry(append(f.base.Registry().Entries(), registration))
	if err != nil {
		t.Fatal(err)
	}
	result := CaptureCandidate(f.base, f.candidate.Owner(), claims, tree, registry)
	candidate, ok := result.Candidate()
	if !ok {
		t.Fatal(result.Findings())
	}
	if len(candidate.Claims()) != 1 {
		t.Fatal("fixture did not use a claim")
	}
	record, err := NewCandidateRecord(f.compiled, candidate, f.classes, f.rows, recordLimits())
	if err != nil {
		t.Fatalf("a claimed new file was not recorded: %v", err)
	}
	got := record.Observation().Candidate()
	if !got.Valid() || !slices.Equal(got.TouchedPaths(), candidate.TouchedPaths()) || len(got.Claims()) != 1 || got.Claims()[0].Request() != request {
		t.Fatalf("decoded claims = %+v, want the recorded request", got.Claims())
	}
	for name, change := range map[string]func(*candidateRecordWire){
		"missing claim": func(w *candidateRecordWire) { w.Claims = []recordClaim{} },
		"other form":    func(w *candidateRecordWire) { w.Claims[0].Form = Doc },
		"duplicate":     func(w *candidateRecordWire) { w.Claims = append(w.Claims, w.Claims[0]) },
		"extra claim": func(w *candidateRecordWire) {
			w.Claims = append(w.Claims, recordClaim{"host:other", "out/other.go", Code})
		},
		"protected path": func(w *candidateRecordWire) { w.Claims[0].Path = ".markitect/state/x.go" },
	} {
		t.Run(name, func(t *testing.T) {
			w := recordWireFrom(t, record)
			change(&w)
			if _, err := DecodeCandidateRecord(recordEncode(t, w), recordLimits()); err == nil {
				t.Fatal("a changed claim was accepted")
			}
		})
	}
}

func TestCandidateRecordManifestBudgetPrecedesManifestCopy(t *testing.T) {
	f := recordSetup(t, false)
	entries := make([]snapshot.Entry, 50000)
	for i := range entries {
		e, err := snapshot.NewEntry(recordPath(t, fmt.Sprintf("old/%05d", i)), snapshot.Regular, nil)
		if err != nil {
			t.Fatal(err)
		}
		entries[i] = e
	}
	tree, err := snapshot.NewTree(entries)
	if err != nil {
		t.Fatal(err)
	}
	facts, err := NewPathFacts(f.compiled, Registry{}, tree, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	base, err := NewCandidateBase(f.compiled, facts)
	if err != nil {
		t.Fatal(err)
	}
	capture := CaptureCandidate(base, f.candidate.Owner(), PathClaims{}, tree, Registry{})
	candidate, ok := capture.Candidate()
	if !ok {
		t.Fatal(capture.Findings())
	}
	for _, limits := range []CandidateRecordLimits{
		{EncodedBytes: 64 << 20, FileBytes: 64 << 20, ManifestEntries: 1, MetadataRows: 1000000},
		{EncodedBytes: 64 << 20, FileBytes: 64 << 20, ManifestEntries: 100000, MetadataRows: 100},
	} {
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		record, err := NewCandidateRecord(f.compiled, candidate, nil, nil, limits)
		runtime.ReadMemStats(&after)
		if !errors.Is(err, ErrCandidateRecordLimit) || record.Valid() {
			t.Fatalf("manifest limit: %v", err)
		}
		// Entries() would copy 50,000 entry structs (>1 MiB) before the old refusal.
		if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 1<<20 {
			t.Fatalf("copied manifest before refusal: %d bytes", allocated)
		}
	}
}
