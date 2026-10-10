package projectcoverage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func Classify(request Request) (Report, error) {
	if request.Universe == nil {
		return Report{}, fmt.Errorf("repository universe is required")
	}
	if err := validateOptions(request.Options); err != nil {
		return Report{}, err
	}
	ignore, err := DecodeIgnore(request.Universe.IgnoreBytes)
	if err != nil {
		return Report{}, err
	}
	if err := validateIgnoreAgainstOptions(ignore, request.Options); err != nil {
		return Report{}, err
	}
	if err := validateIgnoreAgainstRequiredArtifacts(ignore, request.Model); err != nil {
		return Report{}, err
	}
	if err := validateRequiredArtifactOwnership(request.Model, request.Options); err != nil {
		return Report{}, err
	}
	report := Report{APIVersion: APIVersion, Accounted: true, Conforming: true,
		IgnoreBytesDigest: sha256Hex(request.Universe.IgnoreBytes), Entries: []Entry{}, Findings: []Finding{},
		LegacyDiagnostics: legacyDiagnostics(request.LegacyRoots, request.LegacyExclusions), ModelDigest: request.Model.ModelDigest,
		ModelReportDigest: request.Model.Digest}
	for _, finding := range request.Model.Findings {
		if finding.Severity == "error" {
			report.ModelErrors = append(report.ModelErrors, finding)
			report.Conforming = false
		}
	}
	if request.Model.Status == "incomplete" {
		report.Conforming = false
	}
	pathStates := append([]PathState(nil), request.Universe.Paths...)
	sort.Slice(pathStates, func(i, j int) bool { return pathStates[i].Path < pathStates[j].Path })
	for _, state := range pathStates {
		entry, matched := classifyPath(state, request, ignore)
		if !matched {
			report.Accounted = false
			report.Conforming = false
			report.Findings = append(report.Findings, Finding{Code: "coverage.unclassified", Path: state.Path,
				Message: "Repository path has no canonical model, Artifact realization, registered tool owner, ignore reason, or transitional exclusion.", Severity: "error"})
			entry = Entry{Path: state.Path, Artifacts: []string{}, Statements: []string{}, Head: state.Head, Index: state.Index,
				Worktree: state.Worktree, OpaqueBoundary: state.OpaqueBoundary}
		}
		if entry.Class == ClassTransitional {
			report.Conforming = false
		}
		if state.Path != IgnorePath && stableCurrentPresent(state, request.Universe.FixedRevision) && !state.OpaqueBoundary && entry.Class != ClassIgnored && !entry.Operational && entry.Class != ClassTransitional && stateDigest(state, request.Universe.FixedRevision) == "" {
			report.Conforming = false
			report.Findings = append(report.Findings, Finding{Code: "coverage.bytes-unavailable", Path: state.Path,
				Message: "Current nonignored file bytes were not observed, so the coverage basis cannot bind them.", Severity: "error"})
		}
		report.Entries = append(report.Entries, entry)
	}
	for _, modelPath := range request.Options.CanonicalModelPaths {
		found := false
		for _, state := range pathStates {
			if state.Path == modelPath && stableCurrentPresent(state, request.Universe.FixedRevision) {
				found = true
				break
			}
		}
		if !found {
			report.Conforming = false
			report.Findings = append(report.Findings, Finding{Code: "coverage.canonical-model-missing", Path: modelPath,
				Message: "Registered canonical model path is absent from the current snapshot.", Severity: "error"})
		}
	}
	if err := validateRequiredArtifacts(request.Model, pathStates, ignore, request.Universe.FixedRevision, &report); err != nil {
		return Report{}, err
	}
	for _, rule := range ignore.Entries {
		matched := false
		for _, state := range pathStates {
			if selectorMatches(rule.Path, state.Path) {
				matched = true
				break
			}
		}
		if !matched && !strings.HasSuffix(rule.Path, "/") {
			report.Conforming = false
			report.Findings = append(report.Findings, Finding{Code: "coverage.ignore-unused", Path: rule.Path,
				Message: "Exact ignore entry matches no observed repository path.", Severity: "error"})
		}
	}
	if !report.Accounted || len(report.Findings) != 0 || request.Model.Status != "succeeded" {
		report.Conforming = false
	}
	if len(report.LegacyDiagnostics) > 0 {
		// Legacy scopes are shown to aid migration; they never define a complete
		// universe or alter the classifications produced from this census.
	}
	report.UniverseDigest = universeDigest(request.Universe, request.Options)
	report.IgnoreMembersDigest = ignoredMembersDigest(report.Entries)
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Path != report.Findings[j].Path {
			return report.Findings[i].Path < report.Findings[j].Path
		}
		if report.Findings[i].Code != report.Findings[j].Code {
			return report.Findings[i].Code < report.Findings[j].Code
		}
		return report.Findings[i].Message < report.Findings[j].Message
	})
	report.Digest = stableDigest(report, optionsDigest(request.Options))
	return report, nil
}

func Overlay(base *Universe, delta []Delta, options Options) (*Universe, error) {
	if base == nil {
		return nil, fmt.Errorf("base repository universe is required")
	}
	if err := validateOptions(options); err != nil {
		return nil, err
	}
	states := make(map[string]PathState, len(base.Paths)+len(delta))
	for _, state := range base.Paths {
		copy := state
		if base.FixedRevision {
			copy.Worktree = copy.Head
		}
		states[state.Path] = copy
	}
	seen := map[string]string{}
	ignoreBytes := append([]byte(nil), base.IgnoreBytes...)
	for _, change := range delta {
		if err := validatePath(change.Path); err != nil {
			return nil, err
		}
		if strings.HasSuffix(change.Path, "/") {
			return nil, fmt.Errorf("candidate file path %q cannot end with a slash", change.Path)
		}
		key := strings.ToLower(change.Path)
		if old, ok := seen[key]; ok {
			return nil, fmt.Errorf("candidate repeats or case-aliases paths %q and %q", old, change.Path)
		}
		seen[key] = change.Path
		state, exists := states[change.Path]
		if !exists {
			state = PathState{Path: change.Path}
		}
		if change.Delete {
			if change.Mode != "" || change.Digest != "" || len(change.Content) != 0 {
				return nil, fmt.Errorf("deleted candidate path %q cannot carry content, mode, or digest", change.Path)
			}
			state.Worktree = FileState{}
			if change.Path == IgnorePath {
				ignoreBytes = nil
			}
		} else {
			if change.Mode != "100644" && change.Mode != "100755" {
				return nil, fmt.Errorf("candidate path %q has unsupported mode %q", change.Path, change.Mode)
			}
			digest := change.Digest
			if change.Path == IgnorePath {
				if digest != "" && digest != sha256Hex(change.Content) {
					return nil, fmt.Errorf("candidate ignore digest does not match its bytes")
				}
				digest = sha256Hex(change.Content)
				ignoreBytes = append([]byte(nil), change.Content...)
			} else if digest == "" {
				return nil, fmt.Errorf("candidate path %q requires a content digest", change.Path)
			}
			state.Worktree = FileState{Present: true, Mode: change.Mode, Digest: digest}
		}
		states[change.Path] = state
	}
	if _, err := DecodeIgnore(ignoreBytes); err != nil {
		return nil, err
	}
	result := &Universe{Revision: base.Revision, IdentityDigest: base.IdentityDigest, Paths: make([]PathState, 0, len(states)),
		IgnoreBytes: ignoreBytes, IgnoreBytesDigest: sha256Hex(ignoreBytes)}
	for _, state := range states {
		result.Paths = append(result.Paths, state)
	}
	sort.Slice(result.Paths, func(i, j int) bool { return result.Paths[i].Path < result.Paths[j].Path })
	if err := validateUniversePaths(result.Paths); err != nil {
		return nil, err
	}
	result.Digest = universeDigest(result, options)
	return result, nil
}

func ValidateCandidate(base *Universe, delta []Delta, model projectmodel.Report, options Options, legacyRoots []string, legacyExclusions []TransitionalExclusion) (Report, error) {
	overlaid, err := Overlay(base, delta, options)
	if err != nil {
		return Report{}, err
	}
	return Classify(Request{Universe: overlaid, Model: model, Options: options, LegacyRoots: legacyRoots, LegacyExclusions: legacyExclusions})
}

func classifyPath(state PathState, request Request, ignore IgnoreFile) (Entry, bool) {
	entry := Entry{Path: state.Path, Artifacts: []string{}, Statements: []string{}, Head: state.Head, Index: state.Index,
		Worktree: state.Worktree, OpaqueBoundary: state.OpaqueBoundary}
	for _, modelPath := range request.Options.CanonicalModelPaths {
		if state.Path == modelPath {
			entry.Class, entry.Owner = ClassCanonicalModel, "project-model"
			return entry, true
		}
	}
	for _, tool := range request.Options.ToolPaths {
		if selectorMatches(tool.Selector, state.Path) {
			entry.Class, entry.Owner, entry.Operational = ClassToolOwned, tool.Owner, tool.Operational
			return entry, true
		}
	}
	if state.Path == IgnorePath {
		entry.Class, entry.Owner = ClassToolOwned, "projectcoverage"
		return entry, true
	}
	for _, rule := range ignore.Entries {
		if selectorMatches(rule.Path, state.Path) {
			entry.Class, entry.Owner, entry.Reason = ClassIgnored, "repository-ignore", rule.Reason
			clearStateDigests(&entry)
			return entry, true
		}
	}
	for _, exclusion := range request.Options.Transitional {
		selector, _ := normalizeSelector(exclusion.Path)
		if selectorMatches(selector, state.Path) {
			entry.Class, entry.Owner, entry.Reason = ClassTransitional, "project-migration", exclusion.Reason
			clearStateDigests(&entry)
			return entry, true
		}
	}
	entry.Artifacts, entry.Statements, entry.Owner = matchingArtifacts(request.Model, state.Path)
	if len(entry.Artifacts) > 0 && len(entry.Statements) > 0 {
		entry.Class = ClassRealization
		return entry, true
	}
	return entry, false
}

func clearStateDigests(entry *Entry) {
	entry.Head.Digest = ""
	entry.Index.Digest = ""
	entry.Worktree.Digest = ""
}

func matchingArtifacts(model projectmodel.Report, name string) ([]string, []string, string) {
	var artifacts, statements []string
	owner := ""
	for _, artifact := range model.Artifacts {
		for _, raw := range artifact.Paths {
			selector, ok := normalizeSelector(raw)
			if ok && selectorMatches(selector, name) {
				artifacts = append(artifacts, artifact.ID)
				statements = append(statements, artifact.Realizes...)
				if owner == "" {
					owner = artifact.Owner
				}
			}
		}
	}
	return sortedUnique(artifacts), sortedUnique(statements), owner
}

func validateRequiredArtifacts(model projectmodel.Report, states []PathState, ignore IgnoreFile, fixed bool, report *Report) error {
	for _, artifact := range model.Artifacts {
		if !artifact.Required {
			continue
		}
		if len(artifact.Paths) == 0 {
			report.Conforming = false
			report.Findings = append(report.Findings, Finding{Code: "coverage.required-artifact-unmapped", Path: artifact.ID,
				Message: "Required Artifact has no realization path.", Severity: "error"})
			continue
		}
		for _, raw := range artifact.Paths {
			selector, ok := normalizeSelector(raw)
			if !ok {
				report.Conforming = false
				report.Findings = append(report.Findings, Finding{Code: "coverage.artifact-path-invalid", Path: artifact.ID,
					Message: "Required Artifact has an invalid path selector.", Severity: "error"})
				continue
			}
			for _, rule := range ignore.Entries {
				if selectorsOverlap(selector, rule.Path) {
					report.Conforming = false
					report.Findings = append(report.Findings, Finding{Code: "coverage.required-artifact-ignored", Path: artifact.ID,
						Message: "Required Artifact path overlaps an ignore entry: " + rule.Path, Severity: "error"})
				}
			}
			found := false
			for _, state := range states {
				if !selectorMatches(selector, state.Path) || !stableCurrentPresent(state, fixed) {
					continue
				}
				found = true
				break
			}
			if !found {
				report.Conforming = false
				report.Findings = append(report.Findings, Finding{Code: "coverage.required-artifact-missing", Path: artifact.ID,
					Message: "Required Artifact path is absent from the current candidate: " + selector, Severity: "error"})
			}
		}
	}
	return nil
}

func validateRequiredArtifactOwnership(model projectmodel.Report, options Options) error {
	for _, artifact := range model.Artifacts {
		if !artifact.Required {
			continue
		}
		for _, raw := range artifact.Paths {
			selector, ok := normalizeSelector(raw)
			if !ok {
				continue
			}
			for _, canonical := range options.CanonicalModelPaths {
				if selectorsOverlap(selector, canonical) {
					return fmt.Errorf("required Artifact %q path %q overlaps canonical model path %q", artifact.ID, selector, canonical)
				}
			}
			for _, tool := range options.ToolPaths {
				if selectorsOverlap(selector, tool.Selector) {
					return fmt.Errorf("required Artifact %q path %q overlaps tool-owned path %q", artifact.ID, selector, tool.Selector)
				}
			}
		}
	}
	return nil
}

func stableCurrentPresent(state PathState, fixed bool) bool {
	if fixed {
		return state.Head.Present
	}
	return state.Worktree.Present
}

func stateDigest(state PathState, fixed bool) string {
	if fixed {
		return state.Head.Digest
	}
	return state.Worktree.Digest
}

func validateIgnoreAgainstOptions(ignore IgnoreFile, options Options) error {
	for _, rule := range ignore.Entries {
		for _, modelPath := range options.CanonicalModelPaths {
			if selectorsOverlap(rule.Path, modelPath) {
				return fmt.Errorf("ignore entry %q overlaps canonical model path %q", rule.Path, modelPath)
			}
		}
		for _, tool := range options.ToolPaths {
			if selectorsOverlap(rule.Path, tool.Selector) {
				return fmt.Errorf("ignore entry %q overlaps tool-owned path %q", rule.Path, tool.Selector)
			}
		}
	}
	return nil
}

func validateIgnoreAgainstRequiredArtifacts(ignore IgnoreFile, model projectmodel.Report) error {
	for _, artifact := range model.Artifacts {
		if !artifact.Required {
			continue
		}
		for _, raw := range artifact.Paths {
			selector, ok := normalizeSelector(raw)
			if !ok {
				continue
			}
			for _, rule := range ignore.Entries {
				if selectorsOverlap(selector, rule.Path) {
					return fmt.Errorf("ignore entry %q overlaps required Artifact %q path %q", rule.Path, artifact.ID, selector)
				}
			}
		}
	}
	return nil
}

func validateOptions(options Options) error {
	var seen []string
	for _, value := range options.CanonicalModelPaths {
		if err := validatePath(value); err != nil || strings.HasSuffix(value, "/") {
			return fmt.Errorf("canonical model path %q must be an exact normalized file path", value)
		}
		if err := addUniquePath(&seen, value); err != nil {
			return err
		}
	}
	for _, tool := range options.ToolPaths {
		selector, ok := normalizeSelector(tool.Selector)
		if !ok || selector != tool.Selector || strings.TrimSpace(tool.Owner) == "" || tool.Owner != strings.TrimSpace(tool.Owner) {
			return fmt.Errorf("tool path %q requires a normalized selector and nonempty owner", tool.Selector)
		}
		if selectorWithin(selector, ".git/") || selector == IgnorePath || selectorWithin(selector, IgnorePath) {
			return fmt.Errorf("tool path %q overlaps reserved repository metadata", selector)
		}
		if err := addUniquePath(&seen, selector); err != nil {
			return err
		}
	}
	for _, exclusion := range options.Transitional {
		selector, ok := normalizeSelector(exclusion.Path)
		if !ok || selector != exclusion.Path || selectorWithin(selector, ".git/") || selectorWithin(selector, ".markitect/") || strings.TrimSpace(exclusion.Reason) == "" || exclusion.Reason != strings.TrimSpace(exclusion.Reason) {
			return fmt.Errorf("transitional exclusion %q requires a normalized selector and nonempty reason", exclusion.Path)
		}
		if err := addUniquePath(&seen, selector); err != nil {
			return err
		}
	}
	return nil
}

func addUniquePath(seen *[]string, selector string) error {
	key := strings.ToLower(selector)
	for _, previous := range *seen {
		if strings.ToLower(previous) == key {
			return fmt.Errorf("path selector %q duplicates or aliases %q", selector, previous)
		}
	}
	for _, previous := range *seen {
		if selectorsOverlap(strings.ToLower(previous), strings.ToLower(selector)) {
			return fmt.Errorf("path selectors %q and %q overlap", previous, selector)
		}
	}
	*seen = append(*seen, selector)
	return nil
}

func isIgnored(name string, ignore IgnoreFile) bool {
	for _, entry := range ignore.Entries {
		if selectorMatches(entry.Path, name) {
			return true
		}
	}
	return false
}

func isOperational(name string, options Options) bool {
	for _, tool := range options.ToolPaths {
		if tool.Operational && selectorMatches(tool.Selector, name) {
			return true
		}
	}
	return false
}

func isTransitional(name string, options Options) bool {
	for _, exclusion := range options.Transitional {
		if selectorMatches(exclusion.Path, name) {
			return true
		}
	}
	return false
}

func universeDigest(universe *Universe, options Options) string {
	ignore, _ := DecodeIgnore(universe.IgnoreBytes)
	type stablePath struct {
		Path                  string
		Head, Index, Worktree FileState
		OpaqueBoundary        bool
	}
	var paths []stablePath
	for _, state := range universe.Paths {
		if isOperational(state.Path, options) {
			continue
		}
		if isIgnored(state.Path, ignore) || isTransitional(state.Path, options) {
			state.Head.Digest, state.Index.Digest, state.Worktree.Digest = "", "", ""
		}
		paths = append(paths, stablePath{state.Path, state.Head, state.Index, state.Worktree, state.OpaqueBoundary})
	}
	return digest(struct {
		Revision, Identity, IgnoreBytes, Registry string
		Fixed                                     bool
		Paths                                     []stablePath
	}{universe.Revision, universe.IdentityDigest, universe.IgnoreBytesDigest, optionsDigest(options), universe.FixedRevision, paths})
}

func ignoredMembersDigest(entries []Entry) string {
	type member struct{ Path, Reason string }
	var members []member
	for _, entry := range entries {
		if entry.Class == ClassIgnored {
			members = append(members, member{entry.Path, entry.Reason})
		}
	}
	return digest(members)
}

func stableDigest(report Report, toolDigest string) string {
	stableEntries := make([]Entry, 0, len(report.Entries))
	for _, entry := range report.Entries {
		if !entry.Operational {
			// Ignored content digests are not stored in entries. The path and
			// decision remain bound through IgnoreMembersDigest.
			stableEntries = append(stableEntries, entry)
		}
	}
	return digest(struct {
		API, Universe, IgnoreBytes, IgnoreMembers, Model, ModelReport, ToolRegistry string
		Accounted, Conforming                                                       bool
		Entries                                                                     []Entry
		Findings                                                                    []Finding
		ModelErrors                                                                 []projectmodel.Finding
		LegacyDiagnostics                                                           []string
	}{report.APIVersion, report.UniverseDigest, report.IgnoreBytesDigest, report.IgnoreMembersDigest, report.ModelDigest, report.ModelReportDigest, toolDigest,
		report.Accounted, report.Conforming, stableEntries, report.Findings, report.ModelErrors, report.LegacyDiagnostics})
}

func optionsDigest(options Options) string {
	canonical := append([]string(nil), options.CanonicalModelPaths...)
	sort.Strings(canonical)
	tools := append([]ToolPath(nil), options.ToolPaths...)
	sort.Slice(tools, func(i, j int) bool { return tools[i].Selector < tools[j].Selector })
	transitional := append([]TransitionalExclusion(nil), options.Transitional...)
	sort.Slice(transitional, func(i, j int) bool { return transitional[i].Path < transitional[j].Path })
	return digest(struct {
		Canonical    []string
		Tools        []ToolPath
		Transitional []TransitionalExclusion
	}{canonical, tools, transitional})
}

func legacyDiagnostics(roots []string, exclusions []TransitionalExclusion) []string {
	var result []string
	if len(roots) > 0 {
		copy := append([]string(nil), roots...)
		sort.Strings(copy)
		result = append(result, "legacy inventoryRoots remain diagnostic only and do not define a complete repository census: "+strings.Join(copy, ", "))
	}
	if len(exclusions) > 0 {
		copy := append([]TransitionalExclusion(nil), exclusions...)
		sort.Slice(copy, func(i, j int) bool { return copy[i].Path < copy[j].Path })
		for _, exclusion := range copy {
			result = append(result, "legacy exclusion "+exclusion.Path+": "+exclusion.Reason)
		}
	}
	return result
}

func sortedUnique(values []string) []string {
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func digest(value any) string {
	data, _ := json.Marshal(value)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
