// Package cli presents preparation and Apply. It owns command parsing and
// reports, not repository processes or execution policy.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/modeldocs"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/tooling/licenses"
)

// RevisionResolver resolves an explicitly selected ref exactly once. Only its
// full commit identity is passed to the inner fixed-preview source contract.
type RevisionResolver interface {
	ResolveRevision(context.Context, string, string) (string, error)
}

// Documents owns v1 view generation and its independent source/target write
// binding. PreviewSource never supplies a publication guard for this operation.
type Documents interface {
	Preview(context.Context, apply.PreviewRequest) (modeldocs.Preview, error)
	Write(context.Context, apply.PreviewRequest, string) (modeldocs.Preview, error)
}

type Command struct {
	preparation *apply.PreparationService
	agents      apply.AgentProfilePreparer
	checks      apply.CheckProfilePreparer
	revisions   RevisionResolver
	documents   Documents
	apply       ApplyFunc
	extract     ExtractFunc
	setup       SetupFunc
	cockpit     CockpitFunc
	window      WindowOpener
}

func New(preparation *apply.PreparationService, agents apply.AgentProfilePreparer, checks apply.CheckProfilePreparer, revisions RevisionResolver, documents Documents, applyRun ApplyFunc) *Command {
	return &Command{preparation: preparation, agents: agents, checks: checks, revisions: revisions, documents: documents, apply: applyRun}
}

// WithExtract installs the Extract operation. Without it the command reports
// that Extract is unavailable.
func (c *Command) WithExtract(extract ExtractFunc) *Command {
	c.extract = extract
	return c
}

type manifestList []string

func (s *manifestList) String() string         { return fmt.Sprint([]string(*s)) }
func (s *manifestList) Set(value string) error { *s = append(*s, value); return nil }

const help = `Markitect: canonical v1 model, preparation and recursive Apply

  check [options]           structural model, registry and file accounting
  context <package> [options]  actual static selection and inclusion reasons
  docs [options]            preview a generated v1 model view
  apply [options]          realize the bound model: direct or staged work, then Verify
  extract [options]        derive a model candidate from the existing artifacts
  setup --runtime DIR --image REF --credential FILE --agent-model M [options]
                            prepare a Runtime directory with runtime.yaml and run.json
  cockpit --state DIR | --run-config PATH [--repo PATH] [--listen ADDR] [--no-open]
                            open the app window and observe runs in a local web view
  version | licenses | help

Common options:
  --repo PATH              repository root (default .)
  --model PATH             explicit model.yaml (default .markitect/model.yaml)
  --dependency PATH        explicit dependency model.yaml; repeat as needed
  --revision REF           resolve once; inspect exactly that fixed commit
  --metadata               request static profile/settings metadata preparation
  --runtime-profile PATH   explicit absolute Runtime profile document path

Without --metadata, metadata and settings are not requested. A structural pass
does not establish static profile metadata, rendered role context or runtime
support. Fixed previews refuse unbound present-day external Runtime inputs.

docs options:
  --write --expect PREVIEW_DIGEST
  contentDigest identifies rendered bytes; previewDigest binds their sources
  and observed target. Neither is the canonicalModelDigest.
  Writes require every acquired source to be repository-bound. External
  model/control/instruction sources remain preview-only because target
  disjointness has not been established.

apply options:
  --run-config PATH        explicit absolute closed JSON invocation configuration
  --runtime-profile PATH   explicit absolute versioned Runtime profile document
  --repo, --model and repeated --dependency select committed capture inputs.
  Revision previews and --metadata are preparation options, not apply flags.
  Configuration supplies state storage, helper resources, harvest bounds,
  productBuild assertion and dedicated credential references. No ambient defaults.
  The JSON report lists the preparation, the run (passes with each Manager's
  state, basis, result and verdict; findings, questions, problems, turns,
  usage), the publication receipt, the Run outcome (success, failed or
  incomplete) and the evidence directory. Evidence is kept under
  STATE/runs/RUN; only one run holds a state directory. A Run is closed: no
  question or answer crosses Root to the user or the invoking AI, and no
  preloaded owner decision supplies model intent. A Run that needs a material
  decision the model lacks ends failed without publication and reports it;
  amend the model and start a new Run.

extract options:
  --run-config PATH, --runtime-profile PATH and repeated --dependency as for apply;
  --repo selects the repository. There is no --model: the Orchestrator prepares the
  structural manifest and lock .markitect/model.yaml with no local Definitions,
  the imported vocabulary, and .markitect/execution.yaml with budget, profiles,
  ownership areas of the artifacts and checks. Phase one proposes the Packages
  and their ownership areas, phase two derives the model of every Package and
  registers the artifacts. Artifacts are never changed or published; a verified
  model is published as the changed .markitect model, registry shards and
  execution.yaml (ownership and proposed checks). The JSON report lists model
  completion, the final Verify Pass, the realization deviations per Definition
  (artifact findings), the proposedChecks, open questions, per-phase passes,
  turns and usage and the publication. An Extract is closed like an Apply.

setup options:
  --runtime DIR            absolute Runtime directory, outside the repository; created
                           private and holding runtime.yaml, run.json and state/
  --image REF              runtime image; its ID and metadata are observed, never started
  --credential FILE        absolute private Claude runtime token file; checked, never read
  --check PROFILE=EXECUTABLE  absolute executable in the image for each check profile;
                           repeat, one per profile the project selects
  --agent-model M          model of every Agent profile, required (for example claude-haiku-5-5)
  --effort E               low, medium or high (default high)
  --engine PATH            container engine executable (default /usr/bin/docker)
  --store LABEL            repository identity and store label (default: the model id)
  --request TEXT           owner request (default: a neutral sentence)
  --deadline RFC3339       absolute UTC wall-clock limit, ending in Z
  --repo, --model and repeated --dependency select the project as for check.
  Setup reads the profile names from the working draft, writes the two files
  deterministically (unchanged files stay) and prints the files with their
  SHA-256, the bound identities and the exact apply command. It refuses while an
  Apply holds the state directory.

cockpit options:
  --state DIR              state directory whose runs to observe; repeat
  --run-config PATH        observe the state directory of this run configuration
  --repo PATH              repository to observe and propose settings for (default .)
  --model PATH, --dependency PATH
                           the model and dependency models, as for check
  --runtime-profile PATH   with --run-config: enable model sessions with this Runtime profile
  --listen ADDR            loopback address (default 127.0.0.1:7341)
  Serves a local web view of the recorded and running runs until interrupted.
  It reads the evidence under STATE/runs. It writes only a settings or model
  proposal the owner accepted after its structural check passed, through
  Apply's guarded publication while no observed Apply runs; it answers no
  question of a run, commits nothing, starts and stops no run, and replay,
  pause or reconnect never affect a run. With --runtime-profile, the owner's
  prompts start or continue a model session: an Agent of Root's Execute profile
  in a container that may change only .markitect/model and .markitect/registry
  of its workspace and proposes; each prompt is one provider turn.

Exit codes: 0 requested stage passed; 1 known failure; 2 usage;
3 incomplete observation, I/O/budget refusal or unavailable operation.
`

func (c *Command) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		_, err := io.WriteString(stdout, help)
		if err != nil {
			return 3
		}
		return 0
	}
	verb := args[0]
	emit := func(value any) int {
		e := json.NewEncoder(stdout)
		e.SetIndent("", "  ")
		if err := e.Encode(reportValue(value)); err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		return 0
	}
	if verb == "version" {
		return emit(map[string]string{"version": buildinfo.Version()})
	}
	if verb == "licenses" {
		_, err := io.WriteString(stdout, licenses.Text)
		if err != nil {
			return 3
		}
		return 0
	}
	if verb == "setup" {
		return c.setupVerb(ctx, args[1:], stdout, stderr, emit)
	}
	if verb == "cockpit" {
		return c.cockpitVerb(ctx, args[1:], stdout, stderr, emit)
	}
	if verb == "apply" || verb == "extract" {
		return c.execution(ctx, verb, args[1:], stdout, stderr, emit)
	}
	if verb != "check" && verb != "context" && verb != "docs" {
		fmt.Fprintln(stderr, "unknown operation:", verb)
		return 2
	}
	args = args[1:]
	packagePath := ""
	if verb == "context" {
		if len(args) == 0 || len(args[0]) == 0 || args[0][0] == '-' {
			fmt.Fprintln(stderr, "context requires one exact local Package path (or .)")
			return 2
		}
		packagePath, args = args[0], args[1:]
	}
	flags := flag.NewFlagSet(verb, flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "repository root")
	manifest := flags.String("model", ".markitect/model.yaml", "explicit model.yaml")
	revision := flags.String("revision", "", "explicit fixed revision")
	metadata := flags.Bool("metadata", false, "request static metadata and settings")
	runtimeProfile := flags.String("runtime-profile", "", "explicit absolute Runtime profile document path")
	write := flags.Bool("write", false, "write the guarded generated view")
	expect := flags.String("expect", "", "expected source/target-bound previewDigest")
	var dependencies manifestList
	flags.Var(&dependencies, "dependency", "explicit dependency model.yaml; repeat")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			io.WriteString(stdout, help)
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || verb != "docs" && (*write || *expect != "") || verb == "docs" && *metadata || *runtimeProfile != "" && !*metadata && verb != "docs" || *write && *expect == "" || !*write && *expect != "" {
		fmt.Fprintln(stderr, "invalid option combination; see help")
		return 2
	}
	root, err := filepath.Abs(*repo)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	selection := apply.InputSelection{Repository: root}
	selection.Local, err = selectManifest(root, *manifest)
	if err == nil {
		for _, dependency := range dependencies {
			selected, e := selectManifest(root, dependency)
			if e != nil {
				err = e
				break
			}
			selection.Dependencies = append(selection.Dependencies, selected)
		}
	}
	if err == nil && *runtimeProfile != "" {
		if !filepath.IsAbs(*runtimeProfile) || filepath.Clean(*runtimeProfile) != *runtimeProfile {
			err = errors.New("--runtime-profile requires an explicit absolute native document path")
		} else {
			var p snapshot.Path
			p, err = snapshot.NewPath(filepath.Base(*runtimeProfile))
			selection.RuntimeProfiles = []apply.SourceLocation{{Boundary: filepath.Dir(*runtimeProfile), Path: p}}
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	resolved := ""
	if *revision != "" {
		if c.revisions == nil {
			emit(map[string]string{"operation": verb, "status": "not_completed", "error": "revision resolver unavailable"})
			return 3
		}
		resolved, err = c.revisions.ResolveRevision(ctx, root, *revision)
		if err != nil {
			emit(map[string]string{"operation": verb, "status": "not_completed", "error": err.Error()})
			return 3
		}
	}
	request := apply.PreviewRequest{Selection: selection, Revision: resolved}
	if verb == "docs" {
		if c.documents == nil {
			emit(map[string]string{"operation": "docs", "status": "not_completed", "error": "document service unavailable"})
			return 3
		}
		var preview modeldocs.Preview
		if *write {
			preview, err = c.documents.Write(ctx, request, *expect)
		} else {
			preview, err = c.documents.Preview(ctx, request)
		}
		report := map[string]any{"operation": "docs", "requestedRevision": *revision, "resolvedRevision": resolved, "view": preview}
		if err != nil {
			report["error"] = err.Error()
		}
		if emit(report) != 0 {
			return 3
		}
		return exitStatus(preview.Status, err)
	}
	report, err := c.prepare(ctx, request, verb, packagePath, *metadata)
	report.RequestedRevision = *revision
	if err != nil {
		report.Error = err.Error()
	}
	if emit(report) != 0 {
		return 3
	}
	return exitStatus(report.Status, err)
}

func selectManifest(repository, literal string) (apply.ModelSelection, error) {
	// Relative manifest coordinates belong to the selected repository. Keep
	// that relationship explicit instead of passing a second absolute spelling
	// of its directory (for example a Windows short-name alias).
	boundary := "."
	if filepath.IsAbs(literal) {
		if filepath.Clean(literal) != literal {
			return apply.ModelSelection{}, errors.New("model manifest path cannot require normalization")
		}
		boundary, literal = filepath.Dir(literal), filepath.Base(literal)
	}
	p, err := snapshot.NewPath(literal)
	if err != nil {
		return apply.ModelSelection{}, err
	}
	return apply.ModelSelection{Boundary: boundary, ManifestPath: p}, nil
}

func exitStatus(status model.Status, err error) int {
	if status == model.Failed {
		return 1
	}
	if err != nil || status != model.Passed {
		return 3
	}
	return 0
}
