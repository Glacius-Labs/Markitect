package runtimeconfig

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"hash"
	"path"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const (
	maxAgentImageMetadataBytes = 1 << 20
	maxAgentImageLiteralBytes  = 4096
)

func runtimePortMissing(port any) bool {
	if port == nil {
		return true
	}
	v := reflect.ValueOf(port)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}

func observeRuntimeProfile(ctx context.Context, input ProfileInput, agents AgentProfilePreparer, checks CheckProfilePreparer) (PreparedProfile, error) {
	p := PreparedProfile{input: input}
	var err error
	if err = ctx.Err(); err != nil {
		p.observation.Status = model.NotCompleted
	} else if input.namespace == "agentProfiles" && !runtimePortMissing(agents) {
		p.observation, err = agents.PrepareAgentProfile(ctx, p.Input())
	} else if input.namespace == "checkProfiles" && !runtimePortMissing(checks) {
		p.observation, err = checks.PrepareCheckProfile(ctx, p.Input())
	} else {
		p.observation.Status = model.NotCompleted
		p.problem(model.NotCompleted, "adapter_unavailable", "A concrete metadata preparer is required for this profile.")
	}
	p.observation = cloneProfileObservation(p.observation)
	if p.observation.Status == "" {
		p.observation.Status = model.NotCompleted
	}
	if p.observation.Status != model.Passed && p.observation.Status != model.Failed && p.observation.Status != model.NotCompleted {
		p.problem(model.Failed, "adapter_status", "Metadata preparer returned an unknown static status.")
	}
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		p.problem(model.NotCompleted, "adapter_error", err.Error())
	}
	p.validateObservation()
	if p.observation.Status == model.Passed {
		p.identity = p.comparisonIdentity()
	}
	return p, err
}

func (p *PreparedProfile) problem(status model.Status, code, message string) {
	if status == model.Failed || p.observation.Status != model.Failed {
		p.observation.Status = status
	}
	p.observation.Diagnostics = append(p.observation.Diagnostics, RuntimeProfileDiagnostic{Code: "runtime." + code, Source: p.Source(), Message: message})
}

func (p *PreparedProfile) validateObservation() {
	o := &p.observation
	if o.InputBinding != (ProfileBinding{}) && o.InputBinding != p.input.Binding() || o.Status == model.Passed && o.InputBinding == (ProfileBinding{}) {
		p.problem(model.Failed, "input_binding", "Metadata must echo the exact acquired request binding.")
	}
	if o.Status == model.Passed && (!agentImageLiteral(o.Contract.Name) || o.Contract.Version == 0) {
		p.problem(model.Failed, "contract", "Passed requires an explicit named versioned adapter contract.")
	}
	if o.AgentImage != nil {
		p.validateAgentImage()
	}
	if o.CheckImage != nil {
		p.validateCheckImage()
	}
	if o.Status == model.Passed && ((p.input.namespace == "agentProfiles" && (o.AgentImage == nil || o.CheckImage != nil)) || (p.input.namespace == "checkProfiles" && (o.CheckImage == nil || o.AgentImage != nil))) {
		p.problem(model.Failed, "image_missing", "Passed requires exactly the selected native or check image observation.")
	}
	if len(o.Instructions)+len(o.RequiredInstructions) > 10000 {
		p.problem(model.NotCompleted, "instruction_limit", "Instruction observation exceeds finite count bound.")
		return
	}
	for _, ref := range o.Instructions {
		if ref.Boundary == "" || ref.Path.String() == "" {
			p.problem(model.Failed, "instruction_location", "Instruction coordinates must be explicit.")
			continue
		}
		if _, ok := p.instruction(ref); !ok {
			p.problem(model.NotCompleted, "instruction_uncaptured", "Selected instructions require a new complete acquisition.")
		}
	}
	for _, ref := range o.RequiredInstructions {
		if ref.Boundary == "" || ref.Path.String() == "" {
			p.problem(model.Failed, "instruction_location", "Required instruction coordinates must be explicit.")
		} else {
			p.problem(model.NotCompleted, "instruction_required", "A complete new instruction acquisition is required.")
		}
	}
}

func (p *PreparedProfile) validateAgentImage() {
	o := &p.observation
	image := o.AgentImage
	complete := o.Status == model.Passed
	if p.input.namespace != "agentProfiles" || o.CheckImage != nil {
		p.problem(model.Failed, "image_evidence", "Agent image evidence is agent-only and cannot mix with file/inventory evidence.")
	}
	// Each Agent section has exactly one image provider.
	if (complete || image.Provider != "") && image.Provider != agentImageProviders[p.input.section] {
		p.problem(model.Failed, "image_provider", "Agent image provider differs from the selected Agent section.")
	}
	if complete || image.ImageID != "" {
		if !profileSHA256(image.ImageID) {
			p.problem(model.Failed, "image_identity", "Agent image must name an immutable engine sha256 image ID.")
		}
	}
	file := image.Metadata
	if complete || file.Path != "" {
		if !agentImageLiteral(file.Path) || !strings.HasPrefix(file.Path, "/") || file.Path == "/" || strings.Contains(file.Path, "\\") || path.Clean(file.Path) != file.Path {
			p.problem(model.Failed, "image_metadata_path", "Image metadata requires an exact canonical absolute POSIX file path.")
		}
	}
	if complete && file.ContentID == (snapshot.ContentID{}) || file.Size < 0 || complete && file.Size == 0 {
		p.problem(model.Failed, "image_metadata_file", "Image metadata requires its actual whole-file digest and positive size.")
	}
	if file.Size > maxAgentImageMetadataBytes {
		p.problem(model.NotCompleted, "image_metadata_limit", "Image metadata exceeds the finite 1 MiB observation limit.")
	}
	for _, literal := range []string{image.CLIVersion, image.AgentdVersion} {
		if (complete || literal != "") && !agentImageLiteral(literal) {
			p.problem(model.Failed, "image_version_literal", "Recorded CLI and agentd versions must be bounded nonblank literals without control characters.")
		}
	}
	if complete && !agentImageLiteral(o.Contract.Name) {
		p.problem(model.Failed, "image_contract", "Image validation contract name must be a bounded nonblank literal without controls.")
	}
	codex := image.Provider == AgentImageCodexAppServer
	if codex && (complete || image.RecordedCodexSchemaDigest != "") && !profileSHA256(image.RecordedCodexSchemaDigest) {
		p.problem(model.Failed, "image_recorded_schema", "Codex image evidence requires the literal build-recorded sha256 schema digest, not an inventory assertion.")
	}
	if !codex && image.RecordedCodexSchemaDigest != "" {
		p.problem(model.Failed, "image_recorded_schema", "Only Codex image evidence carries a recorded Codex schema digest.")
	}
}

func (p *PreparedProfile) validateCheckImage() {
	o := &p.observation
	image := o.CheckImage
	complete := o.Status == model.Passed
	if p.input.namespace != "checkProfiles" || p.input.section != "container" || o.AgentImage != nil {
		p.problem(model.Failed, "image_evidence", "Check image evidence belongs to a container check and cannot mix with other evidence.")
	}
	if (complete || image.ImageID != "") && !profileSHA256(image.ImageID) {
		p.problem(model.Failed, "image_identity", "A container check must name an immutable engine sha256 image ID.")
	}
	file := image.Executable
	if complete || file.Path != "" {
		if !agentImageLiteral(file.Path) || !strings.HasPrefix(file.Path, "/") || file.Path == "/" || strings.Contains(file.Path, "\\") || path.Clean(file.Path) != file.Path {
			p.problem(model.Failed, "image_executable_path", "The check executable requires an exact canonical absolute POSIX path inside the image.")
		}
	}
	if complete && (file.ContentID == (snapshot.ContentID{}) || file.Size <= 0) || file.Size < 0 {
		p.problem(model.Failed, "image_executable_file", "The check executable requires its actual whole-file digest and positive size.")
	}
}

func agentImageLiteral(value string) bool {
	if len(value) > maxAgentImageLiteralBytes || !utf8.ValidString(value) || strings.TrimSpace(value) == "" {
		return false
	}
	return !strings.ContainsFunc(value, unicode.IsControl)
}

func profileSHA256(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, c := range value[len("sha256:"):] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func (p PreparedProfile) instruction(location project.SourceLocation) (project.SourceFile, bool) {
	var found project.SourceFile
	ok := false
	for _, file := range p.input.instructions {
		if file.Location != location {
			continue
		}
		if ok && found != file {
			return project.SourceFile{}, false
		}
		found, ok = file, true
	}
	return found, ok
}

func runtimeHashText(h hash.Hash, text string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(text)))
	h.Write(length[:])
	h.Write([]byte(text))
}

// Binding includes the complete captured instruction view supplied to the port,
// not just a caller-chosen name or digest. Inline is identified by the original
// whole-file content plus its exact selected field; it is never reserialized.
func (p ProfileInput) Binding() ProfileBinding {
	if p.document.Location.Boundary == "" || p.document.Location.Path.String() == "" || p.inline == nil {
		return ProfileBinding{}
	}
	h := sha256.New()
	text := func(value string) { runtimeHashText(h, value) }
	source := func(value project.SourceFile) {
		text(value.Location.Boundary)
		text(value.Location.Path.String())
		text(value.RepositoryPath.String())
		text(value.File.ContentID().String())
		text(fmt.Sprint(value.File.Mode()))
	}
	text("markitect.runtime-profile/request-v1")
	source(p.document)
	text(p.namespace)
	text(p.name)
	text(p.field)
	text(fmt.Sprint(len(p.instructions)))
	for _, instruction := range p.instructions {
		source(instruction)
	}
	var content snapshot.ContentID
	copy(content[:], h.Sum(nil))
	return ProfileBinding{content: content}
}

func (p PreparedProfile) comparisonIdentity() string {
	h := sha256.New()
	runtimeHashText(h, "markitect.runtime-profile/static-native-v2")
	runtimeHashText(h, p.input.Binding().content.String())
	o := p.observation
	data, _ := json.Marshal(struct {
		Contract ProfileContract
		Agent    *AgentImageObservation
		Check    *CheckImageObservation
	}{o.Contract, o.AgentImage, o.CheckImage})
	h.Write(data)
	for _, ref := range o.Instructions {
		file, _ := p.instruction(ref)
		runtimeHashText(h, ref.Boundary)
		runtimeHashText(h, ref.Path.String())
		runtimeHashText(h, file.File.ContentID().String())
		runtimeHashText(h, fmt.Sprint(file.File.Mode()))
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}
