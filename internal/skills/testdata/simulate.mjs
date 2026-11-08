// Runs the parallel workflow against stub agents and checks its plan, its waves,
// its merges and its stop rule, without any real agent. Usage:
//   node simulate.mjs <path to markitect-parallel.js>
// It exits non-zero on the first failed check. The Go test in this package runs
// it when node is on the PATH.
import fs from "fs"
import assert from "assert"

const src = fs.readFileSync(process.argv[2], "utf8")
assert(src.startsWith("export const meta = {"), "meta comes first")
assert(!/Date\.now|Math\.random|new Date\(\)|import\(/.test(src), "no clock, randomness or import, which the runtime refuses")
const body = `return (async () => {\n${src.replace("export const meta", "const meta")}\n})()`

// world is a project of Packages that the stub agents change, as Claude Code lets
// them: a builder works and commits in its own worktree and returns its branch;
// the merge step brings a committed branch to the run's branch; a reviewer there
// settles its Package's items when that Package's work has come; the record step
// commits and reads status. Options: stuck (builders change nothing), swing (each
// Root record puts items back), hunt, specErrors, inWorktree, branch, noBranch,
// silent (that builder returns nothing), uncommitted (that builder leaves its
// work uncommitted), idle (that builder changes nothing), conflict (that
// strand's merge conflicts), forget (the merge step leaves that strand out of
// its answer), wrongBranch (that builder commits but names a branch that does
// not exist), lose (the nth merge returns nothing), loseStatus.
function world(spec) {
  const pk = structuredClone(spec.packages)
  const committed = new Set()
  const landed = new Set()
  let realization = spec.realization || 0
  let merges = 0
  let swing = 0
  const status = () => ({
    spec_errors: spec.specErrors || 0,
    realization_errors: realization,
    in_worktree: spec.inWorktree !== false,
    branch: spec.noBranch ? undefined : spec.branch || "worktree-markitect-build-1",
    packages: Object.entries(pk).map(([name, p]) => ({ package: name, definitions: p.defs, items: p.items, spec_problems: p.sp || 0, depends_on: p.deps || [] })),
  })
  return {
    respond(prompt, opts) {
      if (opts.agentType === "markitect-builder") {
        const name = prompt.match(/Package (\S+?)[.,]/)[1]
        if (spec.silent === name) return Promise.resolve(null)
        const changed = !spec.stuck && spec.idle !== name
        const ok = changed && spec.uncommitted !== name
        if (ok) committed.add(name)
        return Promise.resolve({
          folder: `/wt/${name}`,
          branch: spec.wrongBranch === name ? `nope-${name}` : `wt-${name}`,
          changed,
          committed: ok,
          summary: `built ${name}`,
          requests: name === "a" ? [{ file: "go.mod", change: "add module x" }] : [],
        })
      }
      if (opts.agentType === "markitect-reviewer" || opts.agentType === "markitect-auditor") {
        const m = prompt.match(/Package (\S+?)[.:]/)
        if (spec.hunt && m && prompt.startsWith("Hunt")) pk[m[1]].items += spec.hunt
        if (opts.agentType === "markitect-reviewer" && m && prompt.startsWith("Verify") && landed.has(m[1])) pk[m[1]].items = pk[m[1]].sp || 0
        return Promise.resolve(`report for ${m ? m[1] : "all"}`)
      }
      if (prompt.includes("You do the Git steps")) {
        merges++
        if (spec.lose === merges) return Promise.resolve(null)
        const out = { merged: [], unchanged: [], not_merged: [] }
        for (const m of prompt.matchAll(/^- (\S+): (\S+)$/gm)) {
          const name = m[1]
          if (spec.wrongBranch === name) {
            out.unchanged.push(name) // no such branch
            continue
          }
          assert.strictEqual(m[2], `wt-${name}`, "the merge names the strand's branch")
          if (spec.forget === name) {
            // left out of every list
          } else if (!committed.has(name)) {
            out.unchanged.push(name)
          } else if (spec.conflict === name) {
            out.not_merged.push({ strand: name, why: "conflict in x.go" })
          } else {
            committed.delete(name)
            landed.add(name)
            if (name === "Root") realization = 0
            out.merged.push(name)
          }
        }
        return Promise.resolve(out)
      }
      if (prompt.includes("You do a Git step")) {
        if (spec.swing && prompt.includes(".Root\"")) {
          // Each Root record puts more back than the round before took away,
          // every other round.
          swing++
          pk.a.items = swing % 2 === 1 ? 5 : 3
        }
        return Promise.resolve(status())
      }
      if (prompt.startsWith("Run `markitect status")) return Promise.resolve(spec.loseStatus ? null : status())
      throw new Error("unexpected agent: " + prompt.slice(0, 80))
    },
  }
}

async function run(args, w) {
  const calls = []
  const agent = (prompt, opts = {}) => {
    calls.push({ prompt, opts })
    return w.respond(prompt, opts)
  }
  const parallel = async (thunks) => Promise.all(thunks.map((t) => t().catch(() => null)))
  const log = (m) => calls.push({ log: m, prompt: "", opts: {} })
  const phase = () => {}
  const fn = new Function("agent", "parallel", "log", "phase", "args", body)
  const result = await fn(agent, parallel, log, phase, args)
  return { result, calls }
}

const builders = (calls) => calls.filter((c) => c.opts.agentType === "markitect-builder")
const reviewers = (calls) => calls.filter((c) => c.opts.agentType === "markitect-reviewer")
const plain = (calls) => calls.filter((c) => c.prompt && !c.opts.agentType)
const logs = (calls) => calls.filter((c) => c.log !== undefined).map((c) => c.log)
const base = () => ({
  packages: {
    Root: { defs: 2, items: 1, deps: ["b"] },
    a: { defs: 5, items: 3, deps: [] },
    b: { defs: 4, items: 2, deps: ["a"] },
    c: { defs: 3, items: 4, deps: ["b"] },
    d: { defs: 2, items: 1, deps: [] },
    e: { defs: 1, items: 2, sp: 2, deps: [] },
  },
  realization: 1,
})
const build = { mode: "build", into: "main" }

// Build: waves along the dependencies, Root last with the requests and the
// errors in the realization; builders in worktrees, returning their branch;
// after each merge a reviewer per strand on the run's branch; a Package whose
// items all wait for the owner gets no strand.
{
  const { result, calls } = await run(build, world(base()))
  assert.deepStrictEqual(logs(calls).slice(0, 4), ["round 1, wave 1: a, d", "round 1, wave 2: b", "round 1, wave 3: c", "round 1, Root"], logs(calls).join("\n"))
  const root = builders(calls).find((c) => c.prompt.includes("Package Root"))
  assert(root.prompt.includes("go.mod: add module x (for a)") && root.prompt.includes("errors in the realization"), root.prompt)
  assert(!calls.some((c) => c.prompt.includes("Package e")))
  assert(builders(calls).every((c) => c.opts.isolation === "worktree" && c.opts.schema && c.opts.schema.required.includes("branch")))
  // Each strand: builder, then the merge, then its reviewer, then the record.
  const order = calls.filter((c) => c.prompt).map((c) => (c.opts.agentType || (c.prompt.includes("Git steps") ? "merge" : c.prompt.includes("Git step") ? "record" : "status")).replace("markitect-", ""))
  assert.deepStrictEqual(order.slice(0, 7), ["status", "builder", "builder", "merge", "reviewer", "reviewer", "record"], order.join(" "))
  assert.deepStrictEqual(reviewers(calls).slice(0, 2).map((c) => c.prompt), ["Verify what markitect status lists in Package a.", "Verify what markitect status lists in Package d."])
  assert(reviewers(calls).every((c) => !c.opts.isolation))
  // No step is asked to reach into another folder with Git.
  assert(!plain(calls).some((c) => /git -C \S|\bcd \S|\/wt\//.test(c.prompt)), "a Git step names another folder")
  // The record is new in a project that has none yet: the record step adds it
  // before it commits it alone, which a bare `git commit -- FILE` refuses.
  const records = plain(calls).filter((c) => c.prompt.includes("You do a Git step"))
  assert(records.length >= 4 && records.every((c) => c.prompt.includes("`git add -- .markitect/review.sum`, then `git commit")), records[0].prompt)
  assert.strictEqual(result.stopped, "spec-problems")
  assert.strictEqual(result.items, 2)
  assert(result.worktrees.length >= 5 && result.worktrees.every((w) => w.merged && w.branch === `wt-${w.strand}`), JSON.stringify(result.worktrees))
}

// Nothing gets done: builders change nothing, the reviewers still verify, and
// three rounds without a shorter list stop the run.
{
  const s = base()
  s.stuck = true
  const { result, calls } = await run(build, world(s))
  assert.strictEqual(result.stopped, "no-progress")
  assert.strictEqual(result.rounds, 3)
  assert(reviewers(calls).length > 0)
  assert.deepStrictEqual(result.not_merged, [])
}

// A list that swings down and up again gets no shorter than it has been, so the
// run stops after three rounds instead of running on.
{
  const s = base()
  s.swing = true
  const { result } = await run(build, world(s))
  assert.strictEqual(result.stopped, "no-progress", JSON.stringify(result))
  // 14, 7, 5 left; then 7, 5, 7: three rounds no shorter than 5.
  assert.strictEqual(result.rounds, 5, `ran ${result.rounds} rounds`)
}

// A strand whose merge conflicts is left on its branch and gets no reviewer; the
// others go on.
{
  const s = base()
  s.conflict = "d"
  const { result, calls } = await run(build, world(s))
  assert(result.not_merged.some((n) => n.strand === "d" && n.why.includes("conflict") && n.why.includes("wt-d")), JSON.stringify(result.not_merged))
  assert(!reviewers(calls).some((c) => c.prompt.includes("Package d.")))
  assert(result.worktrees.some((w) => w.strand === "d" && !w.merged))
  assert(reviewers(calls).some((c) => c.prompt.includes("Package a.")))
}

// A strand the merge step leaves out of its answer counts as not merged: it
// keeps its branch, and gets no reviewer.
{
  const s = base()
  s.forget = "d"
  const { result, calls } = await run(build, world(s))
  assert(result.not_merged.some((n) => n.strand === "d" && n.why.includes("did not say") && n.why.includes("wt-d")), JSON.stringify(result.not_merged))
  assert(result.worktrees.some((w) => w.strand === "d" && !w.merged))
  assert(!reviewers(calls).some((c) => c.prompt.includes("Package d.")))
}

// A builder that committed but named a branch with nothing to merge keeps its
// worktree: its work is there, not here.
{
  const s = base()
  s.wrongBranch = "d"
  const { result, calls } = await run(build, world(s))
  assert(result.not_merged.some((n) => n.strand === "d" && n.why.includes("held nothing")), JSON.stringify(result.not_merged))
  assert(result.worktrees.some((w) => w.strand === "d" && !w.merged))
  assert(!reviewers(calls).some((c) => c.prompt.includes("Package d.")))
}

// A builder that changed nothing has no branch left to merge: the merge step is
// not asked about it, and its Package is verified as it is.
{
  const s = base()
  s.idle = "d"
  const { result, calls } = await run(build, world(s))
  assert(!calls.some((c) => c.prompt.includes("- d: wt-d")))
  assert(reviewers(calls).some((c) => c.prompt.includes("Package d.")))
  assert(!result.not_merged.some((n) => n.strand === "d"))
  assert(!result.worktrees.some((w) => w.strand === "d"))
}

// A builder that leaves its work uncommitted is reported and neither merged nor
// reviewed.
{
  const s = base()
  s.uncommitted = "d"
  const { result, calls } = await run(build, world(s))
  assert(result.not_merged.some((n) => n.strand === "d" && n.why.includes("uncommitted")), JSON.stringify(result.not_merged))
  assert(!calls.some((c) => c.prompt.includes("- d: wt-d")))
  assert(!reviewers(calls).some((c) => c.prompt.includes("Package d.")))
}

// A writing mode refuses to start outside a worktree, on the branch the run
// would come back to, and when the branch is not known.
for (const s of [{ inWorktree: false }, { branch: "main" }, { noBranch: true }]) {
  const { result, calls } = await run(build, world({ ...base(), ...s }))
  assert(result.stopped.startsWith("not-in-a-worktree"), result.stopped)
  assert.strictEqual(calls.length, 1)
}

// The mode is required, and a writing mode needs into; args may come as JSON text.
for (const bad of [{}, { mode: "deploy" }, { mode: "cleanup" }, undefined]) {
  await assert.rejects(run(bad, world(base())), /args\./)
}
{
  const { result } = await run(JSON.stringify({ mode: "review" }), world(base()))
  assert.strictEqual(result.stopped, "reviewed")
}

// A merge that returns nothing stops the run with a report, and names the
// strands whose work is left on their branches.
{
  const s = base()
  s.lose = 1
  const { result } = await run(build, world(s))
  assert(result.stopped.startsWith("lost:"), result.stopped)
  assert.deepStrictEqual(result.not_merged.map((n) => n.strand), ["a", "d"])
  assert(result.not_merged.every((n) => n.why.includes("branch wt-")))
}
// A merge lost later reports the status of the last record that returned, not of
// the start.
{
  const s = base()
  s.lose = 2
  const { result } = await run(build, world(s))
  assert(result.stopped.startsWith("lost:"), result.stopped)
  assert.strictEqual(result.open.find((p) => p.package === "a"), undefined, JSON.stringify(result.open))
  assert.deepStrictEqual(result.not_merged.map((n) => n.strand), ["b"])
}
// A status step that returns nothing ends the run before any work; a builder that
// returns nothing is reported, and the others go on.
{
  const s = base()
  s.loseStatus = true
  const { result, calls } = await run(build, world(s))
  assert(result.stopped.startsWith("lost:"), result.stopped)
  assert.strictEqual(calls.length, 1)
  const t = base()
  t.silent = "d"
  const quiet = await run(build, world(t))
  assert(quiet.result.not_merged.some((n) => n.strand === "d" && n.why.includes("returned nothing")))
  assert(builders(quiet.calls).some((c) => c.prompt.includes("Package c")))
}
// A review with only errors in the realization left says so.
{
  const { result } = await run({ mode: "review" }, world({ packages: { Root: { defs: 1, items: 0, deps: [] }, a: { defs: 1, items: 0, deps: [] } }, realization: 2 }))
  assert(result.stopped.startsWith("nothing-to-review"), result.stopped)
}

// Review: one reviewer per Package with items it can settle, in the session's
// folder; with all, every Package.
{
  const { result, calls } = await run({ mode: "review" }, world(base()))
  assert.strictEqual(reviewers(calls).length, 5)
  assert(reviewers(calls).every((c) => !c.opts.isolation))
  assert.strictEqual(result.stopped, "reviewed")
  const all = await run({ mode: "review", all: true }, world(base()))
  assert.strictEqual(reviewers(all.calls).length, 6)
  assert(all.calls[0].prompt.includes("markitect status --all --json"))
}

// Audit: one auditor per Package with Definitions and one across them.
{
  const { result, calls } = await run({ mode: "audit", focus: "fees" }, world(base()))
  const auditors = calls.filter((c) => c.opts.agentType === "markitect-auditor")
  assert.strictEqual(auditors.length, 7)
  assert(auditors.some((c) => c.prompt.startsWith("Audit the spec across Packages")))
  assert(auditors.every((c) => c.prompt.includes("Focus on what the user asked for: fees")))
  assert.strictEqual(result.stopped, "audited")
}

// Bughunt: hunters per Package, the record committed, then build rounds.
{
  const s = base()
  s.hunt = 1
  const { result, calls } = await run({ mode: "bughunt", into: "main" }, world(s))
  assert.strictEqual(calls.filter((c) => c.prompt.startsWith("Hunt bugs in Package")).length, 6)
  assert(calls.some((c) => c.prompt.includes("Git step") && c.prompt.includes("markitect: hunt")))
  assert.strictEqual(result.stopped, "spec-problems")
}

// Cleanup: every Package with Definitions once, Root last, one round.
{
  const { result, calls } = await run({ mode: "cleanup", into: "main" }, world(base()))
  const strands = builders(calls).map((c) => c.prompt)
  assert.strictEqual(strands.length, 6)
  assert(strands.every((p) => p.startsWith("Clean up Package")))
  assert(strands[strands.length - 1].startsWith("Clean up Package Root"))
  assert.strictEqual(reviewers(calls).length, 6)
  assert.strictEqual(result.rounds, 1)
}

// A wide wave runs in batches of eight strands.
{
  const s = { packages: { Root: { defs: 1, items: 0, deps: [] } } }
  for (let i = 0; i < 19; i++) s.packages["p" + String(i).padStart(2, "0")] = { defs: 1, items: 1, deps: [] }
  const { result, calls } = await run(build, world(s))
  const sizes = logs(calls).filter((l) => l.includes("wave")).map((l) => l.split(": ")[1].split(", ").length)
  assert.deepStrictEqual(sizes, [8, 8, 3])
  assert.strictEqual(result.stopped, "empty")
}

// A spec with errors stops before any work.
{
  const s = base()
  s.specErrors = 2
  const { result, calls } = await run(build, world(s))
  assert.strictEqual(result.stopped, "spec-errors")
  assert.strictEqual(calls.length, 1)
}
console.log("ok: the parallel workflow behaves as planned against stub agents")
