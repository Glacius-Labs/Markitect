package desktop

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Status is what the hub shows about a project: whether Markitect is set up
// in it and what markitect check finds in its working tree.
type Status struct {
	Initialized bool   `json:"initialized"`
	Check       *Check `json:"check,omitempty"`
	// Problem says why the project could not be read, such as a folder that
	// is gone or no longer a Git repository.
	Problem string `json:"problem,omitempty"`
}

// Check is the result of markitect check: the same load, the same errors.
type Check struct {
	OK          bool     `json:"ok"`
	ErrorCount  int      `json:"errorCount"`
	Errors      []string `json:"errors"` // the first maxErrors
	Definitions int      `json:"definitions"`
	Packages    int      `json:"packages"`
	Files       int      `json:"files"`
	Warnings    int      `json:"warnings"` // what markitect lint warns about
	// NotDrawn says that no model is drawn yet: Markitect is set up, its only
	// Definition is the first page, and the only errors are files with no place
	// in a model. On an existing project that is the state right after init,
	// and the next step is to draw the model, not to fix errors.
	NotDrawn bool `json:"notDrawn,omitempty"`
}

const maxErrors = 20

// fileUnregistered is the error of a file with no place in the model.
const fileUnregistered = "file.unregistered"

// Waiting is what the hub's next step looks at once the model compiles. Only
// the hub asks for it; the project list has no use for it.
type Waiting struct {
	// Proposal is the newest chat proposal the owner can decide now; a chat
	// at work may still change its own.
	Proposal *WaitingProposal `json:"proposal,omitempty"`
	// Uncommitted counts the files of Markitect's folder that a Run would not
	// see, since it starts from the tip of the branch.
	Uncommitted int `json:"uncommitted"`
	// Landed says that a Run has landed; Unlanded counts the Definitions,
	// the Libraries' too, that differ from its landing, as impact does.
	Landed   bool `json:"landed"`
	Unlanded int  `json:"unlanded"`
	// Extract is the newest conversation that draws the model, which the hub
	// names while no model is drawn.
	Extract *WaitingExtract `json:"extract,omitempty"`
}

// WaitingExtract names the conversation that draws the model.
type WaitingExtract struct {
	Chat    string `json:"chat"`
	Working bool   `json:"working"`
}

// WaitingProposal names the conversation a proposal waits in.
type WaitingProposal struct {
	Chat  string `json:"chat"`
	Title string `json:"title"`
}

// waitingChats is what the conversations hold for the owner.
func (s *server) waitingChats(p Project) (*Waiting, error) {
	w := &Waiting{}
	chats, err := s.chat.list(p)
	if err != nil {
		return nil, err
	}
	for _, c := range chats {
		if c.Proposal && !c.Working && w.Proposal == nil {
			w.Proposal = &WaitingProposal{Chat: c.ID, Title: c.Title}
		}
		if c.Skill == skillExtract && w.Extract == nil {
			w.Extract = &WaitingExtract{Chat: c.ID, Working: c.Working}
		}
	}
	return w, nil
}

func (s *server) waiting(ctx context.Context, p Project) (*Waiting, error) {
	w, err := s.waitingChats(p)
	if err != nil {
		return nil, err
	}
	ws, err := s.repos.Open(ctx, p.Path)
	if err != nil {
		return nil, err
	}
	files, err := s.repos.Uncommitted(ctx, ws.Root())
	if err != nil {
		return nil, err
	}
	w.Uncommitted = len(files)
	// The Model area's own document, which is kept while nothing changes.
	body, _, err := s.models.view(ctx, s.repos, p, "", nil)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Bases  bases `json:"bases"`
		Impact struct {
			Definitions []analysis.Change `json:"changedDefinitions"`
		} `json:"impact"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	if doc.Bases.Landing != "" {
		w.Landed, w.Unlanded = true, len(doc.Impact.Definitions)
	}
	return w, nil
}

func readStatus(ctx context.Context, repos Repos, p Project) Status {
	ws, err := repos.Open(ctx, p.Path)
	if err != nil {
		return Status{Problem: err.Error()}
	}
	pr, err := project.Load(ws)
	if err != nil {
		return Status{Problem: err.Error()}
	}
	if !slices.Contains(pr.Files, project.ManifestFile) {
		return Status{}
	}
	c := &Check{OK: len(pr.Errors) == 0, ErrorCount: len(pr.Errors), Errors: []string{}}
	unplaced := 0
	for _, d := range pr.Errors {
		if d.Code == fileUnregistered {
			unplaced++
		}
		if len(c.Errors) < maxErrors {
			c.Errors = append(c.Errors, d.String())
		}
	}
	if pr.Model != nil {
		c.Definitions = len(pr.Model.ModelDefinitions())
		c.Packages = len(pr.Model.Packages)
		c.Warnings = len(analysis.Lint(pr))
	}
	if pr.Registry != nil {
		c.Files = len(pr.Registry.Owner)
	}
	c.NotDrawn = !c.OK && unplaced == c.ErrorCount && c.Definitions <= 1
	return Status{Initialized: true, Check: c}
}
