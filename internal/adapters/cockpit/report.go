package cockpit

import (
	"encoding/json"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Report is the part of a run's report.json the cockpit presents. The report
// is written once when the run ends; its absence during a run is no failure.
type Report struct {
	Operation   string       `json:"operation"`
	Status      string       `json:"status"`
	Outcome     string       `json:"outcome"`
	Derived     bool         `json:"derived,omitempty"`
	Run         *reportRun   `json:"run,omitempty"`
	Publication *Publication `json:"publication,omitempty"`
	Retained    string       `json:"retainedWork,omitempty"`
	Error       string       `json:"error,omitempty"`
}

type reportRun struct {
	ID        string           `json:"id"`
	Status    string           `json:"status"`
	Reason    string           `json:"reason,omitempty"`
	Passes    []run.PassRecord `json:"passes"`
	Findings  []run.Finding    `json:"findings"`
	Questions []run.Question   `json:"questions"`
	Usage     run.Usage        `json:"usage"`
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

func decodeReport(data []byte) (*Report, error) {
	var raw struct {
		Operation   string     `json:"operation"`
		Status      string     `json:"status"`
		Outcome     string     `json:"outcome"`
		Run         *reportRun `json:"run"`
		Publication *struct {
			ID      string
			Outcome string
			Paths   []struct{ Path, State string }
			Reason  string
		} `json:"publication"`
		RetainedWork string `json:"retainedWork"`
		Error        string `json:"error"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	r := &Report{Operation: raw.Operation, Status: raw.Status, Outcome: raw.Outcome, Run: raw.Run, Retained: raw.RetainedWork, Error: raw.Error}
	if r.Outcome == "" {
		// Reports of earlier builds carry only the command status; its mapping
		// to the Run outcome is the CLI's (passed success, failed failed,
		// anything else incomplete). Derived marks the value as mapped.
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
