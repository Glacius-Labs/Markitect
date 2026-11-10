package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/guard"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// worktreeBase is the base of the worktrees Claude Code makes, which init sets so
// that a builder in a worktree of its own starts from the owner's branch and not
// from the repository's default branch.
const worktreeBase = "head"

// EnsureSettings adds the hooks to the project's settings, in place of ones an
// earlier markitect wrote, sets the base of Claude Code's worktrees unless the
// settings name one, and keeps what else they hold. Settings it cannot read it
// leaves as they are; a part of them with an unexpected shape it leaves as it is
// and writes the other. Either way it says what to add by hand. It returns the
// lines to say, in order, also when it fails.
func EnsureSettings(f Folder) ([]string, error) {
	settings := map[string]any{}
	addHooks := fmt.Sprintf("add the hooks by hand: %q before Bash for %s, before %s and before %s", guard.HookCommand, guard.MarkRule, guard.EditTools, guard.AgentTools)
	setBase := fmt.Sprintf("set worktree.baseRef to %q by hand", worktreeBase)
	lacks := func(why string) []string {
		return []string{fmt.Sprintf("left %s as it is, %s; %s, and %s", guard.SettingsFile, why, addHooks, setBase)}
	}
	guarded := false
	if f.Exists(guard.SettingsFile) {
		data, err := f.Read(guard.SettingsFile)
		if err != nil {
			return nil, err
		}
		guarded = guard.HasGuard(data)
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		if err := dec.Decode(&settings); err != nil {
			return lacks(fmt.Sprintf("because it is not valid JSON (%v)", err)), nil
		}
		if _, err := dec.Token(); err != io.EOF {
			return lacks("because it holds more than one JSON value"), nil
		}
		if settings == nil {
			return lacks("because it is not a JSON object"), nil
		}
	}
	var said, left []string
	if !guarded {
		if hooks, why := guard.GuardedHooks(settings); why != "" {
			left = append(left, fmt.Sprintf("left the hooks in %s as they are, because %s; %s", guard.SettingsFile, why, addHooks))
		} else {
			settings["hooks"] = hooks
			said = append(said, fmt.Sprintf("added the hooks that hold the agents' rights to %s", guard.SettingsFile))
		}
	}
	setBaseRef := fmt.Sprintf("set worktree.baseRef to %q in %s, so that a builder's worktree starts from your branch", worktreeBase, guard.SettingsFile)
	switch w := settings["worktree"].(type) {
	case nil: // absent, or null
		settings["worktree"] = map[string]any{"baseRef": worktreeBase}
		said = append(said, setBaseRef)
	case map[string]any:
		if w["baseRef"] == nil {
			w["baseRef"] = worktreeBase
			said = append(said, setBaseRef)
		}
	default:
		left = append(left, fmt.Sprintf("left worktree in %s as it is, because it is not an object; %s", guard.SettingsFile, setBase))
	}
	if len(said) == 0 {
		return left, nil
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(settings); err != nil {
		return left, err
	}
	if err := f.Write(guard.SettingsFile, b.Bytes()); err != nil {
		return left, err
	}
	return append(left, said...), nil
}

// InstallAttributes has Git merge the record by keeping the lines of both sides
// (DEC-099): status takes the line that matches the folder.
func InstallAttributes(f Folder) error {
	return f.Write(project.AttributesFile, []byte(project.Attributes))
}

// GitIgnoreFile is Git's ignore file at the project's root, and worktreeIgnore
// the lines EnsureIgnore adds to it: the worktrees Claude Code makes for the
// builders and the parallel runs lie in the project's folder, and are checkouts
// of their own, not files to commit.
const (
	GitIgnoreFile  = ".gitignore"
	worktreeIgnore = "# the worktrees Claude Code makes, as for Markitect's builders\n" + project.WorktreeDir + "\n"
)

// EnsureIgnore adds project.WorktreeDir to the project's .gitignore, unless a line
// there already ignores it, or un-ignores it with !, which is the owner's choice.
// It keeps the rest of the file and its line endings, and says whether it added
// the line.
func EnsureIgnore(f Folder) (bool, error) {
	var text string
	if f.Exists(GitIgnoreFile) {
		data, err := f.Read(GitIgnoreFile)
		if err != nil {
			return false, err
		}
		text = string(data)
	}
	for _, line := range strings.Split(text, "\n") {
		pattern := strings.TrimPrefix(strings.TrimSpace(line), "!")
		pattern = strings.TrimPrefix(strings.TrimPrefix(pattern, "**/"), "/")
		pattern = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(pattern, "**"), "*"), "/")
		if pattern == strings.TrimSuffix(project.WorktreeDir, "/") || pattern == ".claude" {
			return false, nil
		}
	}
	nl := "\n"
	if strings.Contains(text, "\r\n") {
		nl = "\r\n"
	}
	add := strings.ReplaceAll(worktreeIgnore, "\n", nl)
	if text != "" {
		// A blank line apart from what is there, after its last line ends.
		add = nl + add
		if !strings.HasSuffix(text, "\n") {
			add = nl + add
		}
	}
	return true, f.Write(GitIgnoreFile, []byte(text+add))
}
