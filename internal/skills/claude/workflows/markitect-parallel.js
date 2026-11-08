export const meta = {
  name: "markitect-parallel",
  description: "Markitect's parallel strategy: work per Package, in waves along the spec's dependencies, until markitect status lists nothing or three rounds leave it no shorter",
  whenToUse: "Started by the Markitect skills when their strategy is parallel, with args {mode, focus, all}; not by hand",
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
//   - it takes args {mode, focus, all}, mode being build, cleanup, bughunt,
//     review or audit;
//   - it reads what to do from `markitect status --json`, whose packages give
//     each Package's items and the Packages it depends on;
//   - the work is done by the four roles' agents only: markitect-builder,
//     markitect-reviewer, markitect-architect, markitect-auditor. A plain agent
//     does only what a script cannot: run markitect status and the Git steps;
//   - it stops when status lists nothing a builder can settle, or when three
//     rounds in a row leave the list no shorter, and returns what remains.
// The session that starts it enters a worktree of its own first, for the modes
// that change code, since Claude Code branches each builder's worktree from the
// session's HEAD: a later wave then starts from what the earlier waves merged.
// When it has ended, the session merges that branch back, as the skill says.

const MODES = ["build", "cleanup", "bughunt", "review", "audit"]
const MODE = (args && args.mode) || "build"
if (!MODES.includes(MODE)) throw new Error(`mode is one of ${MODES.join(", ")}, not ${MODE}`)
const FOCUS = args && args.focus ? ` Focus on what the user asked for: ${args.focus}` : ""
const ALL = Boolean(args && args.all)
const WRITES = MODE === "build" || MODE === "cleanup" || MODE === "bughunt"

// A strand is one builder and the reviewer it starts. Claude Code runs at most 20
// subagents of a session at once, so 8 strands stay within that limit with room
// for the session's own agents.
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

const READ_STATUS = `Run \`${STATUS_COMMAND}\` in your working folder and return its fields spec_errors, realization_errors and packages exactly as it prints them, each Package with package, definitions, items, spec_problems and depends_on.`

function readStatus(label, checkWorktree) {
  const worktree = checkWorktree
    ? " Also run `git rev-parse --git-dir --git-common-dir` there and set in_worktree to true when the two lines differ, which means the folder is a linked worktree."
    : ""
  return agent(`${READ_STATUS}${worktree} Change nothing.`, { label, phase: "Status", schema: STATUS, effort: "low" })
}

// merge commits each strand's worktree on its branch, merges the branches into
// the run's branch here, and reads status again. With record set it first
// commits a changed record here, as hunters leave it.
function merge(done, label, record) {
  const strands = done.map((d) => `### Strand ${d.name}\n\n${d.report}`).join("\n\n")
  const commitRecord = record
    ? "0. If `git status --porcelain .markitect/review.sum` shows a change here, commit that file alone: `git commit -m \"markitect: hunt\" -- .markitect/review.sum`.\n"
    : ""
  return agent(
    `You do the Git steps of a Markitect run, in your working folder, whose branch is the run's own. Edit no file yourself.

${commitRecord}1. For each strand below, take its folder and branch from its report. If the report names none, or the folder no longer exists, the strand changed nothing: leave it. Otherwise, if \`git -C <folder> status --porcelain\` shows changes, commit them all there: \`git -C <folder> add -A\`, then \`git -C <folder> commit -m "markitect ${MODE}: <strand>"\`.
2. Merge each strand's branch into the branch here, one at a time: \`git merge --no-edit <branch>\`. If a merge stops on a conflict, run \`git merge --abort\`, leave that branch as it is, and note the strand with the files in conflict. If Git refuses because of changes here, note it and merge no more.
3. For each strand you merged, remove its worktree and branch: \`git worktree remove --force <folder>\`, after \`git worktree unlock <folder>\` if Git says it is locked, then \`git branch -D <branch>\`.
4. ${READ_STATUS}

Return in merged the strands you merged, and in not_merged each strand you did not merge, with why and its branch, so that nothing is lost. Return in requests each change a report says a file outside its own Package needs, with the strand, the file and the change.

${strands || "(No strand this time: do step 0 if it applies, then step 4.)"}`,
    { label, phase: "Merge", schema: STATUS },
  )
}

// strand starts one builder in a worktree of its own.
function strand(task, name) {
  return agent(task, { label: `${MODE} ${name}`, phase: "Build", agentType: "markitect-builder", isolation: "worktree" }).then(
    (report) => ({ name, report: report || "(The strand ended without a report.)" }),
  )
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
const notMerged = []
const requested = []

function keep(st) {
  notMerged.push(...(st.not_merged || []))
  requested.push(...(st.requests || []))
  return st
}

// round runs one round of strands: the Packages in waves, a merge after each
// batch, and Root last and alone, which also makes the changes the others asked
// for in files that are not theirs.
async function round(st, n, everything) {
  const task = (name) => (MODE === "cleanup" ? `Clean up Package ${name}.${FOCUS}` : `Build what markitect status lists in Package ${name}.${FOCUS}`)
  const wanted = new Set(
    st.packages.filter((p) => p.package !== ROOT && (everything ? p.definitions > 0 : workable(p))).map((p) => p.package),
  )
  let last = st
  const plan = waves(st.packages, wanted)
  for (let w = 0; w < plan.length; w++) {
    for (let at = 0; at < plan[w].length; at += STRANDS_AT_ONCE) {
      const batch = plan[w].slice(at, at + STRANDS_AT_ONCE)
      log(`round ${n}, wave ${w + 1}: ${batch.join(", ")}`)
      const done = (await parallel(batch.map((name) => () => strand(task(name), name)))).filter(Boolean)
      last = keep(await merge(done, `merge ${n}.${w + 1}`, false))
    }
  }
  const root = last.packages.find((p) => p.package === ROOT)
  const asks = requested.splice(0)
  if (everything || last.realization_errors > 0 || (root && workable(root)) || asks.length > 0) {
    const list = asks.map((r) => `\n- ${r.file}: ${r.change} (for ${r.strand})`).join("")
    const ask = asks.length > 0 ? ` Make too these changes, which the other Packages' strands asked for:${list}` : ""
    log(`round ${n}, Root`)
    const done = await strand(`${task(ROOT)}${ask}`, ROOT)
    last = keep(await merge([done], `merge ${n}.Root`, false))
  }
  return last
}

function finish(stopped, st, rounds, reports) {
  return {
    mode: MODE,
    stopped,
    rounds,
    items: st ? count(st) : null,
    spec_errors: st ? st.spec_errors : null,
    realization_errors: st ? st.realization_errors : null,
    open: st ? st.packages.filter((p) => p.items > 0) : [],
    not_merged: notMerged,
    reports: reports || [],
  }
}

phase("Status")
let st = await readStatus("status", WRITES)
if (st.spec_errors > 0) return finish("spec-errors", st, 0)
if (WRITES && !st.in_worktree) {
  return finish("not-in-a-worktree: enter a worktree of your own before you start this workflow, so that the run has a branch of its own", st, 0)
}

if (MODE === "audit") {
  phase("Audit")
  const pkgs = st.packages.filter((p) => p.definitions > 0).map((p) => p.package)
  const tasks = pkgs.map((name) => ({ name, task: `Audit the spec of Package ${name}.${FOCUS}` }))
  tasks.push({ name: "across Packages", task: `Audit the spec across Packages: look only for contradictions between Packages.${FOCUS}` })
  const reports = await parallel(
    tasks.map((t) => () => agent(t.task, { label: `audit ${t.name}`, phase: "Audit", agentType: "markitect-auditor" }).then((r) => ({ package: t.name, report: r }))),
  )
  return finish("audited", st, 1, reports.filter(Boolean))
}

if (MODE === "review") {
  phase("Review")
  const pkgs = st.packages.filter((p) => p.items > 0).map((p) => p.package)
  if (pkgs.length === 0) return finish("empty", st, 0)
  const task = (name) => (ALL ? `Verify every Definition in Package ${name}: markitect status --all.${FOCUS}` : `Verify what markitect status lists in Package ${name}.${FOCUS}`)
  const reports = await parallel(
    pkgs.map((name) => () => agent(task(name), { label: `review ${name}`, phase: "Review", agentType: "markitect-reviewer" }).then((r) => ({ package: name, report: r }))),
  )
  st = await readStatus("status after the review", false)
  return finish("reviewed", st, 1, reports.filter(Boolean))
}

let reports = []
if (MODE === "bughunt") {
  phase("Hunt")
  const pkgs = st.packages.filter((p) => p.definitions > 0).map((p) => p.package)
  reports = (
    await parallel(
      pkgs.map((name) => () =>
        agent(`Hunt bugs in Package ${name}: look beyond what markitect status lists.${FOCUS}`, { label: `hunt ${name}`, phase: "Hunt", agentType: "markitect-reviewer" }).then((r) => ({ package: name, report: r })),
      ),
    )
  ).filter(Boolean)
  st = keep(await merge([], "commit the hunt", true))
}

if (MODE === "cleanup") {
  st = await round(st, 1, true)
  return finish(count(st) === 0 ? "empty" : "cleaned", st, 1, reports)
}

let rounds = 0
let stale = 0
let previous = count(st)
for (;;) {
  if (count(st) === 0) return finish("empty", st, rounds, reports)
  if (onlyForTheOwner(st)) return finish("spec-problems", st, rounds, reports)
  rounds++
  st = await round(st, rounds, false)
  const now = count(st)
  stale = now < previous ? 0 : stale + 1
  previous = now
  log(`round ${rounds}: ${now} left on the list`)
  if (stale >= STALE_ROUNDS) return finish("no-progress", st, rounds, reports)
}
