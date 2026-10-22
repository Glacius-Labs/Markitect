package run

import (
	"slices"
	"strings"
)

// semanticAssessment is a model-directed review method, not a machine proof or
// a project-specific catalogue of troublesome inputs. It is delivered in the
// assignment so using this method does not depend on an optional lookup.
const semanticAssessment = "Derive outcomes before reading a test's expected result. From each material public contract, distinguish the permitted and rejected value/state classes and the exact advertised interface forms; cross the value domain with the interface syntax, including values that may be mistaken for control syntax. A substituted encoding does not establish the advertised form. Trace a representative material condition through the public entry, responsible logic and persistent or external effect, including rejection without the forbidden effect. Before relying on a green receipt, read its complete invocation and assertions: does the model require that outcome for these preconditions, and which materially different permitted or rejected case could still pass this script? Exercise that countercase when it bears on a required claim; otherwise leave the missing evidence open. Do not infer that an input is invalid because the candidate or its test rejects it. A source-only claim must explain why execution is inapplicable. Do not enumerate every possible input or invent model policy."

// methods are the reusable method guidance every Manager, reviewer and helper
// may consult with look_up topic method: concise names in the standing context,
// details on request. They describe how to do good work; they grant no right,
// change no model, add no participant and override no gate.
var methods = map[string]string{
	"planning":     "For each own obligation, decide how it is shown to be met: the existing artifacts and tests where they already meet it, a new test with each change you plan or where the model requires one; the independent Verify probes the claims itself. Name your own artifacts and interfaces, what you need and from whom, what you provide and to whom, and the order of your work. Depend on another Manager's accepted basis when its contract suffices; depend on its complete result only when you cannot start without it, and wait for the results you integrate when you reach that point. Turn open assumptions into questions early, through your parent. A parent plans its own part and how it integrates its children; the children plan theirs.",
	"preparing":    "Lay only the foundations your plan and your children need: module and build files, interfaces, configuration and test harness, following the project's conventions. Keep them minimal and usable: they must build or load, not yet behave. Register every new file as you create it. Preserve an existing foundation that fits.",
	"implementing": "Derive material conditions, exceptions and interactions inside each bound Definition and required contract; record concise source references and justify their coverage in conditions. Realize each own obligation in your own area, with the tests and documentation the model requires; a change comes with a test that discriminates it, and behavior that already conforms needs no new tests or hardening unless the model requires evidence the artifacts lack. Use other owners' realizations through their contracts instead of rebuilding them. Run the project's build and tests yourself, keep your registry exact, and cover every own obligation with its artifacts and evidence in your result. In Implement, done/preserved public behavior needs discriminating tool-produced receipts through the actual declared entry and its parsing/transport/persistence, on the submitted stand after the last edit. Private validators, source reassurance and suite totals cannot close that behavior; use reasoned source evidence only for static/documentary properties. Unavailable required execution stays open. Implement completion submits to Verify and does not establish verified conformance.",
	"integrating":  "State your own integration conditions, source references, cases and evidence in integration, including at Root with no own Definitions. Establish relevant composed workflows and global invariants; an acceptance list alone is insufficient. Read each child's actual artifacts, assertions and probe receipts, not only its summary or success label, and ask which plausible violation of its claims could still pass them. Check the interfaces between your children and with your own work, and that no Definition is realized outside its owner's artifacts; probe a child's checked proposal when that discriminates. Return a gap to the child that owns it with a concrete reason; never fix a child's area yourself.",
	"reviewing":    "Derive the expectations from the authority of this run before you assess the candidate: the bound model for Apply; for Extract the fixed artifacts and documented intent, where the proposed model is the work product and never evidence for itself. Derive your own model-based conditions and counterexamples before assessing author coverage; every Definition ID can hide omitted material conditions. For each claim, ask under which permitted conditions it could fail: valid inputs wrongly refused, invalid inputs wrongly accepted, unusual permitted and rejected inputs, exact boundaries, state changes, cross-area interactions. Exercise them through the public interface, including parsing, transport and persistence; run them as probes with the tool check, stating claim and expected outcome first, and read the assertions of existing tests rather than counting them. Cite per claim a receipt or an inspected source location; distinguish execution, inspection and inference. Raise a genuine finding with its obligation and observed evidence, keep optional improvements as notes, and reassess the findings you raised earlier on this candidate.",
	"escalating":   "Escalate only what you cannot decide within the model and your authority. State the question, the options, what depends on the answer and your recommendation, and keep working on what does not depend on it. A clarification never changes the model; a model that is erroneous or contradictory is Root's to report.",
}

// methodNames lists the method guidance names in order.
func methodNames() []string {
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// methodIndex is the concise metadata of the method guidance for prompts.
func methodIndex() string {
	return "Method guidance with look_up topic method (id one of " + strings.Join(methodNames(), ", ") + ")."
}
