# Relationship types on the Makerspace specs: measurements — 8 November 2026

This session read every link of the archived Makerspace spec and asked what relation it expresses. It then recomputed what the change steps would put back on the list of `markitect status` if each relation made its own input of the record, and checked every item that would drop for a re-verification that was needed. It repeated the mechanical part on the 5× project. It started no agent beyond an independent reviewer, who judged a blind sample of the classification and recomputed the figures, and it changed no tool and no language. The reading is this session's and decides nothing; the [relations plan](../concepts/relations-plan.md) draws the decision from it ([DEC-104](../concepts/register.md#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them)), under the owner's rule of 8 November 2026 that the spec behaves like code and has no link for reading only. The record measures both forms the plan weighed: two link keywords, `id` and `part`, and the decided form without new syntax.

**What the numbers show:**

- **Two relations of code hide in the plain link.** Of Makerspace's 360 written links between Definitions at c8, 52 name a record by its identifier ("naming an existing [[members/member]]") and 58 are an Interface listing its own operations and records. A further 15 are an Interface's list of the modules it uses, which is a module dependency. On the 5× project the same three patterns find 296, 281 and 137 of 1792.
- **The list would be a sixth shorter on Makerspace and a quarter on the 5× project, with no needed item lost.** With identity references and part lists as their own relations, and the "Uses" lists linking Packages, Makerspace's eight change steps put 113 items on the list instead of 133: besides the 84 Definitions that changed, links put back 29 instead of 49. The 5× project's steps put 348 instead of 468; the step that changed `members/member` and the members Interface falls from 138 items to 42. Each of the 36 and 126 pairs of Definitions behind the dropped items was checked against what its target's change said, and none changed anything the source relies on.
- **The new inputs added nothing.** No target of an identity reference or a part list changed its Kind in any step, and no existing Definition joined or left an Interface's list; only new Definitions did, which are on the list anyway.
- **The gain comes from different places.** In Makerspace it is the part lists and the "Uses" lists; in the 5× project more than half comes from the identity references. A heading convention, which catches the lists only, catches all of Makerspace's gain and less than half of the 5× project's.
- **The same gain needs no new syntax.** Each Makerspace Interface's lists of operations and records are exactly its Package's public Definitions that are not Rules, on all nine steps, so visibility already says what they say. And an identity Definition per named record, which holders link, takes the identity references off the list as well. Together with "Uses" lists linking Packages, the steps put 113 items on the list on Makerspace and 354 on the 5× project, with no change to the language.
- **No module edge rests on these links alone.** Two of Makerspace's 23 module edges, and 35 of the 5× project's 138, rest only on identity references, part lists or "Uses" lists, but every one of them is carried by a "Uses" list too, which keeps it as a Package link.
- **Reliance sharpens the other measures.** Counting only plain links, `members/member` stops being a hub (fan-in 24 to 6; 90 to at most 6 in the 5× project), and three Packages fall into groups that read as sub-areas, which the Interfaces' lists hid.
- **Two alternatives cost more than they give.** Letting every Operation rely on what the `Operation` Kind links would add 64 items on two steps. A link for reading only, for pointers and code terms, would take 6 more items off Makerspace's list, but the owner's rule keeps such links out of the spec.

## Source and method

**Fixtures.** The archive point [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks): `tools/evaluation-tasks/makerspace`, steps `w0` and `c1` to `c8`, each with its manifest, model and the finished registry of `references/<step>`, and `tools/evaluation-tasks/makerspace-large`, the 5× project, steps `w0` to `c8`. Each step was copied into a scratch project folder, `markitect tidy` wrote the lock, and `markitect check --spec` passes on every step. Only public fixtures were read.

**Tool.** `markitect` built from Main `ad237173` (0.23.0-dev). `list --deps --json` gives the Definitions, their Kinds and their Packages. The written links were read from the Definition files, outside code spans and code blocks, each with its line and the bold label of the list item or paragraph it sits under. A probe on a copy of c8 confirmed that `[[id::members/member]]` fails today as `link.syntax`, and that an Interface whose "Uses" list links Packages keeps its five module edges.

**Throwaway scripts,** outside the repository:

| Script | Computes |
|---|---|
| links | Every written link of a step, with its source, target, line and label |
| classify, hand judgements | The role of each written link: six patterns, then the hand judgements below |
| propagate | What each change step puts on the list, today and with relations, and the pairs whose items drop |
| analyses | Package edges, fan-in, every-link reach, cohesion, and Kind links inherited |
| patterns | The same gain without new syntax, and the cost of adopting either form once |

**Roles.** Each link got a role, the reading of why it is there, and the role decides its relation as a keyword; the decided form writes the same roles as identity Definitions, visibility and Package links:

| Role | As a keyword | How found |
|---|---|---|
| Identity: an identifier that names an existing record | `id` | Pattern: "naming an existing [[X]]" |
| Part list: an Interface listing its operations and records | `part` | Pattern: links under **Operations** and **Records** in an Interface |
| "Uses" list: an Interface listing the modules it uses | A link to the Package | Pattern: links under **Uses** in an Interface |
| Format, result shape, field type, capability, a rule applied, content relied on | Plain link | Patterns for formats, results and capabilities; the rest by hand |
| Pointer for the reader, a code written in the source and linked to its explanation, a link from the general back to its user | Plain link, or no link | By hand |

The unit is a written link, keyed by its source, its target and the text of its line, so a line that stays the same over the steps is judged once: 413 distinct lines over Makerspace's nine steps, 250 by six patterns and 163 by hand. The hand judgements asked one question: could a change to what the target states change how the source must be realized, or make its verdict wrong? They found no identity reference or part list that the patterns missed. What they told apart, reliance from pointers and code terms, is a plain link either way under the owner's rule, so the relations on Makerspace come from the three patterns alone. The 5× project was read by the six patterns only; the 579 lines they leave count as plain links, so its gains are lower bounds.

**What goes on the list.** Inputs from the spec only, as in the [7 November record](spec-design-measures-20261107.md): the archive holds no realization, so file inputs are left out. A step's changed Definitions are those whose text differs from the step before, added ones included.

- **Today,** a changed Definition puts back itself, every Definition that links to it, and every Definition of it as Kind. This column repeats the 7 November record's one-step column exactly, on both projects.
- **With relations,** a plain link makes the target's text an input of the source; an `id` or `part` link makes only the target's Kind an input; a `part` link also makes the whole an input of the part, so that a whole gaining or losing the part puts the part back on the list; a link to a Package is no input. A Definition is on the list when any of its inputs differs from the step before.

**Misses.** For each pair whose item drops, a source and a target that changed at that step, this session read the target's diff and the source's linking lines and asked the question above of that change.

**Independent check.** An independent reviewer recomputed the figures with its own link reader and propagation, from the step folders and without this session's scripts or calls; every figure matched but one, which is corrected here (308, [below](#the-links-at-c8)). It also sampled the dropped items of both projects, all 67 identity holders at the 5× project's c4 among them, and found none that was needed. Before seeing this session's calls, it judged a random sample of 50 of the 413 lines, 40 judged here by hand and 10 by pattern, with the same question. It agreed on 49. The one difference is `billing/credit`, which applies credit "as [[billing/issuing-invoices|issuing invoices]] describes in step 5": a pointer here, doubtful, and reliance for the reviewer, also doubtful. Under the owner's rule it is a plain link either way. The sample held 6 identity references and part lists, and the reviewer read all 6 as naming their target only.

## The links at c8

Makerspace c8 has 382 Definition links: 336 pairs of Definitions joined by 360 written links, and 46 uses of a project Kind. Beside them stand 21 links to Packages, which are no inputs already.

| Role | Relation | Written links | Example |
|---|---|---|---|
| format | plain | 62 | ``**Fields**, in the formats of [[values]]`` |
| result shape | plain | 46 | ```reservation`: a [[bookings/reservation]] record`` |
| content the source relies on | plain | 44 | `bookings/check-out`: "A usage [[billing/charge]] is created for the member" with its fields |
| applies a rule | plain | 37 | "Rules of other modules that apply here: [[members/membership]]"; "Its amount follows [[money]]" |
| capability of another module | plain | 8 | ```member_suspended` when the member is not in good standing ([[members/members-interface]])`` |
| field type | plain | 6 | ```tier`: a [[members/tier]]`` |
| pointer | plain, or no link | 21 | `project`: "Four policies apply to every operation: [[authorization]], [[audit]], …" |
| code term | plain | 10 | `order-of-checks`: ``→ otherwise [[refusal-codes\|`invalid_argument`]]`` |
| back reference | plain, or no link | 1 | `values`: "Arguments are checked against them as [[argument-values]] states" |
| part list | `part` | 58 | In `bookings/bookings-interface`: ``**Operations:** [[bookings/create-reservation]], …`` |
| identity | `id` | 52 | ``- `memberId`: identifier, naming an existing [[members/member]]`` |
| "Uses" list | Package link | 15 | ``**Uses:** [[members/members-interface]], [[billing/billing-interface]]`` |

**The pairs.** Of the 336 pairs, 224 are joined by at least one plain link, 39 only by identity references, 58 only by part lists and 15 only by "Uses" lists. So the record would take the text of 270 targets as inputs (224 pairs and 46 Kinds) instead of 382, the Kind alone of 97, and nothing of the 15 that become Package links; and each of the 58 parts would know the whole that lists it.

**Whether the Kinds at both ends tell the relation.** The best rule from the pair of Kinds alone, the more common reading for each pair, tells identity references and part lists from the other links for 308 of the 360 written links. It misses exactly the 52 identity references, where the same pair of Kinds carries both: an Operation links a Concept by identity in 32 places and for its result or format in 91, and a Concept links a Concept by identity in 20 and for a field's type or format in 23.

## The change steps

**Makerspace,** Definitions on the list (spec inputs):

| Step | Changed | Today | `id` and `part`, "Uses" lists as written | `id` and `part`, "Uses" lists linking Packages |
|---|---|---|---|---|
| c1 | 5 | 9 | 9 | 9 |
| c2 | 23 | 35 | 35 | 32 |
| c3 | 2 | 3 | 2 | 2 |
| c4 | 14 | 24 | 19 | 19 |
| c5 | 11 | 20 | 20 | 16 |
| c6 | 12 | 17 | 17 | 16 |
| c7 | 12 | 17 | 12 | 12 |
| c8 | 5 | 8 | 7 | 7 |
| **Total** | **84** | **133** | **121** | **113** |

Besides the 84 changed Definitions, links put back 49 today, 37 with `id` and `part`, and 29 once the "Uses" lists link Packages.

**The 5× project,** six patterns only, the rest counted as plain links:

| Step | Changed | Today | "Uses" lists as written | "Uses" lists linking Packages |
|---|---|---|---|---|
| c1 | 3 | 5 | 5 | 5 |
| c2 | 14 | 29 | 29 | 28 |
| c3 | 13 | 63 | 62 | 55 |
| c4 | 9 | 138 | 71 | 42 |
| c5 | 26 | 86 | 85 | 81 |
| c6 | 9 | 47 | 47 | 46 |
| c7 | 23 | 75 | 74 | 70 |
| c8 | 14 | 25 | 25 | 21 |
| **Total** | **111** | **468** | **398** | **348** |

Besides the 111 changed Definitions, links put back 357 today and 237 with the keywords.

## What drops, and whether it was needed

**Makerspace: 36 pairs, no miss.**

- **27 pairs: an Interface lists a part that changed.** The reports and their rows gained lockout minutes and credits (c2); operations gained a bulk discount (c3), the steward role in their permissions (c4), a refusal (c5), the new meaning of a valid certification (c6), links to the renamed billing Rules (c7), and a new slot granularity and opening hours on every day (c8). An Interface lists the operation or record and restates none of it. The codes it lists are its own module's, and the refusals the operations gained, such as `account_overdue`, are billing's.
- **9 pairs: an Interface's "Uses" list names an Interface that changed.** The billing Interface gained lockout credits (c2) and `account_overdue` (c5), the bookings Interface lockouts (c2), the training Interface a new meaning of a valid certification (c6). That one module uses another stays true. The Operations that rely on the changed capability link it in their refusals, as plain links, and changed themselves at the same step.

**The closest case** is `members/members-interface` at c4, when the steward role came in. It lists `members/user`, which changed, and it offers other modules "whether a member exists", which with a new role must still exclude stewards. That rule changed in `order-of-checks` (check 5) and in `authorization`, which both changed at c4 and are on the list themselves, and the architecture has the protocol module realize the order of checks.

**The 5× project: 126 pairs, no miss.**

- **67 pairs, identity:** at c4, `members/member` gained the standing `paused`. 67 Definitions name an existing member through an identifier, such as ``- `makerId`: identifier, naming an existing [[members/member]]``. None relies on the member's standing; what relies on standing links the members Interface, which changed at the same step.
- **51 pairs, "Uses" lists:** the members Interface gained "whether the member is paused" (c4, 29 pairs), the bookings Interface new operations, uses and codes (c3, c5, c7, c8), and the lockouts, waivers and loans Interfaces new capabilities. That one module uses another stays true.
- **8 pairs, part lists:** a record or report changed a field, such as `finance/charge-kind-row` gaining the kind `noShow`. Its Interface lists it and restates none of it.

**What stays on the list under the owner's rule.** Seven more Makerspace pairs would have dropped with a link for reading only: the project anchor's list of policies (`audit` and `authorization` changed) and its pointer to the `architecture`, `argument-values` pointing to the rules for computed values (three times), and `computed-times` pointing to `identifier-limits`; an eighth, `program` saying that a process starts in the initial state, was a doubtful call. None of their targets' changes reached what the source says, so these 6 items, 7 with the doubtful one, are the price of keeping pointers as plain links; the rule's other answer is to drop such a link.

## Forms of writing it, compared

How much of the gain a form of writing catches, as items on the list over the eight steps, the whole list and those that links put back:

| Form | Makerspace | 5× project |
|---|---|---|
| Today | 133, 49 | 468, 357 |
| The "Uses" lists link the Package, as the language allows today | 129, 45 | 420, 309 |
| A heading convention: links under **Uses**, **Operations** and **Records** | 113, 29 | 415, 304 |
| Identity references only | 133, 49 | 401, 290 |
| The keywords: `id`, `part`, and Package links in the "Uses" lists | 113, 29 | 348, 237 |
| No new syntax: identity Definitions, no part lists, Package links in the "Uses" lists | 113, 29 | 354, 243 |
| For comparison, also a link for reading only, which the owner's rule excludes | 107, 23 | 348, 237 |

A Package link already makes the edge and is no input of the record, so linking `[[members]]` in a "Uses" list in place of the members Interface catches that part today. A heading convention catches the lists and none of the identity references. None of Makerspace's 269 lines with a link between Definitions holds links of two relations, but an identity reference shares its label, **Fields** or **Arguments**, with field lines whose links the field relies on.

## The same gain without new syntax

The reviewer pointed out two ways to reach the gain in today's language; this session checked both.

**The part lists repeat visibility.** On all nine Makerspace steps, each Interface's lists of operations and records are exactly its Package's public Definitions that are not Rules: 72 of 72 Interfaces over the steps. In the 5× project they are on 399 of 417; `purchasing` and `quotes` leave out public status Concepts. A Definition that joins or leaves a module's surface changes its own `internal:` line, so it is on the list without any input of membership. Dropping the lists takes off the same items as `part`.

**Typed identities.** Simulated: each record that another Definition names by identity gets an identity Definition (`members/member-id`), the record links it, and each holder links it with a plain link instead of linking the record. The identity Definition holds the record's identity and links its record, which it names. No identity statement of a record changed in any step, so no identity Definition's text would have changed, and the holders stay off the list as with `id`. On the 5× project the identity Definitions add 6 items: 3 written when new modules first named their records (`loans/loan-id` and `loans/tool-id` at c5, `packs/pack-product-id` at c7), and 3 put back on the list when their record changed (`members/member-id` at c4, `notices/notice-id` at c5, `loans/loan-id` at c6). An identity that named its record only in words would save those 3, 351 items, but `vet.unlinked_mention` would report every such identity as a name without a link; the reviewer counted 354 too.

| | Makerspace | 5× project |
|---|---|---|
| Identity Definitions needed at c8 | 9 | 42 |
| Items over the eight steps | 113 | 354 |

**Adopting either once.** Rewriting the c8 spec to the new form changes the text of many Definitions, and the next `status` lists most of the spec:

| | Makerspace c8 | 5× c8 |
|---|---|---|
| The keywords | 76 of 115 | 375 of 419 |
| No new syntax | 73 of 115, and the 9 new identity Definitions | 358 of 419, and 42 new |

Either costs once more than the eight steps save (20 and 117 to 120 items), so a spec gains most by starting in the new form, or by adopting it with a change that touches its Interfaces anyway.

## Further measures

**Package edges.** Of Makerspace's 23 explicit module edges, 21 rest on at least one plain link. The other two rest only on identity references and "Uses" lists: `safety` on `equipment` (an incident names an existing machine) and `training` on `equipment` (a course and a certification name an existing family). The `architecture` Decision allows both. On the 5× project, 35 of 138 edges rest only on such links: 31 on identity references and a "Uses" list, 4 on a "Uses" list alone. Every one of these edges is carried by a "Uses" list, so it stays as a Package link whatever the identity references become; keeping identity references edges is a choice on principle, so that the architecture sees every module whose records another names, not something the data forces. So `safety` and `training` depend on `equipment` only by identity and by their "Uses" lists: what changes in `equipment` reaches them only when what identifies a machine or a family changes.

**Fan-in,** counting plain links and Kinds only, at c8:

| Definition | Fan-in today | Plain links only |
|---|---|---|
| `values` | 60 | 60 |
| `Operation` (Kind) | 39 | 39 |
| `members/member` | 24 | 6 |
| `equipment/machine` | 14 | 4 |
| `bookings/reservation` | 11 | 9 |
| `equipment/family` | 9 | 2 |
| `members/members-interface` | 9 | 5 |

The p90 falls from 7 to 5. At the threshold of `vet.hub` ([DEC-102](../concepts/register.md#dec-102-what-vet-measures): fan-in + 1 above a fifth of the spec), Makerspace has two hubs today, `values` and `members/member`, and one counting plain links only, `values`. In the 5× project `members/member` falls from 90 to at most 6 and the members Interface from 48 to at most 17, while `values` (269), `billing/charge`, `members/membership` (20 each) and `ledger/postings` (23) keep theirs; `values` stays the one hub.

**Every-link reach,** what the author reviews ([guide](../guides/good-spec.md#why-design-matters-here-links-are-the-build-graph)), on the steps:

| Step | Every link today | Plain links only | Every link, without the back link | Plain links only, without the back link |
|---|---|---|---|---|
| c1 | 93 (86 %) | 86 (79 %) | 55 (50 %) | 47 (43 %) |
| c2 | 98 (88 %) | 92 (82 %) | 74 (66 %) | 67 (60 %) |
| c3 | 4 (3 %) | 2 (1 %) | 4 (3 %) | 2 (1 %) |
| c4 | 58 (51 %) | 50 (44 %) | 58 (51 %) | 50 (44 %) |
| c5 | 99 (87 %) | 92 (81 %) | 60 (53 %) | 53 (46 %) |
| c6 | 97 (85 %) | 91 (80 %) | 59 (52 %) | 52 (46 %) |
| c7 | 20 (17 %) | 12 (10 %) | 20 (17 %) | 12 (10 %) |
| c8 | 10 (8 %) | 7 (6 %) | 10 (8 %) | 7 (6 %) |

The back link is the one from `values` to its user `argument-values`, which closed a cycle at Root; the [7 November record](spec-design-measures-20261107.md#the-split) found the same counts without it (55, 74, 60, 59; its percentages are of the split spec). It dominates every-link reach on four steps, and the new relations take 7 or 8 more Definitions out of each.

**Cohesion.** Joined by every link, every area Package is one group, glued by its Interface, which links every operation and record. Joined only by plain links, the Interface stands alone, since it relies on nothing in its own Package, and three Packages fall into two groups of two or more Definitions:

- `equipment`: the catalog (families, machines, rates; 7) and maintenance (windows, scheduling, cancelling; 4);
- `members`: membership (member, tier, standing, their operations; 7) and the users (`members/user`, the registry, `register-user`; 3);
- `billing`: invoicing (charges, invoices, payments, credit; 11) and the account (`billing/account`, `get-account`; 2).

Each reads as a sub-area of its module, not a second area or a defect.

**Kind links inherited.** If every Definition relied on what its Kind relies on, a change to `audit`, `audit-entry` or `order-of-checks` at c1 would add 36 Operations to the list, and a change to `authorization` and `order-of-checks` at c4 another 28: 64 items. The `architecture` Decision places all of these in the protocol module: it runs the order of checks, appends the audit entries and checks roles and ownership once, before any operation's own rules. Those Rules, on the list themselves, cover that code, and re-verifying each Operation would read the same code again.

## Limits

- Two specs by one author, of one domain. The patterns fit how this author writes; another spec needs its own reading.
- File inputs were left out, since the archive holds no realization. In a real build, changed files put Definitions back on the list too, and some dropped items would come back through them; the relations change only what links add.
- The steps are designed change steps, not a real history. No step removed or renamed a referenced Definition; such a change breaks the link, which `check` reports, so the source changes and goes on the list anyway.
- The new inputs, a target's Kind and a part's membership, never fired, so their cost is measured as nothing and their value is not measured at all.
- The 5× project was read by patterns only, and its misses were checked by the target's diff for every dropped pair, not by reading every source.
- The blind check covered 50 lines; it held only 6 identity references and part lists.
- Whether a shorter list helps an agent build or review was not tested.
