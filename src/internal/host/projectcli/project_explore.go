package projectcli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
)

// runExplore keeps transport path access and JSON formatting in the CLI while
// the typed application operation owns record lifecycle and write guards.
func runExplore(opts options, out io.Writer) error {
	var record *projectexplore.Record
	if opts.input != "" {
		data, err := readRecord(opts.repo, opts.input)
		if err != nil {
			return fmt.Errorf("read exploration input: %w", err)
		}
		decoded, err := projectexplore.DecodeRecordInput(data)
		if err != nil {
			return err
		}
		record = &decoded
	}
	result, err := projectOperations().Explore(projectapp.ExploreOperation{
		Selection:     projectapp.Selection{Root: opts.repo, Revision: opts.revision},
		ExplorationID: opts.explorationID, Record: record, Write: opts.write, ExpectedDigest: opts.expect,
	})
	if err != nil {
		return err
	}
	if result.Record != nil {
		return writeJSON(out, result.Record)
	}
	if result.Plan != nil {
		if result.Persisted != nil {
			return writeJSON(out, struct {
				Record projectexplore.Record    `json:"record"`
				Plan   projectexplore.WritePlan `json:"plan"`
			}{*result.Persisted, *result.Plan})
		}
		return writeJSON(out, result.Plan)
	}
	return writeJSON(out, result.Records)
}

// runReadiness formats the application result. The CLI parses the explicit
// timestamp syntax; projectapp binds and persists the assertion through CAS.
func runReadiness(opts options, out io.Writer) error {
	var acknowledgement *projectapp.StructureAcknowledgementInput
	if opts.acknowledgeStructure {
		if strings.TrimSpace(opts.actor) == "" || strings.TrimSpace(opts.authority) == "" || strings.TrimSpace(opts.decisionRef) == "" {
			return fmt.Errorf("--acknowledge-structure requires --actor, --authority, and --decision-ref")
		}
		if strings.TrimSpace(opts.acknowledgedAt) == "" {
			return fmt.Errorf("--acknowledged-at must be an explicit RFC3339 timestamp")
		}
		recordedAt, err := time.Parse(time.RFC3339Nano, opts.acknowledgedAt)
		if err != nil {
			return fmt.Errorf("--acknowledged-at must be an explicit RFC3339 timestamp")
		}
		acknowledgement = &projectapp.StructureAcknowledgementInput{
			Actor: opts.actor, Authority: opts.authority, Provenance: opts.decisionRef, RecordedAt: recordedAt,
		}
	}
	result, err := projectOperations().Readiness(projectapp.ReadinessOperation{
		Selection:     projectapp.Selection{Root: opts.repo, Revision: opts.revision},
		ExplorationID: opts.explorationID, ScopeID: opts.scope, Acknowledgement: acknowledgement,
		Write: opts.write, ExpectedDigest: opts.expect,
	})
	if err != nil {
		return err
	}
	return writeJSON(out, struct {
		Exploration projectexplore.Record          `json:"exploration"`
		Scope       projectexplore.Scope           `json:"scope"`
		Binding     projectexplore.Binding         `json:"binding"`
		Readiness   projectexplore.ReadinessReport `json:"readiness"`
		WritePlan   *projectexplore.WritePlan      `json:"writePlan,omitempty"`
		Persisted   *projectexplore.Record         `json:"persisted,omitempty"`
	}{result.Exploration, result.Scope, result.Binding, result.Readiness, result.WritePlan, result.Persisted})
}
