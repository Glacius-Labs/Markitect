# The core

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
The Area of class Core; [planning](../../processes/planning/README.md), [the Triager](../../agents/triage.md), [the Recorder](../../agents/record.md)

## Rule
The core is treated like a temple: it decides more than anything how well and how fast the project develops. In the core an Admin usually plans; afterwards it is built like everywhere else, with no check before `main`, and stories with fixed tasks run alone there too. When a feature needs something of the core that does not exist, its person tells their agent; it goes into the context and becomes a finding like any other. When an Admin decides for it, a story arises in the core, and the stories that need it depend on it. Usually nobody creates the story in the core themselves and nobody builds a detour around the core; doing it is a deviation of level `high`. After every change to the core, what depends on it is checked, and everything against the spec. The deeper a place lies, the heavier its smell weighs: a smell in the core weighs more than one in a module.

## Example
The Area Lending needs the catalog's copies to carry a shelf mark. The Developer says so; the Admin decides; a story in the core adds the shelf mark, and the story in Lending depends on it.
