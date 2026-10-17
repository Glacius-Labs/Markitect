package apply

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"reflect"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const maxProfileMetadataFiles = 100000

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
	if o.Status == model.Passed && (len(o.Files) == 0 || p.input.namespace == "agentProfiles" && len(o.Inventories) == 0) {
		p.problem(model.Failed, "metadata_missing", "Passed requires executable metadata and agent schema inventories.")
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
	text("markitect.host-profile/static-v1")
	text(p.input.Binding().content.String())
	source(p.input.document)
	text(p.input.namespace)
	text(p.input.name)
	text(p.input.field)
	text(p.observation.Contract.Name)
	text(fmt.Sprint(p.observation.Contract.Version))
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
