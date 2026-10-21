package apply

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strconv"
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

// requiredDefinition is a checked Definition that required checks name as the
// reason they exist, with the names of those checks.
type requiredDefinition struct {
	Definition model.Definition
	Checks     []string
}

// briefChild is one direct child of a Manager as its parent sees it: where it
// works, which Definitions it must realize there and the Managers below it.
type briefChild struct {
	Path        string
	Areas       []string
	Definitions []string
	// Below lists the child's descendants, parents before their children; they
	// belong to the team and are coordinated by their own parents.
	Below []briefDescendant
}

// briefDescendant is a Manager below a direct child.
type briefDescendant struct {
	Path        string
	Parent      string
	Areas       []string
	Definitions []string
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
	// Required lists the Definitions behind the required checks and Project the
	// inventory of every local Definition with its owner. Both are set for Root,
	// which coordinates the whole model and receives its project and boundary
	// obligations explicitly, attributed to their actual owners.
	Required   []requiredDefinition
	Project    []string
	ProjectIDs []string
	// Children are the Manager's direct children, in team order.
	Children []briefChild
}

// projectObligations lists every local Definition of the checked model with
// its kind, title and owning Package, in identity order.
func projectObligations(checked model.Result) []string {
	var out []string
	for _, owner := range checked.Models {
		if !owner.Local {
			continue
		}
		for _, d := range owner.Definitions {
			out = append(out, fmt.Sprintf("%s (%s): %s, owned by %s", identityText(d.Identity), identityText(d.Kind), definitionTitle(d), d.Identity.Package))
		}
	}
	slices.Sort(out)
	return out
}

// projectIdentities lists the identities of the project inventory in the
// order of projectObligations' entries' identities.
func projectIdentities(checked model.Result) []string {
	var out []string
	for _, owner := range checked.Models {
		if owner.Local {
			for _, d := range owner.Definitions {
				out = append(out, identityText(d.Identity))
			}
		}
	}
	slices.Sort(out)
	return out
}

// coverage is the context record of a rendered Brief: the Definitions it
// states in full (the selection and, for Root, the Definitions of the required
// checks) and those it only lists: Root's inventory and a parent's children's
// and descendants' Definitions.
func (in briefInput) coverage(brief string) ContextBinding {
	c := ContextBinding{Brief: "sha256:" + fmt.Sprintf("%x", sha256.Sum256([]byte(brief))), Full: []string{}, Reference: []string{}}
	for _, d := range in.Selection.Definitions() {
		c.Full = append(c.Full, identityText(d.Definition.Identity))
	}
	for _, r := range in.Required {
		if id := identityText(r.Definition.Identity); !slices.Contains(c.Full, id) {
			c.Full = append(c.Full, id)
		}
	}
	listed := slices.Clone(in.ProjectIDs)
	for _, child := range in.Children {
		for _, d := range child.Definitions {
			listed = append(listed, strings.SplitN(d, ": ", 2)[0])
		}
		for _, below := range child.Below {
			for _, d := range below.Definitions {
				listed = append(listed, strings.SplitN(d, ": ", 2)[0])
			}
		}
	}
	for _, id := range listed {
		if !slices.Contains(c.Full, id) && !slices.Contains(c.Reference, id) {
			c.Reference = append(c.Reference, id)
		}
	}
	return c
}

func identityText(id model.Identity) string { return id.Package + "/" + id.ID }

// selectionOf returns the context selection of a Package, or the empty one.
func selectionOf(selections []organization.ContextSelection, pkg string) (selection organization.ContextSelection) {
	for _, candidate := range selections {
		if candidate.Package() == pkg {
			selection = candidate
		}
	}
	return selection
}

// ownDefinitions lists the Definitions a Package itself defines, each with its
// title, as its parent's Brief shows them.
func ownDefinitions(selections []organization.ContextSelection, pkg string) (own []string) {
	for _, definition := range selectionOf(selections, pkg).Definitions() {
		if definition.Definition.Identity.Package == pkg {
			own = append(own, identityText(definition.Definition.Identity)+": "+definitionTitle(definition.Definition))
		}
	}
	return own
}

// briefChildren describes the direct children of a Manager for its Brief: where
// each works, which Definitions it defines and the Managers below it, parents
// before their children. managers is the whole topology; areas names the
// artifact areas of a Package.
func briefChildren(manager organization.Manager, managers []organization.Manager, selections []organization.ContextSelection, areas func(string) []string) []briefChild {
	childrenOf := map[string][]string{}
	for _, m := range managers {
		childrenOf[m.Package] = m.Children
	}
	var children []briefChild
	for _, path := range manager.Children {
		child := briefChild{Path: path, Areas: areas(path), Definitions: ownDefinitions(selections, path)}
		var below func(string)
		below = func(parent string) {
			for _, d := range childrenOf[parent] {
				child.Below = append(child.Below, briefDescendant{Path: d, Parent: parent, Areas: areas(d), Definitions: ownDefinitions(selections, d)})
				below(d)
			}
		}
		below(path)
		children = append(children, child)
	}
	return children
}

// managerRole is the short description of a Manager that its contacts see.
func managerRole(manager organization.Manager, selection organization.ContextSelection) string {
	if manager.Package == organization.Root {
		return "foundations, coordination and integration of the whole project"
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
	b.WriteString("Only these areas and your local registry are writable in your work turns; everything else in /work is read-only. Ask look_up for the owner of any path.\n")
	if len(in.Children) > 0 {
		b.WriteString("\nYour children\nYour children realize their own Definitions in their areas. You supply the foundations they need within your own area, coordinate them and integrate their results; you do not implement their areas, and you cannot release them from their obligations. Cross-area matters between them are yours to settle: decide which child's realization provides a shared rule or interface, and have the others use it instead of rebuilding it.\n")
		for _, child := range in.Children {
			areas := "none"
			if len(child.Areas) > 0 {
				areas = strings.Join(child.Areas, "; ")
			}
			fmt.Fprintf(&b, "- %s\n  areas: %s\n", child.Path, areas)
			if len(child.Definitions) == 0 {
				b.WriteString("  Definitions: none\n")
			}
			for _, definition := range child.Definitions {
				fmt.Fprintf(&b, "  Definition %s\n", definition)
			}
			for _, d := range child.Below {
				areas := "none"
				if len(d.Areas) > 0 {
					areas = strings.Join(d.Areas, "; ")
				}
				fmt.Fprintf(&b, "  below it: %s, a child of %s, areas: %s\n", d.Path, d.Parent, areas)
				for _, definition := range d.Definitions {
					fmt.Fprintf(&b, "    Definition %s\n", definition)
				}
			}
		}
		b.WriteString("Every area listed here has its owner in the team; the Managers below your children are coordinated through their own parents.\n")
	}
	definitions := in.Selection.Definitions()
	example := "package/id"
	switch {
	case len(definitions) > 0:
		example = identityText(definitions[0].Definition.Identity)
	case len(in.Required) > 0:
		example = identityText(in.Required[0].Definition.Identity)
	}
	fmt.Fprintf(&b, "\nRegistry\nYour local registry %s explains what your artifacts realize. Register every file you create, or a directory holding it, there in the same change; a file that is not registered is unexplained and blocks your submission. %s\n\n", in.Shard, registryShape(example))
	b.WriteString("Protected intent\nThe canonical model, its locks, the ownership, the execution settings and the ignore policy under /work/.markitect are fixed authority inputs. Read them; never change them.\n\n")
	if len(in.Project) > 0 {
		b.WriteString("Project obligations\nYou coordinate the whole model. Every Definition below is realized by its owner; you supply the shared foundations you own, integrate your children's results and keep the project boundary coherent. Use look_up for any Definition's full text.\n")
		for _, line := range in.Project {
			fmt.Fprintf(&b, "- %s\n", line)
		}
		b.WriteString("\n")
	}

	b.WriteString("Model obligations\nYour own Definitions are your obligations. Included Definitions describe contracts of other areas: rely on their owners' realization, coordinated through your parent or children, instead of realizing them again in your area.\n")
	requiredBy := make(map[model.Identity][]string)
	for _, required := range in.Required {
		requiredBy[required.Definition.Identity] = required.Checks
		if !slices.ContainsFunc(definitions, func(d organization.ContextDefinition) bool {
			return d.Definition.Identity == required.Definition.Identity
		}) {
			definitions = append(definitions, organization.ContextDefinition{Definition: required.Definition})
		}
	}
	if len(definitions) == 0 {
		b.WriteString("This Manager has no Definitions of its own and none are included.\n")
	}
	isOwn := func(d organization.ContextDefinition) bool {
		return d.Definition.Identity.Package == in.Manager.Package
	}
	for _, definition := range definitions {
		if isOwn(definition) {
			writeDefinition(&b, definition, true, in.Manager, nil)
		}
	}
	if in.Manager.Package != organization.Root {
		b.WriteString("\nYour own Definitions are yours to realize in your area. Your parent coordinates; it cannot take them over or waive them.\n")
	}
	b.WriteString("Never realize another area's Definitions or touch anything outside your area: use the owner's realization, and ask through the tree when you need something from another area.\n")
	wroteIncluded := false
	for _, definition := range definitions {
		if isOwn(definition) {
			continue
		}
		if !wroteIncluded {
			b.WriteString("\nIncluded Definitions (context from other Packages)\n")
			wroteIncluded = true
		}
		writeDefinition(&b, definition, false, in.Manager, requiredBy[definition.Definition.Identity])
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

// registryShape shows the entries of a registry shard.
func registryShape(example string) string {
	return fmt.Sprintf("Entries use this shape:\n\napiVersion: markitect.registry/v1\nartifacts:\n  - path: path/of/the/file/from/the/repository/root   # or a directory ending in / for the files below it that you own\n    form: code   # one of code, test, doc, build, config\n    realizes: [%s]   # existing Definition identities\n    why: One sentence on what it contributes.", example)
}

func writeDefinition(b *strings.Builder, c organization.ContextDefinition, own bool, reader organization.Manager, requiredBy []string) {
	d := c.Definition
	fmt.Fprintf(b, "\n%s (%s): %s", identityText(d.Identity), identityText(d.Kind), definitionTitle(d))
	if !own {
		fmt.Fprintf(b, " (owned by %s)", d.Identity.Package)
	}
	b.WriteString("\n")
	if strings.TrimSpace(d.Body) != "" {
		fmt.Fprintf(b, "%s\n", strings.TrimSpace(d.Body))
	}
	// A body link names its target through the source's own import aliases;
	// each link as written is paired with the identity it resolves to.
	var links []string
	for _, link := range d.BodyLinks {
		text := identityText(link.Target)
		if c := link.Coordinates; c != nil && c.Body != nil && 0 <= c.Body.StartByte && c.Body.StartByte < c.Body.EndByte && c.Body.EndByte <= len(d.Body) {
			text = d.Body[c.Body.StartByte:c.Body.EndByte] + " = " + text
		}
		if !slices.Contains(links, text) {
			links = append(links, text)
		}
	}
	if len(links) > 0 {
		fmt.Fprintf(b, "  links: %s\n", strings.Join(links, "; "))
	}
	for _, field := range d.Spec {
		fmt.Fprintf(b, "  %s: %s\n", field.Name, valueText(field.Value))
	}
	if !own {
		for _, reason := range c.Reasons {
			fmt.Fprintf(b, "  included because: %s\n", reasonText(reason))
		}
		for _, check := range requiredBy {
			if reader.Package == organization.Root {
				fmt.Fprintf(b, "  included because: required by check %s; realized by %s, integrated and verified by you\n", check, d.Identity.Package)
			} else {
				fmt.Fprintf(b, "  included because: required by check %s; realized by %s\n", check, d.Identity.Package)
			}
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
		return decimalText(number)
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

// decimalText writes an exact decimal coefficient * 10^exponent in plain
// notation, such as 1.5 or 2500, or in e-notation when the exponent is too
// large for that.
func decimalText(d model.Decimal) string {
	exp, err := strconv.Atoi(d.Exponent)
	if err != nil || exp > 30 || exp < -30 {
		return d.Coefficient + "e" + d.Exponent
	}
	sign, digits := "", strings.TrimPrefix(d.Coefficient, "-")
	if digits != d.Coefficient {
		sign = "-"
	}
	if exp >= 0 {
		if digits == "0" {
			return "0"
		}
		return sign + digits + strings.Repeat("0", exp)
	}
	if n := -exp; n >= len(digits) {
		digits = strings.Repeat("0", n-len(digits)+1) + digits
	}
	point := len(digits) + exp
	return sign + digits[:point] + "." + digits[point:]
}
