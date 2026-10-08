package projectwork

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	hostwrite "github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

const maxMutationBytes = 4 << 20
const HumanActor = "user"

// DecodeMutation decodes the closed JSON proposal envelope. Duplicate keys,
// unknown fields, malformed trailing values, and oversized input are refused.
func DecodeMutation(data []byte) (Mutation, error) {
	if len(data) == 0 || len(data) > maxMutationBytes || !utf8.Valid(data) {
		return Mutation{}, fmt.Errorf("mutation JSON must be valid UTF-8 between 1 and %d bytes", maxMutationBytes)
	}
	if err := validateJSONDuplicates(data); err != nil {
		return Mutation{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var mutation Mutation
	if err := decoder.Decode(&mutation); err != nil {
		return Mutation{}, fmt.Errorf("invalid mutation JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Mutation{}, fmt.Errorf("mutation JSON must contain exactly one value")
		}
		return Mutation{}, fmt.Errorf("invalid trailing mutation JSON: %w", err)
	}
	return mutation, nil
}

// EncodeMutation writes the canonical JSON representation used by plan/apply.
func EncodeMutation(mutation Mutation) ([]byte, error) { return json.Marshal(mutation) }

// EncodeEditPlan writes the canonical JSON representation of a reviewed plan.
func EncodeEditPlan(plan EditPlan) ([]byte, error) { return json.Marshal(plan) }

func validateJSONDuplicates(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	rootFields := map[string]bool{"apiVersion": true, "baseDigest": true, "actor": true, "goal": true, "files": true}
	fileFields := map[string]bool{"path": true, "content": true, "delete": true}
	var value func(json.Token, map[string]bool) error
	value = func(token json.Token, fields map[string]bool) error {
		switch delimiter, ok := token.(json.Delim); {
		case !ok:
			return nil
		case delimiter == '{':
			seen := map[string]bool{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("mutation JSON object key is not a string")
				}
				if seen[key] {
					return fmt.Errorf("mutation JSON contains duplicate key %q", key)
				}
				for prior := range seen {
					if strings.EqualFold(prior, key) {
						return fmt.Errorf("mutation JSON contains case-alias keys %q and %q", prior, key)
					}
				}
				if fields == nil || !fields[key] {
					return fmt.Errorf("mutation JSON contains unknown or incorrectly cased field %q", key)
				}
				seen[key] = true
				child, err := decoder.Token()
				if err != nil {
					return err
				}
				var childFields map[string]bool
				if key == "files" {
					childFields = fileFields
				}
				if err := value(child, childFields); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		case delimiter == '[':
			for decoder.More() {
				child, err := decoder.Token()
				if err != nil {
					return err
				}
				if err := value(child, fields); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		default:
			return fmt.Errorf("mutation JSON contains an unexpected delimiter")
		}
	}
	first, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("invalid mutation JSON: %w", err)
	}
	if err := value(first, rootFields); err != nil {
		return fmt.Errorf("invalid mutation JSON: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("mutation JSON must contain exactly one value")
		}
		return fmt.Errorf("invalid trailing mutation JSON: %w", err)
	}
	return nil
}

// PlanEdit validates a prospective full project revision against the active
// model and binds its report, impact, source bytes, schema and tool build.
func PlanEdit(project *Project, mutation Mutation) (EditPlan, error) {
	if project == nil || project.Snapshot == nil {
		return EditPlan{}, fmt.Errorf("active Project is required")
	}
	if mutation.APIVersion != APIVersion {
		return EditPlan{}, fmt.Errorf("mutation apiVersion must be %q", APIVersion)
	}
	if mutation.BaseDigest == "" || mutation.BaseDigest != project.Digest {
		return EditPlan{}, fmt.Errorf("mutation baseDigest does not match the active Project")
	}
	if strings.TrimSpace(mutation.Goal) == "" || mutation.Goal != strings.TrimSpace(mutation.Goal) {
		return EditPlan{}, fmt.Errorf("mutation goal must be nonempty and have no surrounding whitespace")
	}
	if len(mutation.Files) == 0 {
		return EditPlan{}, fmt.Errorf("mutation must contain at least one model or manifest change")
	}
	actor, user, err := resolveActor(project, mutation.Actor)
	if err != nil {
		return EditPlan{}, err
	}
	seen := map[string]string{}
	candidateSnapshot := cloneSnapshot(project.Snapshot)
	for _, change := range mutation.Files {
		if change.Path != ManifestPath && !strings.HasPrefix(change.Path, ModelRoot+"/") {
			return EditPlan{}, fmt.Errorf("mutation path %q is outside the selected project model and manifest", change.Path)
		}
		if err := validateRepoPath(change.Path); err != nil {
			return EditPlan{}, fmt.Errorf("mutation path %q: %w", change.Path, err)
		}
		if old, ok := seen[strings.ToLower(change.Path)]; ok {
			return EditPlan{}, fmt.Errorf("mutation repeats or case-aliases paths %q and %q", old, change.Path)
		}
		seen[strings.ToLower(change.Path)] = change.Path
		if change.Delete {
			if change.Path == ManifestPath || change.Content != "" {
				return EditPlan{}, fmt.Errorf("only model files may be deleted and delete content must be empty")
			}
			if !selectedModelFiles(project.Config)[change.Path] {
				return EditPlan{}, fmt.Errorf("cannot delete unselected model path %q", change.Path)
			}
			delete(candidateSnapshot.Files, change.Path)
			delete(candidateSnapshot.Modes, change.Path)
		} else {
			if !utf8.ValidString(change.Content) || len(change.Content) == 0 || len(change.Content) > source.DefaultMaxFileBytes {
				return EditPlan{}, fmt.Errorf("mutation content for %q must be nonempty UTF-8 and within the source file limit", change.Path)
			}
			candidateSnapshot.Files[change.Path] = []byte(change.Content)
			if _, selected := project.Snapshot.Files[change.Path]; !selected {
				candidateSnapshot.Modes[change.Path] = snapshot.RegularMode
			}
		}
	}
	_, changedManifest := seen[strings.ToLower(ManifestPath)]
	candidateConfig := project.Config
	if changedManifest {
		manifest := candidateSnapshot.Files[ManifestPath]
		decoded, decodeErr := DecodeConfig(manifest)
		if decodeErr != nil {
			return EditPlan{}, fmt.Errorf("prospective project manifest: %w", decodeErr)
		}
		if decoded.Name != project.Config.Name {
			return EditPlan{}, fmt.Errorf("project identity is fixed by the active Project")
		}
		scopeChanged := !reflect.DeepEqual(decoded.InventoryRoots, project.Config.InventoryRoots) || !reflect.DeepEqual(decoded.Exclusions, project.Config.Exclusions)
		if scopeChanged && !mayChangeInventoryScope(project, actor, user) {
			return EditPlan{}, fmt.Errorf("only the explicit user actor or active root Manager may change inventory roots or exclusions")
		}
		candidateConfig = decoded
	}
	if !reflect.DeepEqual(candidateConfig.InventoryRoots, project.Config.InventoryRoots) || !reflect.DeepEqual(candidateConfig.Exclusions, project.Config.Exclusions) {
		inventory, inventoryErr := loadConfiguredInventory(project.Root, project.Revision, project.Provisional, candidateConfig)
		if inventoryErr != nil {
			return EditPlan{}, fmt.Errorf("acquire prospective selected inventory: %w", inventoryErr)
		}
		for file := range candidateSnapshot.Files {
			if !strings.HasPrefix(file, ".markitect/") {
				delete(candidateSnapshot.Files, file)
				delete(candidateSnapshot.Modes, file)
			}
		}
		for file, data := range inventory.Files {
			candidateSnapshot.Files[file] = append([]byte(nil), data...)
			candidateSnapshot.Modes[file] = inventory.Modes[file]
		}
	}
	for _, file := range candidateConfig.ModelFiles {
		if _, present := candidateSnapshot.Files[file]; !present {
			return EditPlan{}, fmt.Errorf("prospective manifest selects missing model file %q", file)
		}
	}
	selected := map[string]bool{}
	for _, file := range candidateConfig.ModelFiles {
		selected[file] = true
	}
	for file := range candidateSnapshot.Files {
		if strings.HasPrefix(file, ModelRoot+"/") && !selected[file] {
			return EditPlan{}, fmt.Errorf("prospective snapshot retains unselected model file %q; remove it from the candidate or select it explicitly", file)
		}
	}
	for _, change := range mutation.Files {
		if change.Path == ManifestPath {
			continue
		}
		namespace, namespaceErr := namespaceForModelPath(change.Path)
		if namespaceErr != nil {
			return EditPlan{}, namespaceErr
		}
		if !user {
			if err := authorizeModelPath(project.Report, actor, namespace, change.Path); err != nil {
				return EditPlan{}, err
			}
		}
	}
	if !user {
		for _, change := range mutation.Files {
			if change.Path == ManifestPath {
				root := nearestManager(project.Report.Managers, "")
				if root == nil || actor.ID != root.ID {
					return EditPlan{}, fmt.Errorf("only the active root Manager or user may update model selection")
				}
			}
		}
	}
	prospective, err := FromSnapshot(project.Root, candidateSnapshot)
	if err != nil {
		return EditPlan{}, fmt.Errorf("prospective project model is invalid: %w", err)
	}
	if prospective.Report.Status == "failed" {
		return EditPlan{}, fmt.Errorf("prospective project report failed structural validation")
	}
	if err := checkManagerChanges(project.Report, prospective.Report, actor, user); err != nil {
		return EditPlan{}, err
	}
	if !user {
		if err := preserveRequiredArtifacts(project.Report, prospective.Report); err != nil {
			return EditPlan{}, err
		}
	}
	impact := projectmodel.Impact(project.Report, prospective.Report)
	plan := EditPlan{
		APIVersion:      APIVersion,
		BaseDigest:      project.Digest,
		Mutation:        mutation,
		CandidateDigest: prospective.Digest,
		Report:          prospective.Report,
		Impact:          impact,
	}
	plan.Digest, err = editPlanDigest(plan)
	if err != nil {
		return EditPlan{}, err
	}
	return plan, nil
}

// ApplyEdit recomputes and applies an exact mutation only while the active
// selected bytes still match the reviewed base digest.
func ApplyEdit(root string, plan EditPlan, expected string) (EditPlan, error) {
	if expected == "" || expected != plan.BaseDigest {
		return plan, fmt.Errorf("expected digest must equal the reviewed plan baseDigest")
	}
	digest, err := editPlanDigest(plan)
	if err != nil {
		return plan, err
	}
	if digest != plan.Digest {
		return plan, fmt.Errorf("edit plan digest does not match its contents")
	}
	current, err := Load(root, "")
	if err != nil {
		return plan, err
	}
	if current.Digest != expected {
		// A plan may be based on the exact current fixed HEAD snapshot. Resolve
		// that same immutable input when the working view differs, then rely on
		// the guarded capture to require working bytes to match before writing.
		head, headErr := source.GitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
		if headErr == nil {
			fixed, fixedErr := Load(root, strings.TrimSpace(string(head)))
			if fixedErr == nil && fixed.Digest == expected {
				current = fixed
			}
		}
		if current.Digest != expected {
			return plan, fmt.Errorf("project changed after planning; current digest is %s", current.Digest)
		}
	}
	recomputed, err := PlanEdit(current, plan.Mutation)
	if err != nil {
		return plan, fmt.Errorf("recompute reviewed edit plan: %w", err)
	}
	if recomputed.Digest != plan.Digest || recomputed.CandidateDigest != plan.CandidateDigest {
		return plan, fmt.Errorf("reviewed edit plan no longer matches the recomputed candidate")
	}
	candidateConfig := current.Config
	for _, change := range plan.Mutation.Files {
		if change.Path == ManifestPath {
			candidateConfig, err = DecodeConfig([]byte(change.Content))
			if err != nil {
				return plan, fmt.Errorf("decode reviewed candidate manifest: %w", err)
			}
		}
	}
	prospectiveInventory, err := loadConfiguredInventory(root, current.Revision, current.Provisional, candidateConfig)
	if err != nil {
		return plan, fmt.Errorf("recheck prospective inventory selection: %w", err)
	}
	paths := make([]string, 0, len(current.Snapshot.Files)+len(plan.Mutation.Files))
	for file := range current.Snapshot.Files {
		paths = append(paths, file)
	}
	for file := range prospectiveInventory.Files {
		paths = append(paths, file)
	}
	for _, change := range plan.Mutation.Files {
		paths = append(paths, change.Path)
	}
	paths = uniqueSorted(paths)
	rootAbs, err := absoluteRoot(root)
	if err != nil {
		return plan, err
	}
	capture, err := hostwrite.CaptureGuardedWrite(rootAbs, paths)
	if err != nil {
		return plan, fmt.Errorf("capture reviewed project inputs: %w", err)
	}
	for file, data := range current.Snapshot.Files {
		actual, found := capture.Files[file]
		if !found || !actual.Exists || !bytes.Equal(actual.Bytes, data) || !sameSnapshotMode(actual.Mode, current.Snapshot.Modes[file]) {
			return plan, fmt.Errorf("selected project input changed before guarded write: %s", file)
		}
	}
	for file, data := range prospectiveInventory.Files {
		actual, found := capture.Files[file]
		if !found || !actual.Exists || !bytes.Equal(actual.Bytes, data) || !sameSnapshotMode(actual.Mode, prospectiveInventory.Modes[file]) {
			return plan, fmt.Errorf("prospectively selected inventory changed before guarded write: %s", file)
		}
	}
	changes := make([]hostwrite.GuardedWriteChange, 0, len(plan.Mutation.Files))
	for _, change := range plan.Mutation.Files {
		actual, found := capture.Files[change.Path]
		if !found {
			return plan, fmt.Errorf("guarded write did not capture mutation path %q", change.Path)
		}
		_, wasSelected := current.Snapshot.Files[change.Path]
		if !wasSelected && actual.Exists {
			return plan, fmt.Errorf("new model path already exists outside the selected Project: %s", change.Path)
		}
		mode := actual.Mode
		if !actual.Exists {
			mode = 0644
		}
		if change.Delete {
			changes = append(changes, hostwrite.GuardedWriteChange{Path: change.Path, Delete: true})
		} else {
			changes = append(changes, hostwrite.GuardedWriteChange{Path: change.Path, Bytes: []byte(change.Content), Mode: mode})
		}
	}
	validateBasis := func() error {
		var latest *Project
		var loadErr error
		if current.Provisional {
			latest, loadErr = Load(root, "")
		} else {
			latest, loadErr = Load(root, current.Revision)
		}
		if loadErr != nil {
			return loadErr
		}
		if latest.Digest != expected {
			return fmt.Errorf("project selection or selected inventory changed under the writer lock")
		}
		if !current.Provisional && latest.Revision != capture.Head {
			return fmt.Errorf("fixed project revision %s is no longer repository HEAD %s", latest.Revision, capture.Head)
		}
		working, workingErr := Load(root, "")
		if workingErr != nil {
			return workingErr
		}
		if !sameSnapshotFiles(current.Snapshot, working.Snapshot) {
			return fmt.Errorf("working selected files or inventory membership differ from the reviewed project snapshot")
		}
		checked, checkErr := PlanEdit(latest, plan.Mutation)
		if checkErr != nil {
			return checkErr
		}
		if checked.Digest != plan.Digest || checked.CandidateDigest != plan.CandidateDigest {
			return fmt.Errorf("prospective inventory selection changed under the writer lock")
		}
		return nil
	}
	result, err := hostwrite.ApplyGuardedWriteChecked(capture.Root, capture, changes, validateBasis)
	if err != nil {
		return plan, fmt.Errorf("project edit partially completed at %s: %w", strings.Join(result.CompletedPaths, ", "), err)
	}
	return plan, nil
}

func resolveActor(project *Project, id string) (*projectmodel.Manager, bool, error) {
	if id == HumanActor {
		return nil, true, nil
	}
	for i := range project.Report.Managers {
		manager := &project.Report.Managers[i]
		if manager.ID == id {
			return manager, false, nil
		}
	}
	return nil, false, fmt.Errorf("actor %q is neither an active Manager ID nor the reserved user actor", id)
}

func mayChangeInventoryScope(project *Project, actor *projectmodel.Manager, user bool) bool {
	if user {
		return true
	}
	root := nearestManager(project.Report.Managers, "")
	return root != nil && actor != nil && actor.ID == root.ID && root.Namespace == ""
}

func authorizeModelPath(report projectmodel.Report, actor *projectmodel.Manager, namespace, file string) error {
	if path.Base(file) != "manager.yaml" {
		owner := nearestManager(report.Managers, namespace)
		if owner == nil || owner.ID != actor.ID {
			return fmt.Errorf("Manager %q does not own model namespace %q under the active model", actor.ID, namespace)
		}
		return nil
	}
	target := managerInNamespace(report.Managers, namespace)
	if target == nil {
		parentNamespace := namespace
		if i := strings.LastIndex(parentNamespace, "."); i >= 0 {
			parentNamespace = parentNamespace[:i]
		} else {
			parentNamespace = ""
		}
		parent := nearestManager(report.Managers, parentNamespace)
		if parent == nil || parent.ID != actor.ID {
			return fmt.Errorf("only the active parent Manager or user may establish a new Manager at namespace %q", namespace)
		}
		return nil
	}
	if actor.ID == target.ID {
		return nil
	}
	if !isManagerAncestor(report.Managers, actor.ID, target.ID) {
		return fmt.Errorf("Manager %q is not the active Manager or an authorized ancestor of %q", actor.ID, target.ID)
	}
	return nil
}

func checkManagerChanges(base, candidate projectmodel.Report, actor *projectmodel.Manager, user bool) error {
	if user {
		return nil
	}
	oldByID := map[string]projectmodel.Manager{}
	newByID := map[string]projectmodel.Manager{}
	for _, manager := range base.Managers {
		oldByID[manager.ID] = manager
	}
	for _, manager := range candidate.Managers {
		newByID[manager.ID] = manager
	}
	oldIDs := make([]string, 0, len(oldByID))
	for id := range oldByID {
		oldIDs = append(oldIDs, id)
	}
	sort.Strings(oldIDs)
	for _, id := range oldIDs {
		old := oldByID[id]
		current, exists := newByID[id]
		if !exists {
			if !isManagerAncestor(base.Managers, actor.ID, id) {
				return fmt.Errorf("Manager %q cannot remove Manager %q", actor.ID, id)
			}
			continue
		}
		if reflect.DeepEqual(old, current) {
			continue
		}
		authorityChanged := old.Parent != current.Parent || old.Namespace != current.Namespace || !reflect.DeepEqual(old.Owns, current.Owns)
		if authorityChanged && !isManagerAncestor(base.Managers, actor.ID, id) {
			return fmt.Errorf("Manager %q cannot change the active mandate of Manager %q", actor.ID, id)
		}
	}
	newIDs := make([]string, 0, len(newByID))
	for id := range newByID {
		newIDs = append(newIDs, id)
	}
	sort.Strings(newIDs)
	for _, id := range newIDs {
		if _, exists := oldByID[id]; !exists && actor.ID != nearestManagerID(base.Managers, newByID[id].Namespace) {
			return fmt.Errorf("Manager %q cannot add Manager %q outside its active delegated namespace", actor.ID, id)
		}
	}
	return nil
}

func preserveRequiredArtifacts(base, candidate projectmodel.Report) error {
	candidateByID := map[string]projectmodel.Artifact{}
	for _, artifact := range candidate.Artifacts {
		candidateByID[artifact.ID] = artifact
	}
	for _, artifact := range base.Artifacts {
		if !artifact.Required {
			continue
		}
		next, exists := candidateByID[artifact.ID]
		if !exists || !next.Required {
			return fmt.Errorf("required Artifact %q cannot disappear or become optional under Manager authority; use the explicit user actor to review that scope removal", artifact.ID)
		}
	}
	return nil
}

func nearestManager(managers []projectmodel.Manager, namespace string) *projectmodel.Manager {
	var best *projectmodel.Manager
	for i := range managers {
		manager := &managers[i]
		if namespace != manager.Namespace && !(manager.Namespace == "" || strings.HasPrefix(namespace, manager.Namespace+".")) {
			continue
		}
		if best == nil || len(manager.Namespace) > len(best.Namespace) {
			best = manager
		}
	}
	return best
}

func managerInNamespace(managers []projectmodel.Manager, namespace string) *projectmodel.Manager {
	for i := range managers {
		if managers[i].Namespace == namespace {
			return &managers[i]
		}
	}
	return nil
}

func nearestManagerID(managers []projectmodel.Manager, namespace string) string {
	if manager := nearestManager(managers, namespace); manager != nil {
		return manager.ID
	}
	return ""
}

func isManagerAncestor(managers []projectmodel.Manager, possibleAncestor, child string) bool {
	byID := map[string]projectmodel.Manager{}
	for _, manager := range managers {
		byID[manager.ID] = manager
	}
	current := byID[child]
	for current.Parent != "" {
		if current.Parent == possibleAncestor {
			return true
		}
		next, exists := byID[current.Parent]
		if !exists || next.ID == current.ID {
			return false
		}
		current = next
	}
	return false
}

func cloneSnapshot(input *snapshot.Snapshot) *snapshot.Snapshot {
	result := &snapshot.Snapshot{ID: input.ID, Provisional: input.Provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
	for path, data := range input.Files {
		result.Files[path] = append([]byte(nil), data...)
		result.Modes[path] = input.Modes[path]
	}
	return result
}

func editPlanDigest(plan EditPlan) (string, error) {
	plan.Digest = ""
	data, err := json.Marshal(plan)
	if err != nil {
		return "", fmt.Errorf("encode edit plan: %w", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func selectedModelFiles(config Config) map[string]bool {
	result := make(map[string]bool, len(config.ModelFiles))
	for _, file := range config.ModelFiles {
		result[file] = true
	}
	return result
}

func sameSnapshotMode(actual fs.FileMode, expected string) bool {
	switch expected {
	case snapshot.RegularMode:
		return actual.Type() == 0 && actual.Perm()&0111 == 0
	case snapshot.ExecutableMode:
		return actual.Type() == 0 && actual.Perm()&0111 != 0
	default:
		return false
	}
}

func sameBytes(left, right []byte) bool { return bytes.Equal(left, right) }

func sameSnapshotFiles(left, right *snapshot.Snapshot) bool {
	if left == nil || right == nil || len(left.Files) != len(right.Files) || len(left.Modes) != len(right.Modes) {
		return false
	}
	for name, data := range left.Files {
		other, ok := right.Files[name]
		if !ok || !bytes.Equal(data, other) || left.Modes[name] != right.Modes[name] {
			return false
		}
	}
	return true
}

func sortedManagerIDs(managers []projectmodel.Manager) []string {
	result := make([]string, len(managers))
	for i, manager := range managers {
		result[i] = manager.ID
	}
	sort.Strings(result)
	return result
}
