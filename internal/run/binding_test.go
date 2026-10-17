package run_test

import (
	"bytes"
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
