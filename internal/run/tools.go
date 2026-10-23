package run

import "encoding/json"

// Orchestrator tool names: the small provider-neutral surface beside the
// provider's own read, edit and shell tools.
const (
	toolWait     = "wait"
	toolDecide   = "decide"
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
			"existing":   schema{"type": "boolean", "description": "Use the existing test at path unchanged on the captured stand; omit content. Omitted args uses the required check's configured invocation. Inspect actual assertions for the claimed behavior; passing suite totals alone do not establish it."},
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

// memberTools lists the tools of a Manager's or reviewer's turn.
func (e *engine) memberTools(m *member) []ToolSpec {
	spec := m.p.spec
	sendText := "Send a question, an answer or information to your parent or a direct child; other Managers are reached through them."
	if m.Parent == "" {
		sendText = "Send a question, an answer or information to a direct child; other Managers are reached through them. Nobody outside the tree answers during the run."
	}
	tools := []ToolSpec{
		{Name: toolWait, Description: "Wait for an actual prerequisite: a named open question, child, or accepted contract/result. Ending a Turn does not itself need a tool.", InputSchema: raw(waitSchema)},
		{Name: toolSend, Description: sendText, InputSchema: raw(sendSchema)},
		{Name: toolEscalate, Description: "Escalate a classified problem and the decision you need to your parent as a question. Root records a confirmed model problem in its final assessment problem field.", InputSchema: raw(escalateSchema)},
		{Name: toolLookUp, Description: "Look up structural facts: your standing context again (brief), a Definition of the bound model in full, the owner of a path, the derived registry view, the states of the Pass, accepted plans and bases, open findings, checks, probe receipts and your questions; and method guidance.", InputSchema: raw(lookUpSchema)},
	}
	if len(m.Children) > 0 || m.basis != nil {
		tools = append(tools, ToolSpec{Name: toolDecide, Description: "Make a necessary accept/return/reopen decision or revise your bound contract.", InputSchema: raw(decisionSchema)})
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
		{Name: toolLookUp, Description: "Look up structural facts: your standing context again (brief), a Definition of the bound model in full, the owner of a path, the derived registry view, the states of the Pass, plans and bases, findings, checks.", InputSchema: raw(lookUpSchema)},
	}
	if writes {
		tools = append(tools, ToolSpec{Name: toolCheck, Description: "File check of your Manager's current area against the accepted stand.", InputSchema: raw(fileCheckSchema)})
	}
	return tools
}
