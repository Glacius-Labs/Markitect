// Package run is the work on a project folder, without Git (DEC-096, DEC-097):
// checklist says what a change since the newest certificate reaches, answer
// stores a reviewer's answers bound to the files they rest on, and gate checks the
// work and, when it passes, writes its certificate. Whoever drives the agents (a
// skill, a workflow or a person) starts them and makes every Git step; nothing
// here starts or drives an agent. What the tool knows of the work between commands
// lives in the user's cache folder.
//
// The work is described in internal/run/README.md.
package run

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

const (
	// repairRounds bounds how often a review's wrong answers go back.
	repairRounds = 2
	// maxRefusals bounds how many errors one refusal lists.
	maxRefusals = 30
)

// ErrUsage marks a request the tool cannot run, such as an unknown base.
var ErrUsage = errors.New("cannot run")

// Engine runs the commands of the work in one project folder.
type Engine struct {
	// Folder reads the project folder, and Cache is where the work's state lives
	// between commands, outside the project.
	Folder Folder
	Cache  string
	// Now defaults to the clock.
	Now func() time.Time
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// Answer stores a reviewer's answers against the folder as it is. What it cannot
// accept is refused at once, with what is still missing.
func (e *Engine) Answer(_ context.Context, dir string, raw []byte) (string, bool, error) {
	return e.answerState(dir, raw)
}

// Status says how the work in the folder stands, and changes nothing.
func (e *Engine) Status(_ context.Context, dir string) (string, error) {
	return e.stateStatus(dir)
}

func listErrors(errs model.Diagnostics) string {
	var b strings.Builder
	for i, d := range errs {
		if i == maxRefusals {
			fmt.Fprintf(&b, "and %d more\n", len(errs)-maxRefusals)
			break
		}
		fmt.Fprintf(&b, "- %s\n", d.String())
	}
	return b.String()
}
