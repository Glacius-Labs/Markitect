//go:build native && windows

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// A failure in Wails ends the process, so the test binary starts itself as the
// app with these set.
const (
	failureEnv = "MARKITECT_TEST_WAILS_FAILURE"
	folderEnv  = "MARKITECT_TEST_CONFIG_FOLDER"
)

func TestMain(m *testing.M) {
	if failure := os.Getenv(failureEnv); failure != "" {
		failInWails(failure, os.Getenv(folderEnv))
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// failInWails has Wails, under the app's own options, report a failure. Nobody
// sees it: there is no tray, and the window is hidden and never gets to show.
func failInWails(failure, dir string) {
	opts := setUpWails(dir)
	switch failure {
	case "no web view":
		// The folder holds no web view, so WebView2 fails as on a machine
		// without it, and Wails ends the app the way it does when the 60
		// seconds run out. This also holds the line a later Wails might change:
		// stdLog looks for it.
		opts.Windows.WebviewBrowserPath = dir
		app := application.New(opts)
		app.Window.NewWithOptions(application.WebviewWindowOptions{Hidden: true})
		_ = app.Run()
	case "fatal":
		application.New(opts)
		application.Fatal("a fatal error")
	case "non-fatal":
		application.New(opts)
		application.NewWindow(application.WebviewWindowOptions{Hidden: true}).Error("an error it lives through")
	}
}

func TestAWailsFailureLeavesALineInTheErrorLog(t *testing.T) {
	tests := []struct {
		failure string
		want    string
		exit    int
	}{
		{"no web view", "no webview2 found", 1},
		{"fatal", "a fatal error", 1},
		{"non-fatal", "an error it lives through", 0},
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.failure, func(t *testing.T) {
			dir := t.TempDir()
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, self)
			cmd.Env = append(os.Environ(), failureEnv+"="+tt.failure, folderEnv+"="+dir)
			out, _ := cmd.CombinedOutput()
			if code := cmd.ProcessState.ExitCode(); code != tt.exit {
				t.Errorf("the app ended with %d, want %d\n%s", code, tt.exit, out)
			}
			log, err := os.ReadFile(filepath.Join(dir, "desktop-error.log"))
			if err != nil {
				t.Fatalf("no record: %v\n%s", err, out)
			}
			t.Logf("the log:\n%s", log)
			lines := strings.Split(strings.TrimSuffix(string(log), "\n"), "\n")
			if last := lines[len(lines)-1]; !strings.HasSuffix(last, tt.want) {
				t.Errorf("the last line is %q, want it to end in %q", last, tt.want)
			}
			if strings.Contains(string(log), "FATAL") {
				t.Errorf("the log holds Wails' banner:\n%s", log)
			}
		})
	}
}
