package apply

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

// briefCheck is one required check as a Manager sees it.
type briefCheck struct {
	Name       string
	Command    string
	Directory  string
	RequiredBy string
}

// briefInput is everything one Manager's standing Brief states. The engine adds
// the mandate, contacts, findings, messages and check results itself.
type briefInput struct {
	Manager   organization.Manager
	Selection organization.ContextSelection
	Settings  organization.ManagerSettings
	Areas     []string
	Shard     string
	Checks    []briefCheck
}

func identityText(id model.Identity) string { return id.Package + "/" + id.ID }

// managerRole is the short description of a Manager that its contacts see.
func managerRole(manager organization.Manager, selection organization.ContextSelection) string {
	if manager.Package == organization.Root {
		return "integration of the whole project and shared files at the artifact roots"
	}
	var titles []string
	for _, definition := range selection.Definitions() {
		if definition.Definition.Identity.Package == manager.Package {
			titles = append(titles, definitionTitle(definition.Definition))
		}
	}
	if len(titles) == 0 {
		return manager.Package
	}
	return manager.Package + ": " + strings.Join(titles, "; ")
}

func definitionTitle(d model.Definition) string {
	if d.Title != nil && *d.Title != "" {
		return *d.Title
	}
	return d.Identity.ID
}

// renderBrief states the Manager's standing context as plain text: who it is,
// what it owns and how it registers files, the protected intent, its
// applicable model obligations with provenance, the required checks and its
// configured instructions.
func renderBrief(in briefInput) string {
	var b strings.Builder
	who := "Package " + in.Manager.Package
	if in.Manager.Package == organization.Root {
		who = "Root"
	}
	fmt.Fprintf(&b, "You are the Manager of %s.\nRole: %s.\n\n", who, managerRole(in.Manager, in.Selection))

	b.WriteString("Your areas\n")
	if len(in.Areas) == 0 {
		b.WriteString("- none\n")
	}
	for _, area := range in.Areas {
		fmt.Fprintf(&b, "- %s\n", area)
	}
	example := "package/id"
	if definitions := in.Selection.Definitions(); len(definitions) > 0 {
		example = identityText(definitions[0].Definition.Identity)
	}
	fmt.Fprintf(&b, "\nRegistry\nEvery file you create is registered by you in your registry shard %s, in the same change; a file that is not registered there is unexplained and blocks the candidate. Entries use this shape:\n\napiVersion: markitect.registry/v1\nartifacts:\n  - path: path/of/the/file/from/the/repository/root\n    form: code   # one of code, test, doc, build, config\n    realizes: [%s]   # existing Definition identities\n    why: One sentence on what the file contributes.\n\n", in.Shard, example)
	b.WriteString("Protected intent\nThe canonical model under /work/.markitect is the binding intent. Read it; never change it.\n\n")

	b.WriteString("Model obligations\nYour own Definitions are your obligations. Included Definitions describe contracts of other areas: rely on their owners' realization, coordinated through your parent or children, instead of realizing them again in your area.\n")
	definitions := in.Selection.Definitions()
	if len(definitions) == 0 {
		b.WriteString("This Manager has no Definitions of its own and none are included.\n")
	}
	wroteIncluded := false
	for _, definition := range definitions {
		own := definition.Definition.Identity.Package == in.Manager.Package
		if !own && !wroteIncluded {
			b.WriteString("\nIncluded Definitions (context from other Packages)\n")
			wroteIncluded = true
		}
		writeDefinition(&b, definition, own)
	}

	b.WriteString("\nRequired checks\n")
	if len(in.Checks) == 0 {
		b.WriteString("- none\n")
	}
	for _, check := range in.Checks {
		fmt.Fprintf(&b, "- %s: %s (directory %s", check.Name, check.Command, check.Directory)
		if check.RequiredBy != "" {
			fmt.Fprintf(&b, ", required by %s", check.RequiredBy)
		}
		b.WriteString(")\n")
	}

	writeInstructions(&b, "Execute", in.Settings.Execute)
	writeInstructions(&b, "Verify", in.Settings.Verify)
	return b.String()
}

func writeDefinition(b *strings.Builder, c organization.ContextDefinition, own bool) {
	d := c.Definition
	fmt.Fprintf(b, "\n%s (%s): %s\n", identityText(d.Identity), identityText(d.Kind), definitionTitle(d))
	if strings.TrimSpace(d.Body) != "" {
		fmt.Fprintf(b, "%s\n", strings.TrimSpace(d.Body))
	}
	for _, field := range d.Spec {
		fmt.Fprintf(b, "  %s: %s\n", field.Name, valueText(field.Value))
	}
	if !own {
		for _, reason := range c.Reasons {
			fmt.Fprintf(b, "  included because: %s\n", reasonText(reason))
		}
	}
}

func reasonText(r organization.ContextReason) string {
	switch r.Kind {
	case organization.OutgoingReference:
		return fmt.Sprintf("%s links to it", identityText(r.From))
	case organization.IncomingReference:
		return fmt.Sprintf("it links to %s", identityText(r.To))
	case organization.KindContract:
		return fmt.Sprintf("kind contract of %s", identityText(r.From))
	case organization.PropertyContract:
		return fmt.Sprintf("property contract used by %s", identityText(r.From))
	case organization.ModeledScope:
		return fmt.Sprintf("modeled scope %s", r.ScopePackage)
	}
	return string(r.Kind)
}

func valueText(v model.Value) string {
	switch v.Kind() {
	case model.StringValue, model.EnumValue:
		text, _ := v.Text()
		return fmt.Sprintf("%q", text)
	case model.BooleanValue:
		value, _ := v.Boolean()
		return fmt.Sprint(value)
	case model.IntegerValue:
		text, _ := v.Integer()
		return text
	case model.NumberValue:
		number, _ := v.Number()
		return number.Coefficient + "e" + number.Exponent
	case model.ReferenceValue:
		reference, _ := v.Reference()
		return identityText(reference)
	case model.ObjectValue:
		fields, _ := v.Fields()
		parts := make([]string, len(fields))
		for i, field := range fields {
			parts[i] = field.Name + ": " + valueText(field.Value)
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case model.ListValue:
		items, _ := v.Items()
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = valueText(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return ""
}

func writeInstructions(b *strings.Builder, role string, settings organization.RoleSettings) {
	if len(settings.Instructions) == 0 {
		return
	}
	fmt.Fprintf(b, "\nConfigured %s instructions\n", role)
	for _, instruction := range settings.Instructions {
		fmt.Fprintf(b, "[%s]\n%s\n", instruction.Value.ID, strings.TrimSpace(string(instruction.Value.Bytes)))
	}
}
