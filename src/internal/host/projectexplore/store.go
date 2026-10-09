package projectexplore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	hostwrite "github.com/Glacius-Labs/Markitect/src/internal/host"
)

const recordsDirectory = ".markitect/state/explorations"

func RecordPath(id string) (string, error) {
	if !safeID.MatchString(id) {
		return "", fmt.Errorf("invalid exploration ID %q", id)
	}
	return recordsDirectory + "/" + id + ".json", nil
}

// CreatePreview prepares a new one-scope exploration record without writing.
// Its initial source binding is retained as immutable provenance.
func CreatePreview(root string, record Record, binding Binding) (WritePlan, error) {
	var empty WritePlan
	binding, err := canonicalBinding(binding)
	if err != nil {
		return empty, err
	}
	if err := validateRecordIdentity(record); err != nil {
		return empty, err
	}
	if err := validateScopeBinding(record, binding); err != nil {
		return empty, err
	}
	if record.Status == "" {
		record.Status = StatusActive
	}
	if record.CreatedAgainst == "" {
		record.CreatedAgainst, err = BindingDigest(binding)
		if err != nil {
			return empty, err
		}
	}
	if record.CreatedAgainst != mustBindingDigest(binding) {
		return empty, errors.New("new exploration creation digest does not match its initial binding")
	}
	record.Digest = ""
	if record.Scopes == nil || record.Decisions == nil || record.Drafts == nil || record.Acknowledgements == nil || record.Completions == nil {
		return empty, errors.New("exploration record collections must be explicit arrays")
	}
	if len(record.Completions) != 0 || record.Status != StatusActive {
		return empty, errors.New("new exploration must be active and have no completion receipts")
	}
	encoded, err := EncodeRecord(record)
	if err != nil {
		return empty, err
	}
	decoded, err := DecodeRecord(encoded)
	if err != nil {
		return empty, err
	}
	path, _ := RecordPath(record.ID)
	capture, err := captureBinding(root, binding, path, true)
	if err != nil {
		return empty, err
	}
	target := capture.Files[path]
	if target.Exists {
		return empty, fmt.Errorf("exploration %q already exists", record.ID)
	}
	plan := WritePlan{
		APIVersion: APIVersion, RepositoryRoot: capture.Identity.Root, Branch: capture.Branch, Head: capture.Head,
		Binding: binding, BindingDigest: mustBindingDigest(binding), VerifyBasis: true, Path: path, Target: TargetBasis{Path: path}, Next: decoded,
	}
	plan.Digest, err = writePlanDigest(plan)
	if err != nil {
		return empty, err
	}
	return plan, nil
}

// UpdatePreview prepares a CAS update. Decision and scope identities cannot be
// silently deleted, and the original creation binding is immutable.
func UpdatePreview(root string, next Record, expectedStateDigest string, binding Binding) (WritePlan, error) {
	return updatePreview(root, next, expectedStateDigest, binding, true, false)
}

func updatePreview(root string, next Record, expectedStateDigest string, binding Binding, verifyBasis, allowCompletion bool) (WritePlan, error) {
	var empty WritePlan
	if !validDigest(expectedStateDigest) {
		return empty, errors.New("exploration update requires the exact current state digest")
	}
	binding, err := canonicalBinding(binding)
	if err != nil {
		return empty, err
	}
	if err := validateScopeBinding(next, binding); err != nil {
		return empty, err
	}
	prior, err := Load(root, next.ID)
	if err != nil {
		return empty, err
	}
	if prior.Digest != expectedStateDigest {
		return empty, errors.New("exploration state changed after it was read; reload before updating")
	}
	if prior.Status != StatusActive {
		return empty, errors.New("completed exploration records are immutable")
	}
	if err := validateTransition(prior, next, allowCompletion); err != nil {
		return empty, err
	}
	if next.CreatedAgainst != prior.CreatedAgainst {
		return empty, errors.New("exploration creation binding is immutable")
	}
	encoded, err := EncodeRecord(next)
	if err != nil {
		return empty, err
	}
	next, err = DecodeRecord(encoded)
	if err != nil {
		return empty, err
	}
	path, _ := RecordPath(next.ID)
	capture, err := captureBinding(root, binding, path, verifyBasis)
	if err != nil {
		return empty, err
	}
	target := capture.Files[path]
	actualRecord, decodeErr := DecodeRecord(target.Bytes)
	if !target.Exists || decodeErr != nil || actualRecord.Digest != expectedStateDigest {
		return empty, errors.New("exploration target changed after it was read")
	}
	plan := WritePlan{
		APIVersion: APIVersion, RepositoryRoot: capture.Identity.Root, Branch: capture.Branch, Head: capture.Head,
		Binding: binding, BindingDigest: mustBindingDigest(binding), VerifyBasis: verifyBasis, Path: path,
		ExpectedStateDigest: expectedStateDigest, Target: TargetBasis{Path: path, Exists: true, Digest: fileDigest(target.Bytes)}, Next: next, allowCompletion: allowCompletion,
	}
	plan.Digest, err = writePlanDigest(plan)
	if err != nil {
		return empty, err
	}
	return plan, nil
}

// Write publishes a preview only when the plan digest, caller's fresh binding,
// state CAS, source bytes, branch, and HEAD still match.
func Write(root string, plan WritePlan, expectPlanDigest string, currentBinding Binding) (Record, error) {
	if expectPlanDigest == "" || plan.Digest == "" || expectPlanDigest != plan.Digest {
		return Record{}, errors.New("exploration write requires the exact preview digest")
	}
	digest, err := writePlanDigest(plan)
	if err != nil || digest != plan.Digest {
		return Record{}, errors.New("exploration write plan digest does not match its contents")
	}
	expectedPath, pathErr := RecordPath(plan.Next.ID)
	if plan.APIVersion != APIVersion || pathErr != nil || plan.Path != expectedPath || plan.Target.Path != plan.Path {
		return Record{}, errors.New("exploration write plan API version or record path is invalid")
	}
	currentBinding, err = canonicalBinding(currentBinding)
	if err != nil {
		return Record{}, err
	}
	currentDigest, err := BindingDigest(currentBinding)
	if err != nil || currentDigest != plan.BindingDigest {
		return Record{}, errors.New("exploration write binding changed after preview")
	}
	if err := validateScopeBinding(plan.Next, currentBinding); err != nil {
		return Record{}, err
	}
	capture, err := captureBinding(root, currentBinding, plan.Path, plan.VerifyBasis)
	if err != nil {
		return Record{}, err
	}
	if capture.Identity.Root != plan.RepositoryRoot || capture.Branch != plan.Branch || capture.Head != plan.Head {
		return Record{}, errors.New("repository identity, branch, or HEAD changed after exploration preview")
	}
	target, ok := capture.Files[plan.Path]
	if !ok || target.Exists != plan.Target.Exists || target.Exists && fileDigest(target.Bytes) != plan.Target.Digest {
		return Record{}, errors.New("exploration record changed after preview")
	}
	if plan.ExpectedStateDigest == "" && target.Exists {
		return Record{}, errors.New("exploration creation target now exists")
	}
	if plan.ExpectedStateDigest != "" {
		actualRecord, decodeErr := DecodeRecord(target.Bytes)
		if !target.Exists || decodeErr != nil || actualRecord.Digest != plan.ExpectedStateDigest {
			return Record{}, errors.New("exploration state digest changed after preview")
		}
	}
	if err := validateWriteTransition(plan, target.Bytes); err != nil {
		return Record{}, err
	}
	encoded, err := EncodeRecord(plan.Next)
	if err != nil {
		return Record{}, err
	}
	_, err = hostwrite.ApplyGuardedWriteChecked(capture.Root, capture, []hostwrite.GuardedWriteChange{{Path: plan.Path, Bytes: encoded, Mode: 0o600}}, func() error {
		if digest, bindErr := BindingDigest(currentBinding); bindErr != nil || digest != plan.BindingDigest {
			return errors.New("current exploration binding changed during guarded write")
		}
		return validateWriteTransition(plan, target.Bytes)
	})
	if err != nil {
		return Record{}, fmt.Errorf("write exploration state: %w", err)
	}
	return DecodeRecord(encoded)
}

// Complete loads, updates, and guarded-writes the current record after the
// caller has verified an immutable successful Apply receipt. The binding and
// acknowledgement retain the pre-run readiness basis; source bytes may differ
// because the successful Apply just changed them.
func Complete(root, id, scopeID string, currentBinding Binding, receipt ApplyReceipt) (Record, error) {
	current, err := Load(root, id)
	if err != nil {
		return Record{}, err
	}
	if current.Status == StatusCompleted {
		prior, ok := completionFor(current, scopeID)
		if ok && prior == receipt {
			return current, nil
		}
		return Record{}, errors.New("completed exploration cannot be changed by a different Apply receipt")
	}
	expectedDigest := current.Digest
	if err := completeScope(&current, scopeID, currentBinding, receipt); err != nil {
		return Record{}, err
	}
	plan, err := updatePreview(root, current, expectedDigest, currentBinding, false, true)
	if err != nil {
		return Record{}, err
	}
	return Write(root, plan, plan.Digest, currentBinding)
}

func validateWriteTransition(plan WritePlan, priorBytes []byte) error {
	if !plan.VerifyBasis && !plan.allowCompletion {
		return errors.New("generic exploration writes must verify their captured source basis")
	}
	if !plan.Target.Exists {
		if plan.ExpectedStateDigest != "" || plan.Next.Status != StatusActive || len(plan.Next.Completions) != 0 || plan.allowCompletion {
			return errors.New("exploration creation must be active and cannot contain Apply completion receipts")
		}
		if plan.Next.CreatedAgainst != plan.BindingDigest {
			return errors.New("exploration creation is not bound to its initial source digest")
		}
		return nil
	}
	prior, err := DecodeRecord(priorBytes)
	if err != nil {
		return fmt.Errorf("read prior exploration state for transition check: %w", err)
	}
	if prior.Digest != plan.ExpectedStateDigest {
		return errors.New("exploration state changed before the guarded transition")
	}
	if plan.Next.Status == StatusCompleted && !plan.allowCompletion {
		return errors.New("generic exploration writes cannot mark a scope completed")
	}
	return validateTransition(prior, plan.Next, plan.allowCompletion)
}

func Load(root, id string) (Record, error) {
	var empty Record
	path, err := RecordPath(id)
	if err != nil {
		return empty, err
	}
	capture, err := hostwrite.CaptureGuardedWrite(root, []string{path})
	if err != nil {
		return empty, err
	}
	file := capture.Files[path]
	if !file.Exists {
		return empty, fmt.Errorf("exploration %q does not exist", id)
	}
	return DecodeRecord(file.Bytes)
}

func List(root string) ([]Record, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	directory := filepath.Join(abs, filepath.FromSlash(recordsDirectory))
	if err := verifyDirectoryChain(abs, directory); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return []Record{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list exploration records: %w", err)
	}
	records := make([]Record, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		record, err := Load(root, id)
		if err != nil {
			return nil, fmt.Errorf("load exploration %s: %w", id, err)
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	return records, nil
}

func captureBinding(root string, binding Binding, targetPath string, verifyBasis bool) (*hostwrite.GuardedWriteCapture, error) {
	paths := make([]string, 0, len(binding.BasisFiles)+1)
	for _, basis := range binding.BasisFiles {
		if basis.Path == targetPath {
			return nil, errors.New("exploration record cannot also be a source basis file")
		}
		paths = append(paths, basis.Path)
	}
	paths = append(paths, targetPath)
	capture, err := hostwrite.CaptureGuardedWrite(root, paths)
	if err != nil {
		return nil, fmt.Errorf("capture exploration binding: %w", err)
	}
	headMatches := capture.Head == binding.Head || binding.Head == "" && strings.HasPrefix(capture.Head, "unborn:") && !binding.ModelAccepted
	if !samePath(capture.Identity.Root, binding.RepositoryRoot) || capture.Branch != binding.Branch || !headMatches {
		return nil, fmt.Errorf("current repository identity, branch, or HEAD differs from exploration binding (root %q/%q, branch %q/%q, HEAD %q/%q)", capture.Identity.Root, binding.RepositoryRoot, capture.Branch, binding.Branch, capture.Head, binding.Head)
	}
	for _, basis := range binding.BasisFiles {
		file, ok := capture.Files[basis.Path]
		if !ok || verifyBasis && (!file.Exists || fileDigest(file.Bytes) != basis.Digest) {
			return nil, fmt.Errorf("binding source file %q is missing or changed", basis.Path)
		}
	}
	return capture, nil
}

func verifyDirectoryChain(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("exploration directory escapes repository root")
	}
	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			return nil
		}
		if statErr != nil {
			return statErr
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("exploration directory component %s must be a real directory", current)
		}
	}
	return nil
}

func writePlanDigest(plan WritePlan) (string, error) {
	plan.Digest = ""
	return digest(plan)
}

func validateRecordIdentity(record Record) error {
	if record.APIVersion != APIVersion || !safeID.MatchString(record.ID) || strings.TrimSpace(record.Request) == "" {
		return errors.New("exploration record requires the current API version, safe ID, and request")
	}
	if len(record.Scopes) != 1 {
		return errors.New("each exploration record must contain exactly one named work-item scope")
	}
	return nil
}

func validateScopeBinding(record Record, binding Binding) error {
	if err := validateRecordIdentity(record); err != nil {
		return err
	}
	scope := record.Scopes[0]
	if scope.ID != binding.ScopeID || scope.Name != binding.ScopeName || scope.Goal != binding.Goal || scope.Operation != binding.Operation || !equalStrings(scope.ManagerIDs, binding.ManagerIDs) {
		return errors.New("exploration named scope differs from the supplied fixed binding")
	}
	return nil
}

func validateTransition(prior, next Record, allowCompletion bool) error {
	if next.APIVersion != APIVersion || next.ID != prior.ID || next.Request != prior.Request || next.Status != StatusActive && !(allowCompletion && next.Status == StatusCompleted) {
		return errors.New("exploration identity, request, or active status cannot be changed")
	}
	if err := validateRecord(next); err != nil {
		return err
	}
	oldScope := prior.Scopes[0]
	newScope := next.Scopes[0]
	if oldScope.ID != newScope.ID {
		return errors.New("exploration scope identity cannot be changed")
	}
	newDecisions := map[string]Decision{}
	for _, decision := range next.Decisions {
		newDecisions[decision.ID] = decision
	}
	for _, decision := range prior.Decisions {
		updated, ok := newDecisions[decision.ID]
		if !ok || updated.Question != decision.Question || updated.Blocking != decision.Blocking || !equalStrings(updated.ScopeIDs, decision.ScopeIDs) {
			return fmt.Errorf("decision %q is durable and cannot be deleted or redefined", decision.ID)
		}
		if decision.Status == "answered" && !sameDecision(updated, decision) {
			return fmt.Errorf("answered decision %q is immutable", decision.ID)
		}
		if decision.Status == "deferred" && !sameDecision(updated, decision) {
			return fmt.Errorf("deferred decision %q is immutable", decision.ID)
		}
		if decision.Status != "open" && updated.Status != decision.Status {
			return fmt.Errorf("decision %q cannot move backward", decision.ID)
		}
	}
	for _, ack := range prior.Acknowledgements {
		found := false
		for _, candidate := range next.Acknowledgements {
			if candidate == ack {
				found = true
				break
			}
		}
		if !found {
			return errors.New("structure acknowledgement history cannot be deleted")
		}
	}
	return nil
}

func sameDecision(left, right Decision) bool {
	return left.ID == right.ID && equalStrings(left.ScopeIDs, right.ScopeIDs) && left.Question == right.Question && left.Blocking == right.Blocking && left.Status == right.Status && left.Answer == right.Answer && left.Reason == right.Reason && left.Authority == right.Authority && left.Provenance == right.Provenance
}

func mustBindingDigest(binding Binding) string {
	digest, _ := BindingDigest(binding)
	return digest
}

func fileDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func samePath(left, right string) bool {
	left, _ = filepath.Abs(left)
	right, _ = filepath.Abs(right)
	left, right = filepath.Clean(left), filepath.Clean(right)
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	if leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo) {
		return true
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
