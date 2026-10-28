package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// maxToolText bounds one tool argument string.
const maxToolText = 64 << 10

// refusal is a tool call the contract refuses; the agent receives its text
// and may call again.
type refusal string

func (r refusal) Error() string { return string(r) }

func refuse(format string, args ...any) error { return refusal(fmt.Sprintf(format, args...)) }

// call serves one tool call. Every answer, a refusal included, is an MCP
// tool result the agent reads; only the tools of the agent's role exist.
func (a *Agent) call(ctx context.Context, call frame.ToolCall) frame.ToolReply {
	text, err := a.dispatch(ctx, call.Name, call.Arguments)
	failed := err != nil
	if failed {
		text = err.Error()
		var r refusal
		if !errors.As(err, &r) {
			text = "the host could not carry out " + call.Name + ": " + text
		}
	}
	result, _ := json.Marshal(map[string]any{
		"content": []map[string]string{{"type": "text", "text": string(a.redact([]byte(text)))}},
		"isError": failed,
	})
	return frame.ToolReply{ID: call.ID, Result: result}
}

func (a *Agent) dispatch(ctx context.Context, name string, raw json.RawMessage) (string, error) {
	offered := false
	for _, t := range a.tools {
		offered = offered || t.Name == name
	}
	if !offered {
		return "", refuse("there is no tool %q for the %s", name, a.spec.Input.Role)
	}
	in := a.spec.Input
	switch {
	case name == harness.ToolSubmit && in.Role == harness.Maker:
		var args struct {
			Summary  string        `json:"summary"`
			Verdicts []verdictArgs `json:"verdicts"`
		}
		if err := decode(raw, &args); err != nil {
			return "", err
		}
		return a.submit(ctx, args.Summary, args.Verdicts)
	case name == harness.ToolSubmit:
		var args struct {
			Verdicts []verdictArgs `json:"verdicts"`
		}
		if err := decode(raw, &args); err != nil {
			return "", err
		}
		return a.verdicts(ctx, args.Verdicts)
	case name == harness.ToolReview:
		var args struct {
			Outcome string      `json:"outcome"`
			Checks  []checkArgs `json:"checks"`
		}
		if err := decode(raw, &args); err != nil {
			return "", err
		}
		return a.review(ctx, args.Outcome, args.Checks)
	case name == harness.ToolAssign:
		var args struct {
			Child string `json:"child"`
			Text  string `json:"text"`
		}
		if err := decode(raw, &args); err != nil {
			return "", err
		}
		if !a.isChild(args.Child) {
			return "", refuse("%q is none of your children in this Run: %s", args.Child, strings.Join(a.children(), ", "))
		}
		return a.message(ctx, name, args.Child, args.Text)
	case name == harness.ToolAsk || name == harness.ToolAnswer:
		var args struct {
			To   string `json:"to"`
			Text string `json:"text"`
		}
		if err := decode(raw, &args); err != nil {
			return "", err
		}
		to, err := a.neighbour(args.To)
		if err != nil {
			return "", err
		}
		return a.message(ctx, name, to, args.Text)
	case name == harness.ToolEscalate:
		var args struct {
			Text string `json:"text"`
		}
		if err := decode(raw, &args); err != nil {
			return "", err
		}
		return a.message(ctx, name, model.ParentOf(in.Package), args.Text)
	}
	return "", refuse("there is no tool %q", name)
}

func decode(raw json.RawMessage, v any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return refuse("the arguments do not match the tool's schema: %v", err)
	}
	if d.More() {
		return refuse("the arguments are more than one JSON object")
	}
	return nil
}

type verdictArgs struct {
	Item     string `json:"item"`
	Verdict  string `json:"verdict"`
	Evidence string `json:"evidence"`
}

type checkArgs struct {
	Item     string `json:"item"`
	Answer   string `json:"answer"`
	Evidence string `json:"evidence"`
}

// id is the Manager as trailers name it: "." for Root.
func (a *Agent) id() string { return runremote.Name(a.spec.Input.Package) }

func (a *Agent) branch() string { return runremote.Branch(a.spec.Input.Package) }

// checkVerdicts requires exactly one verdict per checklist item, with a
// word of the mode and evidence, as the remote's hook will: a maker conforms,
// changed or model-problem, a verify checker conforms, deviates or
// model-problem.
func (a *Agent) checkVerdicts(args []verdictArgs) ([]runremote.Line, error) {
	words := []string{runremote.Conforms, runremote.Changed, runremote.ModelProblem}
	if a.spec.Input.Mode == harness.Verify {
		words[1] = runremote.Deviates
	}
	want := map[string]bool{}
	for _, it := range a.spec.Input.Items {
		want[it.Key] = true
	}
	given := map[string]bool{}
	var lines []runremote.Line
	var problems []string
	for _, v := range args {
		item, word, evidence := strings.TrimSpace(v.Item), strings.TrimSpace(v.Verdict), flat(v.Evidence)
		switch {
		case !want[item]:
			problems = append(problems, fmt.Sprintf("%q is no item of your checklist", item))
		case given[item]:
			problems = append(problems, fmt.Sprintf("%q has more than one verdict", item))
		case word != words[0] && word != words[1] && word != words[2]:
			problems = append(problems, fmt.Sprintf("%q: the verdict %q is not %s", item, word, strings.Join(words, ", ")))
		case evidence == "":
			problems = append(problems, fmt.Sprintf("%q: the verdict gives no evidence", item))
		}
		given[item] = true
		lines = append(lines, runremote.Line{Item: item, Word: word, Evidence: evidence})
	}
	for _, it := range a.spec.Input.Items {
		if !given[it.Key] {
			problems = append(problems, fmt.Sprintf("%q has no verdict", it.Key))
		}
	}
	if len(problems) > 0 {
		return nil, refuse("not submitted: %s", strings.Join(problems, "; "))
	}
	return lines, nil
}

// submit commits the maker's working tree with its verdicts and pushes it.
// A refused push leaves the work as it was before the call.
func (a *Agent) submit(ctx context.Context, summary string, args []verdictArgs) (string, error) {
	summary = flat(summary)
	if summary == "" {
		return "", refuse("not submitted: the summary is empty")
	}
	lines, err := a.checkVerdicts(args)
	if err != nil {
		return "", err
	}
	message, err := commitMessage(summary, runremote.Trailers{Manager: a.id(), Verdicts: lines})
	if err != nil {
		return "", err
	}
	if err := a.git(ctx, []string{gitPath, "add", "--all"}, "", io.Discard); err != nil {
		return "", err
	}
	if err := a.git(ctx, []string{gitPath, "commit", "--quiet", "--allow-empty", "--no-verify", "--file", "-"}, message, io.Discard); err != nil {
		return "", err
	}
	commit, err := a.rev(ctx, "HEAD")
	if err != nil {
		return "", err
	}
	if err := a.push(ctx, "HEAD"); err != nil {
		if undo := a.git(ctx, []string{gitPath, "reset", "--quiet", "--soft", "HEAD~1"}, "", io.Discard); undo != nil {
			return "", errors.Join(err, undo)
		}
		return "", err
	}
	return fmt.Sprintf("submitted %s to %s; the remote accepted it after the required checks", commit, a.branch()), nil
}

// review answers the submission at the tip of the maker's branch with a
// commit that changes no file.
func (a *Agent) review(ctx context.Context, outcome string, args []checkArgs) (string, error) {
	tip, sub, err := a.tip(ctx)
	if err != nil {
		return "", err
	}
	if sub.Kind != runremote.Submission {
		return "", refuse("not reviewed: the tip %s of %s is no submission", tip, a.branch())
	}
	submitted := map[string]bool{}
	for _, v := range sub.Verdicts {
		submitted[v.Item] = true
	}
	given := map[string]bool{}
	disputed := false
	var lines []runremote.Line
	var problems []string
	for _, c := range args {
		item, word, evidence := strings.TrimSpace(c.Item), strings.TrimSpace(c.Answer), flat(c.Evidence)
		switch {
		case !submitted[item]:
			problems = append(problems, fmt.Sprintf("%q has no submitted verdict", item))
		case given[item]:
			problems = append(problems, fmt.Sprintf("%q has more than one answer", item))
		case word != runremote.Confirmed && word != runremote.Disputed:
			problems = append(problems, fmt.Sprintf("%q: the answer %q is not confirmed or disputed", item, word))
		case evidence == "":
			problems = append(problems, fmt.Sprintf("%q: the answer gives no evidence", item))
		}
		given[item] = true
		disputed = disputed || word == runremote.Disputed
		lines = append(lines, runremote.Line{Item: item, Word: word, Evidence: evidence})
	}
	for _, v := range sub.Verdicts {
		if !given[v.Item] {
			problems = append(problems, fmt.Sprintf("%q has no answer", v.Item))
		}
	}
	switch {
	case outcome != runremote.Approved && outcome != runremote.Rejected:
		problems = append(problems, fmt.Sprintf("the outcome %q is not approved or rejected", outcome))
	case outcome == runremote.Approved && disputed:
		problems = append(problems, "an approval disputes no verdict")
	case outcome == runremote.Rejected && !disputed:
		problems = append(problems, "a rejection disputes at least one verdict")
	}
	if len(problems) > 0 {
		return "", refuse("not reviewed: %s", strings.Join(problems, "; "))
	}
	message, err := commitMessage("Review "+a.id(), runremote.Trailers{Manager: a.id(), Review: outcome, Checks: lines})
	if err != nil {
		return "", err
	}
	commit, err := a.emptyCommit(ctx, tip, message)
	if err != nil {
		return "", err
	}
	if err := a.push(ctx, commit); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s with review commit %s", outcome, tip, commit), nil
}

// verdicts pushes a verify checker's verdicts as a commit that changes no
// file on the tip of its branch.
func (a *Agent) verdicts(ctx context.Context, args []verdictArgs) (string, error) {
	lines, err := a.checkVerdicts(args)
	if err != nil {
		return "", err
	}
	tip, _, err := a.tip(ctx)
	if err != nil {
		return "", err
	}
	message, err := commitMessage("Verdicts of "+a.id(), runremote.Trailers{Manager: a.id(), Verdicts: lines})
	if err != nil {
		return "", err
	}
	commit, err := a.emptyCommit(ctx, tip, message)
	if err != nil {
		return "", err
	}
	if err := a.push(ctx, commit); err != nil {
		return "", err
	}
	return fmt.Sprintf("submitted the verdicts as %s", commit), nil
}

// commitMessage writes the subject and the trailers and reads them back as
// the remote's hook will.
func commitMessage(subject string, t runremote.Trailers) (string, error) {
	message := subject + "\n\n" + t.String()
	if _, err := runremote.ParseMessage(message); err != nil {
		return "", refuse("the trailers would be refused: %v", err)
	}
	return message, nil
}

// tip fetches the agent's branch and returns its tip with its trailers.
func (a *Agent) tip(ctx context.Context) (string, runremote.Trailers, error) {
	ref := "refs/remotes/origin/" + a.branch()
	if err := a.git(ctx, []string{gitPath, "fetch", "--quiet", "origin", "+refs/heads/" + a.branch() + ":" + ref}, "", io.Discard); err != nil {
		return "", runremote.Trailers{}, err
	}
	tip, err := a.rev(ctx, ref)
	if err != nil {
		return "", runremote.Trailers{}, err
	}
	var body bytes.Buffer
	if err := a.git(ctx, []string{gitPath, "log", "-1", "--format=%B", tip}, "", &body); err != nil {
		return "", runremote.Trailers{}, err
	}
	t, err := runremote.ParseMessage(body.String())
	if err != nil {
		return "", runremote.Trailers{}, refuse("the tip %s of %s has malformed trailers: %v", tip, a.branch(), err)
	}
	return tip, t, nil
}

// emptyCommit makes a commit on parent with parent's tree, so it changes no
// file whatever the working tree holds.
func (a *Agent) emptyCommit(ctx context.Context, parent, message string) (string, error) {
	var out bytes.Buffer
	if err := a.git(ctx, []string{gitPath, "commit-tree", parent + "^{tree}", "-p", parent, "-F", "-"}, message, &out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func (a *Agent) rev(ctx context.Context, ref string) (string, error) {
	var out bytes.Buffer
	if err := a.git(ctx, []string{gitPath, "rev-parse", "--verify", ref + "^{commit}"}, "", &out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// push sends the commit to the agent's branch. The remote's refusal, with
// its reason and next step, comes back to the agent as a refusal.
func (a *Agent) push(ctx context.Context, commit string) error {
	err := a.git(ctx, []string{gitPath, "push", "--quiet", "origin", commit + ":refs/heads/" + a.branch()}, "", io.Discard)
	if err != nil {
		return refuse("the remote refused the push to %s: %v", a.branch(), err)
	}
	return nil
}

func (a *Agent) isChild(pkg string) bool {
	for _, c := range a.children() {
		if c == pkg {
			return true
		}
	}
	return false
}

// neighbour resolves "parent" or a direct child to a Manager ID.
func (a *Agent) neighbour(to string) (string, error) {
	pkg := a.spec.Input.Package
	switch {
	case to == "parent" && pkg != "":
		return model.ParentOf(pkg), nil
	case to == "parent":
		return "", refuse("Root has no parent")
	case a.isChild(to):
		return to, nil
	}
	return "", refuse("%q is neither parent nor one of your children in this Run (%s)", to, strings.Join(a.children(), ", "))
}

func (a *Agent) message(ctx context.Context, kind, to, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" || len(text) > maxToolText {
		return "", refuse("the text must be between 1 and %d bytes", maxToolText)
	}
	from := a.spec.Input.Package
	if kind == harness.ToolEscalate && from == "" {
		to = ""
	}
	if err := a.spec.Messages.Send(ctx, Message{Kind: kind, From: from, To: to, Text: text}); err != nil {
		return "", refuse("not sent: %v", err)
	}
	if kind == harness.ToolEscalate && from == "" {
		return "escalated: a model defect ends the Run at Root without landing", nil
	}
	return "sent to " + runremote.Name(to), nil
}

// flat writes text on one line, as a trailer value must be.
func flat(s string) string { return strings.Join(strings.Fields(s), " ") }
