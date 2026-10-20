package bootstrap

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cockpit"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

// cockpitModel turns the checked model and the registry records of a
// structural preparation into the cockpit's read-only model view.
func cockpitModel(checked model.Result, records []apply.RegistryRecord, registryStatus model.Status) *cockpit.ModelView {
	view := &cockpit.ModelView{Models: []cockpit.ModelScope{}, Registry: []cockpit.RegistryEntry{}, RegistryStatus: string(registryStatus)}
	for _, scope := range checked.Models {
		s := cockpit.ModelScope{ID: scope.ID, Digest: scope.Digest, Local: scope.Local, Status: string(scope.Status), Packages: []string{}, Definitions: []cockpit.DefinitionView{}}
		for _, p := range scope.Packages {
			s.Packages = append(s.Packages, p.Name)
		}
		for _, d := range scope.Definitions {
			v := cockpit.DefinitionView{
				Package: d.Identity.Package, ID: d.Identity.ID, Kind: d.Kind.Package + "/" + d.Kind.ID,
				Body: d.Body, Source: d.Source.Path, Links: []cockpit.LinkView{}, Spec: []cockpit.FieldView{},
			}
			if d.Title != nil {
				v.Title = *d.Title
			}
			for _, l := range d.BodyLinks {
				v.Links = append(v.Links, cockpit.LinkView{Package: l.Target.Package, ID: l.Target.ID})
			}
			v.Segments = bodySegments(d.Body, d.BodyLinks)
			for _, f := range d.Spec {
				v.Spec = append(v.Spec, cockpit.FieldView{Name: f.Name, Value: valueText(f.Value)})
			}
			s.Definitions = append(s.Definitions, v)
		}
		view.Models = append(view.Models, s)
	}
	for _, r := range records {
		view.Registry = append(view.Registry, cockpit.RegistryEntry{Shard: r.Source.String(), Path: r.Path, Form: r.Form, Realizes: append([]string{}, r.Realizes...), Why: r.Why})
	}
	return view
}

// valueText writes a spec value as the readable model view does.
func valueText(v model.Value) string {
	if text, ok := v.Text(); ok {
		b, _ := json.Marshal(text)
		return string(b)
	}
	if integer, ok := v.Integer(); ok {
		return integer
	}
	if boolean, ok := v.Boolean(); ok {
		return fmt.Sprint(boolean)
	}
	if number, ok := v.Number(); ok {
		return number.Coefficient + "e" + number.Exponent
	}
	if reference, ok := v.Reference(); ok {
		return reference.Package + "/" + reference.ID
	}
	if list, ok := v.Items(); ok {
		var out []string
		for _, item := range list {
			out = append(out, valueText(item))
		}
		return "[" + strings.Join(out, ", ") + "]"
	}
	if fields, ok := v.Fields(); ok {
		var out []string
		for _, f := range fields {
			out = append(out, f.Name+": "+valueText(f.Value))
		}
		return "{" + strings.Join(out, ", ") + "}"
	}
	return ""
}

// bodySegments cuts a body at its links' located spans. The model check
// locates each link in the decoded body; when a link has no location, or the
// locations do not follow one another inside the body, the body stays one
// text segment rather than guessing which text names which target.
func bodySegments(body string, links []model.BodyLink) []cockpit.BodySegment {
	whole := []cockpit.BodySegment{{Text: body}}
	var out []cockpit.BodySegment
	at := 0
	for _, l := range links {
		if l.Coordinates == nil || l.Coordinates.Body == nil {
			return whole
		}
		start, end := l.Coordinates.Body.StartByte, l.Coordinates.Body.EndByte
		if start < at || end <= start || end > len(body) {
			return whole
		}
		if start > at {
			out = append(out, cockpit.BodySegment{Text: body[at:start]})
		}
		shown := l.Target.ID
		if l.Display != nil {
			shown = *l.Display
		}
		out = append(out, cockpit.BodySegment{Text: shown, Link: &cockpit.LinkView{Package: l.Target.Package, ID: l.Target.ID}})
		at = end
	}
	if at < len(body) {
		out = append(out, cockpit.BodySegment{Text: body[at:]})
	}
	if out == nil {
		return whole
	}
	return out
}
