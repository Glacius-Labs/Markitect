package projectconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go.yaml.in/yaml/v3"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxRuntimeConfigBytes = 1 << 20

// UnmarshalYAML requires a human-readable duration with a unit.
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return fmt.Errorf("duration must be a string such as 30s or 5m")
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(node.Value))
	if err != nil || parsed <= 0 {
		return fmt.Errorf("duration must be a positive Go duration: %q", node.Value)
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	return time.Duration(d).String(), nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// LoadRuntime reads and strictly validates the project-owned runtime config.
// It never resolves or starts a provider. Isolation claims are rejected here
// until a host-controlled launcher supplies verified evidence.
func LoadRuntime(root string) (Runtime, error) {
	var config Runtime
	if strings.TrimSpace(root) == "" {
		return config, fmt.Errorf("project root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return config, fmt.Errorf("resolve project root: %w", err)
	}
	if err := rejectReparsePath(root, RuntimePath); err != nil {
		return config, err
	}
	path := filepath.Join(root, filepath.FromSlash(RuntimePath))
	file, err := os.Open(path)
	if err != nil {
		return config, fmt.Errorf("open %s: %w", RuntimePath, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return config, fmt.Errorf("stat %s: %w", RuntimePath, err)
	}
	if !info.Mode().IsRegular() {
		return config, fmt.Errorf("%s must be a regular file", RuntimePath)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxRuntimeConfigBytes+1))
	if err != nil {
		return config, fmt.Errorf("read %s: %w", RuntimePath, err)
	}
	if len(data) > maxRuntimeConfigBytes {
		return config, fmt.Errorf("%s exceeds %d bytes", RuntimePath, maxRuntimeConfigBytes)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("decode %s: %w", RuntimePath, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return config, fmt.Errorf("%s must contain exactly one YAML document", RuntimePath)
		}
		return config, fmt.Errorf("decode trailing YAML in %s: %w", RuntimePath, err)
	}
	if err := ValidateRuntime(config); err != nil {
		return config, err
	}
	return config, nil
}

func rejectReparsePath(root, relative string) error {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("inspect project root: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("project root must be a real directory")
	}
	current := root
	for _, segment := range strings.Split(filepath.FromSlash(relative), string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) && current == filepath.Join(root, ".markitect") {
			return fmt.Errorf("%s does not exist", relative)
		}
		if statErr != nil {
			return fmt.Errorf("inspect %s: %w", relative, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s contains a symlink", relative)
		}
		if current != filepath.Join(root, filepath.FromSlash(relative)) && !info.IsDir() {
			return fmt.Errorf("%s parent must be a directory", relative)
		}
	}
	return nil
}
