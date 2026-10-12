package projectapp

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/application/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectsetup"
)

const gitRequirement = "Git 2.40 or later is required: Explore, Plan and Apply use git check-attr --source."

var gitVersionPattern = regexp.MustCompile(`^git version ([0-9]+)\.([0-9]+)(?:\.[0-9]+)?(?:[-. ][^\r\n]+)?$`)

func readGitVersion(root string) ([]byte, error) {
	return source.GitOutput(root, "--version")
}

func gitVersionCheck(root string, readVersion func(string) ([]byte, error)) projectsetup.Check {
	check := projectsetup.Check{Name: "git-version", Status: "blocked"}
	output, err := readVersion(root)
	if err != nil {
		check.Detail = fmt.Sprintf("Cannot read the Git version: %v. %s", err, gitRequirement)
		return check
	}
	version := strings.TrimSpace(string(output))
	parts := gitVersionPattern.FindStringSubmatch(version)
	if parts == nil {
		check.Detail = fmt.Sprintf("Unrecognized Git version %q. %s", version, gitRequirement)
		return check
	}
	major, majorErr := strconv.ParseUint(parts[1], 10, 64)
	minor, minorErr := strconv.ParseUint(parts[2], 10, 64)
	if majorErr != nil || minorErr != nil {
		check.Detail = fmt.Sprintf("Unrecognized Git version %q. %s", version, gitRequirement)
		return check
	}
	if major < 2 || (major == 2 && minor < 40) {
		check.Detail = version + ". " + gitRequirement + " Upgrade Git before using those operations."
		return check
	}
	check.Status, check.Detail = "available", version
	return check
}

// Check Git before loading a model or inspecting a provider, so an unsupported
// prerequisite yields its own diagnostic even if later operations cannot run.
func (o Operations) doctor(operation DoctorOperation, readVersion func(string) ([]byte, error)) (projectsetup.DoctorReport, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectsetup.DoctorReport{}, err
	}
	gitCheck := gitVersionCheck(operation.Root, readVersion)
	if gitCheck.Status != "available" {
		return projectsetup.DoctorReport{
			APIVersion: "markitect.example.org/project-doctor/v1alpha1", Provider: operation.Options.Provider,
			Authentication: "not-verified", Checks: []projectsetup.Check{gitCheck},
		}, nil
	}
	project, err := projectwork.Load(operation.Root, "")
	if err != nil {
		return projectsetup.DoctorReport{}, err
	}
	report, err := projectsetup.Doctor(project, operation.Options)
	report.Checks = append(report.Checks, gitCheck)
	sort.Slice(report.Checks, func(i, j int) bool { return report.Checks[i].Name < report.Checks[j].Name })
	return report, err
}
