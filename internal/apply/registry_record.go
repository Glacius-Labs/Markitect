package apply

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

func (r *registryPreparation) decodeRecord(source snapshot.Path, node *yaml.Node, owner realization.Owner, identities map[model.Identity]bool, census snapshot.Tree) (realization.Registration, bool) {
	before := len(r.diagnostics)
	fields := r.fields(source, node, "path", "form", "realizes", "why")
	record := RegistryRecord{
		Source: source, Position: model.Position{Line: node.Line, Column: node.Column}, Owner: owner,
		Path: r.text(source, fields, "path", node), Form: r.text(source, fields, "form", node), Why: r.text(source, fields, "why", node),
	}
	var basis []model.Identity
	if links, found := fields["realizes"]; !found || links.Kind != yaml.SequenceNode {
		r.invalid(source, links, "registry.realizes", "Registry realizes must be a list of full Package/ID strings.")
	} else {
		for _, link := range links.Content {
			if link.Kind != yaml.ScalarNode || link.Tag != "!!str" {
				r.invalid(source, link, "registry.realizes", "Registry realizes entries must be full Package/ID strings.")
				continue
			}
			record.Realizes = append(record.Realizes, link.Value)
			i := strings.LastIndexByte(link.Value, '/')
			if i <= 0 || i == len(link.Value)-1 {
				r.invalid(source, link, "registry.definition_address", "Registry realizes requires an exact full Package/ID address.")
				continue
			}
			identity := model.Identity{Package: link.Value[:i], ID: link.Value[i+1:]}
			if !identities[identity] {
				r.invalid(source, link, "registry.definition_missing", fmt.Sprintf("Definition %q is absent from the complete checked model set.", link.Value))
			}
			basis = append(basis, identity)
		}
	}
	r.records = append(r.records, record)
	if len(r.diagnostics) != before {
		return realization.Registration{}, false
	}
	// An entry is an exact repository-relative file or a directory ending in
	// "/" that covers its registrant's eligible files below it.
	name, directory := strings.CutSuffix(record.Path, "/")
	path, err := snapshot.NewPath(name)
	if err != nil {
		r.invalid(source, fields["path"], "registry.artifact_path", err.Error())
		return realization.Registration{}, false
	}
	var artifact realization.Artifact
	if directory {
		artifact, err = realization.NewDirectoryArtifact(path, realization.Form(record.Form), basis, record.Why)
	} else {
		artifact, err = realization.NewArtifact(path, realization.Form(record.Form), basis, record.Why)
	}
	if err != nil {
		r.invalid(source, node, "registry.artifact", err.Error())
		return realization.Registration{}, false
	}
	if _, found := census.Lookup(path); !directory && !found {
		r.invalid(source, fields["path"], "registry.artifact_missing", fmt.Sprintf("Registered artifact %q is absent from the bound Census.", record.Path))
		return realization.Registration{}, false
	}
	registration, err := realization.NewRegistration(owner, artifact)
	if err != nil {
		r.invalid(source, node, "registry.registration", err.Error())
		return realization.Registration{}, false
	}
	return registration, true
}
