package run

import (
	"encoding/json"
	"slices"
)

// Phase names what one admitted step does (core protocol, rules T1 to T11).
// Manager phase names match organization.ManagerPhase and reviewer phase names
// match organization.ContextView.
type Phase string

const (
	PhaseExplore              Phase = "explore"
	PhaseWork                 Phase = "work"
	PhaseIntegrate            Phase = "integrate"
	PhaseReply                Phase = "reply"
	PhaseReviewerExpectations Phase = "reviewer-expectations"
	PhaseReviewerCandidate    Phase = "reviewer-candidate"
)

// Host tool names (core protocol, Host tools).
const (
	ToolReport             = "report"
	ToolSubmit             = "submit"
	ToolWait               = "wait"
	ToolSend               = "send"
	ToolRead               = "read"
	ToolReservePath        = "reserve_path"
	ToolRegister           = "register"
	ToolAssignOwner        = "assign_owner"
	ToolRequestCheck       = "request_check"
	ToolRecordExpectations = "record_expectations"
	ToolSubmitReview       = "submit_review"
)

const (
	StepContractMedia = "application/vnd.markitect.step-contract+json"
	StepOutcomeFormat = "markitect.step-outcome/v1"
	ToolCatalogMedia  = "application/vnd.markitect.tools+json"
)

// StepContract is a step's output contract. A step's outcome reaches the Host
// only through its Host tools; the turn's final text carries no outcome.
// Outcomes lists the tools that end the step; in the reply phase only a send
// of kind reply ends it (core rule T11). Adapters check the format and do not
// interpret the turn's final text.
type StepContract struct {
	Format   string
	Phase    Phase
	Outcomes []string
}

var stepOutcomes = map[Phase][]string{
	PhaseExplore:              {ToolReport, ToolSubmit},
	PhaseWork:                 {ToolSubmit, ToolWait},
	PhaseIntegrate:            {ToolSubmit, ToolWait},
	PhaseReply:                {ToolSend},
	PhaseReviewerExpectations: {ToolRecordExpectations},
	PhaseReviewerCandidate:    {ToolSubmitReview},
}

// NewStepContract returns the output contract of a phase.
func NewStepContract(phase Phase) (Payload, error) {
	outcomes, ok := stepOutcomes[phase]
	if !ok {
		return Payload{}, ErrInvalid
	}
	data, err := json.Marshal(StepContract{Format: StepOutcomeFormat, Phase: phase, Outcomes: outcomes})
	if err != nil {
		return Payload{}, err
	}
	return NewPayload(StepContractMedia, "utf-8", data)
}

// DecodeStepContract checks and decodes a step contract.
func DecodeStepContract(p Payload) (StepContract, error) {
	var c StepContract
	if p.Ref().MediaType != StepContractMedia || p.Ref().Encoding != "utf-8" || strictJSON(p.Bytes(), &c) != nil || c.Format != StepOutcomeFormat {
		return StepContract{}, ErrInvalid
	}
	if want, ok := stepOutcomes[c.Phase]; !ok || !slices.Equal(want, c.Outcomes) {
		return StepContract{}, ErrInvalid
	}
	return c, nil
}

// ToolSpec is one Host tool as an adapter registers it with the provider.
type ToolSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type toolCatalog struct {
	Version int        `json:"version"`
	Tools   []ToolSpec `json:"tools"`
}

// Every enum is also spelled out in its description, because a provider may
// drop schema keywords before the model sees them. The Host validates every
// argument either way.
const (
	textList  = `{"type":"array","items":{"type":"string","minLength":1}}`
	nonBlank  = `{"type":"string","minLength":1}`
	pathField = `{"type":"string","minLength":1,"description":"Repository-relative path with forward slashes."}`
	formField = `{"type":"string","enum":["code","test","doc","build","config"],"description":"One of: code, test, doc, build, config."}`
	defsField = `{"type":"array","items":{"type":"string","minLength":1},"description":"Definitions as package/id."}`
	evidence  = `{"type":"array","items":{"type":"object","additionalProperties":false,"required":["ref","relevance"],"properties":{"ref":{"type":"string","minLength":1,"description":"A Host reference, such as a check receipt or an artifact path."},"relevance":{"type":"string","minLength":1,"description":"Why this evidence would fail if the obligation were violated."}}}}`
)

var toolSpecs = map[string]ToolSpec{
	ToolReport: {ToolReport, "End your Explore step with your initial read-only assessment of your area. Use submit instead when nothing needs to change.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["summary","proposedChanges","prerequisites","uncertainties","questions"],"properties":{"summary":` + nonBlank + `,"proposedChanges":` + textList + `,"prerequisites":` + textList + `,"uncertainties":` + textList + `,"questions":` + textList + `}}`)},
	ToolSubmit: {ToolSubmit, "End your step by submitting your current workspace for independent review, with one assessment for every local Definition. During Explore only a complete no-change result can be submitted. purpose is local or integration; scope is limited or complete.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["purpose","scope","summary","assessments","dependencies"],"properties":{"purpose":{"type":"string","enum":["local","integration"],"description":"One of: local, integration."},"scope":{"type":"string","enum":["limited","complete"],"description":"One of: limited, complete."},"summary":` + nonBlank + `,"dependencies":{"type":"array","items":{"type":"string","minLength":1},"description":"Manager paths whose accepted results this candidate used."},"assessments":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["definition","disposition","artifacts","uses","evidence","reason"],"properties":{"definition":{"type":"string","minLength":1,"description":"A local Definition as package/id."},"disposition":{"type":"string","enum":["realized","constrains","vocabulary","unresolved"],"description":"One of: realized, constrains, vocabulary, unresolved."},"artifacts":` + textList + `,"uses":` + defsField + `,"evidence":` + evidence + `,"reason":` + nonBlank + `,"question":{"type":"string","description":"For unresolved: the ID of the open question that blocks it."}}}}}}`)},
	ToolWait: {ToolWait, "End your step and name what your task waits for. kind is reply (ref: question ID), delivery (ref: producer Manager path), children (ref unused) or owner (ref: question ID). Wait only for something another Manager must deliver first; missing files of your own are your work, not a reason to wait.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["on","reason"],"properties":{"on":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["kind"],"properties":{"kind":{"type":"string","enum":["reply","delivery","children","owner"],"description":"One of: reply, delivery, children, owner."},"ref":{"type":"string"}}}},"reason":` + nonBlank + `}}`)},
	ToolSend: {ToolSend, "Send a message along the Manager tree. kind is question (to: a Manager path, parent or owner), reply (to: the question ID) or hint (to: a direct child).",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["kind","to","body"],"properties":{"kind":{"type":"string","enum":["question","reply","hint"],"description":"One of: question, reply, hint."},"to":` + nonBlank + `,"body":` + nonBlank + `}}`)},
	ToolRead: {ToolRead, "Read a Definition, artifact, delivery report or transcript by identity.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["ref"],"properties":{"ref":` + nonBlank + `}}`)},
	ToolReservePath: {ToolReservePath, "Reserve a new artifact path before writing it. form is one of: code, test, doc, build, config.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["path","form"],"properties":{"path":` + pathField + `,"form":` + formField + `}}`)},
	ToolRegister: {ToolRegister, "Propose the registry record of an owned or reserved artifact: its form and the Definitions it realizes.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["path","form","realizes","why"],"properties":{"path":` + pathField + `,"form":` + formField + `,"realizes":` + defsField + `,"why":` + nonBlank + `}}`)},
	ToolAssignOwner: {ToolAssignOwner, "As the common parent, decide the owner of an artifact that several descendants need, or of an unexplained file.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["path","manager","reason"],"properties":{"path":` + pathField + `,"manager":` + nonBlank + `,"reason":` + nonBlank + `}}`)},
	ToolRequestCheck: {ToolRequestCheck, "Run an allowed project check on the current candidate and get its receipt.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["check"],"properties":{"check":` + nonBlank + `}}`)},
	ToolRecordExpectations: {ToolRecordExpectations, "End your expectations step with the expected behavior and the discriminating cases you derive from the model, before you see the candidate.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["cases"],"properties":{"cases":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["name","expectation","discriminates"],"properties":{"name":` + nonBlank + `,"expectation":` + nonBlank + `,"discriminates":` + nonBlank + `}}}}}`)},
	ToolSubmitReview: {ToolSubmitReview, "End your assessment step with your verdict: pass, changes_requested, needs_decision or failed_assessment. A pass must resolve or mark not-applicable every open blocking finding of the task; needs_decision must state the owner question. An integration review addresses a finding about a child's artifacts to that child, with changes_requested.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["verdict","summary","findings","resolutions","evidence"],"properties":{"verdict":{"type":"string","enum":["pass","changes_requested","needs_decision","failed_assessment"],"description":"One of: pass, changes_requested, needs_decision, failed_assessment."},"summary":` + nonBlank + `,"question":{"type":"string","description":"For needs_decision: the decision the owner must make."},"evidence":` + evidence + `,"findings":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["severity","definitions","grounds","requestedEvidence"],"properties":{"to":{"type":"string","description":"The Manager path the finding is addressed to: the reviewed Manager (the default) or, in an integration review, the descendant that owns the affected artifacts."},"severity":{"type":"string","enum":["blocking","advisory"],"description":"One of: blocking, advisory."},"definitions":` + defsField + `,"grounds":` + nonBlank + `,"requestedEvidence":` + nonBlank + `}}},"resolutions":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["finding","state","grounds","evidence"],"properties":{"finding":{"type":"string","minLength":1,"description":"The ID of an open finding."},"state":{"type":"string","enum":["resolved","not-applicable"],"description":"One of: resolved, not-applicable."},"grounds":` + nonBlank + `,"evidence":` + evidence + `}}}}}`)},
}

// phaseSpecs narrows a tool's description for one phase. The expectations
// step comes before the candidate, so its read reaches Definitions only; the
// Host refuses any other read in that phase.
var phaseSpecs = map[Phase]map[string]ToolSpec{
	PhaseReviewerExpectations: {ToolRead: {ToolRead, "Read a Definition of the model by its identity, package/id. The candidate is not available in this step.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["ref"],"properties":{"ref":{"type":"string","minLength":1,"description":"A Definition as package/id."}}}`)}},
}

var phaseTools = map[Phase][]string{
	PhaseExplore:              {ToolReport, ToolSubmit, ToolRead, ToolSend, ToolAssignOwner},
	PhaseWork:                 {ToolSubmit, ToolWait, ToolSend, ToolRead, ToolReservePath, ToolRegister, ToolAssignOwner, ToolRequestCheck},
	PhaseIntegrate:            {ToolSubmit, ToolWait, ToolSend, ToolRead, ToolReservePath, ToolRegister, ToolAssignOwner, ToolRequestCheck},
	PhaseReply:                {ToolSend, ToolRead},
	PhaseReviewerExpectations: {ToolRecordExpectations, ToolRead},
	PhaseReviewerCandidate:    {ToolSubmitReview, ToolRead, ToolRequestCheck},
}

// ToolCatalog returns the closed Host tool catalogue of a phase, in the format
// adapters register with the provider. It follows the core protocol's Host
// tools table; the Host refuses a call it does not support yet with a reason.
func ToolCatalog(phase Phase) (Payload, error) {
	names, ok := phaseTools[phase]
	if !ok {
		return Payload{}, ErrInvalid
	}
	catalog := toolCatalog{Version: 1}
	for _, name := range names {
		spec := toolSpecs[name]
		if override, ok := phaseSpecs[phase][name]; ok {
			spec = override
		}
		catalog.Tools = append(catalog.Tools, spec)
	}
	data, err := json.Marshal(catalog)
	if err != nil {
		return Payload{}, err
	}
	return NewPayload(ToolCatalogMedia, "utf-8", data)
}

// PhaseTool reports whether a phase offers a tool.
func PhaseTool(phase Phase, tool string) bool { return slices.Contains(phaseTools[phase], tool) }

// OutcomeTool reports whether a tool can end a step of the phase. In the reply
// phase the Host also requires the send to be of kind reply.
func OutcomeTool(phase Phase, tool string) bool { return slices.Contains(stepOutcomes[phase], tool) }
