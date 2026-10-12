package projectapp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/Glacius-Labs/Markitect/src/internal/application/projectadoption"
	"github.com/Glacius-Labs/Markitect/src/internal/application/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/application/projecthistory"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

// OverviewOperation selects the project model and source repository whose
// recorded state is included in the read-only project overview.
type OverviewOperation struct {
	Root       string `json:"root"`
	SourceRoot string `json:"sourceRoot,omitempty"`
}

// Overview is the project status assembled from the compiled model and
// recorded project state. It never writes or starts work.
type Overview struct {
	Project      OverviewProject       `json:"project"`
	Runtime      OverviewRuntime       `json:"runtime"`
	Runs         []OverviewRun         `json:"runs"`
	Explorations []OverviewExploration `json:"explorations"`
	Briefings    []OverviewBriefings   `json:"briefings"`
	Adoptions    []OverviewAdoption    `json:"adoptions"`
}

type OverviewProject struct {
	Name          string `json:"name"`
	Revision      string `json:"revision"`
	Provisional   bool   `json:"provisional"`
	Status        string `json:"status"`
	Errors        int    `json:"errors"`
	Warnings      int    `json:"warnings"`
	CoverageMode  string `json:"coverageMode"`
	Conforming    *bool  `json:"coverageConforming,omitempty"`
	ProjectDigest string `json:"projectDigest"`
}

type OverviewRuntime struct {
	Configured bool   `json:"configured"`
	Digest     string `json:"digest,omitempty"`
}

type OverviewRun struct {
	projectrun.RunSummary
	Next string `json:"next,omitempty"`
}

type OverviewExploration struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Scopes []OverviewScope `json:"scopes"`
}

type OverviewScope struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Acknowledged bool   `json:"acknowledged"`
}

type OverviewBriefings struct {
	Manager string `json:"manager"`
	Pending int    `json:"pending"`
}

type OverviewAdoption struct {
	Session    string `json:"session"`
	Stage      string `json:"stage"`
	Iterations int    `json:"iterations"`
	Adoptions  int    `json:"adoptions"`
}

// Overview returns the model status and recorded project activity without
// mutating state or starting agents or configured checks.
func (o Operations) Overview(operation OverviewOperation) (Overview, error) {
	if operation.SourceRoot == "" {
		operation.SourceRoot = operation.Root
	}
	result, err := o.Check(Selection{Root: operation.Root})
	if err != nil {
		return Overview{}, err
	}
	out := Overview{Project: OverviewProject{
		Name: result.Source.Name, Revision: result.Source.Revision, Provisional: result.Source.Provisional,
		Status: result.Report.Status, CoverageMode: result.Source.CoverageMode, ProjectDigest: result.Source.ProjectDigest,
	}}
	for _, finding := range result.Findings {
		switch finding.Severity {
		case "error":
			out.Project.Errors++
		case "warning":
			out.Project.Warnings++
		}
	}
	if result.Source.CoverageMode == "full" && result.Coverage != nil {
		conforming := result.Coverage.Conforming
		out.Project.Conforming = &conforming
	}
	if data, err := os.ReadFile(filepath.Join(operation.Root, filepath.FromSlash(projectvalue.RuntimePath))); err == nil {
		sum := sha256.Sum256(data)
		out.Runtime = OverviewRuntime{Configured: true, Digest: "sha256:" + hex.EncodeToString(sum[:])}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Overview{}, err
	}
	runs, err := projectrun.ListRuns(operation.Root)
	if err != nil {
		return Overview{}, err
	}
	out.Runs = make([]OverviewRun, 0, len(runs))
	for _, run := range runs {
		out.Runs = append(out.Runs, OverviewRun{RunSummary: run, Next: nextVerb(run.Status)})
	}
	if out.Explorations, err = overviewExplorations(operation.Root); err != nil {
		return Overview{}, err
	}
	if out.Briefings, err = overviewPendingBriefings(operation.Root); err != nil {
		return Overview{}, err
	}
	if out.Adoptions, err = overviewAdoptions(operation.SourceRoot); err != nil {
		return Overview{}, err
	}
	return out, nil
}

// nextVerb names the verb that usually applies to a run in a given status.
func nextVerb(status string) string {
	switch status {
	case projectrun.StatusPlanned:
		return "run"
	case projectrun.StatusInterrupted:
		return "resume"
	case projectrun.StatusBlocked, projectrun.StatusFailed:
		return "repair"
	case projectrun.StatusIntegrated, projectrun.StatusVerifying:
		return "verify"
	case projectrun.StatusVerified:
		return "apply"
	case projectrun.StatusRunning:
		return "status"
	}
	return ""
}

func overviewExplorations(root string) ([]OverviewExploration, error) {
	records, err := projectexplore.List(root)
	if err != nil {
		return nil, err
	}
	out := make([]OverviewExploration, 0, len(records))
	for _, record := range records {
		acknowledged := map[string]bool{}
		for _, item := range record.Acknowledgements {
			acknowledged[item.ScopeID] = true
		}
		scopes := make([]OverviewScope, 0, len(record.Scopes))
		for _, scope := range record.Scopes {
			scopes = append(scopes, OverviewScope{ID: scope.ID, Name: scope.Name, Acknowledged: acknowledged[scope.ID]})
		}
		out = append(out, OverviewExploration{ID: record.ID, Status: record.Status, Scopes: scopes})
	}
	return out, nil
}

// overviewPendingBriefings counts unresolved, undismissed events per affected
// Manager from the recorded store only.
func overviewPendingBriefings(root string) ([]OverviewBriefings, error) {
	read, err := projecthistory.Read(root)
	if err != nil {
		return nil, err
	}
	state := read.Accepted
	dismissed := map[[2]string]bool{}
	for _, item := range state.Dismissals {
		dismissed[[2]string{item.EventID, item.ManagerID}] = true
	}
	pending := map[string]int{}
	seen := map[string]bool{}
	for _, bundle := range state.Briefings {
		for _, event := range bundle.Events {
			if seen[event.ID] || projectbriefing.EventResolutionStatus(read.StatusView, event.ID).Status == projectbriefing.StatusResolved {
				continue
			}
			seen[event.ID] = true
			for _, manager := range event.AffectedManagers {
				if !dismissed[[2]string{event.ID, manager}] {
					pending[manager]++
				}
			}
		}
	}
	out := make([]OverviewBriefings, 0, len(pending))
	for manager, count := range pending {
		out = append(out, OverviewBriefings{Manager: manager, Pending: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manager < out[j].Manager })
	return out, nil
}

// overviewAdoptions reads session records without the session lock, so the
// overview never writes.
func overviewAdoptions(sourceRoot string) ([]OverviewAdoption, error) {
	base := filepath.Join(sourceRoot, filepath.FromSlash(projectadoption.BrownfieldSessionPath))
	entries, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		return []OverviewAdoption{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []OverviewAdoption{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(base, entry.Name(), "session.json"))
		if err != nil {
			continue
		}
		var session projectadoption.BrownfieldSession
		if json.Unmarshal(data, &session) != nil || session.ID != entry.Name() {
			out = append(out, OverviewAdoption{Session: entry.Name(), Stage: "invalid"})
			continue
		}
		stage := "started"
		switch {
		case len(session.Adoptions) != 0:
			stage = "applied"
		case len(session.Iterations) != 0:
			stage = "iterating"
		}
		out = append(out, OverviewAdoption{Session: session.ID, Stage: stage, Iterations: len(session.Iterations), Adoptions: len(session.Adoptions)})
	}
	return out, nil
}
