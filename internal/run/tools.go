package run

import "encoding/json"

// Host tool names. Execute turns send and finish; Verify turns assess.
const (
	toolSend   = "send"
	toolFinish = "finish"
	toolAssess = "assess"
)

const sendSchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["to", "text"],
  "properties": {
    "to": {"type": "string", "description": "A contact: \"parent\", the path of a direct child, or \"owner\" (Root only)."},
    "text": {"type": "string", "description": "The message."}
  }
}`

const finishSchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["status", "summary"],
  "properties": {
    "status": {"type": "string", "enum": ["done", "waiting"], "description": "done: your work for these items is complete. waiting: you wait for an answer or for another Manager."},
    "summary": {"type": "string", "description": "What you did and what is still open."}
  }
}`

const assessSchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["summary", "satisfied"],
  "properties": {
    "summary": {"type": "string", "description": "Checked scope, stand, findings and unresolved questions."},
    "satisfied": {"type": "boolean", "description": "True only when none of your own findings stays open and blocking."},
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["owner", "obligation", "observed", "blocking"],
        "properties": {
          "owner": {"type": "string", "description": "Path of the Manager that must act: yourself or one of your descendants."},
          "obligation": {"type": "string", "description": "The model obligation that is violated or lacks evidence."},
          "observed": {"type": "string", "description": "The observed behavior or source evidence."},
          "blocking": {"type": "boolean"}
        }
      }
    },
    "closed": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["id", "reason"],
        "properties": {
          "id": {"type": "string", "description": "ID of an open finding you raised earlier."},
          "reason": {"type": "string", "description": "Why it is resolved on this candidate."}
        }
      }
    }
  }
}`

func executeTools(root bool) []ToolSpec {
	to := "Send a message to your parent (to \"parent\") or to a direct child (to its path). Other recipients are refused. The answer returns what has arrived for you."
	if root {
		to = "Send a message to a direct child (to its path) or ask the human owner (to \"owner\"). Other recipients are refused. The answer returns what has arrived for you."
	}
	return []ToolSpec{
		{Name: toolSend, Description: to, InputSchema: json.RawMessage(sendSchema)},
		{Name: toolFinish, Description: "End your turn. Call it once when you have done what the items ask.", InputSchema: json.RawMessage(finishSchema)},
	}
}

func verifyTools() []ToolSpec {
	return []ToolSpec{{Name: toolAssess, Description: "Report your assessment of the candidate. Call it exactly once; an invalid call is refused with the reason and you may call it again.", InputSchema: json.RawMessage(assessSchema)}}
}
