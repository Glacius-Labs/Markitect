package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"markitect/internal/source"
)

type GateResult struct {
	Name         string `yaml:"name"`
	ExitCode     int    `yaml:"exitCode"`
	Milliseconds int64  `yaml:"milliseconds"`
	Output       string `yaml:"output,omitempty"`
}

// VerifyRepository runs the existing repository checks from the same immutable
// files used by the graph. This deliberately never runs the recursive wrapper
// that invokes Markitect itself. Profiles map to reviewed, fixed command arrays.
func VerifyRepository(p *Project) ([]GateResult, error) {
	if p.Snapshot.Provisional {
		return nil, fmt.Errorf("verify requires --revision; working trees cannot provide immutable evidence")
	}
	var commands [][]string
	switch p.Graph.Project.Spec.Profile {
	case "konfyra":
		commands = [][]string{{"-B", "scripts/render-governance-adapters.py", "--check"}, {"-B", "-m", "unittest", "discover", "-s", "scripts/tests", "-v"}}
	case "cockpit":
		commands = [][]string{{"-B", "scripts/check_docs.py"}, {"-B", "scripts/render_adapters.py", "--check"}}
	default:
		return nil, fmt.Errorf("profile has no repository gate adapter; check covers the Markitect graph only")
	}
	python, err := exec.LookPath("python")
	if err != nil {
		python, err = exec.LookPath("python3")
	}
	if err != nil {
		return nil, fmt.Errorf("Python is required by this repository profile: %w", err)
	}
	temporary, err := os.MkdirTemp("", "markitect-verify-")
	if err != nil {
		return nil, err
	}
	// Only our newly allocated temporary directory is removed.
	defer os.RemoveAll(temporary)
	temporary, err = filepath.EvalSymlinks(temporary)
	if err != nil {
		return nil, err
	}
	if err = source.Materialize(p.Snapshot, temporary); err != nil {
		return nil, err
	}
	results := make([]GateResult, 0, len(commands))
	for _, args := range commands {
		start := time.Now()
		cmd := exec.Command(python, args...)
		cmd.Dir = temporary
		output, runErr := cmd.CombinedOutput()
		code := 0
		if runErr != nil {
			if failure, ok := runErr.(*exec.ExitError); ok {
				code = failure.ExitCode()
			} else {
				return results, runErr
			}
		}
		if len(output) > 1<<20 {
			return results, fmt.Errorf("repository check output exceeded 1 MiB; no complete evidence was recorded")
		}
		results = append(results, GateResult{Name: args[1], ExitCode: code, Milliseconds: time.Since(start).Milliseconds(), Output: string(output)})
	}
	return results, nil
}
