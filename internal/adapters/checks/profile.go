// Package checks observes pinned executable metadata for project Check profiles.
// It does not execute checks or prepare a process/runtime capability.
package checks

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

// MetadataPreparer implements only Apply's static Check profile metadata port.
// Its zero value is usable. Process/v1 accepts exactly profileVersion: 1,
// executable: <literal absolute native path>, sha256: <64 hexadecimal digits>.
// Observation is bounded to 512 MiB per file and 100000 entries per directory.
// Neither executable format, executable permission, nor runtime readiness is
// inferred from a successful regular-file observation and SHA-256 pin match.
type MetadataPreparer struct {
	// Private deterministic observation seam; production uses the zero value.
	observe func(stage string)
}

var _ apply.CheckProfilePreparer = MetadataPreparer{}

var (
	ErrUnsafeExecutable  = errors.New("check executable is redirected, nonregular or ambiguously spelled")
	ErrExecutableChanged = errors.New("check executable changed during metadata observation")
	ErrMetadataLimit     = errors.New("check executable metadata limit exceeded")
)

func (p MetadataPreparer) PrepareCheckProfile(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
	o := apply.ProfileObservation{InputBinding: input.Binding(), Status: model.NotCompleted, Contract: apply.ProfileContract{Name: "markitect.process-check", Version: 1}}
	issue := func(code, field, message string, node *yaml.Node) {
		location := input.Location()
		locator, _ := json.Marshal([2]string{location.Boundary, location.Path.String()})
		source := organization.SettingSource{File: "source:" + string(locator), ContentID: input.Document().File.ContentID(), Field: input.Field()}
		if field != "" {
			source.Field += "/" + field
		}
		if location.Boundary == "" || location.Path.String() == "" {
			source = organization.SettingSource{}
		}
		var position *model.Position
		if node != nil && node.Line > 0 {
			position = &model.Position{Line: node.Line, Column: node.Column}
		}
		o.Diagnostics = append(o.Diagnostics, apply.HostProfileDiagnostic{Code: "process." + code, Source: source, Position: position, Message: message})
	}
	if err := ctx.Err(); err != nil {
		issue("cancelled", "", err.Error(), nil)
		return o, err
	}
	inline := input.Inline()
	if input.Namespace() != "checkProfiles" || inline == nil || inline.Kind != yaml.MappingNode {
		o.Status = model.Failed
		issue("input", "", "Expected an Apply-selected process Check profile mapping.", inline)
		return o, nil
	}
	fields := make(map[string]*yaml.Node)
	for i := 0; i+1 < len(inline.Content); i += 2 {
		key, value := inline.Content[i], inline.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || fields[key.Value] != nil {
			o.Status = model.Failed
			issue("field", "", "Process profile keys must be unique literal strings.", key)
			continue
		}
		fields[key.Value] = value
		switch key.Value {
		case "profileVersion", "executable", "sha256":
		default:
			o.Status = model.Failed
			issue("unknown_field", "", fmt.Sprintf("Unknown process/v1 field %q.", key.Value), key)
		}
	}
	for _, name := range []string{"profileVersion", "executable", "sha256"} {
		if fields[name] == nil {
			o.Status = model.Failed
			issue("field_missing", name, "Required process/v1 field is missing.", nil)
		}
	}
	version := fields["profileVersion"]
	var decoded uint32
	if version != nil && (version.Kind != yaml.ScalarNode || version.Tag != "!!int" || version.Decode(&decoded) != nil || decoded != 1) {
		o.Status = model.Failed
		issue("version", "profileVersion", "Only the integer process profileVersion 1 is supported.", version)
	}
	literal := func(name string) string {
		n := fields[name]
		if n == nil {
			return ""
		}
		if n.Kind != yaml.ScalarNode || n.Tag != "!!str" || n.Value == "" {
			o.Status = model.Failed
			issue("string", name, "Expected a nonempty literal string.", n)
			return ""
		}
		return n.Value
	}
	executable, pinText := literal("executable"), literal("sha256")
	pin, pinErr := hex.DecodeString(pinText)
	if fields["sha256"] != nil && (pinErr != nil || len(pinText) != 64 || len(pin) != 32) {
		o.Status = model.Failed
		issue("pin", "sha256", "Expected an explicit 64 hexadecimal digit SHA-256 pin.", fields["sha256"])
	}
	if executable != "" && (!filepath.IsAbs(executable) || filepath.Clean(executable) != executable) {
		o.Status = model.Failed
		issue("path", "executable", "Executable must be a literal absolute native file path without normalization.", fields["executable"])
	}
	if o.Status == model.Failed {
		return o, nil
	}
	file, err := observeExecutable(ctx, executable, p.observe)
	if file.ContentID != (snapshot.ContentID{}) {
		o.Files = []apply.ProfileMetadataFile{file}
	}
	if err != nil {
		if errors.Is(err, ErrUnsafeExecutable) {
			o.Status = model.Failed
		}
		issue("observation", "executable", err.Error(), fields["executable"])
		return o, err
	}
	if file.ContentID != snapshot.ContentID(pin) {
		o.Status = model.Failed
		issue("pin_mismatch", "sha256", "Observed executable SHA-256 differs from its explicit pin.", fields["sha256"])
		return o, nil
	}
	o.Status = model.Passed
	return o, nil
}
