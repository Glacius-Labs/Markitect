package organization

import (
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

// ScopeRule selects a modeled scope field, never individual Definitions.
// Kind and Property are exact resolved identities. Scope is read from local
// Definitions only; configuration introduces no new model import visibility.
type ScopeRule struct {
	Kind     model.Identity
	Property model.Identity
}

type ContextReasonKind string

const (
	OwnDefinition     ContextReasonKind = "own"
	KindContract      ContextReasonKind = "kind-contract"
	PropertyContract  ContextReasonKind = "property-contract"
	OutgoingReference ContextReasonKind = "outgoing"
	IncomingReference ContextReasonKind = "incoming"
	ModeledScope      ContextReasonKind = "modeled-scope"
)

// ContextReason explains one inclusion. From/To identify the directed edge;
// At is an RFC 6901 pointer in From's compiled Definition, not a guessed YAML
// coordinate. /bodyLinks/N identifies the original occurrence, including repeats.
// Own uses the Definition itself and an empty At. Scope additionally records
// the selecting configuration field and the modeled covering Package value.
type ContextReason struct {
	Kind         ContextReasonKind
	From         model.Identity
	To           model.Identity
	At           string
	Policy       *SettingSource
	ScopePackage string
}

// ContextDefinition is an original compiled Definition and every inclusion
// reason. ModelDigest binds its owner, independently of the local model digest.
// All model content remains read-only; selection conveys no artifact authority.
type ContextDefinition struct {
	Definition  model.Definition
	ModelDigest string
	Reasons     []ContextReason
}

// ContextSelection is fixed inspectable input selection, not a rendered context,
// mandate, role prompt or proof of semantic completeness. Zero exposes nothing.
type ContextSelection struct {
	topology    Topology
	scope       []Setting[ScopeRule] // Retain all policies, even those selecting no Definition.
	packagePath string
	modelID     string
	modelDigest string
	definitions []ContextDefinition
	index       map[model.Identity]int
}

func (c ContextSelection) Package() string     { return c.packagePath }
func (c ContextSelection) ModelID() string     { return c.modelID }
func (c ContextSelection) ModelDigest() string { return c.modelDigest }

// Definitions returns owned copies: own Definitions first in exact Source.Path
// order, then other Definitions by path, owner and identity. Reasons are stable.
func (c ContextSelection) Definitions() []ContextDefinition {
	result := make([]ContextDefinition, len(c.definitions))
	for i, definition := range c.definitions {
		result[i] = cloneContextDefinition(definition)
	}
	return result
}

func (c ContextSelection) Definition(id model.Identity) (ContextDefinition, bool) {
	i, found := c.index[id]
	if !found {
		return ContextDefinition{}, false
	}
	return cloneContextDefinition(c.definitions[i]), true
}

func cloneContextDefinition(value ContextDefinition) ContextDefinition {
	d := &value.Definition
	if d.Title != nil {
		title := *d.Title
		d.Title = &title
	}
	d.Spec = slices.Clone(d.Spec) // model.Value has immutable, copying accessors.
	d.BodyLinks = slices.Clone(d.BodyLinks)
	for i := range d.BodyLinks {
		link := &d.BodyLinks[i]
		if link.Display != nil {
			display := *link.Display
			link.Display = &display
		}
		if link.Coordinates != nil {
			coordinates := *link.Coordinates
			link.Coordinates = &coordinates
			if coordinates.Body != nil {
				body := *coordinates.Body
				link.Coordinates.Body = &body
			}
			if coordinates.Source != nil {
				source := *coordinates.Source
				link.Coordinates.Source = &source
			}
		}
	}
	value.Reasons = slices.Clone(value.Reasons)
	for i := range value.Reasons {
		if value.Reasons[i].Policy != nil {
			source := *value.Reasons[i].Policy
			value.Reasons[i].Policy = &source
		}
	}
	return value
}
