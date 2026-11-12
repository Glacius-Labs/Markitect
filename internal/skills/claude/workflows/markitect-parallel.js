export const meta = {
  name: "markitect-parallel",
  description: "Markitect's parallel strategy: work per Package, in waves along the spec's dependencies, until markitect status lists nothing or three rounds leave it no shorter",
  whenToUse: "Started by the Markitect skills when their strategy is parallel, with args {mode, focus, all, into}; not by hand",
  phases: [
    { title: "Status", detail: "read markitect status --json" },
    { title: "Hunt", detail: "bughunt: one reviewer per Package hunts beyond the list" },
    { title: "Build", detail: "one builder per Package, each in a worktree of its own, committing there" },
    { title: "Merge", detail: "merge the strands' branches by name into the run's branch" },
    { title: "Review", detail: "one reviewer per Package, on the run's branch" },
    { title: "Record", detail: "commit the verdicts and read status again" },
    { title: "Audit", detail: "one auditor per Package and one across Packages" },
  ],
}

// Markitect's parallel driver (DEC-103). Every strategy, this one and a custom
// one alike, keeps one contract:
//   - it takes args {mode, focus, all, into}: mode is build, cleanup, bughunt,
//     review or audit; focus what the user asked to focus on or to leave
//     alone; all, for review, every Definition; into, for the modes that change
//     code, the branch the session merges the run into, which the run must not
//     work on;
//   - it reads what to do from `markitect status --json`, whose packages give
//     each Package's items and the Packages it depends on, as list --deps has
//     them;
//   - the work is done by the four roles' agents only: markitect-builder,
//     markitect-reviewer, markitect-architect, markitect-auditor. A plain agent
//     does only what a script cannot: run markitect status and the Git steps;
//   - it stops when status lists nothing a builder can settle, or when three
//     rounds in a row leave the list no shorter than it has been, and returns
//     what remains.
// The session that starts it enters a worktree of its own first, for the modes
// that change code, since Claude Code branches each builder's worktree from the
// session's HEAD: a later wave then starts from what the earlier waves merged.
// When it has ended, the session merges that branch back, as the skill says.
//
// Claude Code gives the agents of a workflow no tool to start an agent, and lets
// an agent in a worktree run Git only on its own worktree. So a strand is two
// steps of this script, not one agent that starts another: the builder works and
// commits in its own worktree; the merge step, here on the run's branch, merges
// the strands' branches by name; then a reviewer per strand verifies its Package
// here, where the merged work is, and the record step commits the verdicts.

const ARGS = typeof args === "string" ? JSON.parse(args) : args || {}
const MODES = ["build", "cleanup", "bughunt", "review", "audit"]
const MODE = ARGS.mode
if (!MODES.includes(MODE)) throw new Error(`args.mode is one of ${MODES.join(", ")}, not ${JSON.stringify(MODE)}`)
const FOCUS = ARGS.focus ? ` What the user asked for: ${ARGS.focus}` : ""
const ALL = Boolean(ARGS.all)
const WRITES = MODE === "build" || MODE === "cleanup" || MODE === "bughunt"
if (WRITES && !ARGS.into) throw new Error("args.into names the branch the session merges the run into; the run works on a branch of its own")

// Claude Code runs at most 16 agents of a workflow at once; 8 strands leave room
// for the steps beside them and keep the machine's load and the worktrees in
// bounds.
const STRANDS_AT_ONCE = 8
// A loop stops when this many rounds in a row leave the list no shorter (DEC-100).
const STALE_ROUNDS = 3
const ROOT = "Root"
const STATUS_COMMAND = `markitect status${ALL ? " --all" : ""} --json`

const STATUS = {
  type: "object",
  required: ["clean", "spec_errors", "realization_errors", "packages"],
  properties: {
    clean: { type: "boolean" },
    spec_errors: { type: "integer" },
    realization_errors: { type: "integer" },
    packages: {
      type: "array",
      items: {
        type: "object",
        required: ["package", "definitions", "items", "spec_problems", "depends_on"],
        properties: {
          package: { type: "string" },
          definitions: { type: "integer" },
          items: { type: "integer" },
          spec_problems: { type: "integer" },
          depends_on: { type: "array", items: { type: "string" } },
        },
      },
    },
    in_worktree: { type: "boolean" },
    branch: { type: "string" },
  },
}

// BUILT is what a builder of a strand returns: where its work is, whether it
// changed files and committed them there, and the changes it needs in files
// outside its Package.
const BUILT = {
  type: "object",
  required: ["folder", "branch", "changed", "committed", "summary", "requests"],
  properties: {
    folder: { type: "string" },
    branch: { type: "string" },
    changed: { type: "boolean" },
    committed: { type: "boolean" },
    summary: { type: "string" },
    requests: {
      type: "array",
      items: { type: "object", required: ["file", "change"], properties: { file: { type: "string" }, change: { type: "string" } } },
    },
  },
}

const MERGED = {
  type: "object",
  required: ["merged", "unchanged", "not_merged"],
  properties: {
    merged: { type: "array", items: { type: "string" } },
    unchanged: { type: "array", items: { type: "string" } },
    not_merged: {
      type: "array",
      items: { type: "object", required: ["strand", "why"], properties: { strand: { type: "string" }, why: { type: "string" } } },
    },
  },
}

// Lost stops the run when a step it cannot go on without returns nothing, as when
// the user stops an agent or it fails on an API error.
class Lost extends Error {}

const READ_STATUS = `Run \`${STATUS_COMMAND}\` in your working folder and return its fields clean, spec_errors, realization_errors and packages exactly as it prints them, each Package with package, definitions, items, spec_problems and depends_on.`

async function readStatus(label, checkWorktree) {
  const worktree = checkWorktree
    ? " Also run `git rev-parse --git-dir --git-common-dir` there and set in_worktree to true when the two lines differ, which means the folder is a linked worktree, and set branch to what `git branch --show-current` prints."
    : ""
  const st = await agent(`${READ_STATUS}${worktree} Change nothing.`, { label, phase: "Status", schema: STATUS, effort: "low" })
  if (!st) throw new Lost(`${label}: the status step returned nothing`)
  return st
}

const notMerged = []
const requested = []
const unreviewed = []
// worktrees are the strands' worktrees: the session removes those merged, and
// the others hold work that is left.
const worktrees = []

// strands starts one builder per name, each in a worktree of its own, where it
// commits its work. It returns the strands whose work, if any, is on their
// branch, and notes the others: one that returned nothing, whose worktree may
// still hold its work, and one that left its work uncommitted.
async function strands(tasks) {
  const built = await parallel(
    tasks.map((t) => () =>
      agent(t.task, { label: `${MODE} ${t.name}`, phase: "Build", agentType: "markitect-builder", isolation: "worktree", schema: BUILT }).then((b) =>
        b ? { name: t.name, ...b } : null,
      ),
    ),
  )
  const out = []
  for (let i = 0; i < tasks.length; i++) {
    const b = built[i]
    if (!b) {
      notMerged.push({ strand: tasks[i].name, why: "the builder returned nothing; look for its worktree under .claude/worktrees/" })
      continue
    }
    for (const r of b.requests || []) requested.push({ strand: b.name, file: r.file, change: r.change })
    if (!b.changed && !b.committed) {
      // Claude Code removes the worktree of an agent that changed nothing, and
      // its branch with it: there is nothing to merge, and the Package's items
      // are here to verify as they are.
      out.push({ ...b, nothing: true })
      continue
    }
    if (b.changed && !b.committed) {
      notMerged.push({ strand: b.name, why: `the builder left its work uncommitted in ${b.folder}, on branch ${b.branch}` })
      worktrees.push({ strand: b.name, folder: b.folder, branch: b.branch, merged: false })
      continue
    }
    out.push(b)
  }
  return out
}

// merge merges the strands' branches into the run's branch here, by name: the
// branches are the repository's, but Claude Code lets this step run Git on its
// own worktree only. It returns the strands it merged, those whose branch held
// nothing to merge, and why it left the others.
async function merge(all, label) {
  const nothing = all.filter((b) => b.nothing).map((b) => b.name)
  const built = all.filter((b) => !b.nothing)
  if (built.length === 0) return { merged: [], unchanged: nothing, not_merged: [] }
  const list = built.map((b) => `- ${b.name}: ${b.branch}`).join("\n")
  const m = await agent(
    `You do the Git steps of a Markitect run, in your working folder, whose branch is the run's own. Run each git command here as it stands, with no -C and no cd: the branches below are this repository's, and you touch no other folder. Edit no file.

For each strand below, one at a time, with its branch:
1. If the branch does not exist, or \`git log --oneline HEAD..<branch>\` prints nothing, it holds nothing to merge: return the strand in unchanged.
2. Otherwise run \`git merge --no-edit <branch>\`. If the merge stops on a conflict, run \`git merge --abort\` and return the strand in not_merged, with the files in conflict. If Git refuses because of changes here, return this strand and every later one in not_merged, with what Git said, and merge no more.
3. Return in merged each strand you merged.

${list}`,
    { label, phase: "Merge", schema: MERGED, effort: "low" },
  )
  if (!m) {
    for (const b of built) {
      notMerged.push({ strand: b.name, why: `the merge step returned nothing; its work is on branch ${b.branch}` })
      worktrees.push({ strand: b.name, folder: b.folder, branch: b.branch, merged: false })
    }
    throw new Lost(`${label}: the merge step returned nothing`)
  }
  // Only what the step names as merged or unchanged counts as here: a strand it
  // leaves out keeps its branch, so that the session removes no work.
  const merged = new Set(m.merged || [])
  const unchanged = new Set(m.unchanged || [])
  const why = new Map((m.not_merged || []).map((n) => [n.strand, n.why]))
  const out = { merged: [], unchanged: [...nothing], not_merged: [] }
  for (const b of built) {
    // A builder that committed has something to merge: if its branch held
    // nothing, it named the wrong one, and its worktree keeps the work.
    const empty = unchanged.has(b.name) && !b.committed
    const here = merged.has(b.name) || empty
    worktrees.push({ strand: b.name, folder: b.folder, branch: b.branch, merged: here })
    if (merged.has(b.name)) out.merged.push(b.name)
    else if (empty) out.unchanged.push(b.name)
    else {
      const left = unchanged.has(b.name) ? `the builder committed, but branch ${b.branch} held nothing to merge; look in ${b.folder}` : why.get(b.name) || "the merge step did not say what became of it"
      const n = { strand: b.name, why: `${left}; its work is on branch ${b.branch}` }
      out.not_merged.push(n)
      notMerged.push(n)
    }
  }
  return out
}

// each starts one agent of a role per task, side by side, here on the run's
// branch, and returns the reports of those that returned one.
async function each(tasks, phase, agentType) {
  const done = await parallel(
    tasks.map((t) => () =>
      agent(t.task, { label: `${phase.toLowerCase()} ${t.name}`, phase, agentType }).then((r) => (r ? { package: t.name, report: r } : null)),
    ),
  )
  return done.filter(Boolean)
}

// review starts a reviewer per Package here, where the merged work is, to
// verify the Package's items and record the verdicts, and notes the Packages
// whose reviewer returned nothing.
async function review(names, task) {
  const done = await each(names.map((name) => ({ name, task: task(name) })), "Review", "markitect-reviewer")
  const reviewed = new Set(done.map((d) => d.package))
  unreviewed.push(...names.filter((name) => !reviewed.has(name)))
  return done
}

// record commits the verdicts the reviewers recorded here, alone, and reads
// status again.
async function record(label, message) {
  const st = await agent(
    `You do a Git step of a Markitect run, in your working folder, whose branch is the run's own. Run each git command here as it stands, with no -C and no cd. Edit no file.

1. If \`git status --porcelain -- .markitect/review.sum\` shows a change, commit that file alone, new or changed: \`git add -- .markitect/review.sum\`, then \`git commit -m "${message}" -- .markitect/review.sum\`.
2. ${READ_STATUS}`,
    { label, phase: "Record", schema: STATUS, effort: "low" },
  )
  if (!st) throw new Lost(`${label}: the record step returned nothing`)
  return st
}

const verify = (name) => `Verify what markitect status lists in Package ${name}.${FOCUS}`

// batch runs strands side by side: their builders, the merge, a reviewer per
// strand whose work is here, and the record.
async function batch(names, taskOf, label) {
  const built = await strands(names.map((name) => ({ name, task: taskOf(name) })))
  const m = await merge(built, `merge ${label}`)
  await review([...(m.merged || []), ...(m.unchanged || [])].sort(), verify)
  return record(`record ${label}`, `markitect ${MODE}: ${label}`)
}

// waves orders the Packages so that what is depended on comes first: each wave
// holds the Packages whose dependencies among the wanted ones are done. The spec
// compiles only without cycles between Packages; should one remain, the rest
// goes together.
function waves(packages, wanted) {
  const deps = new Map(packages.map((p) => [p.package, p.depends_on]))
  const left = new Set(wanted)
  const done = new Set()
  const out = []
  while (left.size > 0) {
    let wave = [...left].filter((n) => (deps.get(n) || []).every((d) => d === n || !wanted.has(d) || done.has(d)))
    if (wave.length === 0) wave = [...left]
    wave.sort()
    out.push(wave)
    for (const n of wave) {
      left.delete(n)
      done.add(n)
    }
  }
  return out
}

const count = (st) => st.realization_errors + st.packages.reduce((n, p) => n + p.items, 0)
const workable = (p) => p.items > p.spec_problems
const onlyForTheOwner = (st) => st.realization_errors === 0 && !st.packages.some(workable)
// ended says how a run ends that has nothing left for a builder: status is
// clean, or what is left waits for the owner. That is an error the spec gained
// meanwhile, a spec-problem or, when no item is left at all, a line of the
// record that markitect cannot read.
function ended(st) {
  if (st.clean) return "empty"
  if (st.spec_errors > 0) return "spec-errors"
  return count(st) === 0 ? "record-unreadable: markitect status names the lines of the record the owner fixes" : "spec-problems"
}

// round runs one round of strands: the Packages in waves, in batches, and Root
// last and alone. Every Package lies below Root, so none depends on it as list
// --deps counts; Root's strand also settles the errors in the realization and
// makes the changes the others asked for in files that are not theirs. It keeps
// the newest status in st as it goes, so that a run that stops halfway reports
// what is left then.
async function round(from, n, everything) {
  const verb = (name) => (MODE === "cleanup" ? `Clean up Package ${name}.` : `Build what markitect status lists in Package ${name}.`)
  const wanted = new Set(
    from.packages.filter((p) => p.package !== ROOT && (everything ? p.definitions > 0 : workable(p))).map((p) => p.package),
  )
  let last = from
  const plan = waves(from.packages, wanted)
  for (let w = 0; w < plan.length; w++) {
    for (let at = 0; at < plan[w].length; at += STRANDS_AT_ONCE) {
      const names = plan[w].slice(at, at + STRANDS_AT_ONCE)
      log(`round ${n}, wave ${w + 1}: ${names.join(", ")}`)
      last = await batch(names, (name) => `${verb(name)}${FOCUS}`, `${n}.${w + 1}`)
      st = last
    }
  }
  const root = last.packages.find((p) => p.package === ROOT)
  const asks = requested.splice(0)
  if (everything || last.realization_errors > 0 || (root && workable(root)) || asks.length > 0) {
    const list = asks.map((r) => `\n- ${r.file}: ${r.change} (for ${r.strand})`).join("")
    const errors = MODE === "cleanup" ? "" : " Settle too the errors in the realization that markitect status lists."
    const ask = asks.length > 0 ? ` Make too these changes, which the other Packages' strands asked for:${list}` : ""
    log(`round ${n}, Root`)
    last = await batch([ROOT], () => `${verb(ROOT)}${errors}${ask}${FOCUS}`, `${n}.Root`)
    st = last
  }
  return last
}

let st = null
let rounds = 0
let reports = []

function finish(stopped) {
  return {
    mode: MODE,
    stopped,
    rounds,
    items: st ? count(st) : null,
    spec_errors: st ? st.spec_errors : null,
    realization_errors: st ? st.realization_errors : null,
    open: st ? st.packages.filter((p) => p.items > 0) : [],
    not_merged: notMerged,
    unreviewed,
    worktrees,
    reports,
  }
}

async function run() {
  phase("Status")
  st = await readStatus("status", WRITES)
  if (st.spec_errors > 0) return finish("spec-errors")
  if (WRITES && (!st.in_worktree || !st.branch || st.branch === ARGS.into)) {
    return finish(`not-in-a-worktree: the session enters a worktree of its own, on a branch other than ${ARGS.into}, before it starts this workflow`)
  }

  if (MODE === "audit") {
    phase("Audit")
    const pkgs = st.packages.filter((p) => p.definitions > 0).map((p) => p.package)
    const tasks = pkgs.map((name) => ({ name, task: `Audit the spec of Package ${name}, and triage the unlinked mentions markitect vet --strict lists in it.${FOCUS}` }))
    tasks.push({ name: "across Packages", task: `Audit the spec across Packages: look for contradictions between Packages, and walk the coverage list.${FOCUS}` })
    reports = await each(tasks, "Audit", "markitect-auditor")
    rounds = 1
    return finish(reports.length === tasks.length ? "audited" : `audited, ${tasks.length - reports.length} of ${tasks.length} auditors returned nothing`)
  }

  if (MODE === "review") {
    const pkgs = st.packages.filter((p) => (ALL ? p.items > 0 : workable(p))).map((p) => p.package)
    if (pkgs.length === 0) {
      return finish(onlyForTheOwner(st) ? ended(st) : "nothing-to-review: only errors in the realization are left, which /markitect-build settles")
    }
    const task = (name) => (ALL ? `Verify every Definition in Package ${name}: markitect status --all --package ${name}.${FOCUS}` : verify(name))
    reports = await review(pkgs, task)
    rounds = 1
    st = await readStatus("status after the review", false)
    return finish("reviewed")
  }

  if (MODE === "bughunt") {
    phase("Hunt")
    const pkgs = st.packages.filter((p) => p.definitions > 0).map((p) => p.package)
    const tasks = pkgs.map((name) => ({ name, task: `Hunt bugs in Package ${name}: look beyond what markitect status lists.${FOCUS}` }))
    reports = await each(tasks, "Hunt", "markitect-reviewer")
    st = await record("commit the hunt", "markitect: hunt")
  }

  if (MODE === "cleanup") {
    rounds = 1
    st = await round(st, 1, true)
    return finish(st.clean ? "empty" : "cleaned")
  }

  // The list must get shorter than it has been: a list that swings down and up
  // again counts as no shorter.
  let stale = 0
  let best = count(st)
  for (;;) {
    if (onlyForTheOwner(st)) return finish(ended(st))
    rounds++
    st = await round(st, rounds, false)
    const now = count(st)
    if (now < best) {
      best = now
      stale = 0
    } else {
      stale++
    }
    log(`round ${rounds}: ${now} left on the list`)
    if (stale >= STALE_ROUNDS) return finish("no-progress")
  }
}

try {
  return await run()
} catch (e) {
  if (e instanceof Lost) return finish(`lost: ${e.message}`)
  throw e
}
