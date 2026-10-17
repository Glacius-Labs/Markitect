package run

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// SourceBasis is an opaque Host assertion from committed acquisition, not a Git
// resolver, publication permit, live controller or evidence of a clean checkout.
type SourceBasis struct{ Repository, Revision, Guard, ProductBuild, Operation string }

// Document projects an original acquired input without importing apply. Source
// addresses the whole file; RepositoryPath is absent for external documents.
// No locator is interpreted as a physical path. Include execution, strategy,
// registry/ignore, selected Host profile and instruction documents explicitly.
type Document struct {
	Source         organization.SettingSource
	File           snapshot.Entry
	RepositoryPath snapshot.Path
}

// BindingInput is supplied by the application, never acquired here. ModelInput
// retains the complete original model/lock bytes; Compilation is its producer
// result. Settings and profiles are static facts, never runtime capabilities.
// Source acquisition and compiler correctness remain their producers' duties.
type BindingInput struct {
	Source          SourceBasis
	OwnerRequest    Payload
	ModelInput      model.Input
	Compilation     model.Result
	Census          snapshot.Tree
	Registry        realization.Registry
	Classifications []realization.Classification
	Protections     []realization.Protection
	Settings        organization.Settings
	Profiles        []organization.ProfileFact
	Documents       []Document
	Policy          BudgetPolicy
	PolicySource    organization.SettingSource
}

type compiledBinding struct {
	ID, Digest string
	Local      bool
	Manifest   model.Source
	Canonical  []byte
	Packages   []model.Package
}
type fileBinding struct {
	Path  string
	Mode  snapshot.Mode
	Bytes []byte
}
type documentBinding struct {
	Source               organization.SettingSource
	Path, RepositoryPath string
	Mode                 snapshot.Mode
	Bytes                []byte
}
type registryBinding struct {
	Owner, Path string
	Form        realization.Form
	Basis       []model.Identity
	Why         string
}
type classificationBinding struct {
	Path  string
	Class realization.FileClass
	Why   string
}
type protectionBinding struct{ File, Directory, Why string }
type managerBinding struct {
	Package              string
	Manager, Reviewer    organization.RoleSettings
	StepTimeout          organization.Setting[int64]
	ReviewRounds         organization.Setting[int]
	CreationRoots        []string
	CreationSource       organization.SettingSource
	BatchNoChangeReviews organization.Setting[bool]
}
type bindingWire struct {
	Version         uint32
	Source          SourceBasis
	OwnerRequest    BlobRef
	ModelInput      model.Input
	Compilation     []compiledBinding
	Managers        []organization.Manager
	Settings        []managerBinding
	Profiles        []organization.ProfileFact
	Documents       []documentBinding
	Census          []fileBinding
	Registry        []registryBinding
	Classifications []classificationBinding
	Protections     []protectionBinding
	Policy          BudgetPolicy
	PolicySource    organization.SettingSource
}

// Binding owns its exact serialized basis. Its identity includes empty Package
// inventory, file modes, ordered repeated instructions and opaque source guard.
// It does not grant dispatch, review, artifact ownership or publication authority.
type Binding struct {
	payload, owner Payload
	source         SourceBasis
	policy         BudgetPolicy
	reviews        int64
}

func (b Binding) Ref() BlobRef           { return b.payload.Ref() }
func (b Binding) Payload() Payload       { return b.payload }
func (b Binding) RequiredReviews() int64 { return b.reviews }
func (b Binding) Valid() bool            { return b.payload.Valid() && b.owner.Valid() }

func Bind(input BindingInput) (Binding, error) {
	s := input.Source
	if !text(s.Repository) || !text(s.Revision) || !text(s.Guard) || !text(s.ProductBuild) || s.Operation != "artifact-apply/v1" || !input.OwnerRequest.Valid() {
		return Binding{}, ErrInvalid
	}
	topology, err := organization.BuildTopology(input.Compilation)
	if err != nil || input.Settings.ModelID() != topology.ModelID() || input.Settings.ModelDigest() != topology.ModelDigest() {
		return Binding{}, ErrInvalid
	}
	w := bindingWire{Version: 1, Source: s, OwnerRequest: input.OwnerRequest.Ref(), ModelInput: input.ModelInput, Managers: topology.Managers(), Profiles: slices.Clone(input.Profiles), Policy: input.Policy, PolicySource: input.PolicySource}
	raw := append([]model.SourceModel{input.ModelInput.Root}, input.ModelInput.Dependencies...)
	if len(raw) != len(input.Compilation.Models) {
		return Binding{}, ErrInvalid
	}
	matchedRaw := make(map[int]bool)
	for _, compiled := range input.Compilation.Models {
		i := -1
		for index, source := range raw {
			if source.Origin == compiled.Origin && source.ManifestPath == compiled.Manifest.Path && !matchedRaw[index] {
				i = index
				break
			}
		}
		if i < 0 || compiled.Local != (i == 0) || compiled.Status != model.Passed || len(compiled.Canonical) == 0 || compiled.Digest != fmt.Sprintf("sha256:%x", sha256.Sum256(compiled.Canonical)) {
			return Binding{}, ErrInvalid
		}
		matchedRaw[i] = true
		for _, source := range append([]model.Source{compiled.Manifest}, definitionSources(compiled.Definitions)...) {
			matched := false
			for _, entry := range raw[i].Entries {
				if entry.Kind == model.Regular && entry.Path == source.Path && source.Origin == raw[i].Origin && source.Digest == fmt.Sprintf("sha256:%x", sha256.Sum256(entry.Bytes)) {
					matched = true
					break
				}
			}
			if !matched {
				return Binding{}, ErrInvalid
			}
		}
		w.Compilation = append(w.Compilation, compiledBinding{compiled.ID, compiled.Digest, compiled.Local, compiled.Manifest, compiled.Canonical, compiled.Packages})
	}
	for _, document := range input.Documents {
		if !text(document.Source.File) || document.Source.Field != "" || document.File.Path().String() == "" || document.Source.ContentID != document.File.ContentID() {
			return Binding{}, ErrInvalid
		}
		if document.RepositoryPath.String() != "" {
			entry, ok := input.Census.Lookup(document.RepositoryPath)
			if !ok || entry.ContentID() != document.File.ContentID() || entry.Mode() != document.File.Mode() {
				return Binding{}, ErrInvalid
			}
		}
		w.Documents = append(w.Documents, documentBinding{document.Source, document.File.Path().String(), document.RepositoryPath.String(), document.File.Mode(), document.File.Bytes()})
	}
	covered := func(source organization.SettingSource) bool {
		for _, document := range input.Documents {
			if source.File == document.Source.File && source.ContentID == document.File.ContentID() {
				return true
			}
		}
		return false
	}
	if !covered(input.PolicySource) {
		return Binding{}, ErrInvalid
	}
	for _, profile := range input.Profiles {
		if !text(profile.Name) || !text(profile.Identity) || !covered(profile.Source) {
			return Binding{}, ErrInvalid
		}
	}
	managers := input.Settings.Managers()
	if len(managers) != len(w.Managers) {
		return Binding{}, ErrInvalid
	}
	var areas []realization.CreationArea
	var reviews int64
	for i, manager := range managers {
		if manager.Package != w.Managers[i].Package {
			return Binding{}, ErrInvalid
		}
		if w.Managers[i].HasLocalWork() {
			reviews++
		}
		if len(w.Managers[i].Children) > 0 && !w.Managers[i].PassThrough() {
			reviews++
		}
		for _, role := range []organization.RoleSettings{manager.Manager, manager.Reviewer} {
			if !covered(role.Profile.Source) || !slices.Contains(input.Profiles, role.Profile.Value) {
				return Binding{}, ErrInvalid
			}
			for _, source := range role.InstructionSources {
				if !covered(source) {
					return Binding{}, ErrInvalid
				}
			}
			for _, instruction := range role.Instructions {
				if !covered(instruction.Source) || !covered(instruction.Value.Source) {
					return Binding{}, ErrInvalid
				}
			}
		}
		for _, source := range []organization.SettingSource{manager.StepTimeout.Source, manager.ReviewRounds.Source, manager.CreationRoots.Source, manager.BatchNoChangeReviews.Source} {
			if !covered(source) {
				return Binding{}, ErrInvalid
			}
		}
		owner := realization.RootOwner()
		if manager.Package != organization.Root {
			owner, err = realization.PackageOwner(manager.Package)
			if err != nil {
				return Binding{}, err
			}
		}
		m := managerBinding{Package: manager.Package, Manager: manager.Manager, Reviewer: manager.Reviewer, StepTimeout: organization.Setting[int64]{Value: int64(manager.StepTimeout.Value), Source: manager.StepTimeout.Source}, ReviewRounds: manager.ReviewRounds, CreationSource: manager.CreationRoots.Source, BatchNoChangeReviews: manager.BatchNoChangeReviews}
		for _, directory := range manager.CreationRoots.Value {
			m.CreationRoots = append(m.CreationRoots, directory.String())
			area, e := realization.NewCreationArea(owner, directory)
			if e != nil {
				return Binding{}, e
			}
			areas = append(areas, area)
		}
		w.Settings = append(w.Settings, m)
	}
	if err := input.Policy.validate(reviews); err != nil {
		return Binding{}, err
	}
	if _, err := realization.NewPathFacts(input.Compilation, input.Registry, input.Census, areas, input.Protections); err != nil {
		return Binding{}, err
	}
	for _, file := range input.Census.Entries() {
		w.Census = append(w.Census, fileBinding{file.Path().String(), file.Mode(), file.Bytes()})
	}
	for _, entry := range input.Registry.Entries() {
		a := entry.Artifact()
		w.Registry = append(w.Registry, registryBinding{ownerName(entry.Owner()), a.Path().String(), a.Form(), a.Basis(), a.Why()})
	}
	for _, c := range input.Classifications {
		if c.Path().String() == "" || c.Why() == "" {
			return Binding{}, ErrInvalid
		}
		w.Classifications = append(w.Classifications, classificationBinding{c.Path().String(), c.Class(), c.Why()})
	}
	for _, p := range input.Protections {
		f, _ := p.File()
		d, _ := p.Directory()
		w.Protections = append(w.Protections, protectionBinding{f.String(), d.String(), p.Why()})
	}
	data, err := json.Marshal(w)
	if err != nil {
		return Binding{}, err
	}
	payload, err := NewPayload("application/vnd.markitect.run-basis+json", "utf-8", data)
	if err != nil {
		return Binding{}, err
	}
	return Binding{payload, input.OwnerRequest, s, input.Policy, reviews}, nil
}
func definitionSources(definitions []model.Definition) []model.Source {
	var sources []model.Source
	for _, d := range definitions {
		sources = append(sources, d.Source)
	}
	return sources
}
func ownerName(owner realization.Owner) string {
	if owner.IsRoot() {
		return organization.Root
	}
	name, _ := owner.Package()
	return name
}
