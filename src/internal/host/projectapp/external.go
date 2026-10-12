package projectapp

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
)

// ExternalStartOrResume starts a plan or resumes its existing durable run.
func (o Operations) ExternalStartOrResume(ctx context.Context, operation RunOperation) (projectrun.RunReport, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.RunReport{}, err
	}
	_, err := projectrun.Status(operation.Root, operation.RunID)
	if errors.Is(err, projectrun.ErrNotFound) {
		return projectrun.RunExternal(ctx, o.Host, o.Invoker, operation.Root, operation.RunID)
	}
	if err != nil {
		return projectrun.RunReport{}, err
	}
	return projectrun.ResumeExternal(ctx, o.Host, o.Invoker, operation.Root, operation.RunID)
}

// ExternalPackets returns current pending packets for a durable run.
func (o Operations) ExternalPackets(operation RunOperation) ([]projectrun.ExternalPacket, error) {
	if err := requireRoot(operation.Root); err != nil {
		return nil, err
	}
	return projectrun.ExternalPackets(o.Host, operation.Root, operation.RunID)
}

// ExportExternalPacket returns one packet by ID, or requires exactly one
// pending packet when PacketID is empty.
func (o Operations) ExportExternalPacket(operation ExternalPacketOperation) (projectrun.ExternalPacket, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.ExternalPacket{}, err
	}
	return projectrun.ExportExternalPacket(o.Host, operation.Root, operation.RunID, operation.PacketID)
}

// SubmitExternalResponse durably accepts a response once. Resuming the run is
// a separate explicit operation.
func (o Operations) SubmitExternalResponse(operation ExternalResponseOperation) (projectrun.ExternalPacket, bool, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.ExternalPacket{}, false, err
	}
	return projectrun.SubmitExternalResponse(o.Host, operation.Root, operation.RunID, operation.PacketID, operation.ExpectedDigest, []byte(operation.Response), operation.GitCandidate)
}

type ExternalPacketOperation struct {
	Root     string `json:"root"`
	RunID    string `json:"runId"`
	PacketID string `json:"packetId,omitempty"`
}

type ExternalResponseOperation struct {
	Root           string                           `json:"root"`
	RunID          string                           `json:"runId"`
	PacketID       string                           `json:"packetId"`
	ExpectedDigest string                           `json:"expectedDigest"`
	Response       json.RawMessage                  `json:"response"`
	GitCandidate   *projectrun.ExternalGitCandidate `json:"gitCandidate,omitempty"`
}
