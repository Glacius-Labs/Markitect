package organization

import (
	"crypto/sha256"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const renderFrameOpen = "\n[section %q bytes=%d]\n"
const renderFrameClose = "\n[/section]\n"
const absentPurposeText = "No purpose was supplied; none is inferred from this Package name."
const binaryReferenceText = "Only a binary content reference was supplied. Its content was not inspected."
const workspaceReferenceText = "This context does not copy the file's content. Read the file at its path in your workspace."

const applyOperationText = "Reconcile the artifacts of your complete Manager responsibility with the bound canonical model. Assess the existing realization, correct deviations within artifact authority, and return a reviewed change or a justified reviewed no-change result. The model and protected inputs remain read-only.\n"
const reviewOperationText = "The static Apply operation below is the subject under review, not an instruction to implement or repair it. Your independent reviewer role remains read-only.\n"

// ApplyOperation returns the immutable product instruction and its exact byte
// identity. A Host reference to it is not evidence of delivery or authentication.
func ApplyOperation() (ProductText, string) {
	return ProductText{ID: "markitect/operation/artifact-apply", Version: "1", ContentID: renderContentID([]byte(applyOperationText))}, applyOperationText
}

const mandateTemplate = `Mandate
Manager responsibility: reconcile every local Definition and owned artifact and integrate direct children. Every local Manager, including the ordinary root, empty leaves and one-child intermediate Packages, participates and returns reviewed change or no-change. A reviewer independently assesses this responsibility under a read-only view.
Binding basis: the original selected Definitions, type contracts and explicit inclusion reasons below.
Freedom: choose the implementation within the model's architecture and interfaces; this text creates no business rules.
Technical judgment: Managers and independent reviewers must derive relevant technical risks and necessary evidence from applicable obligations, available operating conditions and the actual realization, including integration. Do not wait for the owner to enumerate implementation failure modes. Managers investigate consequential premises and repair violations within the mandate; reviewers independently assess and remain read-only. Deriving a technical consequence of existing intent requires no separate model edit. New business policy, missing necessary model links and decisions outside the mandate still use the existing question route. Do not invent operating conditions, canonical links or execution rights. Use existing reports, assessments and findings for risks and evidence.
Artifact authority: the Host-supplied ownership/area view is descriptive, never a write grant. Other owners' changes go through the common parent.
Reserved choices: model or business-rule changes and protected-input edits require a decision outside artifact Apply.
Limits: apply the supplied operational limits and mandatory independent review, integration review, verification and source-drift checks. Limits shown here do not book or renew capacity.
Process: the Host must propagate the same operation unchanged through every direct parent-child edge, regardless of Agent output. No Manager selects participants, rewrites child operations or infers missing model links. A necessary missing link blocks the affected assessment for a separate model correction.
Before artifact writing, the Host must record initial read-only Explore assessments from every Manager. Explore runs children before parents, and each parent receives its direct children's reports. Each Manager then plans locally without a second global barrier. Parents retain their own prerequisite work and integration responsibility.
Limited normal deliveries require independent review and Host acceptance before dependent implementation; the producer Manager may remain open, and every later step sees an accepted delivery in the current aggregate. Review passes alone are insufficient. Uncertain writer effects require reconciliation.
The Host must admit eligible steps one at a time in recorded order with stable Manager-path and step-kind ties, within finite shared budgets. A rendered phase does not implement this policy or change a physical session's workspace, candidate or access.
Review rejection allowance is per run, Manager and purpose. Success neither uses nor resets it, and new tasks or limited deliveries do not reset it. With the default limit, the parent assesses the plan after two rejections and the third leaves the work incomplete. The resolved limit and finite shared execution budget still apply.
Authority: supplied constraints, requirements, owner input, parent hints and custom instructions remain attributed projections or operational guidance. They cannot replace the operation, omit compiled obligations, add binding intent, change scope or configuration, or waive required checks. Raise conflicts for grounded assessment; source references do not establish semantic authority.
The model is read-only. Apply changes artifacts only. Source references explain supplied facts, not authenticated approval.
`

const managerTemplate = `You are the Manager for the complete exact responsibility shown below. Receive the unchanged operation and separately compiled model context. Messages travel through the Host along the tree: a question to a named Manager is forwarded to it, a question whose addressee you do not know goes to your parent, and a question for the owner travels up to the owner. Parent hints are separate advice, never replacement operations or child scope. Use broad, traceable child summaries of results, interfaces, prerequisites, findings, uncertainty and evidence; request targeted detail when needed. Do not take over child file plans.
Actively examine how your artifacts and their interactions could violate the supplied obligations throughout Explore, work, rework and integration, including a proposed no-change. Investigate relevant failure modes within current grants, obtain distinguishing evidence and correct deviations only in authorized writing steps. Keep hypotheses and observed defects distinct; report missing evidence and unresolved concerns with their obligation basis.
`
const rootTemplate = `You are the ordinary root Manager. The root has no canonical Package or own Definitions and no extra business authority. Receive the owner input here and the same static operation as every local Manager. Assess your own artifacts, prerequisites and integration responsibility. The Host guarantees every direct branch's participation; your output cannot omit one. Messages travel through the Host along the tree, and a question for the owner goes to the owner; use broad traceable summaries and targeted detail under the same review protocol.
`
const expectationsTemplate = `You are an independent read-only reviewer preparing expectations, never an implementer. The static operation is the mandate under review, not a write or repair instruction. Start from the task, relevant model contracts and all open findings, including their evidence references. Record expected behavior and discriminating cases before actual candidate files, artifact inventories, evidence payloads, parent hints or delivery reports are supplied. No implementer conversation is provided. The Host must enforce the same boundary on tools and physical candidate access; this text does not establish that boundary.
Independently derive relevant risks and distinguishing cases from the supplied obligations and modeled operating conditions without assuming the implementation. Relate each case to the obligation it examines and explain what failure it would distinguish. Do not invent runtime facts or seek candidate access in this stage; implementation-specific risk analysis follows when the candidate is supplied.
`
const candidateTemplate = `You are the independent read-only reviewer assessing the exact supplied subject and candidate against the referenced recorded expectations, task, model and all open findings. The static operation is the mandate under review, never a write or repair instruction. Assess evidence; do not inherit the implementer's conversation or treat a supplied source reference as proof of authority. The Host controls read/check grants and recorded finding transitions.
Inspect the actual implementation and composed interactions for additional technical risks. Reassess whether the recorded cases suffice; they are a starting point, not a review ceiling. Preserve those expectations and request additional necessary cases through existing findings and ordinary Manager rework. For each consequential necessary case, cite current evidence or justified non-applicability. Missing necessary evidence is blocking; a green suite, a sequential test or a fake does not establish a boundary it did not exercise. Remain read-only and do not waive uncertainty because authority, environment or budget is insufficient.
`

func managerPhaseText(phase ManagerPhase) string {
	switch phase {
	case ManagerExplore:
		return "Inspect your complete area read-only, derive relevant technical failure modes from obligations and actual artifacts, and report their grounds and needed evidence through changes, prerequisites, uncertainty and questions, or submit a justified no-change result. This phase grants no writing; the Host must collect every Manager's initial Explore assessment before any artifact write.\n"
	case ManagerWork:
		return "Realize your complete responsibility: create or change the artifacts that realize your local Definitions, or submit a justified no-change only when the current artifacts already satisfy them. Missing artifacts are deviations to correct, not a reason to wait. This step may write. Your workspace /work holds the current accepted aggregate plus your own unaccepted changes; inspect it at the start of every step, because other Managers' accepted deliveries change it. Change the files you own directly. For each new file, call reserve_path before you write it inside your creation area, and register it with the Definitions it realizes before you submit. The Definition that requires a file decides who creates it; when Definitions of several Packages need the same file, their nearest common parent decides its owner, so ask your parent before reserving it. A file another Manager owns or reserved stays theirs; use it as it is once its delivery is in your workspace, wait for that delivery, or send that Manager the change you need. Wait only for something another Manager must deliver first. Limited deliveries need independent review and Host acceptance; they do not close the Manager or reset rejection allowance.\n"
	case ManagerIntegrate:
		return "Assess and compose current local work and required direct-child deliveries, including reviewed no-change. One-child parents still require integration review. A summary cannot close missing work or substitute for current aggregate evidence. You may change only your own artifacts.\n"
	case ManagerReply:
		return "Answer the open questions addressed to you with send, kind reply. This step is read-only and changes no artifact; if an answer needs a change, say so in your reply.\n"
	default:
		return ""
	}
}

func renderContentID(bytes []byte) snapshot.ContentID {
	return snapshot.ContentID(sha256.Sum256(bytes))
}

func roleTemplate(view ContextView, root bool) (ProductText, string) {
	id, text := "manager", managerTemplate
	switch view {
	case ManagerView:
		if root {
			id, text = "root", rootTemplate
		}
	case ReviewerExpectations:
		id, text = "reviewer-expectations", expectationsTemplate
	case ReviewerCandidate:
		id, text = "reviewer-candidate", candidateTemplate
	}
	text += mandateTemplate
	return ProductText{ID: "markitect/context/" + id, Version: "3", ContentID: renderContentID([]byte(text))}, text
}

func contextFormat() ProductText {
	return ProductText{ID: "markitect/context-text", Version: "1", ContentID: renderContentID([]byte(renderFrameOpen + renderFrameClose))}
}

func productLiteral(id, text string) RenderSource {
	return RenderSource{Kind: ProductRenderSource, Product: ProductText{ID: "markitect/context-text/" + id, Version: "1", ContentID: renderContentID([]byte(text))}}
}
