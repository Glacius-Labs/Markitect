package checks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// EngineConfig is the engine executable selected by the captured Host section.
type EngineConfig struct{ Executable string }

// ImageReader observes an immutable image without starting it. The file reader
// must prove real directory ancestors and the required path types in the same
// never-started container, hash the complete regular file within maxBytes, and
// report failed path observations, cancellation and cleanup errors.
type ImageReader interface {
	ImageID(context.Context, string) (string, error)
	ObserveImageFileWithRequiredPaths(context.Context, string, string, int64, []string, []string) (container.ImageFileObservation, error)
}

type ImageReaderFactory func(EngineConfig) (ImageReader, error)

// NewEngineImageReader binds only the explicitly supplied engine executable.
func NewEngineImageReader(config EngineConfig) (ImageReader, error) {
	return container.NewEngine(config.Executable)
}

// NewMetadataPreparer composes container image observation. The zero value still
// prepares process/v1; container/v1 requires this explicit factory.
func NewMetadataPreparer(factory ImageReaderFactory) *MetadataPreparer {
	return &MetadataPreparer{factory: factory}
}

var containerContract = apply.ProfileContract{Name: "markitect.container-check", Version: 1}

type containerPreparation struct {
	input apply.ProfileInput
	out   apply.ProfileObservation
}

func (p *containerPreparation) issue(status model.Status, code, field, message string) {
	if status == model.Failed || p.out.Status != model.Failed {
		p.out.Status = status
	}
	doc := p.input.Document()
	locator, _ := json.Marshal([2]string{doc.Location.Boundary, doc.Location.Path.String()})
	p.out.Diagnostics = append(p.out.Diagnostics, apply.HostProfileDiagnostic{
		Code:    "container_check." + code,
		Source:  organization.SettingSource{File: "source:" + string(locator), ContentID: doc.File.ContentID(), Field: p.input.Field() + field},
		Message: message,
	})
}

func (m MetadataPreparer) prepareContainer(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
	p := containerPreparation{input: input, out: apply.ProfileObservation{
		InputBinding: input.Binding(), Status: model.Passed, Contract: containerContract,
	}}
	incomplete := func(err error) (apply.ProfileObservation, error) {
		p.issue(model.NotCompleted, "acquisition", "/executable", err.Error())
		return p.out, err
	}
	if ctx == nil {
		return incomplete(errors.New("nil container check preparation context"))
	}
	if err := ctx.Err(); err != nil {
		return incomplete(err)
	}
	if input.Namespace() != "checkProfiles" || input.Section() != "container" || input.Binding() == (apply.ProfileBinding{}) {
		p.issue(model.Failed, "input", "", "An exact bound container Check profile section is required.")
		return p.out, nil
	}
	config, ok := p.decode()
	if !ok {
		return p.out, nil
	}
	if m.factory == nil {
		return incomplete(errors.New("an explicit container image-reader factory is required"))
	}
	reader, err := m.factory(EngineConfig{Executable: config.engine})
	if err != nil {
		return incomplete(err)
	}
	if reader == nil {
		return incomplete(errors.New("container image-reader factory returned no reader"))
	}
	if err := ctx.Err(); err != nil {
		return incomplete(err)
	}
	imageID, err := reader.ImageID(ctx, config.profile.ImageID)
	if err != nil {
		return incomplete(err)
	}
	if imageID != config.profile.ImageID {
		p.issue(model.Failed, "image_mismatch", "/imageID", "Observed immutable image ID differs from captured configuration.")
		return p.out, nil
	}
	if err := ctx.Err(); err != nil {
		return incomplete(err)
	}
	// Fixed runtime mounts must not redirect onto the executable's image tree.
	// Fresh lists prevent a reader from changing another preparation's inputs.
	file, err := reader.ObserveImageFileWithRequiredPaths(ctx, imageID, config.profile.Executable, container.MaxImageFileObservationBytes,
		[]string{"/work", "/home", "/home/agent", "/tmp", "/dev", "/proc", "/sys", "/etc"},
		[]string{"/etc/hosts", "/etc/hostname", "/etc/resolv.conf"})
	if err != nil {
		return incomplete(err)
	}
	if err := ctx.Err(); err != nil {
		return incomplete(err)
	}
	if file.SHA256 == ([32]byte{}) || file.Size <= 0 || file.Size > container.MaxImageFileObservationBytes {
		p.issue(model.Failed, "executable_evidence", "/executable", "Image reader must observe a complete nonempty regular file within 512 MiB.")
		return p.out, nil
	}
	p.out.CheckImage = &apply.CheckImageObservation{ImageID: imageID, Executable: apply.AgentImageMetadataFile{
		Path: config.profile.Executable, ContentID: snapshot.ContentID(file.SHA256), Size: file.Size,
	}}
	return p.out, nil
}

// ContainerProfileFromPrepared projects complete container/v1 evidence and its
// captured configuration. It performs no acquisition or execution. Request then
// combines these values with the selected declaration and actual immutable tree.
func ContainerProfileFromPrepared(prepared apply.PreparedProfile) (ContainerProfile, error) {
	input, observation := prepared.Input(), prepared.Observation()
	refuse := func(reason string) (ContainerProfile, error) {
		return ContainerProfile{}, fmt.Errorf("container check profile: %s", reason)
	}
	if _, ok := prepared.Identity(); !ok || observation.Status != model.Passed || input.Namespace() != "checkProfiles" || input.Section() != "container" || input.Binding() == (apply.ProfileBinding{}) || observation.InputBinding != input.Binding() {
		return refuse("complete source-bound container Check preparation is required")
	}
	p := containerPreparation{input: input, out: apply.ProfileObservation{Status: model.Passed}}
	config, ok := p.decode()
	if !ok {
		return refuse("captured container/v1 configuration is invalid")
	}
	image := observation.CheckImage
	if observation.Contract != containerContract || image == nil || image.ImageID != config.profile.ImageID || image.Executable.Path != config.profile.Executable || image.Executable.ContentID == (snapshot.ContentID{}) || image.Executable.Size <= 0 || image.Executable.Size > container.MaxImageFileObservationBytes || observation.AgentImage != nil || len(observation.Files) != 0 || len(observation.Inventories) != 0 || len(observation.Instructions) != 0 || len(observation.RequiredInstructions) != 0 {
		return refuse("prepared contract, image or executable evidence differs from captured container/v1 configuration")
	}
	return config.profile, nil
}
