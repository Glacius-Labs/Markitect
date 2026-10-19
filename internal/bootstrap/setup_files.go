package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	runport "github.com/Glacius-Labs/Markitect/internal/run"
	"go.yaml.in/yaml/v3"
)

// The Agent profiles of a Host are generic: one Claude Code session per turn
// with constant resources and a product prompt that carries no project rules.
const (
	setupSystemPrompt = "You are a Markitect Manager agent in a team that reconciles a repository with its canonical model. Follow the turn input and report only through the offered Host tools. Never print or copy credentials."
	maxSetupFileBytes = 4 << 20
)

// hostDocument is the Host profile document as setup writes it.
type hostDocument struct {
	APIVersion    string                                            `yaml:"apiVersion"`
	AgentProfiles map[string]map[string]claudecode.HostSection      `yaml:"agentProfiles"`
	CheckProfiles map[string]map[string]checks.ContainerHostSection `yaml:"checkProfiles"`
}

// composeHostProfile renders host.yaml: every selected Agent profile as a
// Claude Code profile and every selected check profile as a container check
// profile on the observed image. The YAML encoding is deterministic.
func composeHostProfile(request cli.SetupRequest, selected selection, observed claudecode.RuntimeObservation, store string) ([]byte, error) {
	document := hostDocument{APIVersion: "markitect.host/v1", AgentProfiles: map[string]map[string]claudecode.HostSection{}, CheckProfiles: map[string]map[string]checks.ContainerHostSection{}}
	agent, err := claudecode.HostProfile{
		EngineExecutable: request.Engine, ImageID: observed.ImageID, AgentdVersion: observed.AgentdVersion, CredentialRef: setupCredentialRef, MaxWireBytes: 1 << 20,
		Session: claudecode.SessionProfile{
			Version: 1, Model: request.AgentModel, Effort: request.Effort, SystemPrompt: setupSystemPrompt,
			BuiltinTools: []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash"}, Store: store,
			CPUs: 2, MemoryBytes: 4 << 30, Pids: 1024, TmpBytes: 1 << 30, Network: "",
		},
	}.Section()
	if err != nil {
		return nil, err
	}
	for _, name := range selected.agents {
		document.AgentProfiles[name] = map[string]claudecode.HostSection{"claudeCode": agent}
	}
	for _, name := range selected.checks {
		section, err := checks.ContainerHostProfile{EngineExecutable: request.Engine, Profile: checks.ContainerProfile{
			ImageID: observed.ImageID, Executable: request.Checks[name],
			Limits: runport.CheckLimits{MilliCPUs: 2000, MemoryBytes: 4 << 30, Processes: 1024}, MaxOutputBytes: 1 << 20,
		}}.Section()
		if err != nil {
			return nil, fmt.Errorf("check profile %q: %w", name, err)
		}
		document.CheckProfiles[name] = map[string]checks.ContainerHostSection{"container": section}
	}
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// checkOutsideRepository refuses a path that is inside the repository or
// contains it. A link cannot hide an overlap.
func checkOutsideRepository(path, repository, what string) error {
	if repository == "" {
		return errors.New("the repository root is required")
	}
	if err := checkStateOutsideRepository(path, repository); err != nil {
		return refuse("%s must not overlap the repository: it is inside it or contains it", what)
	}
	return nil
}

// checkCredentialFile checks the credential path without opening it: the path
// rules that Apply applies, a regular file that no link redirects, a plausible
// size, owner-only permissions where the platform has them, and a location
// outside the repository. It never reads, copies or prints the file.
func checkCredentialFile(path, repository string) error {
	if !credentialPath(path) {
		return refuse("the credential path must be an explicit canonical absolute native path")
	}
	if err := checkOutsideRepository(path, repository, "the credential file"); err != nil {
		return err
	}
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		info, err := os.Lstat(dir)
		if err != nil {
			return refuse("the credential file's directory cannot be observed")
		}
		if credentialRedirected(info) || !info.IsDir() {
			return refuse("the credential file's path is redirected or unsafe")
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	info, err := os.Lstat(path)
	switch {
	case err != nil:
		return refuse("the credential file cannot be observed")
	case credentialRedirected(info):
		return refuse("the credential file is redirected by a link")
	case !info.Mode().IsRegular():
		return refuse("the credential path is not a regular file")
	case info.Size() <= 0 || info.Size() > maxRuntimeCredentialBytes:
		return refuse("the credential file is empty or exceeds its byte limit")
	case runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0:
		return refuse("the credential file must be readable by its owner only (mode 0600)")
	}
	return nil
}

type setupFile struct {
	name string
	data []byte
}

// writeSetupFiles creates the private Host directory, takes the state
// directory lock without waiting and writes every file that differs. A file
// with identical bytes stays untouched and is reported as unchanged. The lock
// is taken before any file is touched, so a running Apply leaves the Host as it
// is.
func writeSetupFiles(host, stateDirectory string, files []setupFile) ([]cli.SetupFile, error) {
	if err := os.MkdirAll(host, 0o700); err != nil {
		return nil, err
	}
	if info, err := os.Stat(host); err != nil || !info.IsDir() {
		return nil, refuse("the Host path is not a directory")
	}
	lock, err := lockState(stateDirectory)
	if err != nil {
		if errors.Is(err, errStateLocked) {
			return nil, errors.New("an Apply holds the Host's state directory; wait for it to finish")
		}
		return nil, err
	}
	defer lock.Release()
	// Observe every target first, so that an unusable target stops setup
	// before it replaced another file.
	existing := make([][]byte, len(files))
	for i, file := range files {
		if existing[i], err = observeSetupFile(filepath.Join(host, file.name)); err != nil {
			return nil, err
		}
	}
	report := make([]cli.SetupFile, 0, len(files))
	for i, file := range files {
		path := filepath.Join(host, file.name)
		sum := sha256.Sum256(file.data)
		entry := cli.SetupFile{Path: path, SHA256: hex.EncodeToString(sum[:]), Action: "unchanged"}
		if existing[i] == nil || !bytes.Equal(existing[i], file.data) {
			if err := writeStateFile(path, file.data, true); err != nil {
				return report, err
			}
			entry.Action = "written"
		}
		report = append(report, entry)
	}
	return report, nil
}

// observeSetupFile returns the bytes of an existing regular target, nil for an
// absent one, and refuses a link, a directory or an implausibly large file.
func observeSetupFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxSetupFileBytes || strings.ContainsRune(path, 0) {
		return nil, refuse("%s exists and is not a regular file of plausible size", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = []byte{}
	}
	return data, nil
}
