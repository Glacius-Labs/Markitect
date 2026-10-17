package run

import (
	"encoding/json"
	"slices"
)

// Step phases name what one admitted step does (core protocol, rules T1 to T9).
const (
	PhaseExplore            = "explore"
	PhaseWork               = "work"
	PhaseIntegration        = "integration"
	PhaseReply              = "reply"
	PhaseReviewExpectations = "review-expectations"
	PhaseReviewAssessment   = "review-assessment"
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
// Outcomes lists the tools that end the step. Adapters check the format and
// do not interpret the turn's final text.
type StepContract struct {
	Format   string
	Phase    string
	Outcomes []string
}

var stepOutcomes = map[string][]string{
	PhaseExplore:            {ToolReport, ToolSubmit},
	PhaseWork:               {ToolSubmit, ToolWait},
	PhaseIntegration:        {ToolSubmit, ToolWait},
	PhaseReply:              {ToolSend},
	PhaseReviewExpectations: {ToolRecordExpectations},
	PhaseReviewAssessment:   {ToolSubmitReview},
}

// NewStepContract returns the output contract of a phase.
func NewStepContract(phase string) (Payload, error) {
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

const (
	stringList = `{"type":"array","items":{"type":"string","minLength":1}}`
	nonBlank   = `{"type":"string","minLength":1}`
	pathField  = `{"type":"string","minLength":1,"description":"Repository-relative path with forward slashes."}`
	formField  = `{"type":"string","enum":["code","test","doc","build","config"]}`
)

var toolSpecs = map[string]ToolSpec{
	ToolReport: {ToolReport, "End your Explore step with your initial read-only assessment of your area.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["summary","proposedChanges","prerequisites","uncertainties","questions"],"properties":{"summary":` + nonBlank + `,"proposedChanges":` + stringList + `,"prerequisites":` + stringList + `,"uncertainties":` + stringList + `,"questions":` + stringList + `}}`)},
	ToolSubmit: {ToolSubmit, "End your step by submitting your current workspace for independent review. During Explore, only a complete no-change result can be submitted.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["purpose","scope","summary","assessments"],"properties":{"purpose":{"type":"string","enum":["local","integration"]},"scope":{"type":"string","enum":["limited","complete"]},"summary":` + nonBlank + `,"assessments":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["definition","disposition","artifacts","evidence","reason"],"properties":{"definition":{"type":"string","minLength":1,"description":"package/id of a local Definition."},"disposition":{"type":"string","enum":["realized","constrains","vocabulary","unresolved"]},"artifacts":` + stringList + `,"evidence":` + stringList + `,"reason":` + nonBlank + `}}}}}`)},
	ToolWait: {ToolWait, "End your step and name what your task waits for.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["on","reason"],"properties":{"on":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["kind","ref"],"properties":{"kind":{"type":"string","enum":["reply","delivery","children","owner"]},"ref":` + nonBlank + `}}},"reason":` + nonBlank + `}}`)},
	ToolSend: {ToolSend, "Send a question, reply or hint along the Manager tree.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["kind","to","body"],"properties":{"kind":{"type":"string","enum":["question","reply","hint"]},"to":{"type":"string","minLength":1,"description":"A Manager path, parent, owner, or for a reply the question ID."},"body":` + nonBlank + `}}`)},
	ToolRead: {ToolRead, "Read a Definition, artifact, delivery report or transcript by identity.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["ref"],"properties":{"ref":` + nonBlank + `}}`)},
	ToolReservePath: {ToolReservePath, "Reserve a new artifact path before writing it.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["path","form"],"properties":{"path":` + pathField + `,"form":` + formField + `}}`)},
	ToolRegister: {ToolRegister, "Propose the registry record of an owned or reserved artifact.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["path","form","realizes","why"],"properties":{"path":` + pathField + `,"form":` + formField + `,"realizes":` + stringList + `,"why":` + nonBlank + `}}`)},
	ToolAssignOwner: {ToolAssignOwner, "As the common parent, decide the owner of an artifact.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["path","manager","reason"],"properties":{"path":` + pathField + `,"manager":` + nonBlank + `,"reason":` + nonBlank + `}}`)},
	ToolRequestCheck: {ToolRequestCheck, "Run an allowed project check on the current candidate.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["check"],"properties":{"check":` + nonBlank + `}}`)},
	ToolRecordExpectations: {ToolRecordExpectations, "End your expectations step with the expected behavior and discriminating cases from the model.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["cases"],"properties":{"cases":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["name","expectation","discriminates"],"properties":{"name":` + nonBlank + `,"expectation":` + nonBlank + `,"discriminates":` + nonBlank + `}}}}}`)},
	ToolSubmitReview: {ToolSubmitReview, "End your assessment step with your verdict, findings and evidence.",
		json.RawMessage(`{"type":"object","additionalProperties":false,"required":["verdict","summary","findings","resolutions"],"properties":{"verdict":{"type":"string","enum":["pass","changes_requested","needs_decision","failed_assessment"]},"summary":` + nonBlank + `,"findings":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["severity","grounds","requestedEvidence"],"properties":{"severity":{"type":"string","enum":["blocking","advisory"]},"definition":{"type":"string"},"grounds":` + nonBlank + `,"requestedEvidence":` + nonBlank + `}}},"resolutions":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["finding","state","grounds"],"properties":{"finding":` + nonBlank + `,"state":{"type":"string","enum":["resolved","not_applicable"]},"grounds":` + nonBlank + `}}}}}`)},
}

var phaseTools = map[string][]string{
	PhaseExplore:            {ToolReport, ToolSubmit},
	PhaseWork:               {ToolSubmit, ToolWait, ToolReservePath, ToolRegister},
	PhaseIntegration:        {ToolSubmit, ToolWait, ToolReservePath, ToolRegister},
	PhaseReply:              {ToolSend},
	PhaseReviewExpectations: {ToolRecordExpectations},
	PhaseReviewAssessment:   {ToolSubmitReview},
}

// ToolCatalog returns the closed Host tool catalogue of a phase, in the format
// adapters register with the provider. Tools not yet supported by the core are
// not offered.
func ToolCatalog(phase string) (Payload, error) {
	names, ok := phaseTools[phase]
	if !ok {
		return Payload{}, ErrInvalid
	}
	catalog := toolCatalog{Version: 1}
	for _, name := range names {
		catalog.Tools = append(catalog.Tools, toolSpecs[name])
	}
	data, err := json.Marshal(catalog)
	if err != nil {
		return Payload{}, err
	}
	return NewPayload(ToolCatalogMedia, "utf-8", data)
}

// PhaseTool reports whether a phase offers a tool.
func PhaseTool(phase, tool string) bool { return slices.Contains(phaseTools[phase], tool) }

// OutcomeTool reports whether a tool ends a step of the phase.
func OutcomeTool(phase, tool string) bool { return slices.Contains(stepOutcomes[phase], tool) }
