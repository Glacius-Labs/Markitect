package projectapp

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

func TestDoctorGitVersionBoundaryAndDistributionSuffixes(t *testing.T) {
	for _, tc := range []struct {
		version string
		status  string
	}{
		{"git version 1.99.0", "blocked"},
		{"git version 2.39.99", "blocked"},
		{"git version 2.39.3 (Apple Git-145)", "blocked"},
		{"git version 2.40.0", "available"},
		{"git version 2.40", "available"},
		{"git version 2.52.0.windows.1", "available"},
		{"git version 2.40.1 (Apple Git-200)", "available"},
		{"git version 3.0.0", "available"},
		{"git version 2.40broken", "blocked"},
		{"git version 999999999999999999999999.0", "blocked"},
		{"unrelated version 9.0.0", "blocked"},
		{"", "blocked"},
	} {
		t.Run(tc.version, func(t *testing.T) {
			check := gitVersionCheck("selected-root", func(root string) ([]byte, error) {
				if root != "selected-root" {
					t.Fatalf("version probe root = %q", root)
				}
				return []byte(tc.version + "\n"), nil
			})
			if check.Name != "git-version" || check.Status != tc.status || (tc.version != "" && !strings.Contains(check.Detail, tc.version)) {
				t.Fatalf("Git prerequisite = %+v", check)
			}
			if tc.status == "blocked" && (!strings.Contains(check.Detail, "Git 2.40") || !strings.Contains(check.Detail, "check-attr --source")) {
				t.Fatalf("blocked diagnostic omits the requirement: %+v", check)
			}
		})
	}
}

func TestDoctorRejectsUnsupportedGitBeforeModelOrProviderInspection(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version string
		err     error
	}{
		{"old Git", "git version 2.39.5", nil},
		{"missing Git", "", errors.New("git executable not found")},
		{"unreadable version", "unknown", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// This directory has no Git repository or model. Reaching the loader
			// would yield a different error and lose the Git prerequisite report.
			root := testkit.TempDir(t)
			report, err := NewOperations().doctor(DoctorOperation{
				Root: root, Options: projectsetup.Options{Provider: "process", ProviderExecutable: filepath.Join(root, "must-not-inspect")},
			}, func(string) ([]byte, error) { return []byte(tc.version), tc.err })
			if err != nil || len(report.Checks) != 1 || report.Checks[0].Status != "blocked" || !strings.Contains(report.Checks[0].Detail, "Git 2.40") || report.Authentication != "not-verified" {
				t.Fatalf("unsupported Git report=%+v error=%v", report, err)
			}
			entries, readErr := os.ReadDir(root)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("doctor wrote to the selected root: %v %v", entries, readErr)
			}
		})
	}
}

func TestDoctorReportsInstalledGitWithoutWritingOrStartingAgents(t *testing.T) {
	repo := testkit.NewRepo(t)
	repo.Git("switch", "-c", "doctor-fixture")
	plan, err := projectwork.Init(repo.Dir, "doctor-fixture", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range plan.Files {
		repo.Write(file.Path, file.Content)
	}
	repo.Commit("model")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	invoker := &countingInvoker{}
	operations := NewOperations()
	operations.Invoker = invoker
	report, err := operations.Doctor(DoctorOperation{Root: repo.Dir, Options: projectsetup.Options{Provider: "process", ProviderExecutable: executable}})
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	want := repo.Git("--version")
	found := false
	for _, check := range report.Checks {
		if check.Name == "git-version" {
			found = check.Status == "available" && check.Detail == want
		}
	}
	if !found || invoker.calls != 0 || repo.Git("status", "--porcelain") != "" {
		t.Fatalf("doctor report=%+v invocations=%d", report, invoker.calls)
	}
	for _, path := range []string{".markitect/state", ".markitect/runs"} {
		if _, err := os.Stat(filepath.Join(repo.Dir, filepath.FromSlash(path))); !os.IsNotExist(err) {
			t.Fatalf("doctor created %s: %v", path, err)
		}
	}
}
