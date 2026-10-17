package organization

import (
	"fmt"
	"reflect"
	"unicode/utf8"
)

func renderInvalid(detail string) error { return &RenderRefusal{Code: "invalid-input", Detail: detail} }

func renderText(text string) error {
	if !utf8.ValidString(text) {
		return &RenderRefusal{Code: "invalid-text", Detail: "text must be valid UTF-8; no repair was performed"}
	}
	return nil
}

func hostSource(source HostSource) error {
	if source.RecordID == "" || source.ContentID == renderZeroContentID {
		return renderInvalid("missing Host record/content reference")
	}
	if err := renderText(source.RecordID); err != nil {
		return err
	}
	return renderText(source.Field)
}

func hostText(text HostText) error {
	if err := hostSource(text.Source); err != nil {
		return err
	}
	return renderText(text.Text)
}

func validateRender(c ContextSelection, s Settings, input RenderInput) (ManagerSettings, error) {
	if c.packagePath == "" || c.modelID == "" || c.modelDigest == "" || c.modelID != s.modelID || c.modelDigest != s.modelDigest || len(c.topology.managers) == 0 || len(c.topology.managers) != len(s.managers) {
		return ManagerSettings{}, renderInvalid("selection and settings require the same complete local model/inventory")
	}
	for i, slot := range c.topology.managers {
		if slot.Package != s.managers[i].Package {
			return ManagerSettings{}, renderInvalid("selection/settings Package inventories differ")
		}
	}
	manager, found := s.Manager(c.packagePath)
	if !found {
		return ManagerSettings{}, renderInvalid("selected Manager has no settings")
	}
	if input.View != ManagerView && input.View != ReviewerExpectations && input.View != ReviewerCandidate {
		return ManagerSettings{}, renderInvalid("unsupported context view")
	}
	if input.Task.ID == "" || input.Task.Package != c.packagePath || input.Task.Outcome.Text == "" {
		return ManagerSettings{}, renderInvalid("missing task identity/outcome or mismatched Package")
	}
	if err := hostSource(input.InputBinding); err != nil {
		return ManagerSettings{}, err
	}
	if err := hostText(input.Task.Outcome); err != nil {
		return ManagerSettings{}, err
	}
	for _, texts := range [][]HostText{input.Task.Constraints, input.Task.Requirements, input.RemainingLimits} {
		for _, text := range texts {
			if err := hostText(text); err != nil {
				return ManagerSettings{}, err
			}
		}
	}
	if err := validateFindings(input.Task.Findings); err != nil {
		return ManagerSettings{}, err
	}
	if input.OwnerInput != nil {
		if c.packagePath != Root || input.View != ManagerView {
			return ManagerSettings{}, renderInvalid("owner input belongs only to the root Manager")
		}
		if err := hostText(*input.OwnerInput); err != nil {
			return ManagerSettings{}, err
		}
	}
	if input.MaxContextBytes.Value <= 0 {
		return ManagerSettings{}, renderInvalid("maxContextBytes must be positive")
	}
	validation := s.configuration.clone()
	for _, policy := range c.scope {
		if err := validation.field(policy.Source, policy.Value); err != nil {
			return ManagerSettings{}, renderInvalid(err.Error())
		}
	}
	if err := validation.field(input.MaxContextBytes.Source, input.MaxContextBytes.Value); err != nil {
		return ManagerSettings{}, renderInvalid(err.Error())
	}
	seen := make(map[string]bool)
	for _, fact := range input.Packages {
		if _, found := c.topology.Manager(fact.Package); !found || seen[fact.Package] {
			return ManagerSettings{}, renderInvalid("unknown/duplicate Package projection")
		}
		seen[fact.Package] = true
		if fact.Purpose != nil {
			if err := hostText(*fact.Purpose); err != nil {
				return ManagerSettings{}, err
			}
		}
	}
	seen = make(map[string]bool)
	for _, child := range input.Children {
		slot, found := c.topology.Manager(child.Package)
		if !found || slot.Parent != c.packagePath || seen[child.Package] {
			return ManagerSettings{}, renderInvalid("unknown/non-direct/duplicate child projection")
		}
		seen[child.Package] = true
		if input.View == ReviewerExpectations && (child.State != nil || len(child.Reports) != 0) {
			return ManagerSettings{}, renderInvalid("expectation review forbids child execution state and delivery reports")
		}
		if child.State != nil {
			if err := hostText(*child.State); err != nil {
				return ManagerSettings{}, err
			}
		}
		for _, report := range child.Reports {
			if err := hostText(report); err != nil {
				return ManagerSettings{}, err
			}
		}
	}
	if input.View == ReviewerExpectations && (len(input.OwnArtifacts) != 0 || input.Review != nil) {
		return ManagerSettings{}, renderInvalid("expectation review forbids candidate/inventory/artifact/evidence payload inputs")
	}
	if input.View == ReviewerCandidate && len(input.OwnArtifacts) != 0 {
		return ManagerSettings{}, renderInvalid("candidate review accepts artifact inputs only through its exact review candidate")
	}
	if input.View == ManagerView && input.Review != nil {
		return ManagerSettings{}, renderInvalid("review subject belongs to candidate review")
	}
	if err := validateArtifacts(c, input.OwnArtifacts, true); err != nil {
		return ManagerSettings{}, err
	}
	if input.View == ReviewerCandidate {
		if input.Review == nil || input.Review.Package != c.packagePath {
			return ManagerSettings{}, renderInvalid("candidate review requires its exact subject and recorded expectations")
		}
		for _, source := range []HostSource{input.Review.Subject, input.Review.Candidate, input.Review.Expectations} {
			if err := hostSource(source); err != nil {
				return ManagerSettings{}, err
			}
		}
		if err := validateArtifacts(c, input.Review.Files, false); err != nil {
			return ManagerSettings{}, err
		}
		for _, evidence := range input.Review.Evidence {
			if err := hostText(evidence); err != nil {
				return ManagerSettings{}, err
			}
		}
	}
	return manager, nil
}

func validateFindings(set OpenFindings) error {
	if err := hostSource(set.Source); err != nil {
		return err
	}
	expected := make(map[string]bool)
	for _, id := range set.IDs {
		if id == "" || expected[id] {
			return renderInvalid("empty/duplicate open finding identity")
		}
		expected[id] = true
	}
	seen := make(map[string]bool)
	for _, finding := range set.Items {
		if !expected[finding.ID] || seen[finding.ID] || finding.Grounds.Text == "" {
			return renderInvalid("finding projection is missing grounds or contradicts the complete ID set")
		}
		seen[finding.ID] = true
		if err := hostText(finding.Grounds); err != nil {
			return err
		}
		for _, source := range finding.Evidence {
			if err := hostSource(source); err != nil {
				return err
			}
		}
	}
	if len(seen) != len(expected) {
		return renderInvalid("open finding details are incomplete")
	}
	return nil
}

func validateArtifacts(c ContextSelection, artifacts []ArtifactFact, own bool) error {
	seen := make(map[string]bool)
	for _, artifact := range artifacts {
		if artifact.Path.String() == "" || artifact.ContentID == renderZeroContentID || seen[artifact.Path.String()] {
			return renderInvalid("missing/duplicate artifact path or content identity")
		}
		seen[artifact.Path.String()] = true
		if _, found := c.topology.Manager(artifact.Package); !found || own && artifact.Package != c.packagePath {
			return renderInvalid("artifact owner is not the required exact local Manager")
		}
		if err := hostSource(artifact.Source); err != nil {
			return err
		}
		switch artifact.Content {
		case ArtifactText:
			if artifact.Text == nil {
				return renderInvalid("text artifact requires original text")
			}
			if err := renderText(*artifact.Text); err != nil {
				return err
			}
			if renderContentID([]byte(*artifact.Text)) != artifact.ContentID {
				return renderInvalid("artifact text does not match its content identity")
			}
		case ArtifactBinaryRef:
			if artifact.Text != nil {
				return renderInvalid("binary artifacts permit metadata only, no embedded payload")
			}
		default:
			return renderInvalid("unsupported artifact content form")
		}
	}
	return nil
}

// encoding/json repairs invalid UTF-8 silently. Check all serialized strings
// first, including metadata/locators; model.Value is encoded separately.
func renderStrings(value reflect.Value) error {
	if !value.IsValid() {
		return nil
	}
	switch value.Kind() {
	case reflect.String:
		return renderText(value.String())
	case reflect.Pointer, reflect.Interface:
		if !value.IsNil() {
			return renderStrings(value.Elem())
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			if err := renderStrings(value.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if value.Type().Field(i).IsExported() {
				if err := renderStrings(value.Field(i)); err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		it := value.MapRange()
		for it.Next() {
			if err := renderStrings(it.Key()); err != nil {
				return err
			}
			if err := renderStrings(it.Value()); err != nil {
				return err
			}
		}
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return fmt.Errorf("unsupported rendering metadata")
	}
	return nil
}
