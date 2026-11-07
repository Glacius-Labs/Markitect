package run

// The default strategy's tasks: a short fixed part, the goal, and what the tool
// prints when asked. A task says what to do and how to answer; the process stays
// in the tool, which tells the agent its next step each time it is called.

// ReviewerAgent is the agent type of the reviewer, which the hook on answer
// recognizes and the harness defines; BuilderAgent the builder's.
const (
	ReviewerAgent = "markitect-reviewer"
	BuilderAgent  = "markitect-builder"
)

// reviewerFile and builderFile are the harness's agents, without which the work
// cannot finish.
const (
	reviewerFile = ".claude/agents/" + ReviewerAgent + ".md"
	builderFile  = ".claude/agents/" + BuilderAgent + ".md"
)

// fixedPart is what every agent is told about Markitect.
const fixedPart = `This project is described by a spec in .markitect/model: Markdown files,
linked with [[...]]. The spec is the authority over every artifact in the
project: code, tests, docs, pipelines, config. Where an artifact differs from
the spec, the artifact is wrong. Every file is listed in the registry, or ignored.

markitect check            is the spec and the registry well-formed
markitect impact --base X  what a change reaches
markitect refs ID          who links to a Definition
`

// onModel names the model the manifest asks for, if any.
func onModel(model string) string {
	if model == "" {
		return ""
	}
	return ", on model " + model + ","
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
