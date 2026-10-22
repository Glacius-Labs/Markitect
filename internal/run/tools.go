package run

import "encoding/json"

// Orchestrator tool names: the small provider-neutral surface beside the
// provider's own read, edit and shell tools.
const (
	toolReport   = "report"
	toolSend     = "send"
	toolEscalate = "escalate"
	toolLookUp   = "look_up"
	toolCheck    = "check"
	toolDelegate = "delegate"
)

// Finding subjects of a review Pass with ArtifactFindingsAdvisory: an artifact
// that does not conform to the model, or a defect of the model.
const (
	subjectArtifact = "artifact"
	subjectModel    = "model"
)

type schema = map[string]any

func str(description string) schema { return schema{"type": "string", "description": description} }

func strs(description string) schema {
	return schema{"type": "array", "items": schema{"type": "string"}, "description": description}
}

func object(required []string, properties schema) schema {
	o := schema{"type": "object", "additionalProperties": false, "properties": properties}
	if len(required) > 0 {
		o["required"] = required
	}
	return o
}

func raw(s schema) json.RawMessage {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return b
}

// reportSchema exposes one closed payload per action. Technical identities and
// candidate bindings are supplied by the Host when it records the submission.
func reportSchema(spec PassSpec) schema {
	var corrections []string
	for _, c := range spec.Corrections {
		corrections = append(corrections, string(c))
	}
	var kinds []string
	for _, k := range spec.observations() {
		kinds = append(kinds, string(k))
	}
	finding := schema{
		"kind":       schema{"type": "string", "enum": kinds, "description": "What you claim. deviation: a binding obligation or required check is violated, or a required realization or evidence is missing; it blocks and names its correction. model: a material model problem (Model assessment only); it blocks. suspected: a concrete, plausible violation of a named obligation you could not settle yet; it blocks until a review investigates and classifies it, and names no correction. suggestion: an optional idea beyond what the mandate requires; it never blocks, never causes a correction and grants no write. Default: deviation (model in the assessment) when blocking, suggestion otherwise."},
		"expected":   str("deviation, suspected: the outcome the obligation requires, which the observed text contradicts or for which proof is missing."),
		"owner":      str("Path of the Manager that must act: yourself or one of your descendants."),
		"obligation": str("The model obligation that is violated or lacks evidence."),
		"observed":   str("The observed behavior or source evidence."),
		"blocking":   schema{"type": "boolean", "description": "true for a genuine finding: a model violation, a missing required realization or evidence, a failed required check. false for an optional note."},
		"correction": schema{"type": "string", "enum": corrections, "description": "The earliest kind of work its correction needs; the last one is the default."},
		"receipts":   strs("Probe receipts (R...) that show it; a genuine finding citing an adverse receipt disposes of that receipt as a violation."),
	}
	required := []string{"owner", "obligation", "observed", "blocking"}
	if spec.ArtifactFindingsAdvisory {
		finding["subject"] = schema{"type": "string", "enum": []string{subjectArtifact, subjectModel}, "description": "artifact: an artifact does not conform to the model. model: the model itself is defective."}
		finding["definition"] = str("Required when subject is artifact: the identity of the Definition the artifact deviates from.")
		required = append(required, "subject")
	}
	properties := schema{
		"kind": schema{"type": "string", "enum": []string{kindProgress, kindWait, kindBasis, kindResult, kindAccept, kindReturn, kindReopen, kindRevise, kindProblem},
			"description": "progress: you continue in a later turn. wait: you need the named answer, child or another Manager's accepted basis or result first. basis: what your children need from you before they start. result: your complete integrated result. accept or return: your decision on a child's proposal. reopen: return a child's accepted result. revise: change your accepted basis. problem: record a classified problem."},
		"summary":  str("progress, basis, result: what you did, what it provides and what is still open."),
		"for":      str("wait: the question ID you asked, the path of a child, or the path of another Manager whose accepted basis or result you need first; you are woken when it is accepted."),
		"stage":    schema{"type": "string", "enum": []string{stageNameBasis, stageNameResult}, "description": "wait for another Manager: its accepted basis or (default) result."},
		"proposal": str("accept, return: the proposal ID (P...) of your child."),
		"child":    str("reopen: the path of the child."),
		"reason":   str("return, reopen, revise, wait: why."),
		"obligations": schema{"type": "array", "description": "basis, result: your own obligations; a result covers every one.", "items": object([]string{"definition", "status"}, schema{
			"definition": str("Identity of one of your own Definitions."),
			"status":     schema{"type": "string", "enum": []string{coverDone, coverPreserved, coverNotApplicable, coverOpen}},
			"note":       str("How it is met and with which evidence; required for not_applicable and open, and for every entry of a plan."),
			"conditions": str("Derive material conditions, exceptions and interactions from this Definition and required bound contracts; give concise source references, explain coverage and any uncovered conditions. Required in Implement and artifact review, including the reviewer basis before considering author coverage."),
			"evidence":   evidenceSchema(spec),
		})},
		"integration": object([]string{"status", "conditions", "note"}, schema{
			"status":     schema{"type": "string", "enum": []string{coverDone, coverPreserved, coverOpen}},
			"conditions": str("Every parent, including Root without own Definitions: your authority-derived composed workflows/global invariants, source references and discriminating cases. Basis and Plan result state intended integration; every non-Plan result justifies actual integration with evidence beyond child reports."),
			"note":       str("What your integration establishes and what remains uncovered; route corrections to actual owners."),
			"evidence":   evidenceSchema(spec),
		}),
		"artifacts": strs("basis, result: your own artifacts this contribution created, changed or preserved (repository-relative paths)."),
		"evidence":  strs("basis, result: the checks, tests and observations that support it."),
		"verdict":   schema{"type": "string", "enum": []string{string(Satisfied), string(NeedsCorrection)}, "description": "result: satisfied when its objective is met; needs_correction when a genuine finding names required corrective work."},
		"findings":  schema{"type": "array", "items": object(required, finding), "description": "basis, result: findings within your responsibility."},
		"closed": schema{"type": "array", "description": "basis, result: open findings you raised in an earlier Pass that are resolved or disproved now.", "items": object([]string{"id", "reason", "evidence"}, schema{
			"id":       str("ID of an open finding you raised earlier."),
			"reason":   str("Why it is closed."),
			"as":       schema{"type": "string", "enum": []string{ClosedResolved, ClosedDisproved, ClosedConfirmed}, "description": "resolved (default): the candidate corrects it. disproved: it was erroneous; required on an unchanged candidate. confirmed: a suspicion this submission confirms by raising the deviation or model problem for the same owner."},
			"evidence": str("The observed evidence of the resolution or disproof."),
		})},
		"category": schema{"type": "string", "enum": problemCategories, "description": "problem: its category."},
		"text":     str("problem: the problem."),
		"abort":    schema{"type": "boolean", "description": "problem, Root only, category model: the model is erroneous or contradictory; the run ends without publishing."},
	}
	if spec.Review {
		properties["disputed"] = schema{"type": "array", "description": "basis, result: open genuine review findings raised by another reviewer, below you or about your own responsibility, that you independently judge erroneous; they stay open until a fresh review of their responsibility disposes of them.", "items": object([]string{"id", "reason", "evidence"}, schema{
			"id":       str("ID of the finding."),
			"reason":   str("Why it is erroneous."),
			"evidence": str("The evidence on this candidate."),
		})}
		properties["dispositions"] = schema{"type": "array", "description": "basis, result: dispositions of adverse probe receipts (failed or error) on this candidate within your responsibility. A violation is not listed here: cite the receipt in a genuine finding instead.", "items": object([]string{"receipt", "as"}, schema{
			"receipt":  str("ID of the adverse receipt (R...)."),
			"as":       schema{"type": "string", "enum": []string{DisposedExpectationError, DisposedRerun}, "description": "expectation_error: the probe itself was wrong, not the candidate: its expectation or assertion (failed), or its execution, such as a probe too large for the check's time limit (error, which also names passing). rerun: a Runtime error replaced by a passing exact repetition of its probe (probe.rerun), which also disposes of its erring repetitions."},
			"reason":   str("expectation_error: why the probe, not the candidate, was wrong, derived from the model."),
			"evidence": str("expectation_error: the evidence of the disproof, such as the Definition text and a corrected or narrower passing receipt."),
			"rerun":    str("rerun: the passing receipt that repeated it with probe.rerun."),
			"passing":  str("expectation_error of an error receipt: a passing receipt of the same obligation on this candidate that establishes the claim without the erring probe."),
		})}
	}
	if spec.Plans {
		properties["plan"] = object([]string{"approach", "artifacts", "inputs", "outputs", "dependencies", "order", "blockers", "assumptions"}, schema{
			"approach":  str("Your own concrete approach and, as a parent, how you orchestrate and integrate your children."),
			"artifacts": strs("Your own planned artifacts or directories (repository-relative)."),
			"inputs": schema{"type": "array", "items": object([]string{"need", "owner"}, schema{
				"need":  str("What you need."),
				"owner": str("The Manager that owns it."),
			})},
			"outputs": schema{"type": "array", "items": object([]string{"provides", "consumers"}, schema{
				"provides":  str("What you provide."),
				"consumers": strs("The Managers that use it."),
			})},
			"dependencies": schema{"type": "array", "description": "Start conditions: in the named Pass you start only after the owner's accepted basis or result.", "items": object([]string{"owner", "need", "pass", "stage"}, schema{
				"owner": str("Another Manager."),
				"need":  str("What you need from it."),
				"pass":  str("The later Pass purpose it applies to."),
				"stage": schema{"type": "string", "enum": []string{stageNameBasis, stageNameResult}},
			})},
			"order":       str("Ordering and start conditions of your work."),
			"blockers":    strs("What blocks you."),
			"assumptions": strs("Assumptions that need clarification."),
		})
	}
	if spec.Assesses {
		delete(properties, "integration")
	}
	var actions []schema
	for _, kind := range []string{kindProgress, kindWait, kindBasis, kindResult, kindAccept, kindReturn, kindReopen, kindRevise, kindProblem} {
		fields := schema{"kind": schema{"type": "string", "enum": []string{kind}}}
		for _, name := range reportFields[kind] {
			if value, ok := properties[name]; ok {
				fields[name] = value
			}
		}
		required := []string{"kind"}
		switch kind {
		case kindProgress, kindBasis:
			required = append(required, "summary")
		case kindResult:
			required = append(required, "summary", "verdict")
		case kindWait:
			required = append(required, "for", "reason")
		case kindAccept:
			required = append(required, "proposal")
		case kindReturn:
			required = append(required, "proposal", "reason")
		case kindReopen:
			required = append(required, "child", "reason")
		case kindRevise:
			required = append(required, "reason")
		case kindProblem:
			required = append(required, "category", "text")
		}
		actions = append(actions, object(required, fields))
	}
	return schema{"type": "object", "oneOf": actions}
}

var sendSchema = object([]string{"to", "kind", "text"}, schema{
	"to":    str("\"parent\" or the path of a direct child."),
	"kind":  schema{"type": "string", "enum": []string{sendQuestion, sendAnswer, sendInfo}},
	"text":  str("The message."),
	"about": str("answer: the question ID you answer. question: the question addressed to you that this one serves, when you forward an exchange."),
})

var escalateSchema = object([]string{"category", "text"}, schema{
	"category": schema{"type": "string", "enum": problemCategories},
	"text":     str("The problem and the decision you need."),
})

var lookUpSchema = object([]string{"topic"}, schema{
	"topic":   schema{"type": "string", "enum": lookUpTopics},
	"id":      str("definition: the Definition identity. method: the method's name. probe: the receipt (R...)."),
	"path":    str("owner, registry: a repository-relative path."),
	"manager": str("plan, basis: a Manager path; default yourself (plan) or your parent (basis)."),
})

// evidenceText describes the evidence items of a coverage entry for one Pass.
func evidenceText(spec PassSpec) string {
	if spec.Review || evidenceRequired(spec) {
		return "Required for done and preserved: per distinct claim, a passing probe receipt on this candidate or an inspected source location; a suite total alone is no coverage. One execution may support several claims and Definitions when its actual assertions support them; explain that support and remaining gaps. Open conditions may omit unsupported evidence. Adverse receipts belong in genuine findings or reasoned dispositions, never in coverage evidence."
	}
	return "Optional: per claim, a passing probe receipt run on the stand you submit, or an inspected source location."
}

func evidenceSchema(spec PassSpec) schema {
	return schema{"type": "array", "description": evidenceText(spec), "items": object(nil, schema{
		"kind":      schema{"type": "string", "enum": []string{"public_behavior", "static"}, "description": "Required in Implement and artifact review. public_behavior needs a passing receipt at the actual declared public entry; static needs inspected source and why execution is inapplicable. Never relabel executable public behavior static to waive evidence."},
		"authority": str("Bound model Definition identity; implicit own obligation, required for integration source inspection (pass_mandate refers to the already bound Pass task where no Definition supplies that integration duty, never a new Definition or permission to override the model). State the precise condition/source reference in claim/conditions."),
		"boundary":  str("public_behavior: actual declared public entry exercised by the receipt; omit to inherit probe.boundary, or supply the same value. Inspect the probe's assertions and invocation; a label alone proves no semantic adequacy."),
		"claim":     str("Distinct model-derived condition and case/input class. Omit to use the recorded claim; when reusing a receipt, state the actual supported conclusion."),
		"receipt":   str("A passing tool-produced receipt on this submitted stand; private-validator or suite totals cannot establish the public composition."),
		"source":    str("static: inspected repository-relative source file."),
		"location":  str("static: precise inspected location."),
		"reasoning": str("static: why this location establishes the claim and why behavioral execution is inapplicable."),
	})}
}

// checkSchema is the schema of check: the File check of the caller's area in a
// Pass with writes, or one behavioral probe.
func checkSchema(spec PassSpec) schema {
	properties := schema{
		"probe": object(nil, schema{
			"rerun":      str("Instead of a new probe: an error receipt (R...) of this candidate whose exact probe runs again; give nothing else."),
			"obligation": str("New probe: the Definition whose claim it assesses, yours or one of your descendants'. A new probe gives obligation, claim, expected, check, path, content and args."),
			"claim":      str("The distinct claim, with the case or input class, derived from the model."),
			"boundary":   str("The actual declared public entry the probe exercises, including transport/parsing/persistence when relevant. Required to support a public_behavior claim; a private validator alone cannot do so."),
			"expected":   str("The outcome the model requires, stated before the probe runs."),
			"check":      str("The required check whose execution profile (image, executable, environment, limits) runs the probe."),
			"path":       str("Where the probe file is placed, relative to that check's directory; it must not exist in the stand."),
			"content":    str("The complete standalone probe file, with its own imports, setup and helpers; no variables, files or helper state carry from another probe. Its assertions exit 0 when the expected outcome is observed, nonzero otherwise. An expected refusal of the application passes when the probe asserts it."),
			"args":       strs("Arguments after the profile's fixed arguments, such as the probe file to run."),
			"proposal":   str("Optional, Pass with writes: the checked proposal (P...) of one of your children to probe its submitted stand."),
		}),
	}
	if spec.Writes {
		properties["scope"] = schema{"type": "string", "enum": []string{"own", "all"}, "description": "File check: own (default): your findings in full and the number of others. all: the findings of other Managers too."}
	}
	return object(nil, properties)
}

var fileCheckSchema = object(nil, schema{
	"scope": schema{"type": "string", "enum": []string{"own", "all"}, "description": "own (default): your findings in full and the number of others. all: the findings of other Managers too."},
})

var delegateSchema = object([]string{"role", "task"}, schema{
	"role":  str("One of your helper roles."),
	"task":  str("The bounded task and the result you expect."),
	"paths": strs("Existing files or directories of your own area the helper may change; omit for a read-only task."),
})

var helperReportSchema = object([]string{"summary"}, schema{
	"summary": str("What you did, what you found and what is still open."),
})

// memberTools lists the tools of a Manager's or reviewer's turn.
func (e *engine) memberTools(m *member) []ToolSpec {
	spec := m.p.spec
	sendText := "Send a question, an answer or information to your parent or a direct child; other Managers are reached through them."
	if m.Parent == "" {
		sendText = "Send a question, an answer or information to a direct child; other Managers are reached through them. Nobody outside the tree answers during the run."
	}
	tools := []ToolSpec{
		{Name: toolReport, Description: "Report to the Orchestrator: progress, a named wait, your basis or complete result, a decision on a child's proposal, a reopened child result, a revised basis or a problem. End every turn in which you work on your own contribution with one.", InputSchema: raw(reportSchema(spec))},
		{Name: toolSend, Description: sendText, InputSchema: raw(sendSchema)},
		{Name: toolEscalate, Description: "Escalate a classified problem and the decision you need to your parent as a question. Root has nobody above it: it reports a model problem with report kind problem.", InputSchema: raw(escalateSchema)},
		{Name: toolLookUp, Description: "Look up structural facts: your standing context again (brief), a Definition of the bound model in full, the owner of a path, the derived registry view, the states of the Pass, accepted plans and bases, open findings, checks, probe receipts and your questions; and method guidance.", InputSchema: raw(lookUpSchema)},
	}
	switch probes := e.probesAvailable(spec); {
	case spec.Writes && probes:
		tools = append(tools, ToolSpec{Name: toolCheck, Description: "Without probe: File check of your current area against the accepted stand (unregistered or wrongly registered files, invalid registry entries, missing owners), advice while you work. With probe: run one behavioral probe file in its own container on a stable stand and get the Orchestrator's receipt (R...) of its claim, expectation, bytes, invocation and outcome.", InputSchema: raw(checkSchema(spec))})
	case spec.Writes:
		tools = append(tools, ToolSpec{Name: toolCheck, Description: "File check of your current area against the accepted stand: unregistered or wrongly registered files, invalid registry entries, missing owners. Advice while you work.", InputSchema: raw(fileCheckSchema)})
	case probes:
		tools = append(tools, ToolSpec{Name: toolCheck, Description: "Run one behavioral probe file in its own container on the stand of this Pass and get the Orchestrator's receipt (R...) of its claim, expectation, bytes, invocation and outcome.", InputSchema: raw(checkSchema(spec))})
	}
	if len(m.Helpers) > 0 {
		tools = append(tools, ToolSpec{Name: toolDelegate, Description: "Delegate a bounded task to one of your helper roles; it runs in its own container after this turn and reports its result to you.", InputSchema: raw(delegateSchema)})
	}
	return tools
}

func helperTools(writes bool) []ToolSpec {
	tools := []ToolSpec{
		{Name: toolReport, Description: "Report your result to your Manager when your task is done.", InputSchema: raw(helperReportSchema)},
		{Name: toolLookUp, Description: "Look up structural facts: your standing context again (brief), a Definition of the bound model in full, the owner of a path, the derived registry view, the states of the Pass, plans and bases, findings, checks.", InputSchema: raw(lookUpSchema)},
	}
	if writes {
		tools = append(tools, ToolSpec{Name: toolCheck, Description: "File check of your Manager's current area against the accepted stand.", InputSchema: raw(fileCheckSchema)})
	}
	return tools
}
