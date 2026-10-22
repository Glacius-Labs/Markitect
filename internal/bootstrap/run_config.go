package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	runport "github.com/Glacius-Labs/Markitect/internal/run"
)

var errRunConfig = errors.New("run configuration unavailable, incomplete or invalid; see the closed formatVersion 2 contract")
var errRunConfigIO = errors.New("run configuration file could not be completely observed")

// runConfig supplies explicit invocation assertions and concrete Runtime inputs.
// It adds no project settings or profile defaults. ProductBuild is the
// operator's assertion identifying the frozen build, not a verified source SHA.
// Harvest bounds every observation of a shared working tree. Deadline is the
// optional absolute wall-clock limit (RFC 3339, UTC) that the run must finish
// before; the project's own budget still applies.
type runConfig struct {
	FormatVersion         int                         `json:"formatVersion"`
	RepositoryIdentity    string                      `json:"repositoryIdentity"`
	ProductBuild          string                      `json:"productBuild"`
	OwnerRequest          string                      `json:"ownerRequest"`
	StateDirectory        string                      `json:"stateDirectory"`
	EngineExecutable      string                      `json:"engineExecutable"`
	WorkspaceHelper       container.ContainerSpec     `json:"workspaceHelper"`
	Harvest               harvestLimits               `json:"harvest"`
	WorkDeadline          string                      `json:"workDeadline,omitempty"`
	Deadline              string                      `json:"deadline,omitempty"`
	MaxMessages           int                         `json:"maxMessages,omitempty"`
	MessageTimeoutSeconds int                         `json:"messageTimeoutSeconds,omitempty"`
	Credentials           map[string]credentialSource `json:"credentials"`
}

// harvestLimits are the entry and byte bounds of one working-tree observation.
type harvestLimits struct {
	MaxEntries int   `json:"maxEntries"`
	MaxBytes   int64 `json:"maxBytes"`
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
	root, err := runConfigObject(data)
	if err != nil {
		return runConfig{}, err
	}
	// deadline and workDeadline are optional limit fields. Every other key is required, and any
	// further key is refused, ownerDecisions included: no preloaded owner decision
	// supplies model intent to a Run.
	required := []string{"formatVersion", "repositoryIdentity", "productBuild", "ownerRequest", "stateDirectory", "engineExecutable", "workspaceHelper", "harvest", "credentials"}
	for key := range root {
		if !slices.Contains(required, key) && key != "deadline" && key != "workDeadline" && key != "maxMessages" && key != "messageTimeoutSeconds" {
			return runConfig{}, errRunConfig
		}
	}
	for _, key := range required {
		if root[key] == nil {
			return runConfig{}, errRunConfig
		}
	}
	for _, section := range []struct {
		data json.RawMessage
		keys []string
	}{
		{root["workspaceHelper"], []string{"name", "image", "labels", "user", "cpus", "memoryBytes", "pids", "tmpBytes", "network"}},
		{root["harvest"], []string{"maxEntries", "maxBytes"}},
	} {
		if _, err := runConfigObject(section.data, section.keys...); err != nil {
			return runConfig{}, err
		}
	}
	helper, _ := runConfigObject(root["workspaceHelper"])
	if _, err := runConfigObject(helper["labels"]); err != nil {
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
	if json.Unmarshal(data, &cfg) != nil || root["deadline"] != nil && cfg.Deadline == "" || root["workDeadline"] != nil && cfg.WorkDeadline == "" || !cfg.valid() {
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
	if c.FormatVersion != 2 || !runConfigText(c.RepositoryIdentity) || len(c.RepositoryIdentity) > 256 || !runConfigText(c.ProductBuild) || !utf8.ValidString(c.OwnerRequest) || strings.TrimSpace(c.OwnerRequest) == "" || !credentialPath(c.StateDirectory) || !credentialPath(c.EngineExecutable) {
		return false
	}
	if c.WorkDeadline != "" {
		t, err := time.Parse(time.RFC3339, c.WorkDeadline)
		deadline, derr := time.Parse(time.RFC3339, c.Deadline)
		if err != nil || derr != nil || !strings.HasSuffix(c.WorkDeadline, "Z") || t.IsZero() || !t.Before(deadline) {
			return false
		}
	}
	if c.Deadline != "" {
		if t, err := time.Parse(time.RFC3339, c.Deadline); err != nil || !strings.HasSuffix(c.Deadline, "Z") || t.IsZero() {
			return false
		}
	}
	if c.MaxMessages < 0 || c.MaxMessages > 128 || c.MessageTimeoutSeconds < 0 || c.MessageTimeoutSeconds > 3600 || (c.MaxMessages == 0) != (c.MessageTimeoutSeconds == 0) {
		return false
	}
	h := c.WorkspaceHelper
	if !(runport.HarvestRequest{ID: "harvest", MaxEntries: c.Harvest.MaxEntries, MaxBytes: c.Harvest.MaxBytes}).Valid() {
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

// deadline returns the configured wall-clock limit, or the zero time for none.
func (c runConfig) deadline() time.Time {
	t, _ := time.Parse(time.RFC3339, c.Deadline)
	return t
}

func runConfigText(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && utf8.ValidString(value) && !strings.ContainsFunc(value, unicode.IsControl)
}

// workDeadline reserves the remaining whole-run window for independent Verify.
func (c runConfig) workDeadline() time.Time {
	t, _ := time.Parse(time.RFC3339, c.WorkDeadline)
	return t
}
