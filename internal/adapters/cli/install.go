package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/skills"
)

// claudeConfigDir is Claude Code's own folder: where CLAUDE_CONFIG_DIR says, if
// set, and otherwise .claude in the home folder.
func claudeConfigDir() (string, error) {
	if config := os.Getenv("CLAUDE_CONFIG_DIR"); config != "" {
		return config, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}

// personalSkillFile is where the personal skill lives.
func personalSkillFile() (string, error) {
	config, err := claudeConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(config, "skills", skills.PersonalName, "SKILL.md"), nil
}

// personalSkillState says how the personal skill in place compares with the text
// this binary would write.
func personalSkillState() (file string, state fileState, err error) {
	file, err = personalSkillFile()
	if err != nil {
		return "", current, err
	}
	data, err := os.ReadFile(file)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return file, missing, nil
	case err != nil:
		return file, current, err
	case sameText(data, skills.Personal()):
		return file, current, nil
	}
	return file, outdated, nil
}

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
	file, err := writePersonalSkill()
	if err != nil {
		return Misused, err
	}
	fmt.Fprintf(c.Stdout, "wrote %s\nClaude Code now knows what Markitect is, in every folder. After an upgrade, markitect update brings this file and your projects in line.\n", file)
	c.warnPath()
	return OK, nil
}

// writePersonalSkill writes the personal skill and returns where.
func writePersonalSkill() (string, error) {
	file, err := personalSkillFile()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", err
	}
	return file, os.WriteFile(file, skills.Personal(), 0o644)
}
