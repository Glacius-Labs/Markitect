// Package cli presents preparation and controlled execution. It owns
// command parsing and reports, not repository processes or execution policy.
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
	execute     ExecuteFunc
}

func New(preparation *apply.PreparationService, agents apply.AgentProfilePreparer, checks apply.CheckProfilePreparer, revisions RevisionResolver, documents Documents, execute ExecuteFunc) *Command {
	return &Command{preparation, agents, checks, revisions, documents, execute}
}

type manifestList []string

func (s *manifestList) String() string         { return fmt.Sprint([]string(*s)) }
func (s *manifestList) Set(value string) error { *s = append(*s, value); return nil }

const help = `Markitect: canonical v1 model, preparation and controlled Apply

  check [options]           structural model, registry and file accounting
  context <package> [options]  actual static selection and inclusion reasons
  docs [options]            preview a generated v1 model view
  apply [options]          execute controlled artifact Apply from committed inputs
  answer <question> [options]  record an owner answer and continue that Apply
  version | licenses | help

Common options:
  --repo PATH              repository root (default .)
  --model PATH             explicit model.yaml (default .markitect/model.yaml)
  --dependency PATH        explicit dependency model.yaml; repeat as needed
  --revision REF           resolve once; inspect exactly that fixed commit
  --metadata               request static profile/settings metadata preparation
  --host-profile PATH      explicit absolute Host profile DOCUMENT path

Without --metadata, metadata and settings are not requested. A structural pass
does not establish static profile metadata, rendered role context or runtime
support. Fixed previews refuse unbound present-day external Host inputs.

docs options:
  --write --expect PREVIEW_DIGEST
  contentDigest identifies rendered bytes; previewDigest binds their sources
  and observed target. Neither is the canonicalModelDigest.
  Writes require every acquired source to be repository-bound. External
  model/control/instruction sources remain preview-only because target
  disjointness has not been established.

apply / answer options:
  --run-config PATH        explicit absolute closed JSON invocation configuration
  --host-profile PATH      explicit absolute versioned Host profile document
  --text ANSWER            required for answer; reuse the original run-config
  --repo, --model and repeated --dependency select committed capture inputs.
  Revision previews and --metadata are preparation options, not execution flags.
  Configuration supplies explicit budgets, state storage, helper resources,
  productBuild assertion and dedicated credential references. No ambient defaults.

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
	if verb == "apply" || verb == "answer" {
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
	host := flags.String("host-profile", "", "explicit absolute Host profile document path")
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
	if flags.NArg() != 0 || verb != "docs" && (*write || *expect != "") || verb == "docs" && *metadata || *host != "" && !*metadata && verb != "docs" || *write && *expect == "" || !*write && *expect != "" {
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
	if err == nil && *host != "" {
		if !filepath.IsAbs(*host) || filepath.Clean(*host) != *host {
			err = errors.New("--host-profile requires an explicit absolute native document path")
		} else {
			var p snapshot.Path
			p, err = snapshot.NewPath(filepath.Base(*host))
			selection.HostProfiles = []apply.SourceLocation{{Boundary: filepath.Dir(*host), Path: p}}
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
