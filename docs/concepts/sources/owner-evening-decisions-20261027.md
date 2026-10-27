# Owner decisions: G0 answers, lanes, priority and V1's acceptance scope

Date: 27 October 2026, UTC afternoon and evening. The decisions come from direct owner input, written in English and German, in two Claude Code sessions:
- Sections 1 to 4 are from the session "Markitect evaluation and canonical model design", later renamed "Lanes overview".
- Section 5 is from the coordinator's session, "Lane 1: Host variant lead".

The coordinator read sections 1 to 4 in that session's transcript, and section 5 is in the coordinator's own session.

How the owner's words are recorded:
- The owner statements are verbatim.
- Where the owner chose a prepared answer, the record says so. That covers an option in a question dialog and an answer button. The session wrote the wording of such an answer, and the owner's act is the choice.
- A session's transcript stores a typed message and an answer button's text in the same way.
- The context between statements is the recorder's summary and is labeled as such. The recorded meaning is the recorder's reading and adds no decision.

## 1. A referee in the Git design (15:42 UTC)

Context (recorder's summary): the G0 draft for [DEC-078](../register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review) asked whether a referee is acceptable while a Run goes on. The proposed referee would refuse pushes that break a rule, run the required checks and keep the score, and it would never direct a Manager.

> Yes, a referee that only refuses bad pushes, runs checks and keeps the score is fine. Continue with the draft and its other questions.

## 2. The other G0 questions (16:03 UTC)

Context (recorder's summary): the session asked four questions in a question dialog, each with options the session had written. The owner chose these options. Questions and labels are verbatim:

| Question | Chosen option |
|---|---|
| How fine should each Manager's harness be? | B: free Git + team tools (Recommended) |
| Own branch per Manager, or shared branches? | One per Manager (Recommended) |
| How should the result land on your target branch? | One commit, fail if moved (Recommended) |
| Build a small throwaway prototype before any rebuild? | Yes, prototype first (Recommended) |

## 3. Lanes (16:11 and 16:14 UTC)

> Wait we need to structure first. As you can see we have already three models for markitect. One as we developed all the time with the host as coordinator. Second Claude native Plugin. Third our current Prototype. And of course classic agentic coding. So i think we should create a branch for all of those and implement each one of them. We also need one session or something to create our evaluation suite in a way, that we can hook in each one of them and have the same job and same way of judging the outcome. Also we need one session where we might discuss the windows, linux problems. One session for the canonical model discussions. One session for skills and how to modify the model and how to design it. One for what the compiler can do. One for what agentic commands we can do. And maybe also one for how to continiously modify and apply the canonical model, since at some point we want external triggers being converted into something like tasks, which result in model changes and applications of it. Not all of the changes should be done manually. Also we might want to at some point build the "IDE" or Cockpit as a UI just like Claude Desktop or the ChatGPT App for visualization since at some point we also must understand the canonical model. Those are all areas i can think of.
>
> Id like to partition those so that we have independant workstreams on all of that and dont mix everything in one chat. Also each variant or prototype of Markitect should get an own coordinator or responsible session, which can create subsessions for parallel work. We dont want to loose track here

Context (recorder's summary): the session proposed eleven lanes. Each lane has a responsible session, and the three earlier sessions are reused. Classic agentic coding became the baseline arm of the evaluation lane. Shared code stays on `main`, and topic lanes take stock before they build. The owner replied:

> Yes, start the lanes with this split: reuse the three existing sessions, start the eight new ones, set up the sidebar sections and the overview page, then hand over G0 and the model-quality log.

## 4. Priority and tracking (17:57, 20:15 and 21:06 UTC)

> Look again what each one of them is doing. Then lets plan what we should prioritize. The topics sessions are just there so when i have some time i can discuss those topics and maybe refine them. The working streams and evaluator are currently our primary goal i think. It should achieve that each of them gets a stable version and can be evaluated and then we can decide which one of them performs well in which parameters of a project. So we need a simple plan here that does not get us overwhelmed and chaotic.

Context (recorder's summary): the session proposed a three-step plan for lanes 1 to 4: one yardstick, a stable version per variant, and a comparison. The topic lanes would pause. The workstreams overview, `docs/workstreams.md`, records the plan. The owner wrote:

> Yes, send the plan to the lanes: topic lanes pause after their page PR, L1–L4 follow the three steps.

Context (recorder's summary): the session then proposed that each lane's state page be the only place for that lane's state, and that it stop messaging the lanes on its own. The owner wrote:

> Yes: lane pages as the only state, status from you on demand, no more messages to lanes unless I ask.

## 5. V1's acceptance scope (late evening)

Context (recorder's summary): the coordinator's lane page had asked whether V1's acceptance on `main` covers today's publication, with the Git variant judged after its prototype. The coordinator recommended yes. It put the question to the owner with two prepared answer buttons:

> Soll V1 auf main mit der heutigen Veröffentlichung abgenommen werden?

The owner chose the first button. Its text, written by the coordinator, reads:

> Ja: V1 auf main wird mit der heutigen Veröffentlichung abgenommen; die Git-Variante beurteilen wir nach ihrem Prototyp.

English translation, for reference: "Should V1 on main be accepted with today's publication?" — "Yes: V1 on main is accepted with today's publication; we judge the Git variant after its prototype."

## Recorded meaning

Recorder's reading:

- **G0 (sections 1 and 2).** These are the answers to the G0 questions:
  - A passive referee at the Run remote is acceptable.
  - Each Manager works freely with Git in its own clone and uses team tools for every step that reaches another Manager (harness level B).
  - Each Manager has its own branch.
  - Publication is one commit that fails closed when the target moved.
  - A throwaway prototype comes before any rebuild.
  
  The Git variant's G0 design, `docs/design/agent-git-collaboration.md`, holds the details.
- **Lanes (section 3).** Markitect's work runs in eleven lanes, each with a responsible session, as listed in the workstreams overview, `docs/workstreams.md`. Variant leads may start sub-sessions.
- **Priority and tracking (section 4):**
  - Lanes 1 to 4 build, in the three steps of the overview, and the topic lanes pause once their state page is up.
  - A lane's state page is the only place for that lane's state.
  - The lanes overview session reports status when the owner asks, and messages the lanes only when the owner asks.
- **V1's acceptance scope (section 5):**
  - V1's acceptance on `main` covers today's publication: native publication into the working tree on Linux, and the runner's hand-back as a branch.
  - The Git variant's publication ([DEC-078](../register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)) is judged after its prototype and is not part of V1's acceptance.
  - This answers the delivery plan's question 5 on V1 acceptance. The recorder reads it as the scope of that acceptance, not as the owner's acceptance of V1 itself, which still follows step 5 as a separate decision.
