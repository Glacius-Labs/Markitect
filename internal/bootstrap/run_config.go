package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	runport "github.com/Glacius-Labs/Markitect/internal/run"
)

var errRunConfig = errors.New("run configuration unavailable, incomplete or invalid; see the closed formatVersion 1 contract")
var errRunConfigIO = errors.New("run configuration file could not be completely observed")

// runConfig supplies explicit invocation assertions and concrete Host inputs.
// It adds no project settings or profile defaults. Durations use integer
// nanoseconds, matching the existing producer types. ProductBuild is the
// operator's assertion identifying the frozen build, not a verified source SHA.
type runConfig struct {
	FormatVersion      int                         `json:"formatVersion"`
	RepositoryIdentity string                      `json:"repositoryIdentity"`
	ProductBuild       string                      `json:"productBuild"`
	OwnerRequest       string                      `json:"ownerRequest"`
	StateDirectory     string                      `json:"stateDirectory"`
	Allocation         runport.Capacity            `json:"allocation"`
	RestoreLimits      runport.RestoreLimits       `json:"restoreLimits"`
	EngineExecutable   string                      `json:"engineExecutable"`
	WorkspaceHelper    container.ContainerSpec     `json:"workspaceHelper"`
	EngineLimits       runport.EngineLimits        `json:"engineLimits"`
	Credentials        map[string]credentialSource `json:"credentials"`
}

func loadRunConfig(ctx context.Context, path string) (runConfig, error) {
	if ctx == nil || !credentialPath(path) {
		return runConfig{}, errRunConfig
	}
	// Reuse the existing bounded native regular-file reader. This reads only
	// the explicitly selected configuration, never any referenced credential.
	data, err := readRuntimeCredential(ctx, path)
	if err != nil {
		return runConfig{}, errRunConfigIO
	}
	return decodeRunConfig(data)
}

func decodeRunConfig(data []byte) (runConfig, error) {
	if !utf8.Valid(data) {
		return runConfig{}, errRunConfig
	}
	root, err := runConfigObject(data, "formatVersion", "repositoryIdentity", "productBuild", "ownerRequest", "stateDirectory", "allocation", "restoreLimits", "engineExecutable", "workspaceHelper", "engineLimits", "credentials")
	if err != nil {
		return runConfig{}, err
	}
	for _, section := range []struct {
		data json.RawMessage
		keys []string
	}{
		{root["allocation"], []string{"runs", "sessionStarts", "steps", "executionTime"}},
		{root["restoreLimits"], []string{"pageSize", "batches", "events", "bytes"}},
		{root["workspaceHelper"], []string{"name", "image", "labels", "user", "cpus", "memoryBytes", "pids", "tmpBytes", "network"}},
		{root["engineLimits"], []string{"startTimeout", "stopTimeout", "harvestEntries", "harvestBytes", "record"}},
	} {
		if _, err := runConfigObject(section.data, section.keys...); err != nil {
			return runConfig{}, err
		}
	}
	helper, _ := runConfigObject(root["workspaceHelper"])
	if _, err := runConfigObject(helper["labels"]); err != nil {
		return runConfig{}, err
	}
	limits, _ := runConfigObject(root["engineLimits"])
	if _, err := runConfigObject(limits["record"], "encodedBytes", "fileBytes", "manifestEntries", "metadataRows"); err != nil {
		return runConfig{}, err
	}
	credentials, err := runConfigObject(root["credentials"])
	if err != nil {
		return runConfig{}, err
	}
	for _, source := range credentials {
		if _, err := runConfigObject(source, "provider", "mode", "path"); err != nil {
			return runConfig{}, err
		}
	}
	var cfg runConfig
	if json.Unmarshal(data, &cfg) != nil || !cfg.valid() {
		return runConfig{}, errRunConfig
	}
	// Constructor validation performs no credential file I/O.
	if _, _, err := newCredentialResolvers(cfg.Credentials); err != nil {
		return runConfig{}, errRunConfig
	}
	return cfg, nil
}

// runConfigObject enforces literal field names, one occurrence, non-null
// values and a single JSON object. A nil key list allows reference/label keys.
func runConfigObject(data []byte, keys ...string) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errRunConfig
	}
	out := map[string]json.RawMessage{}
	for d.More() {
		token, err := d.Token()
		key, ok := token.(string)
		if err != nil || !ok || out[key] != nil {
			return nil, errRunConfig
		}
		if len(keys) > 0 {
			found := false
			for _, allowed := range keys {
				found = found || key == allowed
			}
			if !found {
				return nil, errRunConfig
			}
		}
		var raw json.RawMessage
		if d.Decode(&raw) != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil, errRunConfig
		}
		out[key] = raw
	}
	if _, err := d.Token(); err != nil {
		return nil, errRunConfig
	}
	if _, err := d.Token(); !errors.Is(err, io.EOF) {
		return nil, errRunConfig
	}
	for _, required := range keys {
		if out[required] == nil {
			return nil, errRunConfig
		}
	}
	return out, nil
}

var (
	runtimeName  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
	runtimeImage = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	runtimeUser  = regexp.MustCompile(`^[1-9][0-9]{0,9}:[1-9][0-9]{0,9}$`)
	runtimeLabel = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,127}$`)
)

func (c runConfig) valid() bool {
	if c.FormatVersion != 1 || !runConfigText(c.RepositoryIdentity) || len(c.RepositoryIdentity) > 256 || !runConfigText(c.ProductBuild) || !utf8.ValidString(c.OwnerRequest) || strings.TrimSpace(c.OwnerRequest) == "" || !credentialPath(c.StateDirectory) || !credentialPath(c.EngineExecutable) {
		return false
	}
	a, r, e, h := c.Allocation, c.RestoreLimits, c.EngineLimits, c.WorkspaceHelper
	if a.Runs <= 0 || a.SessionStarts <= 0 || a.Steps <= 0 || a.ExecutionTime <= 0 || r.PageSize == 0 || r.PageSize > 256 || r.Batches == 0 || r.Events == 0 || r.Bytes <= 0 || e.StartTimeout <= 0 || e.StopTimeout <= 0 || e.HarvestEntries <= 0 || e.HarvestBytes <= 0 {
		return false
	}
	if e.Record.EncodedBytes <= 0 || e.Record.EncodedBytes > 64<<20 || e.Record.FileBytes <= 0 || e.Record.ManifestEntries <= 0 || e.Record.MetadataRows <= 0 {
		return false
	}
	if !runtimeName.MatchString(h.Name) || !runtimeImage.MatchString(h.Image) || !runtimeUser.MatchString(h.User) || h.CPUs <= 0 || math.IsNaN(h.CPUs) || math.IsInf(h.CPUs, 0) || h.MemoryBytes <= 0 || h.Pids <= 0 || h.TmpBytes <= 0 || h.Network == "host" || h.Network != "" && !runtimeName.MatchString(h.Network) || len(h.Labels) == 0 {
		return false
	}
	for key, value := range h.Labels {
		if !runtimeLabel.MatchString(key) || !runConfigText(value) {
			return false
		}
	}
	return true
}

func runConfigText(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && utf8.ValidString(value) && !strings.ContainsFunc(value, unicode.IsControl)
}
