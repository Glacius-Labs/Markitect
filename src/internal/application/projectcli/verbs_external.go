package projectcli

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Glacius-Labs/Markitect/src/internal/application/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

type packetExportInput struct {
	Run     string `json:"run"`
	Execute bool   `json:"execute,omitempty"`
}

var packetExportVerb = define(verb{
	name: "packet-export", group: "Delivery", effect: effectExecute, inspect: "status", interrupt: true,
	summary:  "Start or resume a run in external mode and export its pending packets.",
	synopsis: "packet-export RUN --execute",
	recovery: "Inspect pending packets with packet-list, submit one with packet-submit, then use packet-export again to continue the run.",
	args:     []arg{{name: "run", value: "RUN", operand: true, required: true, help: "Plan or run ID."}, argRepo, argExecute},
}, func(ctx context.Context, e env, in packetExportInput) (any, error) {
	report, runErr := e.ops.ExternalStartOrResume(ctx, projectapp.RunOperation{Root: e.root, RunID: in.Run})
	if runErr != nil && !errors.Is(runErr, projectrun.ErrAwaitingExternal) {
		return report, runErr
	}
	packets, err := e.ops.ExternalPackets(projectapp.RunOperation{Root: e.root, RunID: in.Run})
	if err != nil {
		return report, err
	}
	return struct {
		Run     projectrun.RunReport        `json:"run"`
		Packets []projectrun.ExternalPacket `json:"packets"`
	}{Run: report, Packets: packets}, nil
})

type packetListInput struct {
	Run string `json:"run"`
}

var packetListVerb = define(verb{
	name: "packet-list", group: "Delivery", effect: effectRead,
	summary:  "List currently pending external packets without starting work.",
	synopsis: "packet-list RUN",
	recovery: "Start or resume the run with `packet-export RUN --execute` to create its pending packet.",
	args:     []arg{{name: "run", value: "RUN", operand: true, required: true, help: "Run ID."}, argRepo},
}, func(_ context.Context, e env, in packetListInput) ([]projectrun.ExternalPacket, error) {
	return e.ops.ExternalPackets(projectapp.RunOperation{Root: e.root, RunID: in.Run})
})

type packetSubmitInput struct {
	Run               string          `json:"run"`
	Packet            string          `json:"packet"`
	Expect            string          `json:"expect"`
	Response          json.RawMessage `json:"response"`
	CandidateRoot     string          `json:"candidateRoot,omitempty"`
	BaseRevision      string          `json:"baseRevision,omitempty"`
	CandidateRevision string          `json:"candidateRevision,omitempty"`
	Execute           bool            `json:"execute,omitempty"`
}

var packetSubmitVerb = define(verb{
	name: "packet-submit", group: "Delivery", effect: effectExecute, inspect: "packet-export",
	summary:  "Submit one response bound to a pending external packet.",
	synopsis: "packet-submit RUN --packet ID --expect DIGEST --response FILE [--candidate-root DIR --base-revision SHA --candidate-revision SHA] --execute",
	notes:    "This stores a protocol-valid response and, when supplied, a Git-observed candidate delta. Task and review validation occurs during the separate `packet-export RUN --execute` continuation; Verify and Apply remain separate operations.",
	recovery: "Export the current packet and refresh its expected digest; a submitted packet cannot accept different response bytes.",
	args: []arg{{name: "run", value: "RUN", operand: true, required: true, help: "Run ID."}, argRepo,
		{name: "packet", value: "ID", required: true, help: "Packet ID from packet-export."},
		{name: "expect", value: "DIGEST", required: true, help: "Exact packet digest from packet-export."},
		{name: "response", kind: kindRecord, value: "FILE", required: true, help: "Complete agent execution response JSON."},
		{name: "candidate-root", value: "DIR", help: "Absolute repository root containing the committed candidate."},
		{name: "base-revision", value: "SHA", help: "Full commit ID for the exact packet base snapshot."},
		{name: "candidate-revision", value: "SHA", help: "Full commit ID for the proposed candidate tree."}, argExecute},
}, func(_ context.Context, e env, in packetSubmitInput) (any, error) {
	if len(in.Response) == 0 || !json.Valid(in.Response) {
		return nil, usagef("--response must contain one complete JSON response")
	}
	gitArgs := []string{in.CandidateRoot, in.BaseRevision, in.CandidateRevision}
	gitArgCount := 0
	for _, value := range gitArgs {
		if value != "" {
			gitArgCount++
		}
	}
	if gitArgCount != 0 && gitArgCount != len(gitArgs) {
		return nil, usagef("--candidate-root, --base-revision, and --candidate-revision must be provided together")
	}
	var candidate *projectrun.ExternalGitCandidate
	if gitArgCount == len(gitArgs) {
		candidate = &projectrun.ExternalGitCandidate{Root: in.CandidateRoot, BaseRevision: in.BaseRevision, Revision: in.CandidateRevision}
	}
	packet, accepted, err := e.ops.SubmitExternalResponse(projectapp.ExternalResponseOperation{
		Root: e.root, RunID: in.Run, PacketID: in.Packet, ExpectedDigest: in.Expect, Response: in.Response, GitCandidate: candidate,
	})
	if err != nil {
		return packet, err
	}
	return struct {
		PacketID      string `json:"packetId"`
		Digest        string `json:"digest"`
		Stored        bool   `json:"stored"`
		NewSubmission bool   `json:"newSubmission"`
	}{PacketID: packet.ID, Digest: packet.Digest, Stored: true, NewSubmission: accepted}, nil
})

// externalVerbs is appended to the Delivery verb table by its owner.
var externalVerbs = []verb{packetExportVerb, packetListVerb, packetSubmitVerb}
