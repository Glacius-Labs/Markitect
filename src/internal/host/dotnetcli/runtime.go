// Package dotnetcli owns the command protocol and staged-file boundary for the
// .NET reference adapter.
package dotnetcli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/consumers/dotnet"
	"go.yaml.in/yaml/v3"
)

const maxCaptureBytes = 64 << 20

// Main is the process entrypoint used by the thin command wrapper.
func Main() { os.Exit(Run(os.Stdin, os.Stdout, os.Stderr)) }

// Run executes one adapter-request document and emits one adapter-result.
func Run(input io.Reader, output io.Writer, stderr io.Writer) int {
	body, err := io.ReadAll(input)
	if err != nil {
		return writeFailure(output, fmt.Errorf("read adapter request: %w", err))
	}
	decoder := yaml.NewDecoder(bytes.NewReader(body))
	var wire host.AdapterRequest
	if err := decoder.Decode(&wire); err != nil {
		return writeFailure(output, fmt.Errorf("decode adapter request: %w", err))
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return writeFailure(output, errors.New("adapter request must contain exactly one YAML document"))
	}
	config, configErr := dotnet.DecodeConfig(wire.Adapter.Parameters)
	request := dotnet.Request{
		APIVersion:    wire.APIVersion,
		Action:        wire.Action,
		Adapter:       dotnet.Identity{Name: wire.Adapter.Name, Type: wire.Adapter.Type, Version: wire.Adapter.Version, Target: wire.Adapter.Target},
		Model:         wire.Model,
		Config:        config,
		Captures:      map[string][]byte{},
		CaptureErrors: map[string]string{},
	}
	if configErr != nil {
		request.ConfigError = configErr.Error()
	} else {
		request.Captures, request.CaptureErrors = readMappedInputs(dotnet.CapturePaths(config))
	}
	if wire.Observation != nil {
		request.Observation, err = decodeResult(wire.Observation)
		if err != nil {
			return writeFailure(output, fmt.Errorf("decode adapter observation: %w", err))
		}
	}
	if wire.Plan != nil {
		request.Plan, err = decodeResult(wire.Plan)
		if err != nil {
			return writeFailure(output, fmt.Errorf("decode adapter plan: %w", err))
		}
	}
	return writeResult(output, dotnet.Run(request))
}

func decodeResult(value *host.AdapterResult) (*dotnet.Result, error) {
	data, err := host.YAML(value)
	if err != nil {
		return nil, err
	}
	var result dotnet.Result
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func readMappedInputs(paths []string) (map[string][]byte, map[string]string) {
	data := make(map[string][]byte, len(paths))
	errorsByPath := make(map[string]string)
	for _, path := range paths {
		value, err := readStagedFile(path, maxCaptureBytes)
		if err != nil {
			errorsByPath[path] = err.Error()
			continue
		}
		data[path] = value
	}
	return data, errorsByPath
}

func readStagedFile(relative string, limit int64) ([]byte, error) {
	if relative == "" || strings.ContainsAny(relative, `\:`) || filepath.IsAbs(relative) {
		return nil, errors.New("projectFile must be a clean staged-input-relative path")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, errors.New("projectFile must stay within staged inputs")
	}
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	current := root
	parts := strings.Split(clean, string(filepath.Separator))
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) {
			return nil, errors.New("staged project input contains a link or reparse point")
		}
		if i < len(parts)-1 && !info.IsDir() {
			return nil, errors.New("staged project input parent is not a directory")
		}
		if i == len(parts)-1 && !info.Mode().IsRegular() {
			return nil, errors.New("project input must be a regular file")
		}
	}
	abs, err := filepath.Abs(current)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, errors.New("project input resolves outside staged inputs")
	}
	before, err := os.Lstat(abs)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, errors.New("project input changed while opening")
	}
	if opened.Size() < 0 || opened.Size() > limit {
		return nil, fmt.Errorf("project input exceeds %d bytes", limit)
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit || int64(len(data)) != opened.Size() {
		return nil, errors.New("project input changed or exceeds size limit")
	}
	return data, nil
}

func isReparsePoint(info os.FileInfo) bool {
	return isPlatformReparsePoint(info)
}

func writeResult(output io.Writer, value any) int {
	encoder := yaml.NewEncoder(output)
	encoder.SetIndent(2)
	if err := encoder.Encode(value); err != nil {
		return 2
	}
	if err := encoder.Close(); err != nil {
		return 2
	}
	return 0
}

func writeFailure(output io.Writer, err error) int {
	_ = writeResult(output, dotnet.InvalidRequest(err.Error()))
	return 2
}
