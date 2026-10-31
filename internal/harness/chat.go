package harness

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/skills"
)

// The chat skills, as Markitect Desktop names them.
const (
	ChatModeling = "modeling"
	ChatExtract  = "extract"
)

// ChatStart is the ref of a chat's copy that holds its start: the working
// tree it was copied from, moved on by each accepted proposal.
const ChatStart = "refs/markitect/start"

// ChatInput is what a chat agent's documents are rendered from.
type ChatInput struct {
	// Skill is ChatModeling or ChatExtract.
	Skill string
	// Earlier is the conversation so far, for an agent that takes over a
	// conversation; empty for a new one.
	Earlier string
}

// chatSkills are the skill files each chat carries, in reading order.
var chatSkills = map[string][]string{
	ChatModeling: {"markitect-modeling/SKILL.md", "markitect-modeling/reference.md", "markitect-modeling/examples.md"},
	ChatExtract:  {"markitect-extract/SKILL.md", "markitect-modeling/SKILL.md", "markitect-modeling/reference.md", "markitect-modeling/examples.md"},
}

// Chat renders a chat agent's documents (concept 2.0 §8): Base holds its role,
// the rules the app enforces and its skill; Task holds the conversation so
// far, which precedes the owner's first message to this agent. A chat agent
// has no Manager tools; it changes files in its copy, and the app proposes
// what it changed.
func Chat(in ChatInput) (Document, error) {
	paths, ok := chatSkills[in.Skill]
	if !ok {
		return Document{}, fmt.Errorf("harness: unknown chat skill %q", in.Skill)
	}
	text := map[string][]byte{}
	for _, f := range skills.Files() {
		text[f.Path] = f.Data
	}
	var b strings.Builder
	b.WriteString(`# Markitect chat

You talk with the owner of a project about its Markitect model and change the model with them. /work is a copy of the owner's working tree, with their uncommitted changes. What you change there reaches the owner as one proposal, which they read in Markitect's views and accept into their working tree or reject. Nothing reaches their files before they accept.

- Change only files below .markitect/: the model, the registry pages, the ignore page and execution.yaml. A change anywhere else is not proposed.
- Do not commit and do not move Git refs. The app reads your changes from the working tree of /work against ` + ChatStart + `, the state the owner has.
- Ask the owner when something is open, and write that part only after their answer. Your answer ends your turn; the owner's next message starts the next one.
- Read the model with markitect check, refs, graph and lint, and see what your change reaches with markitect impact --base ` + ChatStart + `.
- Keep each answer short: what you changed and why, or the one question you need answered.

`)
	for _, p := range paths {
		fmt.Fprintf(&b, "## Skill file %s\n\n%s\n\n", p, strings.TrimSpace(string(text[p])))
	}
	doc := Document{Base: b.String()}
	if in.Earlier != "" {
		doc.Task = "The conversation so far, with an earlier agent:\n\n" + strings.TrimSpace(in.Earlier) + "\n\nThe owner's next message:"
	}
	return doc, nil
}
