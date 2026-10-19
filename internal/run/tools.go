package run

import (
	"encoding/json"
)

// Host tool names. Execute turns send and finish, and check when Config offers
// it; Verify turns assess.
const (
	toolSend   = "send"
	toolFinish = "finish"
	toolCheck  = "check"
	toolAssess = "assess"
)

const checkSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "scope": {"type": "string", "enum": ["own", "all"], "description": "own (default): your findings in full and the number of others. all: the findings of other Managers too."}
  }
}`

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

// Finding subjects of an assess report when Config.ArtifactFindingsAdvisory is
// set: an artifact that does not conform to the model, or a defect of the model.
const (
	subjectArtifact = "artifact"
	subjectModel    = "model"
)

// assessSchemaSubjects is the assess schema when Config.ArtifactFindingsAdvisory
// is set: assessSchema with the two fields every finding needs. It is written out
// so that a change of assessSchema's layout cannot silently break it; a test
// parses both and checks the differences.
const assessSchemaSubjects = `{
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
        "required": ["owner", "subject", "obligation", "observed", "blocking"],
        "properties": {
          "owner": {"type": "string", "description": "Path of the Manager that must act: yourself or one of your descendants."},
          "obligation": {"type": "string", "description": "The model obligation that is violated or lacks evidence."},
          "observed": {"type": "string", "description": "The observed behavior or source evidence."},
          "blocking": {"type": "boolean"},
          "subject": {"type": "string", "enum": ["artifact", "model"], "description": "artifact: an artifact does not conform to the model. model: the model itself is defective."},
          "definition": {"type": "string", "description": "Required when subject is artifact: the identity of the Definition the artifact deviates from."}
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

func executeTools(root, check bool) []ToolSpec {
	to := "Send a message to your parent (to \"parent\") or to a direct child (to its path). Other recipients are refused. The answer returns what has arrived for you."
	finish := "Report how your turn ends: done or waiting, with a summary. Call it when you have done what the items ask; if you call it again, the last call counts. The summary of done reaches your parent, so it needs no separate message."
	if root {
		to = "Send a message to a direct child (to its path) or ask the human owner (to \"owner\"). Other recipients are refused. The answer returns what has arrived for you."
		finish = "Report how your turn ends: done or waiting, with a summary. Call it when you have done what the items ask; if you call it again, the last call counts."
	}
	tools := []ToolSpec{
		{Name: toolSend, Description: to, InputSchema: json.RawMessage(sendSchema)},
		{Name: toolFinish, Description: finish, InputSchema: json.RawMessage(finishSchema)},
	}
	if check {
		tools = append(tools, ToolSpec{Name: toolCheck, Description: "Check the current shared working tree: returns the findings of the project's accounting, yours first. Other Managers may still be writing, so the answer is advice; an uncertain observation is refused with the reason.", InputSchema: json.RawMessage(checkSchema)})
	}
	return tools
}

func verifyTools(subjects bool) []ToolSpec {
	schema := assessSchema
	if subjects {
		schema = assessSchemaSubjects
	}
	return []ToolSpec{{Name: toolAssess, Description: "Report your assessment of the candidate when your assessment is complete. An invalid call is refused with the reason; if you call it again, the last valid call counts.", InputSchema: json.RawMessage(schema)}}
}
