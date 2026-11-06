package run

import (
	"fmt"
	"strings"
)

// The default strategy's task: a short fixed part, the goal, and what the tool
// prints when asked. A task says what to do and how to answer; the process stays
// in the tool, which tells the agent its next step each time it is called.

// ReviewerAgent is the agent type of the reviewer, which the hook on answer
// recognizes and the harness defines; BuilderAgent the builder's.
const (
	ReviewerAgent = "markitect-reviewer"
	BuilderAgent  = "markitect-builder"
)

// fixedPart is what every session is told about Markitect.
const fixedPart = `This project is described by a spec in .markitect/model: Markdown files,
linked with [[...]]. The spec is the authority over every artifact in the
project: code, tests, docs, pipelines, config. Where an artifact differs from
the spec, the artifact is wrong. Every file is listed in the registry, or ignored.

markitect check            is the spec and the registry well-formed
markitect impact --base X  what a change reaches
markitect refs ID          who links to a Definition
`

// implementerTask is the builder's task, which checklist prints each time, so
// that compaction cannot lose it.
func implementerTask(run string) string {
	return fixedPart + fmt.Sprintf(`
You build Markitect run %s, in this folder, and nobody can answer you.

Make this project match the spec. Start with: markitect checklist

You are done when markitect submit says so. Call it when you think the work is done, and do what it tells you each time.

Update the registry when files change: every file is listed on the page of its owning Package with the Definitions it realizes and a short reason, or it is ignored. When a Definition is gone, its files go or get a new justified basis.

Do not change the spec, the manifest, the lock or anything else under .markitect/ except the registry pages, and do not change AGENTS.md, CLAUDE.md or the installed Markitect skills and agents.

The tool handles Git for you: do not commit, push or open a pull request.

If the spec contradicts itself or leaves open something that changes behaviour at the project's boundary, run: markitect stop "why". Nothing lands then, and the owner decides. A gap with no such effect is yours to decide. If something the project needs is missing from this machine, such as a program or a service, run: markitect stop --environment "why". Nobody can answer a question during this run.
`, run)
}

// builderStep tells the owner's session how to build an open Run. The rest of the
// process stays in the tool: the builder takes its task from checklist, and
// submit tells it each next step, starting the reviewer among them.
func builderStep(st *RunState) string {
	return fmt.Sprintf(`Start the %s agent%s with exactly this message:

    Build run %s.

It works in this folder and starts the reviewer itself. When it has ended, run markitect status: while the run is still open, start the builder again with the same message. Do not write the code or answer the checklist yourself.
`, BuilderAgent, onModel(st.Model), st.Run)
}

// verifyStep tells the owner's session how to review an open verify.
func verifyStep(st *RunState) string {
	return fmt.Sprintf(`Start the %s agent%s with exactly this message:

    Review run %s.

It answers the checklist and closes the run. When it has ended, run markitect status: while the run is still open, start the reviewer again with the same message.
`, ReviewerAgent, onModel(st.Model), st.Run)
}

// onModel names the model the manifest asks for, if any.
func onModel(model string) string {
	if model == "" {
		return ""
	}
	return ", on model " + model + ","
}

// reviewPrompt is what checklist --review prints for a review.
func reviewPrompt(st *RunState, rev *Review) string {
	var b strings.Builder
	b.WriteString(fixedPart)
	if st.Command == "verify" {
		fmt.Fprintf(&b, "\nNo one has just changed this project: answer whether it conforms at commit %s.\n\n", rev.Commit)
	} else {
		from := rev.From
		if from == "" {
			from = st.Base
		}
		fmt.Fprintf(&b, "\nAn implementer has just worked on this project, and you owe it no benefit of the doubt. See its work with: git diff %s %s\n\n", from, rev.Commit)
	}
	if len(rev.Earlier) > 0 {
		b.WriteString("An earlier review of this work found these, and they were sent back:\n\n")
		for _, e := range rev.Earlier {
			fmt.Fprintf(&b, "- %s\n", e)
		}
		b.WriteString("\nIts other answers stand. Answer only the items below.\n\n")
	}
	b.WriteString("## Checklist\n\n")
	b.WriteString(renderList(rev.Items))
	b.WriteString(answerFormat)
	return b.String()
}

// answerFormat says how a reviewer answers.
const answerFormat = `
## Your answer

Answer with markitect answer, which reads one line per item from its standard input:

    markitect answer <<'EOF'
    loans/limit: ok  src/loans.go:42, tests/loans_test.go:88 (TestLimit)
    loans/renewal: wrong  src/loans.go:70 -- allows a fourth renewal; the Definition says three
    fees/waiver: not shown -- nothing exercises the waiver
    fees/rounding: spec problem -- rounding says up, its example rounds down
    EOF

The answers are ok, wrong, not shown and spec problem. Only the last three need an explanation, after "--". The file:line names a line you read; a test in brackets names a test in that file, and nothing else goes in brackets. markitect answer refuses what it cannot accept and says what is still missing. You may call it several times; the latest answer to an item counts.
`
