package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var errRunConfig = errors.New("run configuration unavailable, incomplete or invalid; see the closed formatVersion 3 contract")
var errRunConfigIO = errors.New("run configuration file could not be completely observed")

// runConfig holds explicit outer invocation limits and native credential sources.
// ProductBuild is an operator assertion; it does not prove a source revision.
type runConfig struct {
	FormatVersion             int                         `json:"formatVersion"`
	RepositoryIdentity        string                      `json:"repositoryIdentity"`
	ProductBuild              string                      `json:"productBuild"`
	StateDirectory            string                      `json:"stateDirectory"`
	EngineExecutable          string                      `json:"engineExecutable"`
	Deadline                  string                      `json:"deadline"`
	WorkDeadline              string                      `json:"workDeadline,omitempty"`
	MaxManagers               int                         `json:"maxManagers"`
	MaxMessages               int                         `json:"maxMessages"`
	MessageTimeoutSeconds     int                         `json:"messageTimeoutSeconds"`
	CleanupTimeoutSeconds     int                         `json:"cleanupTimeoutSeconds"`
	PersistenceTimeoutSeconds int                         `json:"persistenceTimeoutSeconds"`
	Credentials               map[string]credentialSource `json:"credentials"`
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
	// Only workDeadline is optional. Every other key is required, and any
	// further key is refused, ownerDecisions included: no preloaded owner decision
	// supplies model intent to a Run.
	required := []string{"formatVersion", "repositoryIdentity", "productBuild", "stateDirectory", "engineExecutable", "deadline", "maxManagers", "maxMessages", "messageTimeoutSeconds", "cleanupTimeoutSeconds", "persistenceTimeoutSeconds", "credentials"}
	for key := range root {
		if !slices.Contains(required, key) && key != "workDeadline" {
			return runConfig{}, errRunConfig
		}
	}
	for _, key := range required {
		if root[key] == nil {
			return runConfig{}, errRunConfig
		}
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

func (c runConfig) valid() bool {
	if c.FormatVersion != 3 || !runConfigText(c.RepositoryIdentity) || len(c.RepositoryIdentity) > 256 || !runConfigText(c.ProductBuild) || !credentialPath(c.StateDirectory) || !credentialPath(c.EngineExecutable) {
		return false
	}
	deadline, err := time.Parse(time.RFC3339, c.Deadline)
	if err != nil || !strings.HasSuffix(c.Deadline, "Z") || deadline.IsZero() {
		return false
	}
	if c.WorkDeadline != "" {
		t, err := time.Parse(time.RFC3339, c.WorkDeadline)
		if err != nil || !strings.HasSuffix(c.WorkDeadline, "Z") || t.IsZero() || !t.Before(deadline) {
			return false
		}
	}
	return c.MaxManagers > 0 && c.MaxManagers <= 1024 && c.MaxMessages >= c.MaxManagers && c.MaxMessages <= 4096 && c.MessageTimeoutSeconds > 0 && c.MessageTimeoutSeconds <= 3600 && c.CleanupTimeoutSeconds > 0 && c.CleanupTimeoutSeconds <= 60 && c.PersistenceTimeoutSeconds > 0 && c.PersistenceTimeoutSeconds <= 60
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
