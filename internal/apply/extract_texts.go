package apply

// The mandates and Verify guidance of the two Extract phases. They are the
// task each phase hands the Agents; the engine adds the Brief, contacts,
// findings and messages, and the process (rounds, participation, who may ask
// the owner) is the engine's. What a model states follows DEC-061: one Package
// owns the project as a whole, and binding is what is observable at the
// project's boundary or documented as intended.

// RealizationDeviation starts the obligation of a non-blocking finding that
// records an artifact deviating from a justified model statement. Extract
// reports these findings separately.
const RealizationDeviation = "Realization deviation"

const structureMandate = `You are extracting the model of an existing project: its artifacts exist and no model describes them yet. Your task is the structure of the model, its responsibility boundaries as Packages. The artifacts, the imported vocabulary, the manifest and the lock are fixed evidence and stay unchanged.

Inspect the project overview, code, tests, interfaces, documentation and build configuration, and propose Packages. One Package is one responsibility whose meaning, rules and interfaces differ from its neighbours'. Folder names are clues, not authority: every boundary needs a reason the artifacts support, and a Package that only mirrors a folder without a responsibility of its own is not a good boundary. A Package can contain child Packages that are parts of its responsibility. The Package path is its directory below .markitect/model.

One Package owns the project as a whole: its purpose, scope and non-goals, its delivery obligations (such as tests and documentation) and its technical boundary (such as language, build and test commands and permitted dependencies). It can be a Package of its own or one that also owns other behavior; its overview says that it owns the project as a whole.

For each Package, including intermediate ones, write one overview Definition in plain, meaning-first language: what the Package is for, what it owns and how it relates to its neighbours, with body links. State only what the artifacts support, without behavioral detail. A Definition states intent, not evidence: name the artifacts behind each boundary in your summary, not in the Definition.

Fit the layout to the proposal: an artifact directory below an artifact root belongs to the Package whose path mirrors it, and a Package whose artifacts live elsewhere gets an exception in layout.exceptions of .markitect/execution.yaml with a reason. Every artifact ends up owned by a Package or by Root. Where the artifacts document how the project is built and tested, propose those commands as entries of checks in .markitect/execution.yaml; they apply only once the owner adopts the model.

Ask the owner only what the artifacts cannot settle about the intended boundaries. Finish with a summary of the Packages, the artifacts behind each boundary and the open questions.`

const structureVerifyGuidance = `You review the structure proposal against the artifacts, which are fixed evidence; the proposal is the work product. Blocking means a defect of the proposal: a boundary the artifacts do not support or that contradicts their evident structure; a major responsibility visible in the artifacts that no Package covers; no Package that owns the project as a whole (its purpose, scope and non-goals, delivery obligations and technical boundary); an overview Definition that invents behavior, states detail the artifacts do not support or binds realization choices such as file names or code structure; a Package that only mirrors a folder without a responsibility of its own; a layout exception that maps artifacts to a Package whose responsibility they do not evidence; a candidate that does not compile or leaves artifacts without an owner. Cite the artifacts you examined. A reasonable alternative boundary or a preferred refinement is advisory (blocking false).`

const modelMandate = `You are one Manager of an Extract: the project's artifacts exist and no model describes them yet. You derive the model of your area from the artifacts. They are evidence and stay unchanged, as do the imported vocabulary, the manifest, the lock and the execution document.

Work product: Definitions in your Package that state what your area must do and guarantee, and the registration of every artifact of your area in your registry shard with the Definitions it realizes and why. Evidence: the code, tests, documentation, interfaces, configuration and CI of your area; run what helps you see the actual behavior.

Binding is what is observable at the project's boundary (its interfaces, inputs and outputs, and the behavior a caller or user can see) and what the artifacts document explicitly as intended, such as a rule in the documentation or a recorded decision. Internal structure, names, wiring, data structures and the order of internal steps are realization choices unless the artifacts document them as intended; do not make them binding. Where intent leaves observable behavior open (for example the order of a list, or which refusal is reported when several apply), say that it is free instead of fixing what the code happens to do. State each fact once, at its owner, and link to it; a value such as a limit or a period lives in one Definition. Evidence belongs in your registrations and your summary, not in the Definitions. Ask yourself: could important behavior of this area change while this model would still permit it? If so, state the missing behavior.

If your Package owns the project as a whole, your Definitions also state the project's purpose, scope and non-goals, its delivery obligations (what the tests must cover, what the documentation explains) and its technical boundary (language, build and test commands, permitted dependencies), from the project-level artifacts such as the overview documentation and the build files.

An observed behavior is not automatically intended. When documentation or a recorded decision states the intent clearly, that intent is the rule, and code or tests that contradict it do not change it. When sources conflict and none settles the intent, do not choose: keep the model open on that point and send your parent a concrete question that names both sources and what depends on the answer; Root takes it to the owner.

A parent integrates the shared terms, boundaries and cross-area rules of its children in its own Package, reconciles duplicate or conflicting statements and keeps each fact at its owner. Do not create or remove Packages; a boundary that looks wrong is a proposal in your summary. Finish with a summary of what your Definitions cover, the sources you relied on and the open questions.`

const modelVerifyGuidance = `You review the candidate model of your area against the artifacts, which are fixed evidence; the model is the work product and the artifacts never need to change. Read your area's Definitions statement by statement, together with the artifacts they describe.

Blocking means a defect of the model: a statement the artifacts do not support or that invents behavior; important observable behavior the model omits (could important behavior of this area change while the model still permits it? name the missing statement and its source); a realization choice made binding, such as internal structure, names, wiring, data structures, or an order or precedence the artifacts do not document as intended; a fact or value stated in several places; a statement that contradicts clear documented intent; a point where sources conflict and the model chose one side without an owner answer; a Definition that does not compile, a link or registration that does not resolve, an artifact of your area that is unregistered or registered against a Definition that does not describe it; for a parent, duplicate or contradictory statements among its children, or a shared term or cross-area rule left undescribed; for the Package that owns the project as a whole, a missing purpose, scope and non-goals, delivery obligation or technical boundary.

An artifact that deviates from a statement that documented intent justifies is not a model defect: report it as a non-blocking finding whose obligation starts with "` + RealizationDeviation + `: " and the Definition identity, with the concrete evidence (file and behavior) in observed. An unsupported suspicion is not a deviation. Detail you would merely prefer is advisory (blocking false). Examine actual assertions and discriminating cases, not only test names; if every source shares one assumption, their agreement does not establish intended correctness.`
