package realization

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var (
	ErrInvalidOwner    = errors.New("invalid realization owner")
	ErrInvalidArtifact = errors.New("invalid realization artifact")
)

// Owner is an explicit responsibility reference. Its zero value is invalid.
// The root is not represented as a synthetic canonical Package.
type Owner struct {
	root        bool
	packageName string
}

func RootOwner() Owner { return Owner{root: true} }

// PackageOwner preserves an explicit Package spelling. This checks only the
// relative representation; accounting checks membership in the compiler's
// authoritative local Package inventory, including empty/intermediate Packages.
// It does not reproduce the compiler's naming, Unicode or ownership rules.
func PackageOwner(name string) (Owner, error) {
	if _, err := snapshot.NewPath(name); err != nil {
		return Owner{}, fmt.Errorf("%w: %q", ErrInvalidOwner, name)
	}
	return Owner{packageName: name}, nil
}

func (o Owner) IsRoot() bool            { return o.root }
func (o Owner) Package() (string, bool) { return o.packageName, !o.root && o.packageName != "" }

func (o Owner) valid() bool { return o.root || o.packageName != "" }

// label names the responsibility in diagnostics.
func (o Owner) label() string {
	if o.root {
		return "Root"
	}
	return fmt.Sprintf("Manager %q", o.packageName)
}

// Form is the declared artifact form, independent of its extension or location.
type Form string

const (
	Code   Form = "code"
	Test   Form = "test"
	Doc    Form = "doc"
	Build  Form = "build"
	Config Form = "config"
)

// Artifact is a fixed exact file path and its explained model basis.
// Its zero value is invalid. It grants no write authority.
type Artifact struct {
	path  snapshot.Path
	form  Form
	basis []model.Identity
	why   string
}

// NewArtifact fixes the supplied values. Basis is a set of complete model
// identities: repeated links fail, order is irrelevant, and target existence is
// checked later against compiled model values. Why is preserved without rewriting.
func NewArtifact(path snapshot.Path, form Form, basis []model.Identity, why string) (Artifact, error) {
	invalid := func(reason string) (Artifact, error) {
		return Artifact{}, fmt.Errorf("%w: %q: %s", ErrInvalidArtifact, path, reason)
	}
	if path.String() == "" || statePath(path) {
		return invalid("path is empty or operational state")
	}
	switch form {
	case Code, Test, Doc, Build, Config:
	default:
		return invalid("unsupported form")
	}
	if len(basis) == 0 || strings.TrimSpace(why) == "" {
		return invalid("model basis and explanation are required")
	}
	basis = slices.Clone(basis)
	slices.SortFunc(basis, compareIdentity)
	for i, id := range basis {
		if strings.TrimSpace(id.Package) == "" || strings.TrimSpace(id.ID) == "" {
			return invalid("incomplete Definition identity")
		}
		if i > 0 && id == basis[i-1] {
			return invalid("duplicate Definition identity")
		}
	}
	return Artifact{path: path, form: form, basis: basis, why: why}, nil
}

func (a Artifact) Path() snapshot.Path     { return a.path }
func (a Artifact) Form() Form              { return a.form }
func (a Artifact) Basis() []model.Identity { return slices.Clone(a.basis) }
func (a Artifact) Why() string             { return a.why }

func compareIdentity(a, b model.Identity) int {
	if c := strings.Compare(a.Package, b.Package); c != 0 {
		return c
	}
	return strings.Compare(a.ID, b.ID)
}

func statePath(path snapshot.Path) bool {
	return path.String() == ".markitect/state" || strings.HasPrefix(path.String(), ".markitect/state/")
}
