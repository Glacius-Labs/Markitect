package apply

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"path"
	"reflect"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const maxProfileMetadataFiles = 100000

const (
	maxAgentImageMetadataBytes = 1 << 20
	maxAgentImageLiteralBytes  = 4096
)

func hostPortMissing(port any) bool {
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

func observeHostProfile(ctx context.Context, input ProfileInput, agents AgentProfilePreparer, checks CheckProfilePreparer) (PreparedProfile, error) {
	p := PreparedProfile{input: input}
	var err error
	if err = ctx.Err(); err != nil {
		p.observation.Status = model.NotCompleted
	} else if input.namespace == "agentProfiles" && !hostPortMissing(agents) {
		p.observation, err = agents.PrepareAgentProfile(ctx, p.Input())
	} else if input.namespace == "checkProfiles" && !hostPortMissing(checks) {
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
	p.observation.Diagnostics = append(p.observation.Diagnostics, HostProfileDiagnostic{Code: "host." + code, Source: p.Source(), Message: message})
}

func (p *PreparedProfile) validateObservation() {
	o := &p.observation
	if o.InputBinding != (ProfileBinding{}) && o.InputBinding != p.input.Binding() || o.Status == model.Passed && o.InputBinding == (ProfileBinding{}) {
		p.problem(model.Failed, "input_binding", "Metadata observation does not echo the exact supplied profile request binding.")
	}
	if o.Status == model.Passed && (strings.TrimSpace(o.Contract.Name) == "" || o.Contract.Version == 0) {
		p.problem(model.Failed, "contract", "Passed requires a nonempty literal adapter contract name and an explicit positive version.")
	}
	if o.AgentImage != nil {
		p.validateAgentImage()
	}
	if o.CheckImage != nil {
		p.validateCheckImage()
	}
	image := o.AgentImage != nil || o.CheckImage != nil
	if o.Status == model.Passed && !image && (len(o.Files) == 0 || p.input.namespace == "agentProfiles" && len(o.Inventories) == 0) {
		p.problem(model.Failed, "metadata_missing", "Passed requires executable metadata and agent schema inventories.")
	}
	if o.Status == model.Passed && !image && (p.input.section == "claudeCode" || p.input.section == "container") {
		p.problem(model.Failed, "image_missing", "A claudeCode or container profile is bound to its image, not to Host files.")
	}
	count := len(o.Files)
	for _, inventory := range o.Inventories {
		count += len(inventory.Files)
	}
	if count > maxProfileMetadataFiles || len(o.Inventories) > maxHostProfiles {
		p.problem(model.NotCompleted, "metadata_limit", "Metadata exceeds the finite preparation inventory limit.")
		return
	}
	seen := make(map[SourceLocation]bool)
	validateFile := func(file ProfileMetadataFile) {
		if file.Location.Boundary == "" || file.Location.Path.String() == "" || file.ContentID == (snapshot.ContentID{}) || file.Size < 0 || file.Mode != snapshot.Regular && file.Mode != snapshot.Executable {
			p.problem(model.Failed, "metadata_file", "Metadata requires an exact location, content digest, nonnegative size and regular file mode.")
		}
		if seen[file.Location] {
			p.problem(model.Failed, "metadata_duplicate", "Duplicate external metadata location.")
		}
		seen[file.Location] = true
	}
	less := func(a, b ProfileMetadataFile) int {
		return cmp.Or(cmp.Compare(a.Location.Boundary, b.Location.Boundary), cmp.Compare(a.Location.Path.String(), b.Location.Path.String()))
	}
	for _, file := range o.Files {
		validateFile(file)
	}
	slices.SortFunc(o.Files, less)
	boundaries := make(map[string]bool)
	for i := range o.Inventories {
		inventory := &o.Inventories[i]
		if inventory.Boundary == "" || boundaries[inventory.Boundary] {
			p.problem(model.Failed, "inventory_boundary", "Schema inventory boundaries must be explicit and unique.")
		}
		boundaries[inventory.Boundary] = true
		if o.Status == model.Passed && len(inventory.Files) == 0 {
			p.problem(model.Failed, "inventory_empty", "A passed schema inventory cannot be empty.")
		}
		for _, file := range inventory.Files {
			if file.Location.Boundary != inventory.Boundary {
				p.problem(model.Failed, "inventory_location", "Inventory file must belong to its declared schema boundary.")
			}
			validateFile(file)
		}
		slices.SortFunc(inventory.Files, less)
	}
	slices.SortFunc(o.Inventories, func(a, b ProfileMetadataInventory) int { return cmp.Compare(a.Boundary, b.Boundary) })
	for _, location := range o.Instructions {
		if _, ok := p.instruction(location); !ok {
			o.RequiredInstructions = append(o.RequiredInstructions, location)
			p.problem(model.NotCompleted, "instruction_uncaptured", "Profile instruction requires a new complete source acquisition.")
		}
	}
	for _, location := range o.RequiredInstructions {
		if location.Boundary == "" || location.Path.String() == "" {
			p.problem(model.Failed, "instruction_location", "Required instruction must name an explicit boundary and path.")
		}
		p.problem(model.NotCompleted, "instruction_required", "Host instruction is required in a new complete acquisition; existing inputs are unchanged.")
	}
}

func (p *PreparedProfile) validateAgentImage() {
	o := &p.observation
	image := o.AgentImage
	complete := o.Status == model.Passed
	if p.input.namespace != "agentProfiles" || len(o.Files) != 0 || len(o.Inventories) != 0 {
		p.problem(model.Failed, "image_evidence", "Agent image evidence is agent-only and cannot mix with file/inventory evidence.")
	}
	if complete || image.Provider != "" {
		if image.Provider != AgentImageCodexAppServer && image.Provider != AgentImageClaudeCode {
			p.problem(model.Failed, "image_provider", "Unknown agent image provider discriminator.")
		}
		if want := map[string]AgentImageProvider{"codexAppServer": AgentImageCodexAppServer, "claudeCode": AgentImageClaudeCode}[p.input.section]; image.Provider != want {
			p.problem(model.Failed, "image_provider", "The agent image provider does not match the profile's adapter section.")
		}
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
	switch image.Provider {
	case AgentImageCodexAppServer:
		if (complete || image.RecordedCodexSchemaDigest != "") && !profileSHA256(image.RecordedCodexSchemaDigest) {
			p.problem(model.Failed, "image_recorded_schema", "Codex image evidence requires the literal build-recorded sha256 schema digest, not an inventory assertion.")
		}
	case AgentImageClaudeCode:
		if image.RecordedCodexSchemaDigest != "" {
			p.problem(model.Failed, "image_recorded_schema", "Claude image evidence cannot claim Codex-specific schema evidence.")
		}
	}
}

func (p *PreparedProfile) validateCheckImage() {
	o := &p.observation
	image := o.CheckImage
	complete := o.Status == model.Passed
	if p.input.namespace != "checkProfiles" || p.input.section != "container" || o.AgentImage != nil || len(o.Files) != 0 || len(o.Inventories) != 0 {
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

func (p PreparedProfile) instruction(location SourceLocation) (SourceFile, bool) {
	var found SourceFile
	ok := false
	for _, file := range p.input.instructions {
		if file.Location != location {
			continue
		}
		if ok && found != file {
			return SourceFile{}, false
		}
		found, ok = file, true
	}
	return found, ok
}

func hostHashText(h hash.Hash, text string) {
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
	text := func(value string) { hostHashText(h, value) }
	source := func(value SourceFile) {
		text(value.Location.Boundary)
		text(value.Location.Path.String())
		text(value.RepositoryPath.String())
		text(value.File.ContentID().String())
		text(fmt.Sprint(value.File.Mode()))
	}
	text("markitect.host-profile/request-v1")
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
	text := func(value string) { hostHashText(h, value) }
	location := func(value SourceLocation) { text(value.Boundary); text(value.Path.String()) }
	metadata := func(value ProfileMetadataFile) {
		location(value.Location)
		text(value.ContentID.String())
		text(fmt.Sprint(value.Size))
		text(fmt.Sprint(value.Mode))
	}
	source := func(value SourceFile) {
		location(value.Location)
		text(value.RepositoryPath.String())
		text(value.File.ContentID().String())
		text(fmt.Sprint(value.File.Mode()))
	}
	switch {
	case p.observation.CheckImage != nil:
		text("markitect.host-profile/static-check-image-v1")
	case p.observation.AgentImage == nil:
		// Historical file/inventory identities retain their exact hash sequence.
		text("markitect.host-profile/static-v1")
	default:
		text("markitect.host-profile/static-image-v1")
	}
	text(p.input.Binding().content.String())
	source(p.input.document)
	text(p.input.namespace)
	text(p.input.name)
	text(p.input.field)
	text(p.observation.Contract.Name)
	text(fmt.Sprint(p.observation.Contract.Version))
	if image := p.observation.AgentImage; image != nil {
		text(string(image.Provider))
		text(image.ImageID)
		text(image.Metadata.Path)
		text(image.Metadata.ContentID.String())
		text(fmt.Sprint(image.Metadata.Size))
		text(image.CLIVersion)
		text(image.AgentdVersion)
		text(image.RecordedCodexSchemaDigest)
	}
	if image := p.observation.CheckImage; image != nil {
		text(image.ImageID)
		text(image.Executable.Path)
		text(image.Executable.ContentID.String())
		text(fmt.Sprint(image.Executable.Size))
	}
	text(fmt.Sprint(len(p.observation.Files)))
	for _, file := range p.observation.Files {
		metadata(file)
	}
	text(fmt.Sprint(len(p.observation.Inventories)))
	for _, inventory := range p.observation.Inventories {
		text(inventory.Boundary)
		text(fmt.Sprint(len(inventory.Files)))
		for _, file := range inventory.Files {
			metadata(file)
		}
	}
	text(fmt.Sprint(len(p.observation.Instructions)))
	for _, ref := range p.observation.Instructions {
		file, _ := p.instruction(ref)
		source(file)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}
