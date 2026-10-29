package harness

import (
	"encoding/json"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
)

// The roles and modes of the agent port.
const (
	Maker   = agentport.Maker
	Checker = agentport.Checker
	Apply   = agentport.Apply
	Verify  = agentport.Verify
)

// The Git tools: an agent's work reaches the Run's remote through them. The
// host tools (agentport.Tool*) carry messages and Notes.
const (
	ToolSubmit = "submit"
	ToolReview = "review"
)

// Tool is one Manager tool as offered to the provider: its name, what it
// does and its JSON Schema. Descriptions state the contract, never a
// procedure (concept 1.0 §3.7).
type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage
}

// Tools returns the tools of one start: the Git tool of its role and mode
// (a maker submits, a checker in apply reviews, a checker in verify submits
// its verdicts) and the host tools the start may call.
func Tools(s agentport.Start) []Tool {
	var tools []Tool
	switch {
	case s.Role == Maker:
		tools = append(tools, makerSubmit)
	case s.Mode == Verify:
		tools = append(tools, verifySubmit)
	default:
		tools = append(tools, review)
	}
	for _, t := range s.Tools {
		if tool, ok := hostTools[t]; ok {
			tools = append(tools, tool)
		}
	}
	return tools
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
		"or refuses the push; a refusal returns its reason and nothing is lost. An accepted push is a submission, not an approval: " +
		"once this start has ended, a fresh checker reviews it, and a rejection reaches you with the checker's evidence as a review " +
		"message in a later start. Returns the submitted commit.",
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
	Description: "Answer the submission at the tip of the maker's branch with a commit that changes no file: approved when you " +
		"confirm every verdict, rejected when you dispute at least one. The maker receives a rejection with your evidence. " +
		"Returns the review commit.",
	Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["outcome","checks"],"properties":{` +
		`"outcome":{"type":"string","enum":["approved","rejected"]},` +
		`"checks":{"type":"array","description":"exactly one answer per submitted verdict","items":{"type":"object","additionalProperties":false,"required":["item","answer","evidence"],"properties":{` +
		`"item":{"type":"string"},"answer":{"type":"string","enum":["confirmed","disputed"]},` +
		`"evidence":{"type":"string","description":"what you read or ran; for a dispute, what is wrong and where"}}}}}}`),
}

const textOnly = `{"type":"object","additionalProperties":false,"required":["text"],"properties":{"text":{"type":"string"}}}`

var hostTools = map[agentport.Tool]Tool{
	agentport.ToolAsk: {
		Name:        string(agentport.ToolAsk),
		Description: "Ask your parent a question. Returns at once; the answer arrives with a later start.",
		Schema:      json.RawMessage(textOnly),
	},
	agentport.ToolEscalate: {
		Name: string(agentport.ToolEscalate),
		Description: "Hand your parent a matter you cannot settle within the model; it settles it if the model allows. " +
			"At Root an escalation is a model defect, a contradiction or a gap that changes behavior at the project boundary, and ends the Run without landing. Returns at once.",
		Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["text"],"properties":{` +
			`"text":{"type":"string","description":"the matter, the Definitions involved and why the model does not settle it"}}}`),
	},
	agentport.ToolAnswer: {
		Name:        string(agentport.ToolAnswer),
		Description: "Answer a question or escalation you received, by its message ID. Returns at once.",
		Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["to","text"],"properties":{` +
			`"to":{"type":"string","description":"the ID of the question or escalation"},"text":{"type":"string"}}}`),
	},
	agentport.ToolAssign: {
		Name:        string(agentport.ToolAssign),
		Description: "Send one of your children in this Run a task: a matter you settled that its area must realize, or a correction after your review. Returns at once.",
		Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"required":["child","text"],"properties":{` +
			`"child":{"type":"string","description":"the child's Package path"},"text":{"type":"string"}}}`),
	},
	agentport.ToolNote: {
		Name:        string(agentport.ToolNote),
		Description: "Record a side observation as a Shared Note, with you as its origin. A Note carries no authority.",
		Schema:      json.RawMessage(textOnly),
	},
	agentport.ToolNotes: {
		Name:        string(agentport.ToolNotes),
		Description: "Read the Run's Shared Notes, oldest first, a page at a time; pass the previous page's next cursor to read on.",
		Schema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{` +
			`"after":{"type":"string","description":"the cursor of the next page; empty for the first"}}}`),
	},
}
