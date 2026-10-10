package canonical

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
)

const (
	foundationAPIVersion = "markitect.foundation/v1"
	projectionKind       = "Projection"
	projectionPolicyKind = "ProjectionPolicy"
	maxProjectionTargets = 10000
	maxProjectionBytes   = 128 << 20
)

// ProjectionRequest is the bounded, detached input contract passed from the
// Host to one explicitly selected Projector. It includes only explicitly
// selected Definitions, their directly resolved references, applicable
// selected policies, relevant Kind contracts, and caller-supplied target bytes.
type ProjectionRequest struct {
	Revision         string                `json:"revision"`
	ModelDigest      string                `json:"modelDigest"`
	RequestDigest    string                `json:"requestDigest"`
	Projection       core.Definition       `json:"projection"`
	Binding          ProjectionBinding     `json:"binding"`
	ModulePin        Pin                   `json:"modulePin"`
	Projector        ProjectorRegistration `json:"projector"`
	Definitions      []core.Definition     `json:"definitions"`
	Schemas          []core.Schema         `json:"schemas"`
	Policies         []core.Definition     `json:"policies"`
	Edges            []core.Edge           `json:"edges"`         // Only edges whose endpoints are both explicitly selected.
	ExternalEdges    []core.Edge           `json:"externalEdges"` // Reported as gaps and excluded from Definitions and Edges.
	TargetRepository string                `json:"targetRepository"`
	TargetPath       string                `json:"targetPath"`
	TargetPrefix     string                `json:"targetPrefix"`
	TargetFiles      map[string][]byte     `json:"targetFiles"`
	TargetDigests    map[string]string     `json:"targetDigests"`
}

// ProjectionBinding is runtime/project configuration. It selects an exact
// installed Projection Module for one canonical Projection without changing
// canonical identity or desired representation.
type ProjectionBinding struct {
	Projection core.DefinitionIdentity `json:"projection" yaml:"projection"`
	Module     string                  `json:"module" yaml:"module"`
}

// BindProjection resolves one exact, owner-selected Foundation Projection
// against an already compiled Core model and explicitly activated Modules. It
// performs no I/O, closure traversal, source discovery, or materialization.
func BindProjection(model core.Model, activation Activation, bindings []ProjectionBinding, identity core.DefinitionIdentity, targets map[string][]byte) (ProjectionRequest, error) {
	if err := validateModelEvidence(model); err != nil {
		return ProjectionRequest{}, err
	}
	projection, ok := model.Definition(identity)
	if !ok {
		return ProjectionRequest{}, fmt.Errorf("Projection %s is not present in the compiled model", identity.Key())
	}
	if projection.APIVersion != foundationAPIVersion || projection.Kind != projectionKind {
		return ProjectionRequest{}, fmt.Errorf("selected Definition must be %s/%s", foundationAPIVersion, projectionKind)
	}
	if err := validateSourceDigest(projection.Source.Digest, "Projection "+projection.Identity().Key()); err != nil {
		return ProjectionRequest{}, err
	}

	sourceSpec, ok := objectField(projection.Spec, "source")
	if !ok {
		return ProjectionRequest{}, fmt.Errorf("Projection source must be an object")
	}
	selectedIDs, err := projectionDefinitionIdentities(sourceSpec["definitions"])
	if err != nil {
		return ProjectionRequest{}, err
	}
	selected := make([]core.Definition, 0, len(selectedIDs))
	selectedByKey := make(map[string]core.Definition, len(selectedIDs))
	for _, selectedID := range selectedIDs {
		definition, exists := model.Definition(selectedID)
		if !exists {
			return ProjectionRequest{}, fmt.Errorf("Projection selects unresolved Definition %s", selectedID.Key())
		}
		if err := validateSourceDigest(definition.Source.Digest, "selected Definition "+selectedID.Key()); err != nil {
			return ProjectionRequest{}, err
		}
		selected = append(selected, definition)
		selectedByKey[selectedID.Key()] = definition
	}

	binding, err := exactProjectionBinding(bindings, identity)
	if err != nil {
		return ProjectionRequest{}, err
	}
	moduleName := binding.Module
	representation, ok := stringField(projection.Spec, "representation")
	if !ok || strings.TrimSpace(representation) == "" || strings.TrimSpace(representation) != representation {
		return ProjectionRequest{}, fmt.Errorf("Projection representation must be a nonempty exact target identity")
	}
	registered, err := selectedProjector(activation, moduleName)
	if err != nil {
		return ProjectionRequest{}, err
	}
	if registered.Registration.Target != representation {
		return ProjectionRequest{}, fmt.Errorf("Projection representation %q does not match selected Projection Module target %q", representation, registered.Registration.Target)
	}
	if err := validateProjectorScope(registered.Registration, selected); err != nil {
		return ProjectionRequest{}, err
	}

	targetSpec, ok := objectField(projection.Spec, "target")
	if !ok {
		return ProjectionRequest{}, fmt.Errorf("Projection target must be an object")
	}
	repository, ok := stringField(targetSpec, "repository")
	if !ok || repository != "." {
		return ProjectionRequest{}, fmt.Errorf("Projection target.repository %q is unsupported; this Host binds only repository '.'", repository)
	}
	targetPath, ok := stringField(targetSpec, "path")
	if !ok {
		return ProjectionRequest{}, fmt.Errorf("Projection target.path must be a nonempty path prefix")
	}
	prefix, err := normalizeTargetPrefix(targetPath)
	if err != nil {
		return ProjectionRequest{}, err
	}
	if !targetUnderAllowedRoot(prefix, registered.Registration.AllowedRoots) {
		return ProjectionRequest{}, fmt.Errorf("Projection target path %q is outside Projector %q allowedRoots", targetPath, registered.Registration.ID)
	}
	targetFiles, targetDigests, err := boundedTargetState(targets, prefix)
	if err != nil {
		return ProjectionRequest{}, err
	}

	policies, err := selectedPolicies(model, projection)
	if err != nil {
		return ProjectionRequest{}, err
	}
	for _, policy := range policies {
		if err := validateSourceDigest(policy.Source.Digest, "ProjectionPolicy "+policy.Identity().Key()); err != nil {
			return ProjectionRequest{}, err
		}
	}
	policyKinds, err := validateSelectedPolicies(policies, activation.Schemas, registered.Registration)
	if err != nil {
		return ProjectionRequest{}, err
	}
	schemas := selectedKindSchemas(model, selected, policyKinds)
	for _, schema := range schemas {
		if err := validateSourceDigest(schema.Source.Digest, "Schema "+schema.APIVersion); err != nil {
			return ProjectionRequest{}, err
		}
	}

	inside, outside := selectedEdges(model.Edges, selectedByKey)
	request := ProjectionRequest{
		Revision: model.Revision, ModelDigest: normalizeDigest(model.Digest), Projection: projection, Binding: binding,
		ModulePin: registered.Module, Projector: cloneProjectorRegistration(registered.Registration),
		Definitions: selected, Schemas: schemas, Policies: policies,
		Edges: inside, ExternalEdges: outside,
		TargetRepository: repository, TargetPath: targetPath, TargetPrefix: prefix,
		TargetFiles: targetFiles, TargetDigests: targetDigests,
	}
	request.RequestDigest, err = projectionRequestDigest(request)
	if err != nil {
		return ProjectionRequest{}, fmt.Errorf("encode Projection request digest: %w", err)
	}
	return request, nil
}

func exactProjectionBinding(bindings []ProjectionBinding, identity core.DefinitionIdentity) (ProjectionBinding, error) {
	var matched ProjectionBinding
	found := false
	for _, binding := range bindings {
		if binding.Projection.Key() != identity.Key() {
			continue
		}
		if found {
			return ProjectionBinding{}, fmt.Errorf("Projection %s has duplicate runtime Module bindings", identity.Key())
		}
		matched = binding
		found = true
	}
	if !found {
		return ProjectionBinding{}, fmt.Errorf("Projection %s has no runtime Module binding", identity.Key())
	}
	if matched.Module == "" || strings.TrimSpace(matched.Module) != matched.Module {
		return ProjectionBinding{}, fmt.Errorf("Projection %s runtime Module binding must contain one exact Module name", identity.Key())
	}
	return matched, nil
}

func validateModelEvidence(model core.Model) error {
	if strings.TrimSpace(model.Revision) == "" || strings.TrimSpace(model.Revision) != model.Revision {
		return fmt.Errorf("compiled model must carry an exact nonempty source revision")
	}
	if _, err := digestHex(model.Digest); err != nil {
		return fmt.Errorf("compiled model digest: %w", err)
	}
	compiled, diagnostics := core.Compile(model.Schemas, model.Definitions, model.Revision)
	if len(diagnostics) != 0 {
		return fmt.Errorf("model is not a valid compiled Core model: %s", diagnostics[0].Message)
	}
	if compiled.Digest != model.Digest || !reflect.DeepEqual(compiled.Edges, model.Edges) {
		return fmt.Errorf("model digest or resolved edges do not match the supplied compiled Core model")
	}
	return nil
}

func projectionDefinitionIdentities(value any) ([]core.DefinitionIdentity, error) {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("Projection source.definitions must explicitly select at least one Definition")
	}
	identities := make([]core.DefinitionIdentity, 0, len(items))
	seen := map[string]bool{}
	for index, item := range items {
		fields, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Projection source.definitions[%d] must be an identity object", index)
		}
		api, apiOK := stringField(fields, "apiVersion")
		kind, kindOK := stringField(fields, "kind")
		namespace, namespaceOK := fields["namespace"].(string)
		name, nameOK := stringField(fields, "name")
		if !apiOK || !kindOK || !namespaceOK || !nameOK || api == "" || kind == "" || name == "" || strings.TrimSpace(api) != api || strings.TrimSpace(kind) != kind || strings.TrimSpace(namespace) != namespace || strings.TrimSpace(name) != name {
			return nil, fmt.Errorf("Projection source.definitions[%d] requires exact apiVersion, kind, namespace, and name values", index)
		}
		identity := core.DefinitionIdentity{APIVersion: api, Kind: kind, Namespace: namespace, Name: name}
		if seen[identity.Key()] {
			return nil, fmt.Errorf("Projection repeats selected Definition %s", identity.Key())
		}
		seen[identity.Key()] = true
		identities = append(identities, identity)
	}
	return identities, nil
}

func selectedProjector(activation Activation, moduleName string) (RegisteredProjector, error) {
	var match RegisteredProjector
	found := false
	for _, projector := range activation.Projectors {
		if projector.Module.Name != moduleName {
			continue
		}
		if found {
			return RegisteredProjector{}, fmt.Errorf("Projection Module %q has multiple registered entrypoints; binding requires exactly one", moduleName)
		}
		match, found = projector, true
	}
	if !found {
		return RegisteredProjector{}, fmt.Errorf("Projection selects unavailable Projection Module %q", moduleName)
	}
	selected := false
	for _, pin := range activation.Modules {
		if pin == match.Module {
			selected = true
			break
		}
	}
	if !selected {
		return RegisteredProjector{}, fmt.Errorf("Projection Module %q is not present in the explicit activation", moduleName)
	}
	if activation.ModuleTypes[match.Module] != ModuleTypeProjection {
		return RegisteredProjector{}, fmt.Errorf("selected Module %q is not registered as a Projection Module", moduleName)
	}
	if err := validatePin(match.Module); err != nil {
		return RegisteredProjector{}, fmt.Errorf("registered Projector has invalid exact Module pin: %w", err)
	}
	return match, nil
}

func validateProjectorScope(projector ProjectorRegistration, definitions []core.Definition) error {
	if projector.AllKinds {
		return nil
	}
	supported := map[string]bool{}
	for _, kind := range projector.SupportedKinds {
		supported[kind.Key()] = true
	}
	for _, definition := range definitions {
		kind := core.KindIdentity{APIVersion: definition.APIVersion, Kind: definition.Kind}
		if !supported[kind.Key()] {
			return fmt.Errorf("Projector %q does not support selected Kind %s/%s", projector.ID, kind.APIVersion, kind.Kind)
		}
	}
	return nil
}

func selectedPolicies(model core.Model, projection core.Definition) ([]core.Definition, error) {
	byKey := make(map[string]core.Definition, len(model.Definitions))
	for _, definition := range model.Definitions {
		byKey[definition.Identity().Key()] = definition
	}
	var policies []core.Definition
	seen := map[string]bool{}
	for _, edge := range model.Edges {
		if edge.From != projection.Identity().Key() || (edge.Property != "policies" && !strings.HasPrefix(edge.Property, "policies[")) {
			continue
		}
		definition, ok := byKey[edge.To]
		if !ok || definition.APIVersion != foundationAPIVersion || definition.Kind != projectionPolicyKind {
			return nil, fmt.Errorf("Projection policy reference %q does not resolve to a Foundation ProjectionPolicy", edge.To)
		}
		key := definition.Identity().Key()
		if seen[key] {
			return nil, fmt.Errorf("Projection repeats selected ProjectionPolicy %s", key)
		}
		seen[key] = true
		selected, ok := model.Definition(definition.Identity())
		if !ok {
			return nil, fmt.Errorf("ProjectionPolicy %s is not in the compiled model", definition.Identity().Key())
		}
		policies = append(policies, selected)
	}
	return policies, nil
}

func validateSelectedPolicies(policies []core.Definition, schemas []core.Schema, projector ProjectorRegistration) ([]core.KindIdentity, error) {
	policyByKind := map[string]core.Definition{}
	var policyKinds []core.KindIdentity
	for _, policy := range policies {
		sourceKind, target, _, err := projectionPolicyFields(policy)
		if err != nil {
			return nil, err
		}
		if target != projector.Target {
			return nil, fmt.Errorf("ProjectionPolicy %s targets technology %q but selected Projection Module targets %q", policy.Identity().Key(), target, projector.Target)
		}
		if !containsKind(schemas, sourceKind) {
			return nil, fmt.Errorf("ProjectionPolicy %s references unavailable Kind %s/%s", policy.Identity().Key(), sourceKind.APIVersion, sourceKind.Kind)
		}
		policyKinds = append(policyKinds, sourceKind)
		key := sourceKind.Key() + "\x00" + target
		if prior, exists := policyByKind[key]; exists && prior.Identity().Key() != policy.Identity().Key() {
			return nil, fmt.Errorf("multiple selected ProjectionPolicies map Kind %s/%s to target %q; the Host requires one unambiguous policy mapping and does not judge whether their prose conflicts", sourceKind.APIVersion, sourceKind.Kind, target)
		}
		policyByKind[key] = policy
	}
	return policyKinds, nil
}

func containsKind(schemas []core.Schema, identity core.KindIdentity) bool {
	for _, schema := range schemas {
		if schema.APIVersion == identity.APIVersion {
			_, ok := schema.Kinds[identity.Kind]
			return ok
		}
	}
	return false
}

func projectionPolicyFields(policy core.Definition) (core.KindIdentity, string, string, error) {
	source, ok := policy.Spec["sourceKind"].(map[string]any)
	if !ok {
		return core.KindIdentity{}, "", "", fmt.Errorf("ProjectionPolicy %s sourceKind must be an exact kind identity", policy.Identity().Key())
	}
	api, apiOK := stringField(source, "apiVersion")
	kind, kindOK := stringField(source, "kind")
	target, targetOK := stringField(policy.Spec, "targetTechnology")
	guidance, guidanceOK := stringField(policy.Spec, "guidance")
	if !apiOK || !kindOK || !targetOK || !guidanceOK || api == "" || kind == "" || strings.TrimSpace(target) != target || target == "" {
		return core.KindIdentity{}, "", "", fmt.Errorf("ProjectionPolicy %s has invalid sourceKind, targetTechnology, or guidance", policy.Identity().Key())
	}
	return core.KindIdentity{APIVersion: api, Kind: kind}, target, guidance, nil
}

func selectedKindSchemas(model core.Model, definitions []core.Definition, policyKinds []core.KindIdentity) []core.Schema {
	needed := map[string]map[string]bool{}
	for _, definition := range definitions {
		if needed[definition.APIVersion] == nil {
			needed[definition.APIVersion] = map[string]bool{}
		}
		needed[definition.APIVersion][definition.Kind] = true
	}
	for _, kind := range policyKinds {
		if needed[kind.APIVersion] == nil {
			needed[kind.APIVersion] = map[string]bool{}
		}
		needed[kind.APIVersion][kind.Kind] = true
	}
	var schemas []core.Schema
	for _, schema := range model.Schemas {
		kinds := needed[schema.APIVersion]
		if len(kinds) == 0 {
			continue
		}
		selectedSchema := schema
		selectedSchema.Kinds = make(map[string]core.Kind, len(kinds))
		for kind := range kinds {
			identity := core.KindIdentity{APIVersion: schema.APIVersion, Kind: kind}
			if definition, ok := model.Kind(identity); ok {
				selectedSchema.Kinds[kind] = definition
			}
		}
		schemas = append(schemas, selectedSchema)
	}
	sort.Slice(schemas, func(i, j int) bool { return schemas[i].APIVersion < schemas[j].APIVersion })
	return schemas
}

func selectedEdges(edges []core.Edge, selected map[string]core.Definition) ([]core.Edge, []core.Edge) {
	var inside, outside []core.Edge
	for _, edge := range edges {
		if _, inScope := selected[edge.From]; !inScope {
			continue
		}
		if _, targetSelected := selected[edge.To]; targetSelected {
			inside = append(inside, edge)
		} else {
			outside = append(outside, edge)
		}
	}
	less := func(values []core.Edge) {
		sort.Slice(values, func(i, j int) bool {
			if values[i].From != values[j].From {
				return values[i].From < values[j].From
			}
			if values[i].Property != values[j].Property {
				return values[i].Property < values[j].Property
			}
			return values[i].To < values[j].To
		})
	}
	less(inside)
	less(outside)
	return inside, outside
}

func normalizeTargetPrefix(target string) (string, error) {
	if target == "." {
		return "", nil
	}
	if target == "" || len(target) > 1024 || strings.TrimSpace(target) != target || strings.Contains(target, "\\") || strings.HasPrefix(target, "/") || strings.ContainsAny(target, ":*?<>|\"") {
		return "", fmt.Errorf("Projection target.path %q is not a portable repository-relative prefix", target)
	}
	clean := strings.TrimSuffix(target, "/")
	if clean == "" || strings.HasSuffix(clean, "/") {
		return "", fmt.Errorf("Projection target.path %q has an invalid trailing slash", target)
	}
	for _, component := range strings.Split(clean, "/") {
		if component == "" || component == "." || component == ".." || strings.TrimSpace(component) != component || strings.HasSuffix(component, ".") || strings.EqualFold(component, ".git") || reservedPathComponent(component) {
			return "", fmt.Errorf("Projection target.path %q contains an invalid path component", target)
		}
		for _, char := range component {
			if char < 32 {
				return "", fmt.Errorf("Projection target.path %q contains a control character", target)
			}
		}
	}
	return clean, nil
}

func targetUnderAllowedRoot(prefix string, roots []string) bool {
	for _, root := range roots {
		root = strings.TrimSuffix(root, "/")
		if root == "." || prefix == root || strings.HasPrefix(prefix, root+"/") {
			return true
		}
	}
	return false
}

func boundedTargetState(targets map[string][]byte, prefix string) (map[string][]byte, map[string]string, error) {
	if len(targets) > maxProjectionTargets {
		return nil, nil, fmt.Errorf("supplied target state has %d files, exceeding limit %d", len(targets), maxProjectionTargets)
	}
	files := make(map[string][]byte, len(targets))
	digests := make(map[string]string, len(targets))
	paths := make([]string, 0, len(targets))
	totalBytes := 0
	for target, data := range targets {
		clean, err := normalizeTargetPrefix(target)
		if err != nil || clean == "" || clean != target {
			return nil, nil, fmt.Errorf("supplied target path %q must be a clean exact relative file path", target)
		}
		if prefix != "" && clean != prefix && !strings.HasPrefix(clean, prefix+"/") {
			return nil, nil, fmt.Errorf("supplied target %q is outside Projection target prefix %q", target, prefix)
		}
		totalBytes += len(data)
		if totalBytes > maxProjectionBytes {
			return nil, nil, fmt.Errorf("supplied target state exceeds %d bytes", maxProjectionBytes)
		}
		files[target] = append([]byte(nil), data...)
		digest := sha256.Sum256(data)
		digests[target] = "sha256:" + hex.EncodeToString(digest[:])
		paths = append(paths, target)
	}
	sort.Strings(paths)
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			if pathOverlap(paths[i], paths[j]) {
				return nil, nil, fmt.Errorf("supplied target paths overlap or case-alias: %q and %q", paths[i], paths[j])
			}
		}
	}
	return files, digests, nil
}

func cloneProjectorRegistration(value ProjectorRegistration) ProjectorRegistration {
	value.SupportedKinds = append([]core.KindIdentity(nil), value.SupportedKinds...)
	value.AllowedRoots = append([]string(nil), value.AllowedRoots...)
	value.RequiredChecks = append([]string(nil), value.RequiredChecks...)
	return value
}

func pathOverlap(left, right string) bool {
	a, b := strings.Split(left, "/"), strings.Split(right, "/")
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for i := 0; i < limit; i++ {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}

func reservedPathComponent(value string) bool {
	base := strings.ToUpper(strings.SplitN(value, ".", 2)[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" {
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9'
}

func projectionRequestDigest(request ProjectionRequest) (string, error) {
	type sourceDigest struct {
		Identity string `json:"identity"`
		Digest   string `json:"digest"`
	}
	sources := []sourceDigest{{Identity: request.Projection.Identity().Key(), Digest: request.Projection.Source.Digest}}
	for _, definition := range request.Definitions {
		sources = append(sources, sourceDigest{Identity: definition.Identity().Key(), Digest: definition.Source.Digest})
	}
	for _, policy := range request.Policies {
		sources = append(sources, sourceDigest{Identity: policy.Identity().Key(), Digest: policy.Source.Digest})
	}
	for _, schema := range request.Schemas {
		sources = append(sources, sourceDigest{Identity: schema.APIVersion, Digest: schema.Source.Digest})
	}
	payload := struct {
		Revision         string                `json:"revision"`
		ModelDigest      string                `json:"modelDigest"`
		Projection       core.Definition       `json:"projection"`
		Binding          ProjectionBinding     `json:"binding"`
		ModulePin        Pin                   `json:"modulePin"`
		Projector        ProjectorRegistration `json:"projector"`
		Definitions      []core.Definition     `json:"definitions"`
		Schemas          []core.Schema         `json:"schemas"`
		Policies         []core.Definition     `json:"policies"`
		Edges            []core.Edge           `json:"edges"`
		ExternalEdges    []core.Edge           `json:"externalEdges"`
		TargetRepository string                `json:"targetRepository"`
		TargetPath       string                `json:"targetPath"`
		SourceDigests    []sourceDigest        `json:"sourceDigests"`
		TargetDigests    map[string]string     `json:"targetDigests"`
	}{request.Revision, request.ModelDigest, request.Projection, request.Binding, request.ModulePin, request.Projector, request.Definitions, request.Schemas, request.Policies, request.Edges, request.ExternalEdges, request.TargetRepository, request.TargetPath, sources, request.TargetDigests}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func objectField(fields map[string]any, name string) (map[string]any, bool) {
	value, ok := fields[name]
	if !ok {
		return nil, false
	}
	result, ok := value.(map[string]any)
	return result, ok
}

func stringField(fields map[string]any, name string) (string, bool) {
	value, ok := fields[name].(string)
	return value, ok
}

func digestHex(value string) (string, error) {
	value = strings.TrimPrefix(value, "sha256:")
	if len(value) != 64 {
		return "", fmt.Errorf("must be a SHA-256 digest")
	}
	if _, err := hex.DecodeString(value); err != nil || strings.ToLower(value) != value {
		return "", fmt.Errorf("must be lowercase hexadecimal SHA-256")
	}
	return value, nil
}

func normalizeDigest(value string) string {
	decoded, _ := digestHex(value)
	return "sha256:" + decoded
}

func validateSourceDigest(value, label string) error {
	if _, err := digestHex(value); err != nil {
		return fmt.Errorf("%s source digest: %w", label, err)
	}
	return nil
}
