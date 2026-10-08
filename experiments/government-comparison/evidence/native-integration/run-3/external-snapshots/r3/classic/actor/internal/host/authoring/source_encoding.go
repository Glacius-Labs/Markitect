package authoring

import (
	"fmt"
	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
	"go.yaml.in/yaml/v3"
)

func (r Resource) MarshalYAML() (any, error) {
	var spec any = r.Spec
	if r.Core.APIVersion != core.APIVersion {
		spec = r.Core.Data
	}
	return struct {
		APIVersion string        `yaml:"apiVersion"`
		Kind       string        `yaml:"kind"`
		Metadata   core.Metadata `yaml:"metadata"`
		Spec       any           `yaml:"spec"`
	}{r.Core.APIVersion, r.Core.Kind, r.Core.Metadata, spec}, nil
}

func encodeDomain(domain core.DomainDefinition) ([]byte, error) {
	if domain.Name == "" {
		return nil, fmt.Errorf("Domain name is required")
	}
	spec := struct {
		APIVersion  string                             `yaml:"apiVersion"`
		Kinds       map[string]core.KindDefinition     `yaml:"kinds"`
		Relations   map[string]core.RelationDefinition `yaml:"relations,omitempty"`
		Constraints []core.ConstraintDefinition        `yaml:"constraints,omitempty"`
	}{domain.APIVersion, domain.Kinds, domain.Relations, domain.Constraints}
	envelope := struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
		Metadata   struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
		Spec any `yaml:"spec"`
	}{core.APIVersion, "Domain", struct {
		Name string `yaml:"name"`
	}{domain.Name}, spec}
	return yaml.Marshal(envelope)
}
