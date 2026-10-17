// Package claudecode prepares static metadata from an explicitly selected
// runtime image. Preparation never starts that image, a Session or a provider.
package claudecode

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const (
	runtimeMetadataPath    = "/opt/markitect/runtime.env"
	maxMetadataBytes       = 1 << 20
	supportedClaudeVersion = "2.1.301 (Claude Code)"
)

// MetadataPreparer requires an explicitly composed image-reader factory. Its
// zero value reports incomplete acquisition; it has no ambient engine fallback.
type MetadataPreparer struct{ factory ImageReaderFactory }

func NewMetadataPreparer(factory ImageReaderFactory) *MetadataPreparer {
	return &MetadataPreparer{factory: factory}
}

var _ apply.AgentProfilePreparer = (*MetadataPreparer)(nil)

type preparation struct {
	input apply.ProfileInput
	out   apply.ProfileObservation
	err   error
}

func (p *preparation) issue(status model.Status, code, field, message string) {
	if status == model.Failed || p.out.Status != model.Failed {
		p.out.Status = status
	}
	doc := p.input.Document()
	locator, _ := json.Marshal([2]string{doc.Location.Boundary, doc.Location.Path.String()})
	p.out.Diagnostics = append(p.out.Diagnostics, apply.HostProfileDiagnostic{
		Code:    "claude_image." + code,
		Source:  organization.SettingSource{File: "source:" + string(locator), ContentID: doc.File.ContentID(), Field: p.input.Field() + field},
		Message: message,
	})
}

func (p *preparation) acquisition(err error) {
	if err != nil {
		p.err = errors.Join(p.err, err)
		p.issue(model.NotCompleted, "acquisition", "/imageID", err.Error())
	}
}

func (m *MetadataPreparer) PrepareAgentProfile(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
	p := preparation{input: input, out: apply.ProfileObservation{
		InputBinding: input.Binding(), Status: model.Passed,
		Contract: apply.ProfileContract{Name: "markitect.claude-code.image-metadata", Version: 1},
	}}
	if ctx == nil {
		p.acquisition(errors.New("nil metadata preparation context"))
		return p.out, p.err
	}
	if err := ctx.Err(); err != nil {
		p.acquisition(err)
		return p.out, p.err
	}
	if input.Namespace() != "agentProfiles" || input.Section() != "claudeCode" || input.Binding() == (apply.ProfileBinding{}) {
		p.issue(model.Failed, "input", "", "A bound Agent profile input is required.")
		return p.out, nil
	}
	config, ok := p.decode(input.Inline())
	if !ok {
		return p.out, nil
	}
	for _, location := range config.instructions {
		found := false
		for _, file := range input.Instructions() {
			if file.Location == location {
				found = true
				break
			}
		}
		if found {
			p.out.Instructions = append(p.out.Instructions, location)
		} else {
			p.out.RequiredInstructions = append(p.out.RequiredInstructions, location)
			p.issue(model.NotCompleted, "instruction_uncaptured", "/instructions", "Selected instruction requires a new complete acquisition; no instruction file was read.")
		}
	}
	if m == nil || m.factory == nil {
		p.acquisition(errors.New("an explicit image-reader factory is required"))
		return p.out, p.err
	}
	reader, err := m.factory(EngineConfig{Executable: config.engine})
	if err != nil {
		p.acquisition(err)
		return p.out, p.err
	}
	if reader == nil {
		p.acquisition(errors.New("image-reader factory returned no reader"))
		return p.out, p.err
	}
	if err := ctx.Err(); err != nil {
		p.acquisition(err)
		return p.out, p.err
	}
	imageID, err := reader.ImageID(ctx, config.imageID)
	if err != nil {
		p.acquisition(err)
		return p.out, p.err
	}
	if !sha256Literal(imageID) {
		p.issue(model.Failed, "image_identity", "/imageID", "Engine did not report an immutable sha256 image ID.")
		return p.out, nil
	}
	p.out.AgentImage = &apply.AgentImageObservation{Provider: apply.AgentImageClaudeCode, ImageID: imageID}
	if imageID != config.imageID {
		p.issue(model.Failed, "image_mismatch", "/imageID", "Observed engine image ID differs from the selected immutable image ID.")
		return p.out, nil
	}
	if err := ctx.Err(); err != nil {
		p.acquisition(err)
		return p.out, p.err
	}
	data, err := reader.ReadImageFile(ctx, imageID, runtimeMetadataPath)
	if err != nil {
		p.acquisition(err)
		return p.out, p.err
	}
	image := p.out.AgentImage
	image.Metadata = apply.AgentImageMetadataFile{Path: runtimeMetadataPath, ContentID: snapshot.ContentID(sha256.Sum256(data)), Size: int64(len(data))}
	if err := ctx.Err(); err != nil {
		p.acquisition(err)
		return p.out, p.err
	}
	if len(data) > maxMetadataBytes {
		p.issue(model.NotCompleted, "metadata_limit", "/imageID", "Image metadata exceeds the 1 MiB observation limit.")
		return p.out, nil
	}
	p.parseMetadata(data, config)
	if err := ctx.Err(); err != nil {
		p.acquisition(err)
		return p.out, p.err
	}
	observed, err := reader.ImageID(ctx, config.imageID)
	if err != nil {
		p.acquisition(err)
	} else if observed != imageID {
		p.issue(model.Failed, "image_changed", "/imageID", "Image identity changed across metadata acquisition.")
	}
	p.acquisition(ctx.Err())
	return p.out, p.err
}

func (p *preparation) parseMetadata(data []byte, config metadataConfig) {
	values, err := runtimeMetadata(data)
	if err != nil {
		p.issue(model.Failed, "metadata_format", "/imageID", err.Error())
		return
	}
	image := p.out.AgentImage
	image.CLIVersion, image.AgentdVersion = values["claude"], values["agentd"]
	if image.CLIVersion != supportedClaudeVersion {
		p.issue(model.Failed, "cli_version", "/imageID", "Image must record exactly "+supportedClaudeVersion+".")
	}
	if image.AgentdVersion != config.agentd {
		p.issue(model.Failed, "agentd_version", "/agentdVersion", "Recorded agentd build differs from the selected literal version.")
	}

}

func runtimeMetadata(data []byte) (map[string]string, error) {
	// This is the four-key LF format emitted by packaging/runtime/Dockerfile,
	// not a shell environment to source or a generated schema inventory.
	values := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || values[key] != "" || (key != "codex" && key != "claude" && key != "codexSchema" && key != "agentd") || !literal(value) {
			return nil, errors.New("runtime.env requires exactly four unique codex, claude, codexSchema and agentd literal records")
		}
		values[key] = value
	}
	if len(values) != 4 {
		return nil, errors.New("runtime.env is missing a required metadata record")
	}
	return values, nil
}
