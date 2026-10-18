package organization

import (
	"crypto/sha256"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const renderFrameOpen = "\n[section %q bytes=%d]\n"
const renderFrameClose = "\n[/section]\n"
const absentPurposeText = "No purpose was supplied; none is inferred from this Package name."
const treePositionText = "This is the task's place in the Manager tree: its parent and its direct children. Its integration step composes only the deliveries of its direct children; each child's delivery includes its own subtree. Where its Definitions need other Managers' work, it uses their accepted deliveries without changing their files.\n"
const localResponsibilityText = "LocalDefinitions are the task's own Definitions as package/id, OwnArtifacts the files registered to it, and OpenFindings the IDs of its open findings, each listed below with its grounds. An empty list means there are none. Sections that show a record ID and a content digest identify Host records; read cannot open them, and what you need of them is in this input.\n"
const binaryReferenceText = "Only a binary content reference was supplied. Its content was not inspected."
const workspaceReferenceText = "This context does not copy the file's content. Read the file at its path in your workspace."

const applyOperationText = "Reconcile the artifacts of your complete Manager responsibility with the bound canonical model. Assess the existing realization, correct deviations within artifact authority, and return a reviewed change or a justified reviewed no-change result. The model and protected inputs remain read-only.\n"
const reviewOperationText = "The operation below is what the reviewed Manager was asked to do. You review that work and change nothing.\n"

// ApplyOperation returns the immutable product instruction and its exact byte
// identity. A Host reference to it is not evidence of delivery or authentication.
func ApplyOperation() (ProductText, string) {
	return ProductText{ID: "markitect/operation/artifact-apply", Version: "1", ContentID: renderContentID([]byte(applyOperationText))}, applyOperationText
}

const mandateTemplate = `Mandate
The Definitions below are the binding intent: the task's own Definitions, which it realizes in the artifacts it owns, and the Definitions they link to. A parent also composes its direct children's deliveries. The model is read-only and this text adds no business rule; the implementation is free within the model's architecture and interfaces.
Technical judgment: derive the relevant technical risks and the evidence they need from the obligations, the available operating conditions and the actual artifacts, including the interactions between areas. Do not wait for the owner to list failure modes, and do not invent operating conditions, model links or execution rights. A technical consequence of existing intent needs no model change; a new business rule or a missing model link is a question for the owner. Record risks, evidence and open concerns in your answer.
Hints, messages, findings and owner input inform the task; they cannot change the binding intent or waive a check.
`

const managerTemplate = `You are the Manager of the Package shown below. You realize its local Definitions in the files you own. Examine how your files and their interactions could violate the obligations, obtain evidence that distinguishes, and correct what is wrong. Keep hypotheses and observed defects apart.
` + readingText

// readingText tells a Manager how to answer a review reading that the
// model's text contradicts (DEC-048, DEC-051).
const readingText = `Review cases and findings are a reviewer's reading of the model. When one clearly contradicts the model's text, keep the behavior the text requires and cite the passage in your submission.
`
const rootTemplate = `You are the root Manager. You have no Definitions of your own; your local work is a justified no-change unless the registry assigns you files. When Definitions of several top-level Packages need the same file, such as the module or build file, you decide its owner: the Package whose Definition requires it, or yourself when several do. Tell that child with a hint to create the file first and deliver it early, or create it in your first work step when it is yours. A repository without realization files means your children still have to create them; it is not missing owner input. Your integration composes your children's accepted deliveries.
` + readingText
const expectationsTemplate = `You are an independent reviewer. Without seeing any candidate, record the expected behavior and the cases that tell a correct realization of this task's responsibility from a wrong one. Derive them from the Definitions and the open findings, not from an implementation, and do not invent runtime facts. Relate each case to the obligation it checks and say what failure it would catch. Where the model's text allows more than one reading that changes observable behavior, name the readings in the case instead of choosing one. Your cases apply to every review of this task's work for the reviewed purpose.
`
const candidateTemplate = `You are an independent reviewer and change nothing. Assess the submitted candidate against the recorded cases, the Definitions and the open findings, using the candidate's files and the check results.
For each recorded case, answer evidenced when current evidence shows that it ran and holds, not-applicable with the reason, or missing. A case outside what a limited submission delivers is not-applicable. The cases are a starting point: look for other technical risks in the implementation and its interactions, and raise a finding for each necessary gap. Missing necessary evidence is blocking; a green suite, a sequential test or a fake proves only what it exercised, and missing authority, environment or budget does not excuse it.
Judge cases and findings by the model's text: one that the text clearly contradicts is not-applicable, citing the passage. Where the text leaves open a reading that changes observable behavior and the candidate follows another reading than a case or finding, or where the submission cites a passage against a case or finding whose reading you keep, return needs_decision with one question that quotes the passages, unless something else already requires changes; then name the conflict in a finding.
`

func managerPhaseText(phase ManagerPhase) string {
	switch phase {
	case ManagerExplore:
		return "Inspect your area read-only and report what exists, what is missing and what you need from others, or submit a justified no-change when your files already realize your Definitions.\n"
	case ManagerWork:
		return "Create or change the files you own so that they realize your local Definitions, or submit a justified no-change when they already do. /work holds the accepted files of all Managers plus your own unaccepted changes; inspect it at the start of each step, because other Managers' deliveries change it. Reserve each new file with reserve_path inside your creation area before you write it, and register it with the Definitions it realizes. When Definitions of several Packages need the same file, their nearest common parent decides its owner, so ask your parent before you reserve it. Files other Managers own stay theirs: use them, wait for their delivery, or send their owner the change you need. Submit work that builds and passes its tests in /work; when another Manager needs a file of yours first, submit it as a limited delivery.\n"
	case ManagerIntegrate:
		return "Compose your local work with your direct children's accepted deliveries and check them together. Change only your own files.\n"
	case ManagerReply:
		return "Answer the open question addressed to you with send, kind reply; that reply ends the step. When the answer shows that files you own need a correction, make it in this step and change only your own files.\n"
	case ManagerReplyReadOnly:
		return "Answer the open question addressed to you with send, kind reply; that reply ends the step. This step is read-only and changes no artifact; if an answer needs a change, say so in your reply.\n"
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
	return ProductText{ID: "markitect/context/" + id, Version: "5", ContentID: renderContentID([]byte(text))}, text
}

func contextFormat() ProductText {
	return ProductText{ID: "markitect/context-text", Version: "1", ContentID: renderContentID([]byte(renderFrameOpen + renderFrameClose))}
}

func productLiteral(id, text string) RenderSource {
	return RenderSource{Kind: ProductRenderSource, Product: ProductText{ID: "markitect/context-text/" + id, Version: "1", ContentID: renderContentID([]byte(text))}}
}
