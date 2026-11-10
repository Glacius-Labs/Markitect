package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/guard"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// status says what is to verify, and changes nothing. Exit 1 means it found
// something: an item, an error in the spec or the realization, or a line of the
// record it cannot read; notes, such as a skip, never change the exit code. With
// --all every Definition is an item.
func (c CLI) status(ctx context.Context, args []string) (int, error) {
	var o options
	var all bool
	var pkg string
	fs := c.flags("status", &o)
	fs.BoolVar(&all, "all", false, "list every Definition, to verify everything again")
	fs.StringVar(&pkg, "package", "", "list only the items whose Definition lies directly in this Package, Root for the one at the top")
	if positional, err := parse(fs, args); err != nil {
		return Misused, err
	} else if len(positional) != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	st, err := c.Record.Status(ctx, run.StatusRequest{Dir: o.repo, All: all, Package: pkg})
	if err != nil {
		return Misused, err
	}
	if o.json {
		items, skipped := st.Items, st.Skipped
		if items == nil {
			items = []run.Item{}
		}
		if skipped == nil {
			skipped = []run.Item{}
		}
		unreadable := append([]int{}, st.Unreadable...)
		packages := append([]run.PackageItems{}, st.Packages...)
		if err := c.printJSON(struct {
			envelope
			Clean             bool               `json:"clean"`
			Package           string             `json:"package,omitempty"`
			NotListed         int                `json:"not_listed,omitempty"`
			Definitions       int                `json:"definitions"`
			SpecErrors        int                `json:"spec_errors"`
			RealizationErrors int                `json:"realization_errors"`
			Packages          []run.PackageItems `json:"packages"`
			Items             []run.Item         `json:"items"`
			Diagnostics       []diagnostic       `json:"diagnostics"`
			Skipped           []run.Item         `json:"skipped"`
			Unreadable        []int              `json:"unreadable_lines"`
		}{newEnvelope("status"), st.Clean(), st.Package, st.ErrorsElsewhere + st.UnreadableElsewhere, st.Definitions, len(st.SpecErrors), len(st.Errors), packages, items, append(append(diagnostics(severityError, st.SpecErrors), diagnostics(severityError, st.Errors)...), diagnostics(severityError, st.RecordErrors())...), skipped, unreadable}); err != nil {
			return Misused, err
		}
	} else {
		fmt.Fprint(c.Stdout, st.Text())
	}
	if !st.Clean() {
		return Failed, nil
	}
	return OK, nil
}

// markUsage is mark's usage line.
const markUsage = `markitect mark DEFINITION@FINGERPRINT VERDICT [-e EVIDENCE]... [-m "WHY"] [--cases N [--covered M]]`

// mark records a verdict at the state status printed. Exit 1 means it refused and
// recorded nothing: the inputs changed since, or evidence does not exist. The
// flags may stand before or after the item and the verdict.
func (c CLI) mark(ctx context.Context, args []string) (int, error) {
	req := run.MarkRequest{Dir: "."}
	var positional []string
	asJSON := false
	value := func(i *int, name string) (string, error) {
		if *i+1 >= len(args) {
			return "", fmt.Errorf("%s needs a value\nusage: %s", name, markUsage)
		}
		*i++
		return args[*i], nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, inline, hasInline := strings.Cut(strings.TrimLeft(a, "-"), "=")
		var v string
		var err error
		switch {
		case a == "-h" || a == "--help" || a == "-help":
			fmt.Fprintf(c.Stderr, "usage: %s\nverdicts: %s\n", markUsage, strings.Join(run.Verdicts, ", "))
			return Misused, flag.ErrHelp
		case !strings.HasPrefix(a, "-") || a == "-":
			positional = append(positional, a)
			continue
		case name == "json" && !hasInline:
			asJSON = true
			continue
		case hasInline:
			v = inline
		case name == "e" || name == "m" || name == "repo" || name == "cases" || name == "covered":
			v, err = value(&i, a)
		}
		if err != nil {
			return Misused, err
		}
		switch name {
		case "e":
			req.Evidence = append(req.Evidence, v)
		case "m":
			req.Why = v
		case "repo":
			req.Dir = v
		case "cases", "covered":
			n, err := strconv.Atoi(v)
			if err != nil {
				return Misused, fmt.Errorf("--%s takes a count, not %q\nusage: %s", name, v, markUsage)
			}
			if name == "cases" {
				req.Cases = &n
			} else {
				req.Covered = &n
			}
		default:
			return Misused, fmt.Errorf("unknown flag %s\nusage: %s", a, markUsage)
		}
	}
	if len(positional) != 2 {
		return Misused, fmt.Errorf("names the item, DEFINITION@FINGERPRINT as status prints it, and the verdict; give each piece of evidence with -e and why with -m\nusage: %s", markUsage)
	}
	req.Item, req.Verdict = positional[0], positional[1]
	text, recorded, err := c.Record.Mark(ctx, req)
	if err != nil {
		if errors.Is(err, run.ErrUsage) {
			return Misused, fmt.Errorf("%s\nusage: %s", strings.TrimPrefix(err.Error(), run.ErrUsage.Error()+": "), markUsage)
		}
		return Misused, err
	}
	if asJSON {
		if err := c.printJSON(struct {
			envelope
			Recorded bool   `json:"recorded"`
			Item     string `json:"item"`
			Verdict  string `json:"verdict"`
			Message  string `json:"message"`
		}{newEnvelope("mark"), recorded, req.Item, req.Verdict, strings.TrimSpace(text)}); err != nil {
			return Misused, err
		}
	} else {
		fmt.Fprint(c.Stdout, text)
	}
	if !recorded {
		return Failed, nil
	}
	return OK, nil
}

// hook is what the project's hook and the plugin's call before a tool call. It
// reads the provider's input on standard input and always exits 0: a problem of
// its own must not stop an agent.
func (c CLI) hook(_ context.Context, args []string) (int, error) {
	if len(args) != 1 || args[0] != "pretool" {
		return Misused, errors.New("names its event: markitect hook pretool")
	}
	input, err := io.ReadAll(c.Stdin)
	if err != nil {
		return OK, nil
	}
	if out := guard.Guard(input); len(out) > 0 {
		c.Stdout.Write(out)
	}
	return OK, nil
}

// repoArg is the folder a command line names with --repo, or the current folder.
func repoArg(args []string) string {
	dir := "."
	for i, a := range args {
		if (a == "--repo" || a == "-repo") && i+1 < len(args) {
			dir = args[i+1]
		}
		for _, prefix := range []string{"--repo=", "-repo="} {
			if v, ok := strings.CutPrefix(a, prefix); ok {
				dir = v
			}
		}
	}
	return dir
}
