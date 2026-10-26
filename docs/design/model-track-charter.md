# Canonical model track: charter v0.2

**Status:** approved by the owner on 26 October 2026, with the recommendations for D1–D5 ([source](../concepts/sources/model-track-approval-20261026.md)). v0.2 applied an independent review of the first draft and one point from the Evaluation session. Recording it here changed only its status, tense, link paths, the name of the leading session, a "content Packages" precision and one grammar fix; the approved text is otherwise unchanged.

**Basis:**
- the owner's [founding intent](../concepts/sources/founding-intent-20261026.md);
- an earlier session draft of model quality criteria, not in the repository;
- the register entries named below.

## Goal

Make the canonical model serve its purpose: **typed programming in prose.**
- People decide intent.
- AI agents write the model correctly through skills and tooling.
- The compiler catches structural errors.
- Models are cut like good code, so that the Manager hierarchy and every evaluation build on a sound model.

## Why now

The models that exist do not look like typed programs.
- **Makerspace c9** ([Phase C model](../../tools/comparison/phase-c/model/README.md)):
  - only the generic `software-basics` Kinds Concept, Rule and UseCase;
  - every `spec` empty;
  - no architecture: its program Rule leaves "internal package layout and design" to the implementer;
  - all 15 Packages form one **link** cycle (102 cross-Package edges);
  - 49 backticked field names are used in 3 or more Packages without a Definition of their own.

  The last two figures come from a throwaway session script, not in the repository.
- **Self-model:**
  - 26 content Definitions and no links at all;
  - 16 of its 18 content Packages hold one Definition each;
  - it embeds its own copy of `markitect/core` with `dependencies: {}`.

The evaluation depends on this track. In the owner's words: "Our evaluations of course are not good if the canonical model is bad by itself". D5 decides how the next study waits for it.

## Register entries this track builds on

| Entry | What it already says | Consequence here |
|---|---|---|
| [DEC-003](../concepts/register.md#dec-003-the-compiler-is-a-borrowed-concept), concept §1.3 | Structural compilation is deterministic; the typed-language analogy covers early structural errors | Matches the founding intent; nothing to change |
| [DEC-005](../concepts/register.md#dec-005-model-granularity-is-the-authors-responsibility) | Granularity is the author's choice | Criteria guide it; they don't forbid it |
| [ENH-002](../concepts/register.md#enh-002-model-quality-linter-and-metrics), [OQ-002](../concepts/register.md#open-questions) | A model-quality linter is an accepted direction: findings and recommendations, **not hard gates by default** | W3 output is warnings in the product. Hard rules apply only to our own reference and evaluation models |
| [ENH-003](../concepts/register.md#enh-003-test-to-statement-binding-and-test-review) | Binding tests to statements is a direction, not built | CRAP-style coverage waits for it |
| [DEC-027](../concepts/register.md#dec-027-strict-links-with-an-optional-optimizer) | Authored connections are the baseline for context; an optional Optimizer may propose missing ones | The linter can find missing connections; Apply still never adds them |
| [DEC-028](../concepts/register.md#dec-028-context-follows-need-to-know-and-starts-simple), [OQ-006](../concepts/register.md#open-questions) | Context starts simple; how much context a Manager needs is open | W1 question 4 answers OQ-006 with evidence before anything changes |
| [DEC-061](../concepts/register.md#dec-061-what-an-extracted-model-contains) | A project Package owns project-wide obligations; binding versus free choices | W2 builds on it |
| [DEC-011](../concepts/register.md#dec-011-detailed-planning-of-future-ideas-waits) | Detailed planning of future ideas waits for four conditions | The approval lifts the hold for this bounded track, as [DEC-065](../concepts/register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration) did for the cockpit |
| [DEC-072](../concepts/register.md#dec-072-stable-v1-justified-by-data) | A stable V1 justified by data, including the comparison | See D5 |

## Workstreams

| # | Workstream | Deliverable | Done when |
|---|---|---|---|
| W1 | **Fitness review** of core, compiler and context selection against the typed-language analogy | Short report: what fits, what is missing, a recommendation per gap. Any language gap is fixed in the core, because Libraries get no compiler semantics | The questions below are answered with evidence. A scratch `software-api` model (four Kinds, three remodeled operations; not in the repository) passes `markitect check`, and deliberate structural errors are shown to be caught |
| W2 | **What a well-modeled project contains**, building on DEC-061 | Guide: which concerns a model covers (domain, API, data, architecture, way of working), the level of detail, a worked example | One pilot model written to the guide and reviewed against it |
| W3 | **Quality criteria v0.2** in the code analogy | Hard rules and signals with a check each, plus an anti-pattern catalog. In scope: imports and links (cycles, coupling), size, missing types, unused typed fields. Later research: CRAP-style coverage (needs ENH-003) and model mutation (needs Apply runs) | Reviewed. Thresholds are provisional until W6 |
| W4 | **Standard Libraries, first outline** | Library list and Kinds, oriented on Go/C# (`values`, `api`, `data`, `architecture`, `practice`); versioning and a compatibility promise; Kubernetes and Terraform as orientation for desired state, plan, apply and drift | Owner review of the outline; `api` and `values` drafted and compiled |
| W5 | **Modeling skill and tooling** | A skill that makes an agent model to W2 and W3 with the Libraries. A separate tool under `tools/`, not yet a product CLI change, for W3 signals and "find all references" / impact | An agent with the skill writes a model from neutral requirements that passes W3 and an independent fidelity review against the requirements, without manual repair |
| W6 | **Validation on a pilot** | Makerspace remodeled beside the frozen Phase C models (w0, c1–c9 stay unchanged). Amended on 26 October 2026 ([owner decision](../concepts/sources/self-model-pilot-deferred-20261026.md)): the self-model pilot is deferred until Markitect is stable, after V1 acceptance and the studies, so that Markitect's own build does not depend on Markitect while it is in development | A sanity check, not a calibration: bounded Apply runs on the old and new Makerspace model, under a dispatch, compare delivery, coordination and cost |

**Order:**
1. W1 first.
2. W2, W3 and W4 in parallel; they inform each other.
3. W5 builds on them.
4. W6 closes the loop.

**How the evaluation consumes the result:** portfolio models are written with the W5 skill, pass W3 and get an independent fidelity review. If models carry typed fields, the renderers that produce the single-agent arm's documentation ([requirements.py](../../tools/comparison/requirements.py), [phase-c/spec/render.py](../../tools/comparison/phase-c/spec/render.py)) must render them faithfully, and the equivalence check covers that.

## W1 questions

1. **Expressiveness.** Do object, list, enum and reference with cardinality cover what Libraries need? Known gaps:
   - maps;
   - variants or unions;
   - Kind extension, such as a project Kind that refines a Library Kind;
   - references that accept several target Kinds;
   - string formats;
   - constraints across fields.

   The [language contract](language-contract.md) (§11) deliberately excludes user-defined metaclasses and a general condition language.
2. **Dependencies.** Three candidate mechanisms:
   - file-local Package imports, which are checked and required for every cross-Package reference or link;
   - typed `spec` references;
   - body links, which "alone" declare no dependency (language contract §4.3).

   Language contract §8.1 allows import and reference cycles inside one model. Context selection treats references and links alike. Which of them is "the dependency" in the analogy, and what should the linter read?
3. **Encapsulation.** Code has exported and internal symbols. Does a Package need an explicit interface for clean boundaries, or is that a Library convention?
4. **Context selection.** A Manager gets:
   - its own Definitions;
   - one hop of outgoing references and links, plus local incoming ones;
   - **modeled scope**: Definitions whose scope field covers the Package, configured in `execution.yaml` ([context_scope.go](../../internal/organization/context_scope.go));
   - type contracts, followed transitively.

   The observed Makerspace limit is the second hop: rule bodies reached only through the permission matrix do not reach each operation's Manager ([Phase C model README](../../tools/comparison/phase-c/model/README.md)). Would modeled scope solve that? Answer OQ-006 with evidence, within DEC-028.
5. **Libraries in practice.** Today every dependency is passed with a repeated `--dependency PATH` and locked by version and digest. Is that enough for standard Libraries, or does it need a module-proxy-like source?
6. **Tooling gaps.** What does `context` already give, and what is missing: "find all references", impact of a model diff, a linter?

## Constraints

- Language semantics, compiler, snapshot values, organization derivation and accounting are a protected foundation (concept §5.5). Changing them needs an explicit owner decision, never a side effect.
- Running work is not disturbed: Phase C, the S1–S3 product fixes, and the K-S/K-R probe (an exploratory study of a self-maintaining single agent against one agent given rendered docs). No local Go during quiet-machine measurements.
- Real Apply runs only under a concrete bounded dispatch, under the standing evidence authorization.
- Documentation in English (DEC-069). Owner statements stay verbatim in sources.

## Owner decisions (decided 26 October 2026)

The owner approved every recommendation below ([source](../concepts/sources/model-track-approval-20261026.md)).

| # | Question | Decision |
|---|---|---|
| D1 | Is this track part of V1, or parallel? V1 §2.2 excludes interactive model authoring | **Parallel**, not blocking V1 technical acceptance. The approval also lifts DEC-011 for this bounded track |
| D2 | DEC-003 already matches the founding intent | **No new entry.** At most a cross-link from DEC-003 to the founding-intent source |
| D3 | Dependency mechanism and body-link semantics (W1 question 2) | Decided after W1, with evidence |
| D4 | Where Libraries live. Today `markitect-core` is under `testdata/core` and `software-basics` 0.2.0 under `testdata/library/vocabulary` | A new top-level `libraries/` folder with one owner shard, versioned like `markitect-core`. `software-basics` is absorbed into the new Libraries and kept frozen only for the existing fixtures |
| D5 | Does the next comparison study (v0.3, the planned main study, and its follow-up; drafts not in the repository) wait for W1–W6, or run on the current models with a stated limitation? Under DEC-072 the comparison is V1 evidence | **It waits at least for W1–W4**, because models written the old way test Markitect with the problem it was built to solve |

## Working mode

- The session "Markitect evaluation and canonical model design" leads W1 and W3 and drafts W2 and W4.
- Design-heavy pieces run on Opus, others on Sonnet with high effort.
- Every deliverable gets an independent review before it reaches the owner.
- Shared files (concept, register, plan, AGENTS.md) change only through the Coordinator.
- First step after approval: W1, starting with the scratch `markitect check` once the probe has ended.
