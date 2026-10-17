package run_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestBindingFixesCompleteInputsAndOwnsOriginalBytes(t *testing.T) {
	input := fixture(t)
	b, err := run.Bind(input)
	if err != nil {
		t.Fatal(err)
	}
	if b.RequiredReviews() != 4 {
		t.Fatal("local/integration review floor", b.RequiredReviews())
	}
	for name, change := range map[string]func(*run.BindingInput){
		"guard":    func(i *run.BindingInput) { i.Source.Guard = "different" },
		"revision": func(i *run.BindingInput) { i.Source.Revision = "different" },
		"product":  func(i *run.BindingInput) { i.Source.ProductBuild = "different" },
		"original lock bytes": func(i *run.BindingInput) {
			for n := range i.ModelInput.Root.Entries {
				e := &i.ModelInput.Root.Entries[n]
				if e.Path == "model-lock.yaml" {
					e.Bytes = append([]byte("# same lock selections\r\n"), e.Bytes...)
				}
			}
		},
		"file mode": func(i *run.BindingInput) {
			es := i.Census.Entries()
			for n, e := range es {
				if e.Path().String() == "out/library.go" {
					es[n] = file(t, e.Path().String(), snapshot.Executable, string(e.Bytes()))
				}
			}
			i.Census = tree(t, es)
		},
		"artifact bytes": func(i *run.BindingInput) {
			es := i.Census.Entries()
			for n, e := range es {
				if e.Path().String() == "out/library.go" {
					es[n] = file(t, e.Path().String(), e.Mode(), "package changed\n")
				}
			}
			i.Census = tree(t, es)
		},
		"registry": func(i *run.BindingInput) {
			old := i.Registry.Entries()[0]
			a, _ := realization.NewArtifact(old.Artifact().Path(), old.Artifact().Form(), old.Artifact().Basis(), "Different explanation")
			r, _ := realization.NewRegistration(old.Owner(), a)
			i.Registry, _ = realization.NewRegistry([]realization.Registration{r})
		},
		"classification": func(i *run.BindingInput) {
			c := i.Classifications[0]
			i.Classifications[0], _ = realization.NewClassification(c.Path(), c.Class(), "Different input explanation")
		},
		"protection": func(i *run.BindingInput) {
			p, _ := realization.ProtectFile(i.Documents[0].File.Path(), "Additional reason")
			i.Protections = append(i.Protections, p)
		},
		"static profile": func(i *run.BindingInput) {
			i.Profiles = append(i.Profiles, organization.ProfileFact{Name: "other", Identity: "static-other-v1", Source: source(i.Documents[1], "/other")})
		},
		"policy": func(i *run.BindingInput) { i.Policy.HostOperationTimeout += time.Second },
	} {
		t.Run(name, func(t *testing.T) {
			changed := fixture(t)
			change(&changed)
			other, err := run.Bind(changed)
			if err != nil {
				t.Fatal(err)
			}
			if other.Ref() == b.Ref() {
				t.Fatal("changed input omitted from binding")
			}
		})
	}
	original := b.Payload().Bytes()
	input.ModelInput.Root.Entries[0].Bytes[0] = 'X'
	input.Compilation.Models[0].Canonical[0] = 'X'
	input.Compilation.Models[0].Packages[0].Name = "changed"
	input.Profiles[0].Identity = "changed"
	b.Payload().Bytes()[0] = 'X'
	if !bytes.Equal(original, b.Payload().Bytes()) || !b.Valid() {
		t.Fatal("binding aliases caller/view data")
	}
}

// Only the fields under test are decoded. RawMessage preserves every other
// closed field verbatim and in its original order; the round-trip control below
// proves malformed probes are not rejected merely for a different JSON shape.
type closureRecord struct {
	Version                                                                                                                 uint32
	Source, OwnerRequest                                                                                                    json.RawMessage
	ModelInput                                                                                                              model.Input
	Compilation                                                                                                             []closureScope
	Managers, Settings, Ceilings, Profiles, Documents, Census, Registry, Classifications, Protections, Policy, PolicySource json.RawMessage
}
type closureScope struct {
	ID, Digest string
	Local      bool
	Manifest   model.Source
	Canonical  []byte
	Packages   []model.Package
}

func rewrittenPreparation(t *testing.T, batch run.AppendBatch, basis run.BlobRef) run.AppendBatch {
	t.Helper()
	return alteredBatch(t, batch, func(w *struct {
		Version          uint32
		ID               string
		ExpectedRevision uint64
		Events           []run.Event
	}) {
		w.ID = "new-prepare"
		for n := range w.Events {
			e := &w.Events[n]
			e.Body.ID = w.ID
			e.Body.Basis = basis
			data, err := json.Marshal(e.Body)
			if err != nil {
				t.Fatal(err)
			}
			suffix := byte(1)
			if e.Kind == "assignment.created" {
				suffix = 0
			}
			// Offer a newly identified v1 record, including fresh event identities.
			// The test does not preserve or try to bypass any old content hash.
			e.ID = fmt.Sprintf("%x", sha256.Sum256(append(data, suffix)))
		}
	})
}

func TestRecordedRawClosureMustMatchCompiledManifestReferences(t *testing.T) {
	i := fixture(t)
	basis, err := run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := run.NewCoordinator("repo")
	proposal, err := c.Propose(run.PrepareCommand("prepare", "a", "r", "", observed, basis))
	if err != nil {
		t.Fatal(err)
	}
	manifestEntry := func(t *testing.T, raw *model.SourceModel) *model.Entry {
		t.Helper()
		for n := range raw.Entries {
			if raw.Entries[n].Path == raw.ManifestPath {
				return &raw.Entries[n]
			}
		}
		t.Fatal("fixture lacks selected manifest")
		return nil
	}
	cases := map[string]func(*closureRecord){
		"unchanged":                  func(*closureRecord) {},
		"zero input":                 func(w *closureRecord) { w.ModelInput = model.Input{} },
		"missing root":               func(w *closureRecord) { w.ModelInput.Root = model.SourceModel{} },
		"missing dependency":         func(w *closureRecord) { w.ModelInput.Dependencies = w.ModelInput.Dependencies[:1] },
		"duplicate raw source":       func(w *closureRecord) { w.ModelInput.Dependencies[1] = w.ModelInput.Dependencies[0] },
		"duplicate compiled mapping": func(w *closureRecord) { w.Compilation[2].Manifest = w.Compilation[1].Manifest },
		"root selection mismatch": func(w *closureRecord) {
			w.ModelInput.Root, w.ModelInput.Dependencies[0] = w.ModelInput.Dependencies[0], w.ModelInput.Root
		},
		"local flag mismatch":             func(w *closureRecord) { w.Compilation[0].Local = false },
		"manifest origin mismatch":        func(w *closureRecord) { w.Compilation[0].Manifest.Origin = "other-origin" },
		"manifest path mismatch":          func(w *closureRecord) { w.Compilation[0].Manifest.Path = "other/model.yaml" },
		"selected manifest path mismatch": func(w *closureRecord) { w.ModelInput.Root.ManifestPath = "other/model.yaml" },
		"manifest digest mismatch":        func(w *closureRecord) { w.Compilation[0].Manifest.Digest = "sha256:wrong" },
		"missing manifest original": func(w *closureRecord) {
			var entries []model.Entry
			for _, e := range w.ModelInput.Root.Entries {
				if e.Path != w.ModelInput.Root.ManifestPath {
					entries = append(entries, e)
				}
			}
			w.ModelInput.Root.Entries = entries
		},
		"changed manifest original": func(w *closureRecord) {
			e := manifestEntry(t, &w.ModelInput.Root)
			e.Bytes = append([]byte("# changed original\n"), e.Bytes...)
		},
		"nonregular manifest original": func(w *closureRecord) { manifestEntry(t, &w.ModelInput.Root).Kind = model.Directory },
		"duplicate manifest original": func(w *closureRecord) {
			e := *manifestEntry(t, &w.ModelInput.Root)
			w.ModelInput.Root.Entries = append(w.ModelInput.Root.Entries, e)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			var w closureRecord
			if err := json.Unmarshal(basis.Payload().Bytes(), &w); err != nil {
				t.Fatal(err)
			}
			original, err := json.Marshal(w)
			if err != nil || !bytes.Equal(original, basis.Payload().Bytes()) {
				t.Fatal("test codec changed closed record shape", err)
			}
			change(&w)
			data, err := json.Marshal(w)
			if err != nil {
				t.Fatal(err)
			}
			payload, err := run.NewPayload(basis.Ref().MediaType, basis.Ref().Encoding, data)
			if err != nil {
				t.Fatal(err)
			}
			if !payload.Valid() {
				t.Fatal("probe lacks a correct new byte identity")
			}
			batch := rewrittenPreparation(t, proposal.Batch(), payload.Ref())
			t.Run("BindingOwnerRef", func(t *testing.T) {
				ref, err := run.BindingOwnerRef(payload)
				if name == "unchanged" {
					if err != nil || ref != i.OwnerRequest.Ref() {
						t.Fatal("valid historical record changed", err)
					}
				} else if !errors.Is(err, run.ErrInvalid) {
					t.Fatal("inconsistent raw closure accepted", err)
				}
			})
			t.Run("Restore", func(t *testing.T) {
				// Persist the newly identified offered record directly, independently
				// of Bind/Append admission, as an adversarial physical read can do.
				store := journal()
				store.blobs[payload.Ref()] = payload
				store.blobs[i.OwnerRequest.Ref()] = i.OwnerRequest
				store.batches = []run.AppendBatch{batch}
				store.head = batch.Revision()
				reads := &readJournal{store: store}
				out, err := run.Restore(context.Background(), reads, "repo", restoreLimits)
				if reads.writes != 0 {
					t.Fatal("reconstruction wrote history")
				}
				if name == "unchanged" {
					if err != nil || out == nil || out.Current().State != run.Prepared || out.Budget().Consumed.Runs != 1 {
						t.Fatal("valid historical preparation not reconstructed", err)
					}
				} else if !errors.Is(err, run.ErrInvalid) || out != nil {
					t.Fatal("inconsistent raw closure became executable", err)
				}
			})
		})
	}
}

func TestBindUsesTheSameRawClosureReferenceChecks(t *testing.T) {
	for name, change := range map[string]func(*run.BindingInput){
		"zero input":       func(i *run.BindingInput) { i.ModelInput = model.Input{} },
		"duplicate source": func(i *run.BindingInput) { i.ModelInput.Dependencies[1] = i.ModelInput.Dependencies[0] },
		"duplicate mapping": func(i *run.BindingInput) {
			i.Compilation.Models[2].Manifest = i.Compilation.Models[1].Manifest
			i.Compilation.Models[2].Origin = i.Compilation.Models[1].Origin
		},
		"wrong root": func(i *run.BindingInput) {
			i.ModelInput.Root, i.ModelInput.Dependencies[0] = i.ModelInput.Dependencies[0], i.ModelInput.Root
		},
		"missing original": func(i *run.BindingInput) {
			var entries []model.Entry
			for _, e := range i.ModelInput.Root.Entries {
				if e.Path != i.ModelInput.Root.ManifestPath {
					entries = append(entries, e)
				}
			}
			i.ModelInput.Root.Entries = entries
		},
		"duplicate original": func(i *run.BindingInput) {
			for _, e := range i.ModelInput.Root.Entries {
				if e.Path == i.ModelInput.Root.ManifestPath {
					i.ModelInput.Root.Entries = append(i.ModelInput.Root.Entries, e)
					return
				}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			i := fixture(t)
			change(&i)
			b, err := run.Bind(i)
			if !errors.Is(err, run.ErrInvalid) || b.Valid() {
				t.Fatal("inconsistent source references prepared", err)
			}
		})
	}
}

func TestRawClosureMatchesExactOriginAndManifestPair(t *testing.T) {
	i := fixtureWith(t, func(input *model.Input) {
		rename := func(source *model.SourceModel, prefix string) {
			source.Origin = "shared-opaque-origin"
			source.ManifestPath = prefix + "/" + source.ManifestPath
			for n := range source.Entries {
				source.Entries[n].Path = prefix + "/" + source.Entries[n].Path
			}
		}
		rename(&input.Root, "root")
		for n := range input.Dependencies {
			rename(&input.Dependencies[n], fmt.Sprintf("dependency-%d", n))
		}
	}, nil)
	b, err := run.Bind(i)
	if err != nil {
		t.Fatal("distinct manifest paths under one opaque Origin refused", err)
	}
	c, _ := run.NewCoordinator("repo")
	j := journal()
	commit(t, c, j, run.PrepareCommand("prepare", "a", "r", "", observed, b))
	c2 := restored(t, &readJournal{store: j})
	sameProjection(t, c, c2)
}

func TestBindingRequiresCompleteProducerAccounting(t *testing.T) {
	for name, change := range map[string]func(*run.BindingInput){
		"unexplained file": func(i *run.BindingInput) {
			i.Census = tree(t, append(i.Census.Entries(), file(t, "notes.txt", snapshot.Regular, "unclassified")))
		},
		"missing input classification": func(i *run.BindingInput) { i.Classifications = i.Classifications[1:] },
		"classification without file": func(i *run.BindingInput) {
			cl, _ := realization.NewClassification(file(t, "absent.txt", snapshot.Regular, "").Path(), realization.IgnoredFile, "Absent file")
			i.Classifications = append(i.Classifications, cl)
		},
		"registered artifact conflict": func(i *run.BindingInput) {
			cl, _ := realization.NewClassification(i.Registry.Entries()[0].Artifact().Path(), realization.IgnoredFile, "Cannot ignore registered artifact")
			i.Classifications = append(i.Classifications, cl)
		},
	} {
		t.Run(name, func(t *testing.T) {
			i := fixture(t)
			change(&i)
			if realization.AccountFiles(i.Compilation, i.Census, i.Registry, i.Classifications).Status() == realization.Accounted {
				t.Fatal("fixture did not distinguish accounting failure")
			}
			b, err := run.Bind(i)
			if err == nil || b.Valid() {
				t.Fatal("unaccounted source prepared")
			}
		})
	}
}

func TestBindingIncludesCeilingsValuesAndEverySource(t *testing.T) {
	base := fixture(t)
	original, err := run.Bind(base)
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*organization.SettingsFacts){
		"step timeout":  func(f *organization.SettingsFacts) { f.Ceilings.StepTimeout.Value = 3 * time.Minute },
		"review rounds": func(f *organization.SettingsFacts) { f.Ceilings.ReviewRounds.Value = 4 },
		"creation roots": func(f *organization.SettingsFacts) {
			f.Ceilings.CreationRoots.Value = []snapshot.DirectoryBoundary{snapshot.RootDirectory()}
		},
	} {
		t.Run(name+" value", func(t *testing.T) {
			changed := fixtureWith(t, nil, change)
			if !reflect.DeepEqual(base.Settings.Managers(), changed.Settings.Managers()) {
				t.Fatal("fixture changed effective selection")
			}
			b, err := run.Bind(changed)
			if err != nil {
				t.Fatal(err)
			}
			if b.Ref() == original.Ref() {
				t.Fatal("changed unused ceiling value omitted")
			}
		})
	}
	for _, field := range []string{"step timeout", "review rounds", "creation roots"} {
		t.Run(field, func(t *testing.T) {
			e := file(t, "separate-ceilings.yaml", snapshot.Regular, "Separate ceiling input bytes\r\n")
			d := run.Document{Source: organization.SettingSource{File: "host/separate-ceilings.yaml", ContentID: e.ContentID()}, File: e}
			i := fixtureWith(t, nil, func(f *organization.SettingsFacts) {
				switch field {
				case "step timeout":
					f.Ceilings.StepTimeout.Source = source(d, "/stepTimeout")
				case "review rounds":
					f.Ceilings.ReviewRounds.Source = source(d, "/reviewRounds")
				case "creation roots":
					f.Ceilings.CreationRoots.Source = source(d, "/creationRoots")
				}
			})
			if !reflect.DeepEqual(base.Settings.Managers(), i.Settings.Managers()) {
				t.Fatal("fixture changed effective selection")
			}
			if b, err := run.Bind(i); err == nil || b.Valid() {
				t.Fatal("missing original ceiling document accepted")
			}
			i.Documents = append(i.Documents, d)
			b, err := run.Bind(i)
			if err != nil {
				t.Fatal(err)
			}
			if b.Ref() == original.Ref() {
				t.Fatal("ceiling provenance omitted")
			}
			// Same originals and effective values, only the selected field differs.
			i2 := fixtureWith(t, nil, func(f *organization.SettingsFacts) {
				switch field {
				case "step timeout":
					f.Ceilings.StepTimeout.Source = source(d, "/otherStepTimeout")
				case "review rounds":
					f.Ceilings.ReviewRounds.Source = source(d, "/otherReviewRounds")
				case "creation roots":
					f.Ceilings.CreationRoots.Source = source(d, "/otherCreationRoots")
				}
			})
			i2.Documents = append(i2.Documents, d)
			b2, err := run.Bind(i2)
			if err != nil {
				t.Fatal(err)
			}
			if b2.Ref() == b.Ref() {
				t.Fatal("ceiling field provenance omitted")
			}
		})
	}
}

func TestBindingRejectsUncoveredOrInconsistentFactsBeforePreparation(t *testing.T) {
	for name, change := range map[string]func(*run.BindingInput){
		"insufficient review starts": func(i *run.BindingInput) { i.Policy.Reserve.SessionStarts = 3 },
		"insufficient review steps":  func(i *run.BindingInput) { i.Policy.Reserve.Steps = 3 },
		"unbounded time":             func(i *run.BindingInput) { i.Policy.Assignment.ExecutionTime = 0 },
		"over ceiling":               func(i *run.BindingInput) { i.Policy.Assignment.Runs = i.Policy.Ceilings.Runs + 1 },
		"uncovered profiles":         func(i *run.BindingInput) { i.Documents = append(i.Documents[:1], i.Documents[2:]...) },
		"wrong source bytes":         func(i *run.BindingInput) { i.ModelInput.Root.Entries[1].Bytes = []byte("changed source") },
		"wrong canonical bytes":      func(i *run.BindingInput) { i.Compilation.Models[0].Canonical = []byte("changed canonical") },
		"missing closure":            func(i *run.BindingInput) { i.ModelInput.Dependencies = i.ModelInput.Dependencies[:1] },
		"failed compiler":            func(i *run.BindingInput) { i.Compilation.Status = model.Failed },
		"document mismatch":          func(i *run.BindingInput) { i.Documents[0].File = i.Documents[1].File },
		"wrong operation":            func(i *run.BindingInput) { i.Source.Operation = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			i := fixture(t)
			change(&i)
			b, err := run.Bind(i)
			if err == nil || b.Valid() {
				t.Fatal("invalid preparation admitted")
			}
		})
	}
}
