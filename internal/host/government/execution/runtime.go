package execution

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
)

const RuntimeVersion = "markitect.government-execution/v1alpha1"

const (
	maxRuntimeJSONBytes = 4 << 20
	maxFingerprintFile  = 256 << 20
	maxActorTimeout     = 600
	maxRunTimeout       = 1800
	// These collection bounds cap configuration and orchestration fan-out while
	// remaining comfortably above a useful initial Government cabinet/check set.
	maxRuntimeRessorts = 128
	maxRuntimeChecks   = 128
)

// RunnerSpec names one literal external process and the exact runtime bytes
// that agentexec must bind into its receipt. A slot is a configured role
// assignment, not an authenticated identity claim.
type RunnerSpec struct {
	SlotID          string                  `json:"slotId"`
	Command         string                  `json:"command"`
	Args            []string                `json:"args"`
	Model           string                  `json:"model"`
	ModelOptions    json.RawMessage         `json:"modelOptions"`
	ProviderVersion string                  `json:"providerVersion"`
	TimeoutSeconds  int                     `json:"timeoutSeconds"`
	MaxStdoutBytes  int                     `json:"maxStdoutBytes"`
	MaxStderrBytes  int                     `json:"maxStderrBytes"`
	RuntimeFiles    []agentexec.RuntimeFile `json:"runtimeFiles"`
}

// RessortRunner freezes a configured process slot for one canonical Ressort.
type RessortRunner struct {
	Ressort core.DefinitionIdentity `json:"ressort"`
	Runner  RunnerSpec              `json:"runner"`
}

// Runtime is the complete execution configuration consumed by the Host. The
// caller declares each actor's complete runtime dependencies; fingerprints do
// not discover transitive libraries or establish an operating-system sandbox.
// The active reference is a managed-ref selector; promotion code owns CAS.
type Runtime struct {
	APIVersion         string            `json:"apiVersion"`
	ActiveRef          string            `json:"activeRef"`
	ExpectedBase       string            `json:"expectedBase"`
	TimeoutSeconds     int               `json:"timeoutSeconds"`
	StateDirectory     string            `json:"stateDirectory"`
	TemporaryDirectory string            `json:"temporaryDirectory"`
	Executor           RunnerSpec        `json:"executor"`
	Verifier           RunnerSpec        `json:"verifier"`
	Ressorts           []RessortRunner   `json:"ressorts"`
	Checks             []authoring.Check `json:"checks"`
	Recursion          *RecursiveRuntime `json:"recursion,omitempty"`
	Amendment          *AmendmentRuntime `json:"amendment,omitempty"`
}

// AmendmentRuntime opts an explicitly issued amend-model order into bounded
// candidate repair. It does not grant authority; the active prior model does.
type AmendmentRuntime struct {
	MaxRepairs int `json:"maxRepairs"`
}

// DecodeRuntime decodes the closed JSON runtime contract. Exact and
// case-folded duplicate keys are rejected because encoding/json otherwise
// accepts ambiguous struct-field aliases.
func DecodeRuntime(data []byte) (Runtime, error) {
	if len(data) == 0 || len(data) > maxRuntimeJSONBytes || !utf8.Valid(data) {
		return Runtime{}, errors.New("runtime JSON is empty, invalid UTF-8, or exceeds 4 MiB")
	}
	if err := rejectRuntimeDuplicateKeys(data); err != nil {
		return Runtime{}, err
	}
	if err := rejectNullCheckTimeouts(data); err != nil {
		return Runtime{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var runtime Runtime
	if err := decoder.Decode(&runtime); err != nil {
		return Runtime{}, fmt.Errorf("invalid runtime JSON: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Runtime{}, errors.New("runtime JSON contains trailing data")
	}
	if err := ValidateRuntime(runtime); err != nil {
		return Runtime{}, err
	}
	return runtime, nil
}

func rejectRuntimeDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanRuntimeJSON(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("runtime JSON contains trailing data")
	}
	return nil
}

func scanRuntimeJSON(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("invalid runtime JSON: %w", err)
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]string{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return errors.New("invalid runtime JSON object key")
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("invalid runtime JSON object key")
			}
			folded := strings.Map(foldRuntimeRune, key)
			if previous, exists := seen[folded]; exists {
				return fmt.Errorf("duplicate runtime JSON key %q conflicts with %q", key, previous)
			}
			seen[folded] = key
			if err := scanRuntimeJSON(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("invalid runtime JSON object")
		}
	case '[':
		for decoder.More() {
			if err := scanRuntimeJSON(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("invalid runtime JSON array")
		}
	default:
		return errors.New("invalid runtime JSON delimiter")
	}
	return nil
}

func foldRuntimeRune(value rune) rune {
	folded := value
	for next := unicode.SimpleFold(folded); next != value; next = unicode.SimpleFold(next) {
		if next < folded {
			folded = next
		}
	}
	return folded
}

func runtimeKeyEqual(left, right string) bool {
	return strings.Map(foldRuntimeRune, left) == strings.Map(foldRuntimeRune, right)
}

func rejectNullCheckTimeouts(data []byte) error {
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("invalid runtime JSON: %w", err)
	}
	return rejectNullCheckTimeoutsAt(document, "")
}

func rejectNullCheckTimeoutsAt(value any, parent string) error {
	switch object := value.(type) {
	case map[string]any:
		names := make([]string, 0, len(object))
		for name := range object {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			child := object[name]
			if runtimeKeyEqual(name, "checks") {
				checks, ok := child.([]any)
				if ok {
					for index, rawCheck := range checks {
						check, ok := rawCheck.(map[string]any)
						if !ok {
							continue // the strict decode reports malformed check values
						}
						keys := make([]string, 0, len(check))
						for key := range check {
							keys = append(keys, key)
						}
						sort.Strings(keys)
						for _, key := range keys {
							timeout := check[key]
							if runtimeKeyEqual(key, "timeoutSeconds") && timeout == nil {
								return fmt.Errorf("%s[%d].timeoutSeconds must be omitted or an integer, not null", joinRuntimePath(parent, name), index)
							}
						}
					}
				}
			}
			if err := rejectNullCheckTimeoutsAt(child, joinRuntimePath(parent, name)); err != nil {
				return err
			}
		}
	case []any:
		for index, child := range object {
			if err := rejectNullCheckTimeoutsAt(child, fmt.Sprintf("%s[%d]", parent, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func joinRuntimePath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

// ValidateRuntime checks structural completeness and conservative process,
// path, and budget bounds. It does not confer authority or prove that an
// external runner is independent or trustworthy.
func ValidateRuntime(runtime Runtime) error {
	if runtime.APIVersion != RuntimeVersion {
		return fmt.Errorf("apiVersion must be %q", RuntimeVersion)
	}
	if !safeRef(runtime.ActiveRef) {
		return errors.New("activeRef must be a safe fully-qualified Git ref")
	}
	if !validFullObjectID(runtime.ExpectedBase) {
		return errors.New("expectedBase must be a full lowercase 40- or 64-character Git object ID")
	}
	if runtime.TimeoutSeconds < 1 || runtime.TimeoutSeconds > maxRunTimeout {
		return fmt.Errorf("timeoutSeconds must be between 1 and %d", maxRunTimeout)
	}
	state, err := absoluteDirectory(runtime.StateDirectory)
	if err != nil {
		return fmt.Errorf("stateDirectory: %w", err)
	}
	temporary, err := absoluteDirectory(runtime.TemporaryDirectory)
	if err != nil {
		return fmt.Errorf("temporaryDirectory: %w", err)
	}
	if directoriesOverlap(state, temporary) {
		return errors.New("stateDirectory and temporaryDirectory must be disjoint")
	}
	if err := validateRunner(runtime.Executor); err != nil {
		return fmt.Errorf("executor: %w", err)
	}
	if err := validateRunner(runtime.Verifier); err != nil {
		return fmt.Errorf("verifier: %w", err)
	}
	if runtime.Executor.SlotID == runtime.Verifier.SlotID {
		return errors.New("executor and verifier must use distinct slots")
	}
	if len(runtime.Ressorts) == 0 || len(runtime.Ressorts) > maxRuntimeRessorts {
		return fmt.Errorf("configured Ressort runners must contain 1 to %d entries", maxRuntimeRessorts)
	}
	slots := map[string]struct{}{runtime.Executor.SlotID: {}, runtime.Verifier.SlotID: {}}
	ressorts := make(map[string]struct{}, len(runtime.Ressorts))
	for index, configured := range runtime.Ressorts {
		if configured.Ressort.APIVersion == "" || configured.Ressort.Kind != "Ressort" || configured.Ressort.Name == "" {
			return fmt.Errorf("ressorts[%d] requires a complete Ressort identity", index)
		}
		key := configured.Ressort.Key()
		if _, exists := ressorts[key]; exists {
			return fmt.Errorf("ressorts[%d] duplicates a Ressort identity", index)
		}
		ressorts[key] = struct{}{}
		if err := validateRunner(configured.Runner); err != nil {
			return fmt.Errorf("ressorts[%d]: %w", index, err)
		}
		if _, exists := slots[configured.Runner.SlotID]; exists {
			return fmt.Errorf("runner slot %q is assigned more than once", configured.Runner.SlotID)
		}
		slots[configured.Runner.SlotID] = struct{}{}
	}
	if len(runtime.Checks) == 0 || len(runtime.Checks) > maxRuntimeChecks {
		return fmt.Errorf("deterministic checks must contain 1 to %d entries", maxRuntimeChecks)
	}
	checks := make(map[string]struct{}, len(runtime.Checks))
	for index, check := range runtime.Checks {
		if err := authoring.ValidateCheck(check); err != nil {
			return fmt.Errorf("checks[%d]: %w", index, err)
		}
		if _, exists := checks[check.Name]; exists {
			return fmt.Errorf("checks[%d] duplicates a check name", index)
		}
		checks[check.Name] = struct{}{}
	}
	if runtime.Recursion != nil {
		if err := validateRecursiveRuntime(*runtime.Recursion, slots); err != nil {
			return fmt.Errorf("recursion: %w", err)
		}
	}
	if runtime.Amendment != nil && (runtime.Amendment.MaxRepairs < 0 || runtime.Amendment.MaxRepairs > 2) {
		return errors.New("amendment.maxRepairs must be between 0 and 2")
	}
	return nil
}

func validateRunner(runner RunnerSpec) error {
	if !safeSlot(runner.SlotID) {
		return errors.New("slotId is required and must be a safe identifier")
	}
	if strings.TrimSpace(runner.Command) == "" || strings.ContainsRune(runner.Command, 0) {
		return errors.New("command is required")
	}
	if runner.TimeoutSeconds < 1 || runner.TimeoutSeconds > maxActorTimeout {
		return fmt.Errorf("timeoutSeconds must be between 1 and %d", maxActorTimeout)
	}
	if runner.MaxStdoutBytes < 1 || runner.MaxStdoutBytes > 16<<20 || runner.MaxStderrBytes < 1 || runner.MaxStderrBytes > 16<<20 {
		return errors.New("stdout and stderr byte limits must be between 1 and 16 MiB")
	}
	if runner.Model == "" || runner.ProviderVersion == "" {
		return errors.New("model and providerVersion must be explicit")
	}
	for _, arg := range runner.Args {
		if strings.ContainsRune(arg, 0) {
			return errors.New("args cannot contain NUL")
		}
	}
	if err := validateRunnerArguments(runner); err != nil {
		return err
	}
	return nil
}

func validateRunnerArguments(runner RunnerSpec) error {
	declared := make(map[string]string, len(runner.RuntimeFiles))
	for _, file := range runner.RuntimeFiles {
		if filepath.IsAbs(file.Path) {
			declared[filepath.Clean(file.Path)] = file.Digest
		}
	}
	for _, arg := range runner.Args {
		literalPath := arg
		if !filepath.IsAbs(literalPath) {
			if _, suffix, hasEquals := strings.Cut(arg, "="); hasEquals && filepath.IsAbs(suffix) {
				literalPath = suffix
			} else {
				continue
			}
		}
		if err := validateAbsoluteFileArgument(literalPath, declared); err != nil {
			return fmt.Errorf("argument %q: %w", arg, err)
		}
	}

	if !knownScriptInterpreter(runner.Command) {
		return nil
	}
	scriptIndex, err := interpreterScriptIndex(runner.Command, runner.Args)
	if err != nil {
		return err
	}
	for index, arg := range runner.Args {
		if isInlineInterpreterFlag(arg) {
			if index+1 >= len(runner.Args) || runner.Args[index+1] == "" {
				return errors.New("interpreter inline-code flag requires non-empty argv code")
			}
			if scriptIndex < 0 || index < scriptIndex {
				return nil // argv is covered by the runner configuration fingerprint.
			}
			break
		}
	}
	if scriptIndex < 0 || !filepath.IsAbs(runner.Args[scriptIndex]) {
		return errors.New("configured interpreter requires a declared absolute script file or explicit inline-code argv")
	}
	if err := requireDeclaredRuntimeFile(runner.Args[scriptIndex], declared); err != nil {
		return fmt.Errorf("interpreter script: %w", err)
	}
	return nil
}

func requireDeclaredRuntimeFile(path string, declared map[string]string) error {
	if !filepath.IsAbs(path) {
		return errors.New("file path must be absolute")
	}
	clean := filepath.Clean(path)
	info, err := os.Lstat(clean)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("absolute file argument must exist as a regular non-symlink file")
	}
	if _, exists := declared[clean]; !exists {
		return errors.New("absolute file argument is not declared in runtimeFiles")
	}
	digest, err := digestFile(clean)
	if err != nil || digest != declared[clean] {
		return errors.New("absolute file argument does not match its declared runtime digest")
	}
	return nil
}

func validateAbsoluteFileArgument(path string, declared map[string]string) error {
	if !filepath.IsAbs(path) {
		return errors.New("file path must be absolute")
	}
	info, err := os.Lstat(filepath.Clean(path))
	if err != nil {
		return nil // only existing regular file arguments need byte pins
	}
	if info.IsDir() {
		return nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, targetErr := os.Stat(path)
		if targetErr == nil && target.Mode().IsRegular() {
			return errors.New("absolute file argument cannot be a symlink")
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	return requireDeclaredRuntimeFile(path, declared)
}

func knownScriptInterpreter(command string) bool {
	name := strings.ToLower(filepath.Base(command))
	name = strings.TrimSuffix(name, ".exe")
	if strings.HasPrefix(name, "python") {
		version := strings.TrimPrefix(name, "python")
		if version == "" {
			return true
		}
		validVersion := true
		for _, r := range version {
			if !(r >= '0' && r <= '9' || r == '.') {
				validVersion = false
				break
			}
		}
		return validVersion
	}
	switch name {
	case "node", "pwsh", "powershell", "sh", "bash", "ruby", "perl":
		return true
	default:
		return false
	}
}

func interpreterScriptIndex(command string, args []string) (int, error) {
	name := strings.TrimSuffix(strings.ToLower(filepath.Base(command)), ".exe")
	if strings.HasPrefix(name, "python") {
		name = "python"
	}
	if name == "pwsh" || name == "powershell" {
		for index, arg := range args {
			if strings.EqualFold(arg, "-file") || strings.EqualFold(arg, "--file") {
				if index+1 >= len(args) {
					return -1, errors.New("interpreter script-file flag requires a path")
				}
				return index + 1, nil
			}
		}
		return -1, nil
	}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if strings.HasPrefix(arg, "-") {
			if interpreterOptionTakesValue(name, arg) {
				index++
			}
			continue
		}
		return index, nil
	}
	return -1, nil
}

func interpreterOptionTakesValue(interpreter, arg string) bool {
	option := strings.ToLower(arg)
	switch interpreter {
	case "python", "python2", "python3":
		return option == "-x" || option == "-w" || option == "-q"
	case "node":
		return option == "-r" || option == "--require" || option == "--import" || option == "--loader" || option == "--experimental-loader"
	case "ruby":
		return option == "-i" || option == "-r"
	case "perl":
		return option == "-i" || option == "-m"
	default:
		return false
	}
}

func isInlineInterpreterFlag(arg string) bool {
	switch strings.ToLower(arg) {
	case "-c", "-e", "-command":
		return true
	default:
		return false
	}
}

func safeSlot(slot string) bool {
	if len(slot) == 0 || len(slot) > 128 || strings.TrimSpace(slot) != slot {
		return false
	}
	for _, r := range slot {
		if !(r == '-' || r == '_' || r == '.' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func safeRef(value string) bool {
	const prefix = "refs/markitect/government/active/"
	if !strings.HasPrefix(value, prefix) || len(value) > 1024 || strings.HasSuffix(value, "/") || strings.HasSuffix(value, ".") || strings.Contains(value, "..") || strings.Contains(value, "//") || strings.Contains(value, "@{") || strings.ContainsAny(value, " ~^:?*[\\") {
		return false
	}
	component := strings.TrimPrefix(value, prefix)
	if len(component) == 0 || len(component) > 64 || !((component[0] >= 'a' && component[0] <= 'z') || (component[0] >= 'A' && component[0] <= 'Z') || (component[0] >= '0' && component[0] <= '9')) || strings.HasSuffix(component, ".lock") || strings.Contains(component, "/") {
		return false
	}
	for _, r := range component {
		if !(r == '-' || r == '_' || r == '.' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func validFullObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func absoluteDirectory(value string) (string, error) {
	if value == "" || !filepath.IsAbs(value) {
		return "", errors.New("must be an absolute path")
	}
	return filepath.Clean(value), nil
}

func directoriesOverlap(a, b string) bool {
	rel, err := filepath.Rel(a, b)
	if err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return true
	}
	rel, err = filepath.Rel(b, a)
	return err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

type runnerFingerprint struct {
	SlotID             string `json:"slotId"`
	ConfigDigest       string `json:"configDigest"`
	CommandDigest      string `json:"commandDigest"`
	ExecutableDigest   string `json:"executableDigest"`
	RuntimeFilesDigest string `json:"runtimeFilesDigest"`
}

type checkFingerprint struct {
	Name             string   `json:"name"`
	Run              []string `json:"run"`
	TimeoutSeconds   int      `json:"timeoutSeconds"`
	Executable       string   `json:"executable"`
	ExecutableDigest string   `json:"executableDigest"`
}

type runtimeFingerprint struct {
	APIVersion         string            `json:"apiVersion"`
	ActiveRef          string            `json:"activeRef"`
	ExpectedBase       string            `json:"expectedBase"`
	TimeoutSeconds     int               `json:"timeoutSeconds"`
	StateDirectory     string            `json:"stateDirectory"`
	TemporaryDirectory string            `json:"temporaryDirectory"`
	Executor           runnerFingerprint `json:"executor"`
	Verifier           runnerFingerprint `json:"verifier"`
	Ressorts           []struct {
		Ressort core.DefinitionIdentity `json:"ressort"`
		Runner  runnerFingerprint       `json:"runner"`
	} `json:"ressorts"`
	Checks    []checkFingerprint           `json:"checks"`
	Recursion *recursiveRuntimeFingerprint `json:"recursion,omitempty"`
	Amendment *AmendmentRuntime            `json:"amendment,omitempty"`
}

// FingerprintRuntime returns a deterministic identity for the full runtime,
// including current executable and declared-runtime-file bytes for each actor
// and every deterministic check executable. It performs no actor invocation.
func FingerprintRuntime(runtime Runtime) (string, error) {
	if err := ValidateRuntime(runtime); err != nil {
		return "", err
	}
	value := runtimeFingerprint{
		APIVersion: runtime.APIVersion, ActiveRef: filepath.Clean(runtime.ActiveRef),
		ExpectedBase: runtime.ExpectedBase, TimeoutSeconds: runtime.TimeoutSeconds, StateDirectory: filepath.Clean(runtime.StateDirectory),
		TemporaryDirectory: filepath.Clean(runtime.TemporaryDirectory),
	}
	var err error
	if value.Executor, err = fingerprintRunner(runtime.Executor); err != nil {
		return "", fmt.Errorf("executor fingerprint: %w", err)
	}
	if value.Verifier, err = fingerprintRunner(runtime.Verifier); err != nil {
		return "", fmt.Errorf("verifier fingerprint: %w", err)
	}
	ressorts := append([]RessortRunner(nil), runtime.Ressorts...)
	sort.Slice(ressorts, func(i, j int) bool { return ressorts[i].Ressort.Key() < ressorts[j].Ressort.Key() })
	for _, configured := range ressorts {
		fingerprint, err := fingerprintRunner(configured.Runner)
		if err != nil {
			return "", fmt.Errorf("Ressort %q fingerprint: %w", configured.Ressort.Name, err)
		}
		value.Ressorts = append(value.Ressorts, struct {
			Ressort core.DefinitionIdentity `json:"ressort"`
			Runner  runnerFingerprint       `json:"runner"`
		}{configured.Ressort, fingerprint})
	}
	checks := append([]authoring.Check(nil), runtime.Checks...)
	sort.Slice(checks, func(i, j int) bool { return checks[i].Name < checks[j].Name })
	for _, check := range checks {
		path, digest, err := fingerprintExecutable(check.Run[0])
		if err != nil {
			return "", fmt.Errorf("check %q executable fingerprint: %w", check.Name, err)
		}
		timeout := authoring.DefaultCheckTimeoutSeconds
		if check.TimeoutSeconds != nil {
			timeout = *check.TimeoutSeconds
		}
		value.Checks = append(value.Checks, checkFingerprint{check.Name, append([]string(nil), check.Run...), timeout, path, digest})
	}
	if runtime.Recursion != nil {
		fingerprint, err := fingerprintRecursiveRuntime(*runtime.Recursion)
		if err != nil {
			return "", err
		}
		value.Recursion = &fingerprint
	}
	if runtime.Amendment != nil {
		amendment := *runtime.Amendment
		value.Amendment = &amendment
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", errors.New("runtime fingerprint could not be encoded")
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func fingerprintRunner(spec RunnerSpec) (runnerFingerprint, error) {
	config := toAgentConfig(spec)
	digest, err := agentexec.Fingerprint(config)
	if err != nil {
		return runnerFingerprint{}, err
	}
	resolved, err := exec.LookPath(spec.Command)
	if err != nil {
		return runnerFingerprint{}, errors.New("configured runner command is unavailable")
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		return runnerFingerprint{}, errors.New("configured runner command could not be resolved")
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return runnerFingerprint{}, errors.New("configured runner path is invalid")
	}
	executableDigest, err := digestFile(resolved)
	if err != nil {
		return runnerFingerprint{}, err
	}
	filesJSON, _ := json.Marshal(spec.RuntimeFiles)
	fileSum := sha256.Sum256(filesJSON)
	commandJSON, _ := json.Marshal(struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}{spec.Command, spec.Args})
	commandSum := sha256.Sum256(commandJSON)
	return runnerFingerprint{
		SlotID: spec.SlotID, ConfigDigest: digest,
		CommandDigest:      "sha256:" + hex.EncodeToString(commandSum[:]),
		ExecutableDigest:   executableDigest,
		RuntimeFilesDigest: "sha256:" + hex.EncodeToString(fileSum[:]),
	}, nil
}

func fingerprintExecutable(command string) (string, string, error) {
	resolved, err := exec.LookPath(command)
	if err != nil {
		return "", "", errors.New("configured check command is unavailable")
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		return "", "", errors.New("configured check command could not be resolved")
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", "", errors.New("configured check path is invalid")
	}
	digest, err := digestFile(resolved)
	return resolved, digest, err
}

func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", errors.New("configured executable could not be fingerprinted")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxFingerprintFile {
		return "", errors.New("configured executable is not a bounded regular file")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, io.LimitReader(file, maxFingerprintFile+1)); err != nil {
		return "", errors.New("configured executable could not be read")
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func toAgentConfig(spec RunnerSpec) agentexec.Config {
	return agentexec.Config{
		Command: spec.Command, Args: append([]string(nil), spec.Args...), Model: spec.Model,
		ModelOptions: append(json.RawMessage(nil), spec.ModelOptions...), ProviderVersion: spec.ProviderVersion,
		Timeout:        time.Duration(spec.TimeoutSeconds) * time.Second,
		MaxStdoutBytes: spec.MaxStdoutBytes, MaxStderrBytes: spec.MaxStderrBytes,
		RuntimeFiles: append([]agentexec.RuntimeFile(nil), spec.RuntimeFiles...),
	}
}

// Invoke starts exactly one fresh agentexec process. Callers provide an
// independent Request for each slot and must use executor/verifier roles;
// semantic observation and vote validation belongs to the orchestrator.
func Invoke(ctx context.Context, spec RunnerSpec, request agentexec.Request, workspace, stateDirectory, temporaryDirectory string) (agentexec.RunResult, error) {
	if err := validateRunner(spec); err != nil {
		return agentexec.RunResult{}, err
	}
	if request.Role != agentexec.RoleExecutor && request.Role != agentexec.RoleVerifier {
		return agentexec.RunResult{}, errors.New("government actor role must be executor or verifier")
	}
	if workspace == "" || !filepath.IsAbs(workspace) {
		return agentexec.RunResult{}, errors.New("workspace must be an absolute path")
	}
	state, err := absoluteDirectory(stateDirectory)
	if err != nil {
		return agentexec.RunResult{}, fmt.Errorf("stateDirectory: %w", err)
	}
	temporary, err := absoluteDirectory(temporaryDirectory)
	if err != nil {
		return agentexec.RunResult{}, fmt.Errorf("temporaryDirectory: %w", err)
	}
	if directoriesOverlap(state, temporary) {
		return agentexec.RunResult{}, errors.New("state and temporary directories must be disjoint")
	}
	return agentexec.Run(ctx, toAgentConfig(spec), request, agentexec.RunOptions{
		InputRoots: []string{workspace}, TempParent: temporary, PrivateLogDirectory: state,
	})
}
