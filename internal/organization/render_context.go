package organization

import (
	"cmp"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

// RenderContext prepares one deterministic UTF-8 text package. It checks the
// shared exact Package inventory as well as model identity/digest. The Host's
// binding and authority projections are displayed claims, not verified grants.
// The Host must enforce review phases on tools too. No content is acquired,
// sent, interpreted as policy, silently repaired or truncated by this function.
func RenderContext(selection ContextSelection, settings Settings, input RenderInput) (ContextPackage, error) {
	manager, err := validateRender(selection, settings, input)
	if err != nil {
		return ContextPackage{}, err
	}
	template, text := roleTemplate(input.View, selection.Package() == Root)
	product := RenderSource{Kind: ProductRenderSource, Product: template}
	w := contextWriter{limit: input.MaxContextBytes}
	w.add("role", strings.TrimSuffix(text, mandateTemplate), product)
	w.add("mandate", mandateTemplate, product)
	operation, instruction := ApplyOperation()
	if input.View != ManagerView {
		w.add("operation-is-review-subject", reviewOperationText, productLiteral("review-operation", reviewOperationText))
	}
	w.add("static-apply-operation", instruction, RenderSource{Kind: ProductRenderSource, Product: operation})
	w.reference("operation-record-reference-not-authentication", input.Task.Outcome.Source)
	w.reference("captured-basis-reference-not-verification", input.InputBinding)
	w.json("task-identity", struct{ ID, Package, ModelID, ModelDigest string }{input.Task.ID, input.Task.Package, selection.ModelID(), selection.ModelDigest()}, hostRender(input.InputBinding))
	if input.View != ManagerView && input.Task.Purpose != "" {
		w.json("reviewed-submission-purpose", input.Task.Purpose, hostRender(input.InputBinding))
	}
	parent, children := selection.treePosition()
	w.json("tree-position", struct {
		Manager, Parent string
		DirectChildren  []string
	}{selection.Package(), parent, children}, hostRender(input.InputBinding))
	w.add("tree-position-meaning", treePositionText, productLiteral("tree-position", treePositionText))
	if input.Process != nil {
		w.json("host-manager-phase-not-admission", input.Process.Phase, hostRender(input.Process.Source))
		phaseText := managerPhaseText(input.Process.Phase)
		w.add("manager-phase-instruction-not-grant", phaseText, productLiteral("phase/"+string(input.Process.Phase), phaseText))
	}
	for _, hint := range input.ParentHints {
		w.json("asserted-direct-parent-hint-sender", hint.Package, hostRender(hint.Advice.Source))
		w.host("parent-hint-not-authority", hint.Advice)
	}
	for _, m := range input.Messages {
		w.json("message-identity", struct{ ID, Kind, From, To, About string }{m.ID, m.Kind, m.From, m.To, m.About}, hostRender(m.Body.Source))
		w.host("message-body-not-authority", m.Body)
	}
	if input.OwnerInput != nil {
		w.host("root-owner-input", *input.OwnerInput)
	}
	for _, text := range input.Task.Constraints {
		w.host("task-constraint", text)
	}
	for _, text := range input.Task.Requirements {
		w.host("required-review-or-check", text)
	}
	w.reference("complete-open-finding-set-reference", input.Task.Findings.Source)
	findings := slices.Clone(input.Task.Findings.Items)
	slices.SortFunc(findings, func(a, b FindingFact) int { return cmp.Compare(a.ID, b.ID) })
	for _, finding := range findings {
		w.json("open-finding-identity", finding.ID, hostRender(input.Task.Findings.Source))
		w.host("open-finding-grounds", finding.Grounds)
		for _, ref := range finding.Evidence {
			w.reference("finding-evidence-reference-not-payload", ref)
		}
	}
	w.json("operational-settings-not-grants", struct {
		StepTimeout          string
		RejectionLimit       int
		CreationAreas        []string
		BatchNoChangeReviews bool
	}{manager.StepTimeout.Value.String(), manager.RejectionLimit.Value, directoryStrings(manager.CreationRoots.Value), manager.BatchNoChangeReviews.Value}, configRender(manager.StepTimeout.Source), configRender(manager.RejectionLimit.Source), configRender(manager.CreationRoots.Source), configRender(manager.BatchNoChangeReviews.Source))
	ceilings := settings.Ceilings()
	w.json("project-ceilings", struct {
		StepTimeout    string
		RejectionLimit int
		CreationAreas  []string
	}{ceilings.StepTimeout.Value.String(), ceilings.RejectionLimit.Value, directoryStrings(ceilings.CreationRoots.Value)}, configRender(ceilings.StepTimeout.Source), configRender(ceilings.RejectionLimit.Source), configRender(ceilings.CreationRoots.Source))
	w.json("max-context-bytes", input.MaxContextBytes.Value, configRender(input.MaxContextBytes.Source))
	for _, text := range input.RemainingLimits {
		w.host("remaining-limit-view-not-accounting", text)
	}
	role := manager.Manager
	if input.View != ManagerView {
		role = manager.Reviewer
	}
	w.json("selected-profile-not-provider-wire", role.Profile.Value, configRender(role.Profile.Source), configRender(role.Profile.Value.Source))
	for _, entry := range selection.Definitions() {
		source := RenderSource{Kind: ModelRenderSource, Definition: entry.Definition.Identity, ModelID: entry.Definition.ModelID, ModelDigest: entry.ModelDigest, Model: entry.Definition.Source}
		sources := []RenderSource{source}
		for _, reason := range entry.Reasons {
			if reason.Policy != nil {
				sources = append(sources, configRender(*reason.Policy))
			}
		}
		w.json("definition-identity-contract-and-inclusion", struct {
			Identity, Kind       model.Identity
			ModelID, ModelDigest string
			Source               model.Source
			Reasons              []ContextReason
		}{entry.Definition.Identity, entry.Definition.Kind, entry.Definition.ModelID, entry.ModelDigest, entry.Definition.Source, entry.Reasons}, sources...)
		if entry.Definition.Title != nil {
			w.add("original-title", *entry.Definition.Title, source)
		}
		w.add("original-body", entry.Definition.Body, source)
		spec, err := renderSpec(entry.Definition.Spec)
		if err != nil {
			return ContextPackage{}, err
		}
		w.json("lossless-typed-spec-not-original-yaml", spec, source)
		w.json("original-body-link-occurrences", entry.Definition.BodyLinks, source)
	}
	renderDirectory(&w, selection, input)
	renderArtifacts(&w, input.OwnArtifacts, "own")
	if input.Review != nil {
		w.reference("exact-review-subject-reference", input.Review.Subject)
		w.reference("exact-candidate-reference", input.Review.Candidate)
		w.reference("recorded-expectations-reference", input.Review.Expectations)
		renderArtifacts(&w, input.Review.Files, "review-candidate")
		for _, evidence := range input.Review.Evidence {
			w.host("review-evidence-payload", evidence)
		}
	}
	for _, source := range role.InstructionSources {
		w.json("custom-instruction-selection-source", source, configRender(source))
	}
	// Append original custom instruction bytes last, without newline/BOM repair
	// or deduplication. Frames are outside their exact TextStart/TextEnd range.
	for _, instruction := range role.Instructions {
		w.add("original-custom-instruction", string(instruction.Value.Bytes), configRender(instruction.Source), configRender(instruction.Value.Source))
	}
	if w.err != nil {
		return ContextPackage{}, w.err
	}
	if w.tooLarge {
		return ContextPackage{}, &RenderRefusal{Code: "context-too-large", Detail: "split the Package or explicitly revise its context limit; nothing was truncated", RequiredBytes: w.count, LimitBytes: w.limit.Value, LimitSource: w.limit.Source, Sections: slices.Clone(w.sizes)}
	}
	return ContextPackage{view: input.View, packagePath: selection.Package(), template: template, format: contextFormat(), bytes: w.bytes, sections: w.sections}, nil
}
