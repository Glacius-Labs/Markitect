package projectwork

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

// A test that installs this binary on PATH under the name git sets these
// variables; the binary then records each call and runs the real Git.
const (
	gitCountLogEnv = "MARKITECT_TEST_GIT_COUNT_LOG"
	realGitEnv     = "MARKITECT_TEST_REAL_GIT"
)

// TestMain shields git in these tests from the machine's Git configuration.
func TestMain(m *testing.M) {
	if strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") == "git" && os.Getenv(gitCountLogEnv) != "" {
		os.Exit(runCountingGit(os.Args[1:]))
	}
	testkit.Main(m)
}

func runCountingGit(args []string) int {
	log, err := os.OpenFile(os.Getenv(gitCountLogEnv), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err == nil {
		_, err = log.WriteString(strings.Join(args, " ") + "\n")
		if closeErr := log.Close(); err == nil {
			err = closeErr
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "counting git:", err)
		return 128
	}
	cmd := exec.Command(os.Getenv(realGitEnv), args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "counting git:", err)
		return 128
	}
	return 0
}
