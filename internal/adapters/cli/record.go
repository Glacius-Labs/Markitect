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

// record is the engine of status and review, which read the project folder and
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

// reviewUsage is review's usage line.
const reviewUsage = "markitect review DEFINITION VERDICT --at FINGERPRINT [EVIDENCE...] [-- WHY]"

// review records a verdict at the state status printed. Exit 1 means it refused
// and recorded nothing: the inputs changed since, or evidence does not exist.
// Flags may come anywhere before "--", after which the rest is why.
func (c CLI) review(ctx context.Context, args []string) (int, error) {
	req := run.ReviewRequest{Dir: "."}
	var positional []string
	value := func(i *int, name string) (string, error) {
		if *i+1 >= len(args) {
			return "", fmt.Errorf("%s needs a value", name)
		}
		*i++
		return args[*i], nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, inline, hasInline := strings.Cut(strings.TrimLeft(a, "-"), "=")
		var err error
		switch {
		case a == "--":
			req.Why = strings.Join(args[i+1:], " ")
			i = len(args)
		case a == "-h" || a == "--help" || a == "-help":
			fmt.Fprintf(c.Stderr, "usage: %s\nverdicts: %s\n", reviewUsage, strings.Join(run.Verdicts, ", "))
			return Misused, flag.ErrHelp
		case !strings.HasPrefix(a, "-") || a == "-":
			positional = append(positional, a)
		case name == "at" && hasInline:
			req.At = inline
		case name == "at":
			req.At, err = value(&i, "--at")
		case name == "repo" && hasInline:
			req.Dir = inline
		case name == "repo":
			req.Dir, err = value(&i, "--repo")
		default:
			return Misused, fmt.Errorf("unknown flag %s\nusage: %s", a, reviewUsage)
		}
		if err != nil {
			return Misused, err
		}
	}
	if len(positional) < 2 {
		return Misused, fmt.Errorf("names a Definition and a verdict\nusage: %s", reviewUsage)
	}
	req.Definition, req.Verdict, req.Evidence = positional[0], positional[1], positional[2:]
	e, err := c.record()
	if err != nil {
		return Misused, err
	}
	text, recorded, err := e.Review(ctx, req)
	if err != nil {
		if errors.Is(err, run.ErrUsage) {
			return Misused, fmt.Errorf("%s\nusage: %s", strings.TrimPrefix(err.Error(), run.ErrUsage.Error()+": "), reviewUsage)
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
