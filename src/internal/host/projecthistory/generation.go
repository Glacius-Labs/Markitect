package projecthistory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
)

// Generate loads the named committed revisions and derives their briefing.
func Generate(root, sinceRevision, revision string, provenance projectbriefing.Provenance) (projectbriefing.Bundle, error) {
	return generate(root, sinceRevision, revision, provenance, nil)
}

func generate(root, sinceRevision, revision string, provenance projectbriefing.Provenance, state *projectbriefing.Store) (projectbriefing.Bundle, error) {
	if strings.TrimSpace(sinceRevision) == "" || strings.TrimSpace(revision) == "" || sinceRevision == revision {
		return projectbriefing.Bundle{}, projectbriefing.ErrUncommittedModel
	}
	before, err := projectwork.Load(root, sinceRevision)
	if err != nil {
		return projectbriefing.Bundle{}, fmt.Errorf("load since revision: %w", err)
	}
	after, err := projectwork.Load(root, revision)
	if err != nil {
		return projectbriefing.Bundle{}, fmt.Errorf("load revision: %w", err)
	}
	if before.Provisional || after.Provisional || before.Revision == after.Revision {
		return projectbriefing.Bundle{}, projectbriefing.ErrUncommittedModel
	}
	if err := requireAncestor(root, before.Revision, after.Revision); err != nil {
		return projectbriefing.Bundle{}, err
	}
	if state == nil {
		stored, _, err := readStore(root)
		if err != nil {
			return projectbriefing.Bundle{}, err
		}
		state = &stored
	}
	predecessors, err := lastEvents(root, *state, before.Revision)
	if err != nil {
		return projectbriefing.Bundle{}, err
	}
	return projectbriefing.BuildWithPredecessors(revisionModel(before), revisionModel(after), provenance, predecessors)
}

func revisionModel(project *projectvalue.Project) projectbriefing.RevisionModel {
	return projectbriefing.RevisionModel{Revision: project.Revision, Provisional: project.Provisional, Model: project.Model, Report: project.Report}
}

func lastEvents(root string, state projectbriefing.Store, revision string) (map[string]string, error) {
	revisions, err := firstParentRevisions(root, revision)
	if err != nil {
		return nil, fmt.Errorf("enumerate briefed history before %s: %w", revision, err)
	}
	positions := make(map[string]int, len(revisions))
	for i, item := range revisions {
		positions[item] = i
	}
	last, at := map[string]string{}, map[string]int{}
	for _, bundle := range state.Briefings {
		i, onLine := positions[bundle.Revision]
		if !onLine {
			continue
		}
		for _, event := range bundle.Events {
			key := event.DefinitionID.Key()
			if prior, seen := at[key]; !seen || i > prior {
				last[key], at[key] = event.ID, i
			}
		}
	}
	return last, nil
}

func requireAncestor(root, sinceRevision, revision string) error {
	if sinceRevision == revision {
		return projectbriefing.ErrUncommittedModel
	}
	if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", sinceRevision, revision); err != nil {
		return fmt.Errorf("%w: %s..%s: %v", projectbriefing.ErrRevisionNotAncestor, sinceRevision, revision, err)
	}
	return nil
}

func hash(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
