package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type event struct {
	AtUTC string   `json:"atUtc"`
	Phase string   `json:"phase"`
	Args  []string `json:"args"`
}

func main() {
	realGit := os.Getenv("MARKITECT_G5_REAL_GIT")
	if realGit == "" {
		fatal(errors.New("MARKITECT_G5_REAL_GIT is required"))
	}
	args := os.Args[1:]
	mode := os.Getenv("MARKITECT_G5_GIT_GATE")
	phase := ""
	if mode != "" && isUpdateRef(args) {
		phase = mode
	}
	if phase == "before" {
		writeEvent("attempt", args)
		pause("before")
	}
	cmd := exec.Command(realGit, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		fatal(err)
	}
	if isUpdateRef(args) {
		writeEvent("delegated", args)
	}
	if phase == "after" {
		pause("after")
	}
}

func isUpdateRef(args []string) bool {
	for _, arg := range args {
		if arg == "update-ref" {
			return true
		}
	}
	return false
}

func pause(phase string) {
	gate := os.Getenv("MARKITECT_G5_GATE_DIRECTORY")
	if gate == "" {
		fatal(errors.New("MARKITECT_G5_GATE_DIRECTORY is required"))
	}
	if err := os.MkdirAll(gate, 0o755); err != nil {
		fatal(err)
	}
	marker := filepath.Join(gate, "paused-"+phase)
	data, err := json.Marshal(event{AtUTC: time.Now().UTC().Format(time.RFC3339Nano), Phase: phase, Args: os.Args[1:]})
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(marker, append(data, '\n'), 0o600); err != nil {
		fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(gate, "release-"+phase)); err == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func writeEvent(phase string, args []string) {
	path := os.Getenv("MARKITECT_G5_GIT_TRACE")
	if path == "" {
		return
	}
	data, err := json.Marshal(event{AtUTC: time.Now().UTC().Format(time.RFC3339Nano), Phase: phase, Args: args})
	if err != nil {
		fatal(err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fatal(err)
	}
	defer file.Close()
	if _, err := fmt.Fprintln(file, strings.TrimSpace(string(data))); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
