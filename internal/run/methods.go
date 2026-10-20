package run

import (
	"slices"
	"strings"
)

// methods are the reusable method guidance every Manager, reviewer and helper
// may consult with look_up topic method: concise names in the standing context,
// details on request. They describe how to do good work; they grant no right,
// change no model, add no participant and override no gate.
var methods = map[string]string{
	"planning":     "For each own obligation, decide the method and the evidence that will show it is met: the test, check or observation. Name your own artifacts and interfaces, what you need and from whom, what you provide and to whom, and the order of your work. Turn open assumptions into questions early, through your parent. A parent plans its own part and how it integrates its children; the children plan theirs.",
	"preparing":    "Lay only the foundations your plan and your children need: module and build files, interfaces, configuration and test harness, following the project's conventions. Keep them minimal and usable: they must build or load, not yet behave. Register every new file as you create it. Preserve an existing foundation that fits.",
	"implementing": "Realize each own obligation in your own area, with the tests and documentation the model requires; prefer tests that discriminate the required behavior. Use other owners' realizations through their contracts instead of rebuilding them. Run the project's build and tests yourself, keep your registry exact, and cover every own obligation with its artifacts and evidence in your result.",
	"integrating":  "Read each child's actual artifacts and evidence, not only its summary. Check the interfaces between your children and with your own work, and that no Definition is realized outside its owner's artifacts. Return a gap to the child that owns it with a concrete reason; never fix a child's area yourself.",
	"reviewing":    "Derive the expectations from the authoritative sources of this Pass before assessing its candidate (the bound model for Apply; fixed artifacts and documented intent for Extract), then inspect the actual behavior: run the checks, read the assertions of the tests, try discriminating cases and the interaction across areas. Check unusual permitted inputs as well as invalid inputs at the public boundary: parsing and transport must preserve the domain contract. Record precise claims, reproducible methods, expected results and actual observations per Definition; inspect assertions rather than treating a passing suite as complete coverage. Cite the obligation and the observed evidence for every genuine finding, keep optional improvements as notes, and reassess the findings you raised earlier on this candidate.",
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
