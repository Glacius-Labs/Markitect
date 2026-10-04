package host

import (
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

// Temporary compiler names keep this checkpoint buildable while use cases
// move to Host. Core owns the single wire/IR definition; no copied DTO remains.
const SemanticModelVersion = core.SemanticModelVersion

type SemanticModel = core.SemanticModel
type ModelSnapshot = core.ModelSnapshot
type ModelResource = core.ModelResource
type ModelIdentity = core.ModelIdentity
type ModelSource = core.ModelSource
type ModelRelationship = core.ModelRelationship
type ModelDomainInput = core.ModelDomainInput
type ModelDomain = core.ModelDomain

func CompileModel(p *Project) (SemanticModel, error) {
	if p == nil || p.Graph == nil || p.Graph.Project == nil || p.Snapshot == nil {
		return SemanticModel{}, fmt.Errorf("a parsed project and fixed snapshot are required")
	}
	config, err := YAML(p.Graph.Project.Spec)
	if err != nil {
		return SemanticModel{}, err
	}
	model := SemanticModel{
		APIVersion:   SemanticModelVersion,
		Snapshot:     ModelSnapshot{ID: p.Snapshot.ID, Provisional: p.Snapshot.Provisional, Digest: p.Snapshot.Digest()},
		ConfigDigest: hashBytes(config), Resources: make([]ModelResource, 0, len(p.Graph.Resources)),
	}
	model.PolicyResults = clonePolicyResults(p.Graph.PolicyResults)
	if p.Graph.Registry != nil {
		for _, d := range p.Graph.Registry.Domains() {
			domain := ModelDomain{Name: d.Name, APIVersion: d.APIVersion, Kinds: d.Kinds, Relations: d.Relations, Constraints: d.Constraints}
			encoded, marshalErr := YAML(domain)
			if marshalErr != nil {
				return SemanticModel{}, marshalErr
			}
			var cloned ModelDomain
			if unmarshalErr := yaml.Unmarshal(encoded, &cloned); unmarshalErr != nil {
				return SemanticModel{}, unmarshalErr
			}
			model.Domains = append(model.Domains, cloned)
		}
	}
	for _, input := range p.DomainInputs {
		model.DomainInputs = append(model.DomainInputs, ModelDomainInput{APIVersion: input.APIVersion, Name: input.Name, Path: input.Path, Package: input.Package,
			PackageVersion: p.packageVersion(input.Package), Digest: Hash(p.fileBytes(input.Package, input.Path))})
	}
	structuralDiagnostics := p.StructuralDiagnostics()
	if len(p.Diagnostics) == 0 {
		model.ValidationStatus = "passed"
	} else {
		model.ValidationStatus = "failed"
		model.Diagnostics = cloneDiagnostics(p.Diagnostics)
	}
	if len(structuralDiagnostics) == 0 {
		model.StructuralStatus = "passed"
	} else {
		model.StructuralStatus = "failed"
	}
	model.PolicyStatus = policyStatus(model.StructuralStatus, model.PolicyResults)
	keys := make([]string, 0, len(p.Graph.Resources))
	for key := range p.Graph.Resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		r := p.Graph.Resources[key]
		if r == nil {
			continue
		}
		var data map[string]any
		if r.Data == nil {
			// Built-in resources lower into the existing typed Spec. MarshalYAML
			// emits their normalized authoring representation; adapters still see
			// only the generic `data` object in this DTO.
			encoded, marshalErr := yaml.Marshal(r)
			if marshalErr != nil {
				return SemanticModel{}, marshalErr
			}
			var envelope struct {
				Spec map[string]any `yaml:"spec"`
			}
			if unmarshalErr := yaml.Unmarshal(encoded, &envelope); unmarshalErr != nil {
				return SemanticModel{}, unmarshalErr
			}
			data, err = cloneModelMap(envelope.Spec)
		} else {
			data, err = cloneModelMap(r.Data)
		}
		if err != nil {
			return SemanticModel{}, err
		}
		labels := map[string]string(nil)
		if r.Metadata.Labels != nil {
			labels = map[string]string{}
			for name, value := range r.Metadata.Labels {
				labels[name] = value
			}
		}
		model.Resources = append(model.Resources, ModelResource{
			Identity: ModelIdentity{APIVersion: r.APIVersion, Kind: r.Kind, Namespace: r.Metadata.Namespace, Name: r.Metadata.Name, Package: r.Package, Key: key},
			Labels:   labels, Data: data,
			Source: ModelSource{Path: r.Path, Line: r.Line, Digest: Hash(p.resourceBytes(r))},
			Area:   p.Graph.ResourceAreas[key].Name,
		})
	}
	for _, relation := range p.Graph.Relationships {
		sourceDigest := ""
		if sourceResource := p.Graph.Resources[relation.From]; sourceResource != nil {
			sourcePath := relation.Path
			if sourcePath == "" {
				sourcePath = sourceResource.Path
			}
			sourceDigest = Hash(p.fileBytes(sourceResource.Package, sourcePath))
		}
		model.Relationships = append(model.Relationships, ModelRelationship{
			From: relation.From, To: relation.To, Type: relation.Relation,
			Source:  ModelSource{Path: relation.Path, Line: relation.Line, Digest: sourceDigest},
			Context: relation.Context, Invalidate: relation.Invalidate, Acyclic: relation.Acyclic,
		})
	}
	// Desired-model identity excludes volatile full-snapshot metadata, but includes
	// every normalized resource, relationship, domain definition and exact input hash.
	stable := struct {
		APIVersion    string              `yaml:"apiVersion"`
		ConfigDigest  string              `yaml:"configDigest"`
		DomainInputs  []ModelDomainInput  `yaml:"domainInputs,omitempty"`
		Domains       []ModelDomain       `yaml:"domains,omitempty"`
		Resources     []ModelResource     `yaml:"resources"`
		Relationships []ModelRelationship `yaml:"relationships,omitempty"`
	}{model.APIVersion, model.ConfigDigest, model.DomainInputs, model.Domains, model.Resources, model.Relationships}
	unsigned, err := YAML(stable)
	if err != nil {
		return SemanticModel{}, err
	}
	model.ModelDigest = hashBytes(unsigned)
	return model, nil
}

// clonePolicyResults detaches the ordered relation traces from the canonical
// graph. These traces are nested slices and must not let adapter-side mutation
// alter subsequent model or context evidence.
func clonePolicyResults(results []core.PolicyResult) []core.PolicyResult {
	cloned := append([]core.PolicyResult(nil), results...)
	for i := range cloned {
		if results[i].Comparison == nil {
			continue
		}
		comparison := *results[i].Comparison
		comparison.Left.Relations = append([]string(nil), results[i].Comparison.Left.Relations...)
		comparison.Left.Steps = append([]core.TargetStep(nil), results[i].Comparison.Left.Steps...)
		comparison.Right.Relations = append([]string(nil), results[i].Comparison.Right.Relations...)
		comparison.Right.Steps = append([]core.TargetStep(nil), results[i].Comparison.Right.Steps...)
		cloned[i].Comparison = &comparison
	}
	return cloned
}

func cloneModelMap(input map[string]any) (map[string]any, error) {
	if input == nil {
		return map[string]any{}, nil
	}
	data, err := yaml.Marshal(input)
	if err != nil {
		return nil, err
	}
	var cloned map[string]any
	if err = yaml.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}
	if cloned == nil {
		return map[string]any{}, nil
	}
	return cloned, nil
}
