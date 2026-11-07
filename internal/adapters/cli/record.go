package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// record is the engine of status and mark, which read the project folder and
// its record (DEC-099).
func (c CLI) record() (*run.Engine, error) {
	if c.Record == nil {
		return nil, errors.New("this build reads no record")
	}
	return c.Record, nil
}

// status says what is to verify, and changes nothing. Exit 1 means it found
// something: an item, an error in the spec or the realization, or a line of the
// record it cannot read; notes, such as a skip, never change the exit code. With
// --all every Definition is an item.
func (c CLI) status(ctx context.Context, args []string) (int, error) {
	var o options
	var all bool
	fs := flag.NewFlagSet("markitect status", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder of the project")
	fs.BoolVar(&all, "all", false, "list every Definition, to verify everything again")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	e, err := c.record()
	if err != nil {
		return Misused, err
	}
	st, err := e.Status(ctx, run.StatusRequest{Dir: o.repo, All: all})
	if err != nil {
		return Misused, err
	}
	fmt.Fprint(c.Stdout, st.Text())
	if !st.Clean() {
		return Failed, nil
	}
	return OK, nil
}

// markUsage is mark's usage line.
const markUsage = `markitect mark DEFINITION@FINGERPRINT VERDICT [-e EVIDENCE]... [-m "WHY"]`

// mark records a verdict at the state status printed. Exit 1 means it refused and
// recorded nothing: the inputs changed since, or evidence does not exist. The
// flags may stand before or after the item and the verdict.
func (c CLI) mark(ctx context.Context, args []string) (int, error) {
	req := run.MarkRequest{Dir: "."}
	var positional []string
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
		case hasInline:
			v = inline
		case name == "e" || name == "m" || name == "repo":
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
		default:
			return Misused, fmt.Errorf("unknown flag %s\nusage: %s", a, markUsage)
		}
	}
	if len(positional) != 2 {
		return Misused, fmt.Errorf("names the item, DEFINITION@FINGERPRINT as status prints it, and the verdict; give each piece of evidence with -e and why with -m\nusage: %s", markUsage)
	}
	req.Item, req.Verdict = positional[0], positional[1]
	e, err := c.record()
	if err != nil {
		return Misused, err
	}
	text, recorded, err := e.Mark(ctx, req)
	if err != nil {
		if errors.Is(err, run.ErrUsage) {
			return Misused, fmt.Errorf("%s\nusage: %s", strings.TrimPrefix(err.Error(), run.ErrUsage.Error()+": "), markUsage)
		}
		return Misused, err
	}
	fmt.Fprint(c.Stdout, text)
	if !recorded {
		return Failed, nil
	}
	return OK, nil
}

// retired is what checklist, answer and gate say for one release (DEC-099,
// decision 7): on standard output and with exit 0, so that an old project's skill,
// which reads what its command printed, tells the session what to do instead of
// failing silently.
const retired = "markitect %s is gone: since 0.21, markitect records what was verified in .markitect/review.sum, and markitect status says what is left to verify. This project's skills and agents come from an older markitect. Stop, and run markitect update: it installs the /markitect-build and /markitect-review skills and writes the record from the newest certificate.\n"

// hook is what the project's hook calls before a tool call. It reads the
// provider's input on standard input and always exits 0: a problem of its own
// must not stop an agent.
func (c CLI) hook(_ context.Context, args []string) (int, error) {
	if len(args) != 1 || args[0] != "pretool" {
		return Misused, errors.New("names its event: markitect hook pretool")
	}
	input, err := io.ReadAll(c.Stdin)
	if err != nil {
		return OK, nil
	}
	if out := run.Guard(input); len(out) > 0 {
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
