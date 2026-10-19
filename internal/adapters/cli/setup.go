package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

// Defaults of the setup flags. The engine default is the conventional Linux
// Docker path. The product has no default model: the owner chooses it.
const (
	defaultSetupEngine = "/usr/bin/docker"
	defaultSetupEffort = "high"
)

// SetupRequest contains the parsed invocation. Credential is the path of a
// private token file; its bytes never enter a request or a report.
type SetupRequest struct {
	Selection      apply.InputSelection
	Host           string
	Image          string
	Credential     string
	Checks         map[string]string
	AgentModel     string
	Effort         string
	Engine         string
	Store          string
	OwnerRequest   string
	OwnerDecisions []string
	Deadline       string
	// ProductBuild is the build identity of this CLI, written into the run
	// configuration as the operator's assertion.
	ProductBuild string
}

// SetupBound lists the identities that setup observed and wrote into the Host
// files.
type SetupBound struct {
	ImageID       string `json:"imageID"`
	AgentdVersion string `json:"agentdVersion"`
	CLIVersion    string `json:"cliVersion"`
	EngineVersion string `json:"engineVersion"`
}

// SetupFile is one file setup composed. Action is "written" or "unchanged".
type SetupFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Action string `json:"action"`
}

// SetupReport states what setup bound and wrote and how to run Apply with it.
type SetupReport struct {
	FormatVersion int          `json:"formatVersion"`
	Operation     string       `json:"operation"`
	Status        model.Status `json:"status"`
	Bound         *SetupBound  `json:"bound,omitempty"`
	AgentProfiles []string     `json:"agentProfiles"`
	CheckProfiles []string     `json:"checkProfiles"`
	Files         []SetupFile  `json:"files"`
	ApplyCommand  string       `json:"applyCommand,omitempty"`
	Error         string       `json:"error,omitempty"`
}

type SetupFunc func(context.Context, SetupRequest) (SetupReport, error)

// WithSetup installs the setup operation. Without it the command reports that
// setup is unavailable.
func (c *Command) WithSetup(setup SetupFunc) *Command {
	c.setup = setup
	return c
}

// checkList collects repeated PROFILE=EXECUTABLE flags.
type checkList []string

func (s *checkList) String() string         { return fmt.Sprint([]string(*s)) }
func (s *checkList) Set(value string) error { *s = append(*s, value); return nil }

func (c *Command) setupVerb(ctx context.Context, args []string, stdout, stderr io.Writer, emit func(any) int) int {
	if c.setup == nil {
		emit(map[string]string{"operation": "setup", "status": "not_completed", "reason": "Setup is unavailable because its concrete composition is not configured."})
		return 3
	}
	flags := flag.NewFlagSet("setup", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "repository root")
	manifest := flags.String("model", ".markitect/model.yaml", "explicit model.yaml")
	host := flags.String("host", "", "absolute Host directory to prepare")
	image := flags.String("image", "", "runtime image reference")
	credential := flags.String("credential", "", "absolute path of the private Claude runtime token file")
	agentModel := flags.String("agent-model", "", "model of every Agent profile (required)")
	effort := flags.String("effort", defaultSetupEffort, "effort of every Agent profile")
	engine := flags.String("engine", defaultSetupEngine, "absolute container engine executable")
	store := flags.String("store", "", "store label (default: the model id)")
	ownerRequest := flags.String("request", "", "owner request text")
	deadline := flags.String("deadline", "", "absolute RFC 3339 UTC wall-clock limit")
	var dependencies manifestList
	var decisions manifestList
	var checkFlags checkList
	flags.Var(&dependencies, "dependency", "explicit dependency model.yaml; repeat")
	flags.Var(&decisions, "decision", "owner decision text; repeat")
	flags.Var(&checkFlags, "check", "PROFILE=EXECUTABLE for a check profile; repeat")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			io.WriteString(stdout, help)
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *host == "" || *image == "" || *credential == "" || *agentModel == "" {
		fmt.Fprintln(stderr, "setup requires --host, --image, --credential and --agent-model; see help")
		return 2
	}
	for _, path := range []string{*host, *credential, *engine} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			fmt.Fprintln(stderr, "--host, --credential and --engine require explicit absolute native paths")
			return 2
		}
	}
	checks := map[string]string{}
	for _, entry := range checkFlags {
		profile, executable, ok := strings.Cut(entry, "=")
		if _, duplicate := checks[profile]; !ok || profile == "" || executable == "" || duplicate {
			fmt.Fprintln(stderr, "--check requires one PROFILE=EXECUTABLE per check profile")
			return 2
		}
		checks[profile] = executable
	}
	root, err := filepath.Abs(*repo)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	selection := apply.InputSelection{Repository: root}
	selection.Local, err = selectManifest(root, *manifest)
	for _, dependency := range dependencies {
		if err != nil {
			break
		}
		var selected apply.ModelSelection
		selected, err = selectManifest(root, dependency)
		selection.Dependencies = append(selection.Dependencies, selected)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	report, err := c.setup(ctx, SetupRequest{
		Selection: selection, Host: *host, Image: *image, Credential: *credential, Checks: checks,
		AgentModel: *agentModel, Effort: *effort, Engine: *engine, Store: *store, OwnerRequest: *ownerRequest,
		OwnerDecisions: decisions, Deadline: *deadline, ProductBuild: buildinfo.Build(),
	})
	report.FormatVersion, report.Operation = 1, "setup"
	report.AgentProfiles, report.CheckProfiles = nonNil(report.AgentProfiles), nonNil(report.CheckProfiles)
	if report.Files == nil {
		report.Files = []SetupFile{}
	}
	if err != nil {
		report.Error = err.Error()
	} else if report.Status == model.Passed {
		report.ApplyCommand = applyCommand(root, *manifest, dependencies, filepath.Join(*host, "host.yaml"), filepath.Join(*host, "run.json"))
	}
	if emit(report) != 0 {
		return 3
	}
	return exitStatus(report.Status, err)
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// applyCommand is the exact apply invocation that uses the written files.
func applyCommand(repo, manifest string, dependencies []string, hostProfile, runConfig string) string {
	words := []string{"markitect", "apply", "--repo", repo}
	if manifest != ".markitect/model.yaml" {
		words = append(words, "--model", manifest)
	}
	for _, dependency := range dependencies {
		words = append(words, "--dependency", dependency)
	}
	words = append(words, "--host-profile", hostProfile, "--run-config", runConfig)
	for i, word := range words {
		if strings.ContainsAny(word, " \t\"'$`&;|<>()") {
			words[i] = quoteWord(word)
		}
	}
	return strings.Join(words, " ")
}

// quoteWord quotes one command line word for the shell of the current platform.
func quoteWord(word string) string {
	if runtime.GOOS == "windows" {
		return `"` + strings.ReplaceAll(word, `"`, `\"`) + `"`
	}
	return "'" + strings.ReplaceAll(word, "'", `'\''`) + "'"
}
