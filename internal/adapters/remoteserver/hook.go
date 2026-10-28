package remoteserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// Entry is one line of the remote's decision log: every push the hook saw,
// accepted or refused, with its checks.
type Entry struct {
	Time    string             `json:"time"`
	User    string             `json:"user"`
	Updates []runremote.Update `json:"updates"`
	runremote.Decision
}

// Hook is the pre-receive hook. Git runs it with the pushed updates on stdin
// and the pusher's user in REMOTE_USER, which only the server sets. It prints
// the decision for the pusher and logs it; a non-zero result refuses the
// whole push.
func Hook(l Layout, stdin io.Reader, stderr io.Writer) int {
	user := os.Getenv("REMOTE_USER")
	var updates []runremote.Update
	s := bufio.NewScanner(stdin)
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) == 3 {
			updates = append(updates, runremote.Update{Old: f[0], New: f[1], Ref: f[2]})
		}
	}
	plan, err := LoadPlan(l)
	git, gerr := exec.LookPath("git")
	var dec runremote.Decision
	switch {
	case err != nil || gerr != nil:
		dec = runremote.Decision{Reason: fmt.Sprintf("the remote cannot read its plan or Git: %v %v. Next: report it to your parent.", err, gerr)}
	case user == "":
		dec = runremote.Decision{Reason: "the push carries no authenticated user. Next: push over the Run's remote URL with your credentials."}
	default:
		repo := gitRepo{git: git, dir: l.Repo(), env: os.Environ()}
		dec = runremote.Decide(plan, runremote.NewPatternOwnership(plan), repo, newCheckRunner(l, repo, stderr), user, updates)
	}
	if err := appendLog(l, Entry{Time: time.Now().UTC().Format(time.RFC3339Nano), User: user, Updates: updates, Decision: dec}); err != nil && dec.Accepted {
		dec = runremote.Decision{Reason: fmt.Sprintf("the remote could not record the push: %v. Next: push again.", err)}
	}
	for _, c := range dec.Checks {
		state := "passed"
		if !c.Passed {
			state = "failed"
		}
		fmt.Fprintf(stderr, "check %s %s in %.1fs\n", c.Name, state, c.Seconds)
	}
	if !dec.Accepted {
		fmt.Fprintf(stderr, "Markitect refused this push:\n%s\n", dec.Reason)
		return 1
	}
	fmt.Fprintf(stderr, "Markitect accepted this push (%s).\n", dec.Kind)
	return 0
}

func appendLog(l Layout, e Entry) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(l.Log(), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
