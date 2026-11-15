package analysis

import "github.com/Glacius-Labs/Markitect/internal/spec"

// The std Kinds whose fields vet reads beyond their form (DEC-117). Like the
// Operations and Reactions that doc --read --bound walks, these notes are
// std's: a project's own Kinds get the form notes alone.
const (
	stdOperation = "std/Operation"
	stdOutcome   = "std/Outcome"
	stdState     = "std/State"
	stdConcept   = "std/Concept"
	stdRole      = "std/Role"
	stdRule      = "std/Rule"
	stdRefusal   = "std/Refusal"
	fieldChecked = "Checked in this order"
	fieldLeaves  = "Leaves in"
	fieldWho     = "Who"
)

// formNotes notes a field value out of the form its Kind declares where the
// Kind leaves the form to vet, so that a form added to a field that specs
// already use starts as a note (DEC-117).
func formNotes(sp *spec.Spec, notes *spec.Diagnostics) {
	for _, p := range sp.ValueProblems() {
		if p.Vet {
			notes.Add("vet.field_value", p.File, p.Line, "%s", p.Message)
		}
	}
}

// stdNotes notes what std's typed fields show across Definitions: a refusal
// code of an Operation that no Rule or Refusal states, a State of a Concept
// that no Outcome leaves in where its Concept's Outcomes say where they
// leave, and a Who that links no Role in a spec that has Roles.
func stdNotes(sp *spec.Spec, notes *spec.Diagnostics) {
	stated := map[string]bool{}
	hasRoles := false
	reached := map[string]bool{}  // States some Outcome leaves in
	concepts := map[string]bool{} // Concepts some Outcome says it leaves in a State of
	ids := sp.OwnDefinitions()
	for _, id := range ids {
		d := sp.Definitions[id]
		switch d.Kind {
		case stdRule, stdRefusal:
			for _, c := range spec.Codes(body(d.Data)) {
				stated[c] = true
			}
		case stdRole:
			hasRoles = true
		case stdOutcome:
			for _, l := range d.Links {
				if t := sp.Definitions[l.Target]; l.Field == fieldLeaves && !l.Package && t != nil && t.Kind == stdState {
					reached[l.Target] = true
					if w := sp.Definitions[t.Whole]; w != nil && w.Kind == stdConcept {
						concepts[w.ID] = true
					}
				}
			}
		}
	}
	for _, id := range ids {
		d := sp.Definitions[id]
		switch d.Kind {
		case stdOperation:
			for _, f := range d.Fields {
				switch f.Label {
				case fieldChecked:
					for i, entry := range f.Entries() {
						if c := spec.Code(entry.Text); c != "" && !stated[c] {
							notes.Add("vet.refusal_code", d.File, entry.Line, "entry %d refuses with `%s`, which no Rule or Refusal of the spec states; add it where the spec lists its refusal codes, or use one it states", i+1, c)
						}
					}
				case fieldWho:
					if hasRoles && !linksKind(sp, d, f.Label, stdRole) {
						notes.Add("vet.who_role", d.File, f.Line, "Who links no Role, while the spec has Roles; link the Roles that may call it")
					}
				}
			}
		case stdState:
			if w := sp.Definitions[d.Whole]; w != nil && concepts[w.ID] && !reached[id] {
				notes.Add("vet.unreached_state", d.File, 0, "no Outcome leaves the %s in this State, while Outcomes say which of its States they leave in; name it in Leaves in of the Outcome that leads into it", w.Title)
			}
		}
	}
}

// linksKind says whether a field of a Definition links a Definition of a Kind.
func linksKind(sp *spec.Spec, d *spec.Definition, field, kind string) bool {
	for _, l := range d.Links {
		if t := sp.Definitions[l.Target]; l.Field == field && !l.Kind && !l.Package && t != nil && t.Kind == kind {
			return true
		}
	}
	return false
}
