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
func cockpitModel(checked model.Result, records []apply.RegistryRecord) *cockpit.ModelView {
	view := &cockpit.ModelView{Models: []cockpit.ModelScope{}, Registry: []cockpit.RegistryEntry{}}
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
