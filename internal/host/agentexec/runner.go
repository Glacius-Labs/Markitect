package agentexec

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxOutputBound      = 16 << 20
	maxRuntimeFiles     = 32
	maxRuntimeFileBytes = 256 << 20
	maxSnapshotFiles    = 200000
	maxSnapshotBytes    = 1 << 30
)

type limitedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	overflow bool
	cancel   context.CancelFunc
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.buffer.Len()
	if remaining <= 0 {
		b.overflow = true
		b.cancel()
		return 0, ErrOutputTooLarge
	}
	if len(p) > remaining {
		_, _ = b.buffer.Write(p[:remaining])
		b.overflow = true
		b.cancel()
		return remaining, ErrOutputTooLarge
	}
	return b.buffer.Write(p)
}

type fileFact struct {
	path    string
	kind    string
	mode    uint32
	size    int64
	modTime int64
	digest  string
}

type workspaceSnapshot struct {
	files      []fileFact
	totalBytes int64
}

type runtimeState struct {
	path   string
	mode   string
	digest string
}

func run(parent context.Context, cfg Config, request Request, opts RunOptions) (RunResult, error) {
	if parent == nil {
		return RunResult{}, errors.New("execution context is required")
	}
	req, requestJSON, err := normalizeRequest(request)
	if err != nil {
		return RunResult{}, err
	}
	roots, err := normalizeRoots(opts.InputRoots)
	if err != nil {
		return RunResult{}, err
	}
	tempParent := opts.TempParent
	if tempParent == "" {
		tempParent = os.TempDir()
	}
	tempParent, err = canonicalDir(tempParent)
	if err != nil {
		return RunResult{}, fmt.Errorf("temporary parent: %w", err)
	}
	if pathWithinAny(tempParent, roots) {
		return RunResult{}, errors.New("temporary parent is within an audited input root")
	}
	privateLogDir, err := preparePrivateLogDirectory(opts.PrivateLogDirectory, roots)
	if err != nil {
		return RunResult{}, err
	}

	cfg, executable, runtimeBefore, runtimeDigest, executableDigest, configDigest, err := fingerprintConfig(cfg)
	if err != nil {
		return RunResult{}, err
	}
	commandBytes, _ := json.Marshal(struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}{cfg.Command, cfg.Args})
	commandDigest := digest(commandBytes)

	runID, err := randomID()
	if err != nil {
		return RunResult{}, errors.New("could not allocate an invocation identity")
	}
	nonce, err := randomID()
	if err != nil {
		return RunResult{}, errors.New("could not allocate an invocation nonce")
	}
	inputDigest := digest(requestJSON)
	invocation := Invocation{
		APIVersion:  APIVersion,
		RunID:       runID,
		Nonce:       nonce,
		InputDigest: inputDigest,
		Request:     req,
	}
	inputJSON, err := json.Marshal(invocation)
	if err != nil || len(inputJSON) > maxJSONBytes {
		return RunResult{}, errors.New("invocation could not be encoded within the protocol bound")
	}

	before, err := snapshotRoots(roots)
	if err != nil {
		return RunResult{}, errors.New("input audit before invocation failed")
	}
	runDir, err := os.MkdirTemp(tempParent, "markitect-agentexec-")
	if err != nil {
		return RunResult{}, errors.New("could not create a fresh private invocation directory")
	}
	defer os.RemoveAll(runDir)
	if err := os.Chmod(runDir, 0700); err != nil {
		return RunResult{}, errors.New("could not restrict the invocation directory")
	}
	if pathWithinAny(runDir, roots) || pathWithinAny(runDir, []string{privateLogDir}) {
		return RunResult{}, errors.New("invocation directory overlaps a protected directory")
	}
	privateLogPath := filepath.Join(privateLogDir, runID+".jsonl")
	if _, err := os.Lstat(privateLogPath); !errors.Is(err, os.ErrNotExist) {
		return RunResult{}, errors.New("private log path already exists")
	}

	ctx, cancel := context.WithTimeout(parent, cfg.Timeout)
	defer cancel()
	stdout := &limitedBuffer{limit: cfg.MaxStdoutBytes, cancel: cancel}
	stderr := &limitedBuffer{limit: cfg.MaxStderrBytes, cancel: cancel}
	cmd := exec.Command(executable, cfg.Args...)
	cmd.Args[0] = cfg.Command
	cmd.Dir = runDir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = setEnvironment(os.Environ(), "MARKITECT_AGENT_PRIVATE_LOG", privateLogPath)
	configJSON, _ := json.Marshal(struct {
		Model           string          `json:"model"`
		ModelOptions    json.RawMessage `json:"modelOptions"`
		ProviderVersion string          `json:"providerVersion"`
	}{cfg.Model, cfg.ModelOptions, cfg.ProviderVersion})
	cmd.Env = setEnvironment(cmd.Env, "MARKITECT_AGENT_CONFIG_JSON", string(configJSON))
	start := time.Now()
	err = runGuardedProcess(ctx, cmd, inputJSON)
	wallTime := time.Since(start)
	after, auditErr := snapshotRoots(roots)
	result := RunResult{}
	receipt := Receipt{
		APIVersion:            APIVersion,
		RunID:                 runID,
		InputDigest:           inputDigest,
		ContextDigest:         digest(req.Context),
		ConfigDigest:          configDigest,
		CommandDigest:         commandDigest,
		ExecutableDigest:      executableDigest,
		RuntimeFilesDigest:    runtimeDigest,
		ProviderVersion:       cfg.ProviderVersion,
		ProviderVersionDigest: digest([]byte(cfg.ProviderVersion)),
		StdoutDigest:          digest(stdout.buffer.Bytes()),
		StderrDigest:          digest(stderr.buffer.Bytes()),
		Outcome:               OutcomeIncomplete,
		WallTimeMilliseconds:  wallTime.Milliseconds(),
		RetryCount:            0,
	}
	logBytes, logErr := readOptionalBoundedFile(privateLogPath, maxOutputBound)
	if logErr == nil && logBytes != nil {
		receipt.PrivateLogDigest = digest(logBytes)
	}
	if !sameSnapshot(before, after) || auditErr != nil {
		result.Receipt = receipt
		return result, ErrInputChanged
	}
	if runtimeErr := verifyRuntimeFiles(runtimeBefore); runtimeErr != nil {
		result.Receipt = receipt
		return result, ErrInputChanged
	}
	if err := verifyExecutableDigest(executable, executableDigest); err != nil {
		result.Receipt = receipt
		return result, ErrInputChanged
	}
	if logErr != nil {
		result.Receipt = receipt
		return result, errors.New("private log could not be read within its configured bound")
	}
	if stdout.overflow || stderr.overflow {
		result.Receipt = receipt
		return result, ErrOutputTooLarge
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		result.Receipt = receipt
		return result, errors.New("external runner timed out or was cancelled")
	}
	if err != nil {
		receipt.Outcome = OutcomeFailed
		result.Receipt = receipt
		return result, errors.New("external runner failed")
	}

	var response Response
	if err := strictDecode(stdout.buffer.Bytes(), &response); err != nil {
		result.Receipt = receipt
		return result, errors.New("external runner returned an invalid response")
	}
	if err := validateResponse(response, req, invocation); err != nil {
		result.Receipt = receipt
		return result, err
	}
	receipt.Outcome = response.Outcome
	receipt.Usage = cloneUsage(response.Usage)
	result.Response = response
	result.Receipt = receipt
	return result, nil
}

// Fingerprint returns the configuration identity that Run records in its receipt.
// It resolves and hashes the configured executable and every declared runtime file
// without invoking the external runner.
func Fingerprint(input Config) (string, error) {
	_, _, _, _, _, value, err := fingerprintConfig(input)
	return value, err
}

func fingerprintConfig(input Config) (Config, string, []runtimeState, string, string, string, error) {
	cfg, err := normalizeConfig(input)
	if err != nil {
		return Config{}, "", nil, "", "", "", err
	}
	executable, err := exec.LookPath(cfg.Command)
	if err != nil {
		return Config{}, "", nil, "", "", "", errors.New("configured runner command is unavailable")
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return Config{}, "", nil, "", "", "", errors.New("configured runner command could not be resolved")
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return Config{}, "", nil, "", "", "", errors.New("configured runner command path is invalid")
	}
	if isShellExecutable(executable) {
		return Config{}, "", nil, "", "", "", errors.New("runner command must be a direct executable, not a shell")
	}
	executableBytes, err := readBoundedFile(executable, 256<<20)
	if err != nil {
		return Config{}, "", nil, "", "", "", errors.New("configured runner executable could not be fingerprinted")
	}
	executableDigest := digest(executableBytes)
	runtimeBefore, runtimeDigest, err := snapshotRuntimeFiles(cfg.RuntimeFiles)
	if err != nil {
		return Config{}, "", nil, "", "", "", err
	}
	configBytes, err := json.Marshal(cfg)
	if err != nil {
		return Config{}, "", nil, "", "", "", err
	}
	configDigest := digest(append(append([]byte(nil), configBytes...), []byte("executable:"+executableDigest+"runtime:"+runtimeDigest)...))
	return cfg, executable, runtimeBefore, runtimeDigest, executableDigest, configDigest, nil
}

func setEnvironment(values []string, key string, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(values)+1)
	inserted := false
	for _, item := range values {
		currentKey, _, found := strings.Cut(item, "=")
		if found && strings.EqualFold(currentKey, key) {
			if !inserted {
				out = append(out, prefix+value)
				inserted = true
			}
			continue
		}
		out = append(out, item)
	}
	if !inserted {
		out = append(out, prefix+value)
	}
	return out
}
func normalizeConfig(input Config) (Config, error) {
	cfg := input
	if strings.TrimSpace(cfg.Command) == "" || strings.ContainsRune(cfg.Command, 0) {
		return Config{}, errors.New("runner command is required")
	}
	if cfg.Model == "" || len(cfg.Model) > maxFieldBytes || !utf8.ValidString(cfg.Model) {
		return Config{}, errors.New("explicit model identity is required")
	}
	if cfg.ProviderVersion == "" || len(cfg.ProviderVersion) > maxFieldBytes || !utf8.ValidString(cfg.ProviderVersion) {
		return Config{}, errors.New("explicit provider version is required")
	}
	if cfg.Timeout <= 0 || cfg.Timeout > 10*time.Minute {
		return Config{}, errors.New("runner timeout must be at most ten minutes")
	}
	if cfg.MaxStdoutBytes <= 0 || cfg.MaxStdoutBytes > maxOutputBound ||
		cfg.MaxStderrBytes <= 0 || cfg.MaxStderrBytes > maxOutputBound {
		return Config{}, errors.New("stdout and stderr limits must be between 1 byte and 16 MiB")
	}
	for _, arg := range cfg.Args {
		if strings.ContainsRune(arg, 0) {
			return Config{}, errors.New("runner arguments cannot contain NUL")
		}
	}
	if len(cfg.ModelOptions) == 0 {
		cfg.ModelOptions = json.RawMessage("{}")
	}
	var err error
	cfg.ModelOptions, err = canonicalObject(cfg.ModelOptions, "modelOptions", 64<<10)
	if err != nil {
		return Config{}, err
	}
	cfg.Args = append([]string(nil), cfg.Args...)
	if len(cfg.RuntimeFiles) > maxRuntimeFiles {
		return Config{}, errors.New("runtimeFiles exceed the configured count bound")
	}
	cfg.RuntimeFiles = append([]RuntimeFile(nil), cfg.RuntimeFiles...)
	sort.Slice(cfg.RuntimeFiles, func(i, j int) bool { return cfg.RuntimeFiles[i].Path < cfg.RuntimeFiles[j].Path })
	seen := make(map[string]struct{}, len(cfg.RuntimeFiles))
	for i := range cfg.RuntimeFiles {
		file := &cfg.RuntimeFiles[i]
		if !filepath.IsAbs(file.Path) {
			return Config{}, errors.New("runtime file paths must be absolute")
		}
		clean, err := filepath.Abs(filepath.Clean(file.Path))
		if err != nil {
			return Config{}, errors.New("runtime file path is invalid")
		}
		file.Path = clean
		key := strings.ToLower(clean)
		if _, ok := seen[key]; ok {
			return Config{}, errors.New("runtime files contain a duplicate path")
		}
		seen[key] = struct{}{}
		if file.Mode != "0444" && file.Mode != "0644" && file.Mode != "0600" && file.Mode != "0755" {
			return Config{}, errors.New("runtime file mode must be 0444, 0600, 0644, or 0755")
		}
		if !validDigest(file.Digest) {
			return Config{}, errors.New("runtime file digest must be sha256")
		}
	}
	return cfg, nil
}

func validateResponse(response Response, req Request, invocation Invocation) error {
	if response.APIVersion != APIVersion || response.RunID != invocation.RunID ||
		response.Nonce != invocation.Nonce || response.Role != req.Role ||
		response.InputDigest != invocation.InputDigest {
		return errors.New("external response does not bind to this invocation")
	}
	if response.CandidateFiles == nil || response.EvidenceRefs == nil || response.VerifierObservations == nil || response.Uncertainty == nil {
		return errors.New("external response is missing a required array")
	}
	if len(response.CandidateFiles) > maxArtifactCount || len(response.EvidenceRefs) > maxScopeCount ||
		len(response.VerifierObservations) > maxScopeCount || len(response.Uncertainty) > maxScopeCount {
		return errors.New("external response exceeds a collection bound")
	}
	allowedOutcome := false
	switch req.Role {
	case RoleExecutor:
		allowedOutcome = response.Outcome == OutcomeProposed || response.Outcome == OutcomeFailed || response.Outcome == OutcomeIncomplete || response.Outcome == OutcomeEscalated
		if response.Outcome == OutcomeProposed && len(response.CandidateFiles) == 0 {
			return errors.New("executor proposal contains no candidate files")
		}
		if len(response.VerifierObservations) != 0 || len(response.CandidateJSON) != 0 {
			return errors.New("executor response contains verifier or inference output")
		}
	case RoleVerifier:
		allowedOutcome = response.Outcome == OutcomePassed || response.Outcome == OutcomeFailed ||
			response.Outcome == OutcomeIncomplete || response.Outcome == OutcomeEscalated
		if len(response.CandidateFiles) != 0 || len(response.CandidateJSON) != 0 {
			return errors.New("verifier response contains candidate or inference output")
		}
		if (response.Outcome == OutcomePassed || response.Outcome == OutcomeFailed) && len(response.VerifierObservations) == 0 {
			return errors.New("verifier result requires explicit observations")
		}
	case RoleInfer:
		allowedOutcome = response.Outcome == OutcomeProposed || response.Outcome == OutcomeFailed || response.Outcome == OutcomeIncomplete || response.Outcome == OutcomeEscalated
		if len(response.CandidateFiles) != 0 || len(response.VerifierObservations) != 0 {
			return errors.New("inference response contains execution or verifier output")
		}
		if response.Outcome == OutcomeProposed && len(response.CandidateJSON) == 0 {
			return errors.New("inference proposal requires candidateJson")
		}
	}
	if !allowedOutcome {
		return errors.New("external response outcome is not valid for its role")
	}
	if len(response.CandidateJSON) != 0 {
		if _, err := canonicalObject(response.CandidateJSON, "candidateJson", maxContextBytes); err != nil {
			return err
		}
	}
	seen := make(map[string]struct{}, len(response.CandidateFiles))
	total := 0
	for _, file := range response.CandidateFiles {
		clean, key, err := portablePath(file.Path)
		if err != nil {
			return fmt.Errorf("candidate path: %w", err)
		}
		if clean != file.Path {
			return errors.New("candidate path must be canonical")
		}
		if _, ok := seen[key]; ok {
			return errors.New("candidate paths contain a duplicate or alias")
		}
		seen[key] = struct{}{}
		if file.Mode != "0644" && file.Mode != "0755" && file.Mode != "0600" {
			return errors.New("candidate mode is unsupported")
		}
		if !utf8.ValidString(file.Content) {
			return errors.New("candidate content is not UTF-8")
		}
		total += len(file.Content)
		if total > maxArtifactBytes {
			return errors.New("candidate bytes exceed the 8 MiB response bound")
		}
	}
	evidenceRefs := make(map[string]struct{}, len(req.Artifacts)+len(req.ScopeIDs)+len(req.PolicyIDs))
	for _, artifact := range req.Artifacts {
		evidenceRefs[artifact.Path] = struct{}{}
	}
	for _, id := range req.ScopeIDs {
		evidenceRefs[id] = struct{}{}
	}
	for _, id := range req.PolicyIDs {
		evidenceRefs[id] = struct{}{}
	}
	seenEvidence := make(map[string]struct{}, len(response.EvidenceRefs))
	for _, ref := range response.EvidenceRefs {
		if ref == "" || len(ref) > maxFieldBytes || !utf8.ValidString(ref) {
			return errors.New("evidence reference is invalid")
		}
		if _, ok := evidenceRefs[ref]; !ok {
			return errors.New("evidence reference was not supplied in the request")
		}
		if _, ok := seenEvidence[ref]; ok {
			return errors.New("evidence references contain a duplicate")
		}
		seenEvidence[ref] = struct{}{}
	}
	for _, observation := range response.VerifierObservations {
		if observation.Subject == "" || observation.Detail == "" ||
			len(observation.Subject) > maxFieldBytes || len(observation.Detail) > maxFieldBytes ||
			!utf8.ValidString(observation.Subject) || !utf8.ValidString(observation.Detail) {
			return errors.New("verifier observation is invalid")
		}
		switch observation.Outcome {
		case OutcomePassed, OutcomeFailed, OutcomeIncomplete, OutcomeEscalated:
		default:
			return errors.New("verifier observation outcome is invalid")
		}
	}
	for _, uncertainty := range response.Uncertainty {
		if uncertainty == "" || len(uncertainty) > maxFieldBytes || !utf8.ValidString(uncertainty) {
			return errors.New("uncertainty entry is invalid")
		}
	}
	if response.Usage != nil {
		if response.Usage.Source != "provider-reported" ||
			!nonnegative(response.Usage.InputTokens) || !nonnegative(response.Usage.OutputTokens) ||
			!nonnegative(response.Usage.CachedTokens) || !nonnegative(response.Usage.ToolCalls) ||
			(response.Usage.InputTokens == nil && response.Usage.OutputTokens == nil &&
				response.Usage.CachedTokens == nil && response.Usage.ToolCalls == nil) {
			return errors.New("usage must be explicitly provider-reported and nonnegative")
		}
	}
	return nil
}

func nonnegative(value *int64) bool { return value == nil || *value >= 0 }

func cloneUsage(input *Usage) *Usage {
	if input == nil {
		return nil
	}
	copy := *input
	return &copy
}

func normalizeRoots(values []string) ([]string, error) {
	roots := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		root, err := canonicalInputPath(value)
		if err != nil {
			return nil, errors.New("selected input path is invalid")
		}
		// Preserve distinct paths on case-sensitive filesystems. Dropping one
		// selected root here would leave it outside the before/after audit.
		key := root
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		roots = append(roots, root)
	}
	return roots, nil
}

func canonicalInputPath(value string) (string, error) {
	if value == "" {
		return "", errors.New("path is required")
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	evaluated, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(evaluated)
	if err != nil || (!info.IsDir() && !info.Mode().IsRegular()) {
		return "", errors.New("selected input must be a regular file or directory")
	}
	return filepath.Clean(evaluated), nil
}

func canonicalDir(value string) (string, error) {
	if value == "" {
		return "", errors.New("path is required")
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	evaluated, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(evaluated)
	if err != nil || !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return filepath.Clean(evaluated), nil
}

func pathWithinAny(path string, roots []string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(root, path)
		if err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))) {
			return true
		}
	}
	return false
}

func preparePrivateLogDirectory(value string, roots []string) (string, error) {
	if value == "" {
		return "", errors.New("a private log directory is required")
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", errors.New("private log directory path is invalid")
	}
	absolute = filepath.Clean(absolute)
	info, statErr := os.Lstat(absolute)
	if statErr == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("private log directory must be a real directory")
		}
		canonical, err := canonicalDir(absolute)
		if err != nil || pathWithinAny(canonical, roots) {
			return "", errors.New("private log directory must be outside audited input roots")
		}
		if err := os.Chmod(canonical, 0700); err != nil {
			return "", errors.New("private log directory permissions could not be restricted")
		}
		return canonical, nil
	}
	if !errors.Is(statErr, os.ErrNotExist) {
		return "", errors.New("private log directory could not be inspected")
	}
	parent := filepath.Dir(absolute)
	for {
		_, err := os.Lstat(parent)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", errors.New("private log directory parent could not be inspected")
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", errors.New("private log directory parent could not be resolved")
		}
		parent = next
	}
	canonicalParent, err := canonicalDir(parent)
	if err != nil {
		return "", errors.New("private log directory parent must be a real directory")
	}
	relative, err := filepath.Rel(parent, absolute)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return "", errors.New("private log directory path is invalid")
	}
	candidate := filepath.Join(canonicalParent, relative)
	if pathWithinAny(candidate, roots) {
		return "", errors.New("private log directory must be outside audited input roots")
	}
	if err := os.MkdirAll(absolute, 0700); err != nil {
		return "", errors.New("private log directory could not be created")
	}
	canonical, err := canonicalDir(absolute)
	if err != nil || pathWithinAny(canonical, roots) {
		return "", errors.New("private log directory must be outside audited input roots")
	}
	if err := os.Chmod(canonical, 0700); err != nil {
		return "", errors.New("private log directory permissions could not be restricted")
	}
	return canonical, nil
}

// runGuardedProcess withholds the invocation until the operating-system process
// tree guard owns the child. This is lifecycle containment for the configured
// caller, not a sandbox: arbitrary programs that spawn before reading stdin can
// still create an unguarded descendant. The supported Codex wrapper reads the
// complete closed request before launching its provider process.
func runGuardedProcess(ctx context.Context, cmd *exec.Cmd, input []byte) (resultErr error) {
	guard, err := newProcessTreeGuard()
	if err != nil {
		return errors.New("runner process-tree guard could not be established")
	}
	var waitDone chan error
	waitConsumed := false
	defer func() {
		if err := guard.close(); err != nil && resultErr == nil {
			resultErr = errors.New("runner process-tree cleanup failed")
		}
		if waitDone != nil && !waitConsumed {
			select {
			case <-waitDone:
				waitConsumed = true
			case <-time.After(2 * time.Second):
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				select {
				case <-waitDone:
					waitConsumed = true
				case <-time.After(2 * time.Second):
					resultErr = errors.New("runner process did not stop within its bound")
				}
			}
		}
	}()
	if err := guard.prepare(cmd); err != nil {
		return errors.New("runner process-tree guard could not be prepared")
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.New("runner input pipe could not be opened")
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}
	if err := guard.attach(cmd); err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		waitDone = make(chan error, 1)
		go func() { waitDone <- cmd.Wait() }()
		select {
		case <-waitDone:
			waitConsumed = true
		case <-time.After(2 * time.Second):
			return errors.New("runner process could not be reaped after guard refusal")
		}
		return errors.New("runner process-tree guard could not be attached")
	}
	waitDone = make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	inputDone := make(chan error, 1)
	go func() {
		_, writeErr := stdin.Write(input)
		closeErr := stdin.Close()
		if writeErr != nil {
			inputDone <- writeErr
			return
		}
		inputDone <- closeErr
	}()

	var waitErr error
	select {
	case waitErr = <-waitDone:
		waitConsumed = true
	case <-ctx.Done():
		if err := guard.terminate(); err != nil {
			_ = cmd.Process.Kill()
			return errors.New("runner process tree could not be stopped")
		}
		_ = stdin.Close()
		select {
		case waitErr = <-waitDone:
			waitConsumed = true
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
			_ = guard.terminate()
			select {
			case <-waitDone:
				waitConsumed = true
			case <-time.After(2 * time.Second):
				return errors.New("runner process did not stop within its bound")
			}
		}
	}
	if err := guard.terminate(); err != nil {
		return errors.New("runner process tree could not be stopped after completion")
	}
	_ = stdin.Close()
	select {
	case <-inputDone:
	case <-time.After(2 * time.Second):
		return errors.New("runner input pipe did not close within its bound")
	}
	return waitErr
}

func snapshotRoots(roots []string) (workspaceSnapshot, error) {
	snapshot := workspaceSnapshot{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(name string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			info, err := os.Lstat(name)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, name)
			if err != nil {
				return err
			}
			fact := fileFact{
				path:    root + string(filepath.Separator) + rel,
				mode:    uint32(info.Mode().Perm()),
				size:    info.Size(),
				modTime: info.ModTime().UnixNano(),
			}
			switch {
			case info.Mode()&os.ModeSymlink != 0:
				target, err := os.Readlink(name)
				if err != nil {
					return err
				}
				fact.kind = "symlink:" + target
			case info.IsDir():
				fact.kind = "directory"
			case info.Mode().IsRegular():
				fact.kind = "file"
				if info.Size() < 0 || snapshot.totalBytes+info.Size() > maxSnapshotBytes {
					return errors.New("audited input exceeds the 1 GiB snapshot bound")
				}
				content, err := readBoundedFile(name, maxSnapshotBytes)
				if err != nil {
					return err
				}
				fact.digest = digest(content)
				snapshot.totalBytes += int64(len(content))
			default:
				fact.kind = "special"
			}
			snapshot.files = append(snapshot.files, fact)
			if len(snapshot.files) > maxSnapshotFiles {
				return errors.New("audited input exceeds the file count bound")
			}
			return nil
		})
		if err != nil {
			return workspaceSnapshot{}, err
		}
	}
	sort.Slice(snapshot.files, func(i, j int) bool { return snapshot.files[i].path < snapshot.files[j].path })
	return snapshot, nil
}

func sameSnapshot(a, b workspaceSnapshot) bool {
	if len(a.files) != len(b.files) || a.totalBytes != b.totalBytes {
		return false
	}
	for i := range a.files {
		if a.files[i] != b.files[i] {
			return false
		}
	}
	return true
}

func snapshotRuntimeFiles(files []RuntimeFile) ([]runtimeState, string, error) {
	states := make([]runtimeState, 0, len(files))
	total := int64(0)
	for _, file := range files {
		real, err := filepath.Abs(file.Path)
		if err != nil {
			return nil, "", errors.New("declared runtime file path is invalid")
		}
		info, err := os.Lstat(real)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return nil, "", errors.New("runtime file must be a regular non-symlink file")
		}
		if info.Size() < 0 || info.Size() > maxRuntimeFileBytes || total+info.Size() > maxRuntimeFileBytes {
			return nil, "", errors.New("runtime files exceed the 256 MiB bound")
		}
		content, err := readBoundedFile(real, maxRuntimeFileBytes)
		if err != nil || digest(content) != file.Digest || runtimeFileMode(info.Mode()) != file.Mode {
			return nil, "", errors.New("runtime file differs from its declared digest or mode")
		}
		total += int64(len(content))
		states = append(states, runtimeState{path: real, mode: file.Mode, digest: file.Digest})
	}
	encoded, _ := json.Marshal(files)
	return states, digest(encoded), nil
}

func runtimeFileMode(mode os.FileMode) string {
	if runtime.GOOS == "windows" {
		if mode.Perm()&0200 == 0 {
			return "0444"
		}
		return "0644"
	}
	return fmt.Sprintf("%04o", mode.Perm())
}
func verifyRuntimeFiles(expected []runtimeState) error {
	files := make([]RuntimeFile, 0, len(expected))
	for _, state := range expected {
		files = append(files, RuntimeFile{Path: state.path, Mode: state.mode, Digest: state.digest})
	}
	_, _, err := snapshotRuntimeFiles(files)
	return err
}

// verifyExecutableDigest detects an executable that remains changed or becomes
// unavailable during the invocation. It narrows the fingerprint window but
// cannot prove which bytes the OS loaded if the path is replaced and restored
// between checks.
func verifyExecutableDigest(path, expected string) error {
	content, err := readBoundedFile(path, 256<<20)
	if err != nil {
		return err
	}
	if digest(content) != expected {
		return errors.New("configured runner executable changed during invocation")
	}
	return nil
}

func readBoundedFile(name string, limit int64) ([]byte, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("file exceeds read bound")
	}
	return data, nil
}

func readOptionalBoundedFile(name string, limit int64) ([]byte, error) {
	data, err := readBoundedFile(name, limit)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err == nil && data == nil {
		return []byte{}, nil
	}
	return data, err
}

func randomID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func isShellExecutable(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	switch name {
	case "cmd", "cmd.exe", "powershell", "powershell.exe", "pwsh", "pwsh.exe",
		"sh", "bash", "zsh", "fish", "dash", "ash":
		return true
	default:
		return false
	}
}
