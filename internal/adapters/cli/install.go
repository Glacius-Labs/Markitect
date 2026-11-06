package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/skills"
)

// install writes the personal skill from the binary, so that its text cannot
// drift. Only something outside a project reaches a session in a new folder; init
// stays in the project and never writes here.
func (c CLI) install(args []string) (int, error) {
	fs := flag.NewFlagSet("markitect install", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	// Claude Code keeps its own folder where CLAUDE_CONFIG_DIR says, if set.
	config := os.Getenv("CLAUDE_CONFIG_DIR")
	if config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Misused, err
		}
		config = filepath.Join(home, ".claude")
	}
	file := filepath.Join(config, "skills", skills.PersonalName, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return Misused, err
	}
	if err := os.WriteFile(file, skills.Personal(), 0o644); err != nil {
		return Misused, err
	}
	fmt.Fprintf(c.Stdout, "wrote %s\nClaude Code now knows what Markitect is, in every folder. Run this again after an upgrade.\n", file)
	return OK, nil
}
