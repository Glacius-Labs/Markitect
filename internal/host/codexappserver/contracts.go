// Package codexappserver owns the concrete native Codex transport boundary.
// P02 supplies configuration contracts only; no server is started here.
package codexappserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// Config is explicit App Server configuration, separate from subprocess Args.
// Empty PermissionProfile preserves the existing user's configured boundary.
// The adapter must not translate absence into broader sandbox/approval flags.
type Config struct {
	Command           string        `json:"command"`
	ProviderVersion   string        `json:"providerVersion"`
	Model             string        `json:"model"`
	ReasoningEffort   string        `json:"reasoningEffort"`
	PermissionProfile string        `json:"permissionProfile,omitempty"`
	Helpers           HelperPolicy  `json:"helpers"`
	Timeout           time.Duration `json:"timeout"`
	MaxEventBytes     int64         `json:"maxEventBytes"`
}

type HelperPolicy struct {
	Enabled          bool `json:"enabled"`
	MaxStartRequests int  `json:"maxStartRequests"`
	MaxDepth         int  `json:"maxDepth"`
}

func (c Config) Validate() error {
	for _, value := range []string{c.Command, c.ProviderVersion, c.Model, c.ReasoningEffort, c.PermissionProfile} {
		if !utf8.ValidString(value) {
			return errors.New("App Server configuration text must be valid UTF-8")
		}
	}
	if !filepath.IsAbs(c.Command) || strings.TrimSpace(c.ProviderVersion) == "" || strings.TrimSpace(c.Model) == "" || strings.TrimSpace(c.ReasoningEffort) == "" {
		return errors.New("App Server requires an absolute executable and explicit provider version, model and reasoning effort")
	}
	if c.Timeout <= 0 || c.MaxEventBytes <= 0 {
		return errors.New("App Server requires finite positive time and event byte limits")
	}
	if c.Helpers.MaxStartRequests < 0 || c.Helpers.MaxDepth < 0 || (c.Helpers.Enabled && (c.Helpers.MaxStartRequests == 0 || c.Helpers.MaxDepth == 0)) || (!c.Helpers.Enabled && (c.Helpers.MaxStartRequests != 0 || c.Helpers.MaxDepth != 0)) {
		return errors.New("helper policy must explicitly disable helpers or bound start requests and depth")
	}
	return nil
}

// Digest binds requested settings only. The implemented adapter's Invoker
// fingerprint must also bind executable bytes and instruction/runtime-file pins.
func (c Config) Digest() (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
