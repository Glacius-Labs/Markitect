package claudecode

import (
	"context"
	"errors"
	"fmt"
)

// RuntimeObservation is what an immutable image records for Claude sessions:
// its engine image ID and the build identities of its runtime metadata.
type RuntimeObservation struct {
	ImageID       string
	AgentdVersion string
	CLIVersion    string
}

// ObserveRuntime resolves an image reference to its immutable engine image ID
// and reads the runtime metadata the image records, without starting it. It
// refuses an image that does not record the supported Claude Code version, so
// a Host profile written from the observation passes metadata preparation.
func ObserveRuntime(ctx context.Context, reader ImageReader, ref string) (RuntimeObservation, error) {
	if reader == nil {
		return RuntimeObservation{}, errors.New("an image reader is required")
	}
	imageID, err := reader.ImageID(ctx, ref)
	if err != nil {
		return RuntimeObservation{}, err
	}
	if !sha256Literal(imageID) {
		return RuntimeObservation{}, errors.New("the engine did not report an immutable sha256 image ID")
	}
	data, err := reader.ReadImageFile(ctx, imageID, runtimeMetadataPath)
	if err != nil {
		return RuntimeObservation{}, err
	}
	if len(data) > maxMetadataBytes {
		return RuntimeObservation{}, errors.New("the image runtime metadata exceeds its 1 MiB observation limit")
	}
	values, err := runtimeMetadata(data)
	if err != nil {
		return RuntimeObservation{}, err
	}
	if values["claude"] != supportedClaudeVersion {
		return RuntimeObservation{}, fmt.Errorf("the image records the Claude Code version %q; %q is required", values["claude"], supportedClaudeVersion)
	}
	if again, err := reader.ImageID(ctx, ref); err != nil {
		return RuntimeObservation{}, err
	} else if again != imageID {
		return RuntimeObservation{}, errors.New("the image identity changed during observation")
	}
	return RuntimeObservation{ImageID: imageID, AgentdVersion: values["agentd"], CLIVersion: values["claude"]}, nil
}

// HostProfile is the explicit content of one Claude Code Host profile. Setup
// writes it into the closed section that the metadata decoder reads back.
type HostProfile struct {
	EngineExecutable string
	ImageID          string
	AgentdVersion    string
	CredentialRef    string
	MaxWireBytes     int64
	Session          SessionProfile
}

// HostInstruction names one captured instruction file of a profile.
type HostInstruction struct {
	Boundary string `yaml:"boundary"`
	Path     string `yaml:"path"`
}

// HostSection is the closed `claudeCode` section of a Host document in the
// field order and spelling of the decoder. It has no instruction files.
type HostSection struct {
	ProfileVersion   int               `yaml:"profileVersion"`
	EngineExecutable string            `yaml:"engineExecutable"`
	ImageID          string            `yaml:"imageID"`
	AgentdVersion    string            `yaml:"agentdVersion"`
	Instructions     []HostInstruction `yaml:"instructions"`
	CredentialRef    string            `yaml:"credentialRef"`
	MaxWireBytes     int64             `yaml:"maxWireBytes"`
	Session          SessionProfile    `yaml:"session"`
}

// Section validates the profile as the adapter will and returns its Host
// document section. It performs no engine, image or credential access.
func (p HostProfile) Section() (HostSection, error) {
	if !literal(p.EngineExecutable) || !literal(p.AgentdVersion) || !literal(p.CredentialRef) {
		return HostSection{}, fmt.Errorf("%w: engine, agentd version and credential reference must be nonempty literals", ErrSession)
	}
	profile := Profile{ImageID: p.ImageID, CredentialRef: p.CredentialRef, MaxWireBytes: p.MaxWireBytes, Session: p.Session}
	if err := profile.validate(); err != nil {
		return HostSection{}, err
	}
	return HostSection{
		ProfileVersion: 1, EngineExecutable: p.EngineExecutable, ImageID: p.ImageID, AgentdVersion: p.AgentdVersion,
		Instructions: []HostInstruction{}, CredentialRef: p.CredentialRef, MaxWireBytes: p.MaxWireBytes, Session: p.Session,
	}, nil
}
