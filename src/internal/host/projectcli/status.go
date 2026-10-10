package projectcli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

// overview is the project status: it reads only recorded state and the
// compiled working-tree model, and starts nothing.
type overview struct {
	Project      overviewProject       `json:"project"`
	Runtime      overviewRuntime       `json:"runtime"`
	Runs         []overviewRun         `json:"runs"`
	Explorations []overviewExploration `json:"explorations"`
	Briefings    []overviewBriefings   `json:"briefings"`
	Adoptions    []overviewAdoption    `json:"adoptions"`
}

type overviewProject struct {
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

type overviewRuntime struct {
	Configured bool   `json:"configured"`
	Digest     string `json:"digest,omitempty"`
}

type overviewRun struct {
	projectrun.RunSummary
	Next string `json:"next,omitempty"`
}

type overviewExploration struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Scopes []overviewScope `json:"scopes"`
}

type overviewScope struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Acknowledged bool   `json:"acknowledged"`
}

type overviewBriefings struct {
	Manager string `json:"manager"`
	Pending int    `json:"pending"`
}

type overviewAdoption struct {
	Session    string `json:"session"`
	Stage      string `json:"stage"`
	Iterations int    `json:"iterations"`
	Adoptions  int    `json:"adoptions"`
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

func projectOverview(e env) (overview, error) {
	result, err := e.ops.Check(projectapp.Selection{Root: e.root})
	if err != nil {
		return overview{}, err
	}
	out := overview{Project: overviewProject{
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
	if data, err := os.ReadFile(filepath.Join(e.root, filepath.FromSlash(projectwork.RuntimePath))); err == nil {
		sum := sha256.Sum256(data)
		out.Runtime = overviewRuntime{Configured: true, Digest: "sha256:" + hex.EncodeToString(sum[:])}
	} else if !errors.Is(err, os.ErrNotExist) {
		return overview{}, err
	}
	runs, err := projectrun.ListRuns(e.root)
	if err != nil {
		return overview{}, err
	}
	out.Runs = make([]overviewRun, 0, len(runs))
	for _, run := range runs {
		out.Runs = append(out.Runs, overviewRun{RunSummary: run, Next: nextVerb(run.Status)})
	}
	if out.Explorations, err = overviewExplorations(e.root); err != nil {
		return overview{}, err
	}
	if out.Briefings, err = overviewPendingBriefings(e.root); err != nil {
		return overview{}, err
	}
	if out.Adoptions, err = overviewAdoptions(e.sourceRoot); err != nil {
		return overview{}, err
	}
	return out, nil
}

func overviewExplorations(root string) ([]overviewExploration, error) {
	records, err := projectexplore.List(root)
	if err != nil {
		return nil, err
	}
	out := make([]overviewExploration, 0, len(records))
	for _, record := range records {
		acknowledged := map[string]bool{}
		for _, item := range record.Acknowledgements {
			acknowledged[item.ScopeID] = true
		}
		scopes := make([]overviewScope, 0, len(record.Scopes))
		for _, scope := range record.Scopes {
			scopes = append(scopes, overviewScope{ID: scope.ID, Name: scope.Name, Acknowledged: acknowledged[scope.ID]})
		}
		out = append(out, overviewExploration{ID: record.ID, Status: record.Status, Scopes: scopes})
	}
	return out, nil
}

// overviewPendingBriefings counts unresolved, undismissed events per affected
// Manager from the recorded store only.
func overviewPendingBriefings(root string) ([]overviewBriefings, error) {
	state, _, err := projectbriefing.Read(root)
	if err != nil {
		return nil, err
	}
	dismissed := map[[2]string]bool{}
	for _, item := range state.Dismissals {
		dismissed[[2]string{item.EventID, item.ManagerID}] = true
	}
	pending := map[string]int{}
	seen := map[string]bool{}
	for _, bundle := range state.Briefings {
		for _, event := range bundle.Events {
			if seen[event.ID] || projectbriefing.EventResolutionStatus(state, event.ID).Status == "resolved" {
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
	out := make([]overviewBriefings, 0, len(pending))
	for manager, count := range pending {
		out = append(out, overviewBriefings{Manager: manager, Pending: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manager < out[j].Manager })
	return out, nil
}

// overviewAdoptions reads session records without the session lock, so the
// overview never writes.
func overviewAdoptions(sourceRoot string) ([]overviewAdoption, error) {
	base := filepath.Join(sourceRoot, filepath.FromSlash(projectadoption.BrownfieldSessionPath))
	entries, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		return []overviewAdoption{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []overviewAdoption{}
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
			out = append(out, overviewAdoption{Session: entry.Name(), Stage: "invalid"})
			continue
		}
		stage := "started"
		switch {
		case len(session.Adoptions) != 0:
			stage = "applied"
		case len(session.Iterations) != 0:
			stage = "iterating"
		}
		out = append(out, overviewAdoption{Session: session.ID, Stage: stage, Iterations: len(session.Iterations), Adoptions: len(session.Adoptions)})
	}
	return out, nil
}
