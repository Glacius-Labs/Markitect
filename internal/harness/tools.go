package harness

import "encoding/json"

// Role is an agent's part in its Manager.
type Role string

const (
	Maker   Role = "maker"
	Checker Role = "checker"
)

// Mode is the Command a Run serves.
type Mode string

const (
	// Apply: makers change the repository, checkers review their submissions.
	Apply Mode = "apply"
	// Verify: checkers only, each giving its checklist verdicts on the base.
	Verify Mode = "verify"
)

// Tool names. The adapters serve them; the harness names them in the answer
// format.
const (
	ToolSubmit   = "submit"
	ToolReview   = "review"
	ToolAssign   = "assign"
	ToolAsk      = "ask"
	ToolAnswer   = "answer"
	ToolEscalate = "escalate"
)

// Tool is one Manager tool as offered to the provider: its name, what it
// does and its JSON Schema. Descriptions state the contract, never a
// procedure (concept 1.0 §3.7).
type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage
}

// Tools returns the tools of one agent. A maker in apply submits and sends
// messages along the tree, and assigns only when it has children in the Run.
// A checker in apply reviews; a checker in verify submits its verdicts.
func Tools(mode Mode, role Role, children bool) []Tool {
	switch {
	case role == Checker && mode == Verify:
		return []Tool{verifySubmit}
	case role == Checker:
		return []Tool{review}
	}
	tools := []Tool{makerSubmit}
	if children {
		tools = append(tools, assign)
	}
	return append(tools, ask, answer, escalate)
}

// verdictSchema is one verdict with the words of a mode (runremote).
func verdictSchema(words string) string {
	return `{"type":"object","additionalProperties":false,"required":["item","verdict","evidence"],"properties":{` +
		`"item":{"type":"string","description":"the checklist item's key"},` +
		`"verdict":{"type":"string","enum":` + words + `},` +
		`"evidence":{"type":"string","description":"files, tests or Definitions that show the verdict holds"}}}`
}

var makerSubmit = Tool{
	Name: ToolSubmit,
	Description: "Commit every change in /work, deletions included, as one submission with a verdict for each checklist item, " +
		"and push it to your branch. The Run's remote then runs the required checks on the branch's full state and accepts " +
		"or refuses the push; a refusal returns its reason and nothing is lost. Returns the submitted commit.",
	Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["summary","verdicts"],"properties":{` +
		`"summary":{"type":"string","description":"one line: what the submission does"},` +
		`"verdicts":{"type":"array","description":"exactly one verdict per checklist item","items":` + verdictSchema(`["conforms","changed","model-problem"]`) + `}}}`),
}

var verifySubmit = Tool{
	Name: ToolSubmit,
	Description: "Push your verdicts for every checklist item as one commit that changes no file. " +
		"conforms: the repository conforms to the item; deviates: it does not, and the evidence says what you observed and what the model expects; " +
		"model-problem: the model contradicts itself or leaves open behavior at the project boundary. Returns the commit.",
	Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["verdicts"],"properties":{` +
		`"verdicts":{"type":"array","description":"exactly one verdict per checklist item","items":` + verdictSchema(`["conforms","deviates","model-problem"]`) + `}}}`),
}

var review = Tool{
	Name: ToolReview,
	Description: "Answer the submission with a commit that changes no file, pushed to the maker's branch: approved when you " +
		"confirm every verdict, rejected when you dispute at least one. The maker receives a rejection with your evidence. " +
		"Returns the review commit.",
	Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["outcome","checks"],"properties":{` +
		`"outcome":{"type":"string","enum":["approved","rejected"]},` +
		`"checks":{"type":"array","description":"exactly one answer per submitted verdict","items":{"type":"object","additionalProperties":false,"required":["item","answer","evidence"],"properties":{` +
		`"item":{"type":"string"},"answer":{"type":"string","enum":["confirmed","disputed"]},` +
		`"evidence":{"type":"string","description":"what you read or ran; for a dispute, what is wrong and where"}}}}}}`),
}

var assign = Tool{
	Name:        ToolAssign,
	Description: "Send an assignment to one of your direct children: a matter you settled that its area must realize. Returns at once.",
	Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["child","text"],"properties":{` +
		`"child":{"type":"string","description":"the child's Package path"},"text":{"type":"string"}}}`),
}

var ask = Tool{
	Name:        ToolAsk,
	Description: "Send a question to your parent or to one of your direct children. Returns at once; the answer arrives as a later message.",
	Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["to","text"],"properties":{` +
		`"to":{"type":"string","description":"parent, or a direct child's Package path"},"text":{"type":"string"}}}`),
}

var answer = Tool{
	Name:        ToolAnswer,
	Description: "Answer a question you received from your parent or a direct child. Returns at once.",
	Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["to","text"],"properties":{` +
		`"to":{"type":"string","description":"parent, or a direct child's Package path"},"text":{"type":"string"}}}`),
}

var escalate = Tool{
	Name: ToolEscalate,
	Description: "Escalate a matter you cannot settle within the model to your parent, which settles it if the model allows. " +
		"At Root, an escalation is a model defect: a contradiction, or a gap that changes behavior at the project boundary; " +
		"it ends the Run without landing. Returns at once.",
	Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["text"],"properties":{` +
		`"text":{"type":"string","description":"the matter, the Definitions involved and why the model does not settle it"}}}`),
}
