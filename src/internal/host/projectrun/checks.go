package projectrun

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func bindCheckExecutables(checks []CheckPlan, config Runtime) ([]CheckPlan, []string, error) {
	var findings []string
	for i := range checks {
		agent, ok := config.Agents[checks[i].Owner]
		if !ok {
			findings = append(findings, "check "+checks[i].ID+" owner has no runtime agent environment policy")
			continue
		}
		path, digest, err := resolveCheckExecutable(checks[i].Command[0], agent.Environment)
		if err != nil {
			findings = append(findings, "check "+checks[i].ID+" executable cannot be pinned: "+err.Error())
			continue
		}
		checks[i].ExecutablePath = path
		checks[i].ExecutableDigest = digest
	}
	return checks, uniqueSorted(findings), nil
}

func resolveCheckExecutable(command string, allowlist []string) (string, string, error) {
	if strings.ContainsAny(command, "/\\:") {
		return "", "", fmt.Errorf("check executable must be a bare literal name")
	}
	var pathValue, extValue string
	hasPath := false
	hasExt := false
	for _, name := range allowlist {
		if strings.EqualFold(name, "PATH") {
			pathValue, _ = os.LookupEnv(name)
			hasPath = true
		}
		if strings.EqualFold(name, "PATHEXT") {
			extValue, _ = os.LookupEnv(name)
			hasExt = true
		}
	}
	if !hasPath || strings.TrimSpace(pathValue) == "" {
		return "", "", fmt.Errorf("runtime must explicitly allow PATH to pin check executables")
	}
	var suffixes []string
	if runtime.GOOS == "windows" {
		if hasExt && extValue != "" {
			suffixes = strings.Split(extValue, ";")
		} else {
			suffixes = []string{".EXE", ".COM"}
		}
		if filepath.Ext(command) != "" {
			suffixes = []string{""}
		}
	} else {
		suffixes = []string{""}
	}
	for _, directory := range filepath.SplitList(pathValue) {
		if strings.TrimSpace(directory) == "" {
			continue
		}
		for _, suffix := range suffixes {
			candidate := filepath.Join(directory, command+suffix)
			resolved, raw, err := readPinnedExecutable(candidate)
			if err != nil {
				continue
			}
			if !nativeExecutable(raw, resolved) {
				continue
			}
			sum := sha256.Sum256(raw)
			return resolved, "sha256:" + hex.EncodeToString(sum[:]), nil
		}
	}
	return "", "", fmt.Errorf("%q was not found as a supported native executable on the allowlisted PATH", command)
}

func readPinnedExecutable(path string) (string, []byte, error) {
	linkBefore, err := os.Lstat(path)
	if err != nil {
		return "", nil, err
	}
	if linkBefore.Mode()&os.ModeSymlink != 0 && !linkBefore.Mode().IsRegular() {
		// A PATH symlink is acceptable only after resolving it to one fixed,
		// regular native executable. The resolved target, never the PATH alias,
		// is pinned and executed.
	} else if !linkBefore.Mode().IsRegular() {
		return "", nil, fmt.Errorf("not a regular file")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", nil, err
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", nil, err
	}
	before, err := os.Stat(resolved)
	if err != nil || !before.Mode().IsRegular() {
		return "", nil, fmt.Errorf("resolved executable is not a regular file")
	}
	if before.Size() < 0 || before.Size() > 256<<20 {
		return "", nil, fmt.Errorf("resolved executable exceeds 256 MiB pinning limit")
	}
	f, err := os.Open(resolved)
	if err != nil {
		return "", nil, err
	}
	raw, readErr := io.ReadAll(io.LimitReader(f, (256<<20)+1))
	opened, statErr := f.Stat()
	closeErr := f.Close()
	if readErr != nil {
		return "", nil, readErr
	}
	if statErr != nil {
		return "", nil, statErr
	}
	if closeErr != nil {
		return "", nil, closeErr
	}
	linkAfter, err := os.Lstat(path)
	if err != nil || !os.SameFile(linkBefore, linkAfter) {
		return "", nil, fmt.Errorf("PATH executable alias changed while pinning")
	}
	after, err := os.Stat(resolved)
	if err != nil || !os.SameFile(before, opened) || !os.SameFile(opened, after) || len(raw) != int(before.Size()) {
		return "", nil, fmt.Errorf("executable changed while pinning")
	}
	return resolved, raw, nil
}

func nativeExecutable(raw []byte, path string) bool {
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))
		return (ext == ".exe" || ext == ".com") && len(raw) >= 2 && raw[0] == 'M' && raw[1] == 'Z'
	}
	if len(raw) >= 4 && string(raw[:4]) == "\x7fELF" {
		return true
	}
	if len(raw) >= 4 {
		magic := []byte{raw[0], raw[1], raw[2], raw[3]}
		if string(magic) == "\xfe\xed\xfa\xce" || string(magic) == "\xce\xfa\xed\xfe" || string(magic) == "\xfe\xed\xfa\xcf" || string(magic) == "\xcf\xfa\xed\xfe" || string(magic) == "\xca\xfe\xba\xbe" {
			return true
		}
	}
	return false
}

func validateCheckExecutables(plan PlanRecord) error {
	for _, check := range plan.Checks {
		if check.ExecutablePath == "" || check.ExecutableDigest == "" {
			return fmt.Errorf("check %s has no bound executable", check.ID)
		}
		resolved, raw, err := readPinnedExecutable(check.ExecutablePath)
		if err != nil {
			return fmt.Errorf("check %s executable unavailable: %w", check.ID, err)
		}
		if resolved != check.ExecutablePath || rawContentDigest(raw) != check.ExecutableDigest {
			return fmt.Errorf("check %s executable changed since planning", check.ID)
		}
	}
	return nil
}
