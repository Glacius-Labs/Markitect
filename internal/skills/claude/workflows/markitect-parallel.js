export const meta = {
  name: "markitect-parallel",
  description: "Markitect's parallel strategy: work per Package, in waves along the spec's dependencies, until markitect status lists nothing or three rounds leave it no shorter",
  whenToUse: "Started by the Markitect skills when their strategy is parallel, with args {mode, focus, all, into}; not by hand",
  phases: [
    { title: "Status", detail: "read markitect status --json" },
    { title: "Hunt", detail: "bughunt: one reviewer per Package hunts beyond the list" },
    { title: "Build", detail: "one builder per Package, each in a worktree of its own, with its own reviewer" },
    { title: "Merge", detail: "commit and merge the strands, then read status again" },
    { title: "Review", detail: "one reviewer per Package" },
    { title: "Audit", detail: "one auditor per Package and one across Packages" },
  ],
}

// Markitect's parallel driver (DEC-103). Every strategy, this one and a custom
// one alike, keeps one contract:
//   - it takes args {mode, focus, all, into}: mode is build, cleanup, bughunt,
//     review or audit; focus what the user asked to focus on; all, for review,
//     every Definition; into, for the modes that change code, the branch the
//     session merges the run into, which the run must not work on;
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

const ARGS = typeof args === "string" ? JSON.parse(args) : args || {}
const MODES = ["build", "cleanup", "bughunt", "review", "audit"]
const MODE = ARGS.mode
if (!MODES.includes(MODE)) throw new Error(`args.mode is one of ${MODES.join(", ")}, not ${JSON.stringify(MODE)}`)
const FOCUS = ARGS.focus ? ` Focus on what the user asked for: ${ARGS.focus}` : ""
const ALL = Boolean(ARGS.all)
const WRITES = MODE === "build" || MODE === "cleanup" || MODE === "bughunt"
if (WRITES && !ARGS.into) throw new Error("args.into names the branch the session merges the run into; the run works on a branch of its own")

// A strand is one builder and the reviewer it starts. Claude Code runs at most 16
// workflow agents and 20 subagents of a session at once, so 8 strands, 16
// agents, stay within both.
const STRANDS_AT_ONCE = 8
// A loop stops when this many rounds in a row leave the list no shorter (DEC-100).
const STALE_ROUNDS = 3
const ROOT = "Root"
const STATUS_COMMAND = `markitect status${ALL ? " --all" : ""} --json`

const STATUS = {
  type: "object",
  required: ["spec_errors", "realization_errors", "packages"],
  properties: {
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
    merged: { type: "array", items: { type: "string" } },
    not_merged: {
      type: "array",
      items: { type: "object", required: ["strand", "why"], properties: { strand: { type: "string" }, why: { type: "string" } } },
    },
    requests: {
      type: "array",
      items: {
        type: "object",
        required: ["strand", "file", "change"],
        properties: { strand: { type: "string" }, file: { type: "string" }, change: { type: "string" } },
      },
    },
  },
}

// Lost stops the run when a step it cannot go on without returns nothing, as when
// the user stops an agent or it fails on an API error.
class Lost extends Error {}

const READ_STATUS = `Run \`${STATUS_COMMAND}\` in your working folder and return its fields spec_errors, realization_errors and packages exactly as it prints them, each Package with package, definitions, items, spec_problems and depends_on.`

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

// merge commits each strand's worktree on its branch, merges the branches into
// the run's branch here, and reads status again. With record set it first
// commits a changed record here, as hunters leave it.
async function merge(done, label, record) {
  const strands = done.map((d) => `### Strand ${d.name}\n\n${d.report}`).join("\n\n")
  const commitRecord = record
    ? "0. If `git status --porcelain .markitect/review.sum` shows a change here, commit that file alone: `git commit -m \"markitect: hunt\" -- .markitect/review.sum`.\n"
    : ""
  const st = await agent(
    `You do the Git steps of a Markitect run, in your working folder, whose branch is the run's own. Edit no file yourself.

${commitRecord}1. For each strand below, take its folder and branch from its report. If the report names none, or the folder no longer exists, the strand changed nothing: leave it. Otherwise, if \`git -C <folder> status --porcelain\` shows changes, commit them all there: \`git -C <folder> add -A\`, then \`git -C <folder> commit -m "markitect ${MODE}: <strand>"\`.
2. Merge each strand's branch into the branch here, one at a time: \`git merge --no-edit <branch>\`. If a merge stops on a conflict, run \`git merge --abort\`, leave that branch as it is, and note the strand with the files in conflict. If Git refuses because of changes here, note it and merge no more.
3. For each strand you merged, remove its worktree and branch: \`git worktree remove --force <folder>\`, after \`git worktree unlock <folder>\` if Git says it is locked, then \`git branch -D <branch>\`.
4. ${READ_STATUS}

Return in merged the strands you merged, and in not_merged each strand you did not merge, with why and its branch, so that nothing is lost. Return in requests each change a report says a file outside its own Package needs, with the strand, the file and the change.

${strands || "(No strand this time: do step 0 if it applies, then step 4.)"}`,
    { label, phase: "Merge", schema: STATUS },
  )
  if (!st) {
    for (const d of done) notMerged.push({ strand: d.name, why: `the merge step returned nothing; the strand's worktree and branch are as its report says: ${d.report.slice(0, 300)}` })
    throw new Lost(`${label}: the merge step returned nothing`)
  }
  notMerged.push(...(st.not_merged || []))
  requested.push(...(st.requests || []))
  return st
}

// strands starts one builder per name, each in a worktree of its own, and notes
// a strand that returned nothing, whose worktree may still hold its work.
async function strands(tasks) {
  const done = await parallel(
    tasks.map((t) => () =>
      agent(t.task, { label: `${MODE} ${t.name}`, phase: "Build", agentType: "markitect-builder", isolation: "worktree" }).then((report) =>
        report ? { name: t.name, report } : null,
      ),
    ),
  )
  for (let i = 0; i < tasks.length; i++) {
    if (!done[i]) notMerged.push({ strand: tasks[i].name, why: "the builder returned nothing; look for its worktree under .claude/worktrees/" })
  }
  return done.filter(Boolean)
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

// round runs one round of strands: the Packages in waves, a merge after each
// batch, and Root last and alone. Every Package lies below Root, so none
// depends on it as list --deps counts; Root's strand also settles the errors in
// the realization and makes the changes the others asked for in files that are
// not theirs. It keeps the newest status in st as it goes, so that a run that
// stops halfway reports what is left then.
async function round(from, n, everything) {
  const verb = (name) => (MODE === "cleanup" ? `Clean up Package ${name}.` : `Build what markitect status lists in Package ${name}.`)
  const wanted = new Set(
    from.packages.filter((p) => p.package !== ROOT && (everything ? p.definitions > 0 : workable(p))).map((p) => p.package),
  )
  let last = from
  const plan = waves(from.packages, wanted)
  for (let w = 0; w < plan.length; w++) {
    for (let at = 0; at < plan[w].length; at += STRANDS_AT_ONCE) {
      const batch = plan[w].slice(at, at + STRANDS_AT_ONCE)
      log(`round ${n}, wave ${w + 1}: ${batch.join(", ")}`)
      const done = await strands(batch.map((name) => ({ name, task: `${verb(name)}${FOCUS}` })))
      last = await merge(done, `merge ${n}.${w + 1}`, false)
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
    const done = await strands([{ name: ROOT, task: `${verb(ROOT)}${errors}${ask}${FOCUS}` }])
    last = await merge(done, `merge ${n}.Root`, false)
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
    tasks.push({ name: "across Packages", task: `Audit the spec across Packages: look only for contradictions between Packages.${FOCUS}` })
    reports = (
      await parallel(
        tasks.map((t) => () => agent(t.task, { label: `audit ${t.name}`, phase: "Audit", agentType: "markitect-auditor" }).then((r) => (r ? { package: t.name, report: r } : null))),
      )
    ).filter(Boolean)
    rounds = 1
    return finish(reports.length === tasks.length ? "audited" : `audited, ${tasks.length - reports.length} of ${tasks.length} auditors returned nothing`)
  }

  if (MODE === "review") {
    phase("Review")
    const pkgs = st.packages.filter((p) => (ALL ? p.items > 0 : workable(p))).map((p) => p.package)
    if (pkgs.length === 0) {
      if (count(st) === 0) return finish("empty")
      return finish(onlyForTheOwner(st) ? "spec-problems" : "nothing-to-review: only errors in the realization are left, which /markitect-build settles")
    }
    const task = (name) => (ALL ? `Verify every Definition in Package ${name}: markitect status --all.${FOCUS}` : `Verify what markitect status lists in Package ${name}.${FOCUS}`)
    reports = (
      await parallel(
        pkgs.map((name) => () => agent(task(name), { label: `review ${name}`, phase: "Review", agentType: "markitect-reviewer" }).then((r) => (r ? { package: name, report: r } : null))),
      )
    ).filter(Boolean)
    rounds = 1
    st = await readStatus("status after the review", false)
    return finish("reviewed")
  }

  if (MODE === "bughunt") {
    phase("Hunt")
    const pkgs = st.packages.filter((p) => p.definitions > 0).map((p) => p.package)
    reports = (
      await parallel(
        pkgs.map((name) => () =>
          agent(`Hunt bugs in Package ${name}: look beyond what markitect status lists.${FOCUS}`, { label: `hunt ${name}`, phase: "Hunt", agentType: "markitect-reviewer" }).then((r) =>
            r ? { package: name, report: r } : null,
          ),
        ),
      )
    ).filter(Boolean)
    st = await merge([], "commit the hunt", true)
  }

  if (MODE === "cleanup") {
    rounds = 1
    st = await round(st, 1, true)
    return finish(count(st) === 0 ? "empty" : "cleaned")
  }

  // The list must get shorter than it has been: a list that swings down and up
  // again counts as no shorter.
  let stale = 0
  let best = count(st)
  for (;;) {
    if (count(st) === 0) return finish("empty")
    if (onlyForTheOwner(st)) return finish("spec-problems")
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
