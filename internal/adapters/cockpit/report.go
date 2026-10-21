package cockpit

import (
	"encoding/json"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Report is the part of a run's report.json the cockpit presents. The report
// is written once when the run ends; its absence during a run is no failure.
//
// Status is the command status the report carries. Outcome is the Run outcome
// as the report states it when it does; the report of this build does not
// write that field, so the outcome is then mapped from the command status by
// the rule the producer documents (passed success, failed failed, anything
// else incomplete), and the Run's own status (report.run.status) is what the
// view presents with the publication.
type Report struct {
	Operation   string       `json:"operation"`
	Head        string       `json:"head,omitempty"`
	Status      string       `json:"status"`
	Outcome     string       `json:"outcome"`
	Derived     bool         `json:"derived,omitempty"`
	Run         *ReportRun   `json:"run,omitempty"`
	Publication *Publication `json:"publication,omitempty"`
	Retained    string       `json:"retainedWork,omitempty"`
	Error       string       `json:"error,omitempty"`
}

// ReportRun is the report's run: its Passes, findings, questions and problems
// as the Orchestrator ended them, the counts of the turns and probes it holds
// and the usage. The turns and the probe receipts themselves stay in the
// events and the report file.
type ReportRun struct {
	ID        string           `json:"id"`
	Status    string           `json:"status"`
	Reason    string           `json:"reason,omitempty"`
	Passes    []run.PassRecord `json:"passes"`
	Findings  []run.Finding    `json:"findings"`
	Questions []run.Question   `json:"questions"`
	Problems  []run.Problem    `json:"problems"`
	Turns     int              `json:"turns"`
	Probes    ProbeCounts      `json:"probes"`
	Usage     run.Usage        `json:"usage"`
}

// ProbeCounts summarizes the probe receipts of a run. Undisposed counts
// adverse receipts without an accepted disposition.
type ProbeCounts struct {
	Total      int `json:"total"`
	Passed     int `json:"passed"`
	Failed     int `json:"failed"`
	Error      int `json:"error"`
	Undisposed int `json:"undisposed"`
}

// Publication is the publisher's receipt as reported: its outcome and the
// read-back state of every requested path.
type Publication struct {
	ID      string            `json:"id"`
	Outcome string            `json:"outcome"`
	Paths   []PublicationPath `json:"paths"`
	Reason  string            `json:"reason,omitempty"`
}

// PublicationPath is one requested path: "after" holds the published content,
// "before" the captured content, "other" anything else or unreadable.
type PublicationPath struct {
	Path  string `json:"path"`
	State string `json:"state"`
}

// rawReport and rawRun are the fields of report.json the cockpit reads, named
// as the producer writes them. A test compares their JSON names with the
// producer's types, so a renamed field cannot silently empty the view.
type rawReport struct {
	Operation    string          `json:"operation"`
	Status       string          `json:"status"`
	Outcome      string          `json:"outcome"`
	Run          *rawRun         `json:"run"`
	Publication  *rawPublication `json:"publication"`
	RetainedWork string          `json:"retainedWork"`
	Error        string          `json:"error"`
	Preparation  struct {
		Attempts []struct {
			Head string `json:"head"`
		} `json:"attempts"`
	} `json:"preparation"`
}

type rawRun struct {
	ID        string             `json:"id"`
	Status    string             `json:"status"`
	Reason    string             `json:"reason"`
	Passes    []run.PassRecord   `json:"passes"`
	Findings  []run.Finding      `json:"findings"`
	Questions []run.Question     `json:"questions"`
	Problems  []run.Problem      `json:"problems"`
	Turns     []json.RawMessage  `json:"turns"`
	Probes    []run.ProbeReceipt `json:"probes"`
	Usage     run.Usage          `json:"usage"`
}

// rawPublication has no JSON tags, as the producer's receipt has none: the
// decoder matches the exported names without regard to case.
type rawPublication struct {
	ID      string
	Outcome string
	Paths   []struct{ Path, State string }
	Reason  string
}

func decodeReport(data []byte) (*Report, error) {
	var raw rawReport
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	r := &Report{Operation: raw.Operation, Status: raw.Status, Outcome: raw.Outcome, Retained: raw.RetainedWork, Error: raw.Error}
	// The last preparation attempt is the one the run was bound to.
	if attempts := raw.Preparation.Attempts; len(attempts) > 0 {
		r.Head = attempts[len(attempts)-1].Head
	}
	if raw.Run != nil {
		x := raw.Run
		r.Run = &ReportRun{ID: x.ID, Status: x.Status, Reason: x.Reason, Passes: x.Passes, Findings: x.Findings, Questions: x.Questions, Problems: x.Problems, Turns: len(x.Turns), Usage: x.Usage}
		for _, pr := range x.Probes {
			r.Run.Probes.Total++
			switch pr.Outcome {
			case run.ProbePassed:
				r.Run.Probes.Passed++
			case run.ProbeFailed:
				r.Run.Probes.Failed++
			default:
				r.Run.Probes.Error++
			}
			if pr.Outcome != run.ProbePassed && (pr.Disposition == nil || pr.Disposition.Pending) {
				r.Run.Probes.Undisposed++
			}
		}
	}
	if r.Outcome == "" {
		// The report of this build does not write the Run outcome. It follows
		// from the command status by the producer's rule; Derived marks the
		// value as mapped, and the view presents the Run's own status with
		// the publication beside it.
		r.Derived = true
		switch r.Status {
		case "passed":
			r.Outcome = "success"
		case "failed":
			r.Outcome = "failed"
		default:
			r.Outcome = "incomplete"
		}
	}
	if p := raw.Publication; p != nil {
		r.Publication = &Publication{ID: p.ID, Outcome: p.Outcome, Reason: p.Reason, Paths: []PublicationPath{}}
		for _, path := range p.Paths {
			r.Publication.Paths = append(r.Publication.Paths, PublicationPath{Path: path.Path, State: path.State})
		}
	}
	return r, nil
}
