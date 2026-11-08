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

// world is a project of Packages that the stub agents change: a builder settles
// its Package's items when its strand is merged, a reviewer records, a merge
// reads status. Options: stuck (builders settle nothing), swing (each round puts
// items back), hunt, specErrors, inWorktree, branch, lose (the nth merge returns
// nothing).
function world(spec) {
  const pk = structuredClone(spec.packages)
  const pending = new Set()
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
        if (!spec.stuck) pending.add(name)
        const asks = name === "a" ? "\nIt needs a change in go.mod: add module x." : ""
        return Promise.resolve(`Done. Folder /wt/${name}, branch wt-${name}.${asks}`)
      }
      if (opts.agentType === "markitect-reviewer" || opts.agentType === "markitect-auditor") {
        const m = prompt.match(/Package (\S+?)[.:]/)
        if (spec.hunt && m && prompt.startsWith("Hunt")) pk[m[1]].items += spec.hunt
        if (opts.agentType === "markitect-reviewer" && m && prompt.startsWith("Verify")) pk[m[1]].items = pk[m[1]].sp || 0
        return Promise.resolve(`report for ${m ? m[1] : "all"}`)
      }
      if (prompt.includes("Git steps")) {
        merges++
        if (spec.lose === merges) return Promise.resolve(null)
        const merged = []
        for (const m of prompt.matchAll(/### Strand (\S+)/g)) {
          const name = m[1]
          if (pending.has(name)) {
            pk[name].items = pk[name].sp || 0
            if (name === "Root") realization = 0
            pending.delete(name)
          }
          merged.push(name)
        }
        if (spec.swing && merged.includes("Root")) {
          // Each round settles the list, then the merge puts more back than the
          // round before took away, every other round.
          swing++
          pk.a.items = swing % 2 === 1 ? 5 : 3
        }
        const st = status()
        st.merged = merged
        st.requests = merged.includes("a") ? [{ strand: "a", file: "go.mod", change: "add module x" }] : []
        return Promise.resolve(st)
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
// errors in the realization; builders in worktrees; a Package whose items all
// wait for the owner gets no strand.
{
  const { result, calls } = await run(build, world(base()))
  assert.deepStrictEqual(logs(calls).slice(0, 4), ["round 1, wave 1: a, d", "round 1, wave 2: b", "round 1, wave 3: c", "round 1, Root"], logs(calls).join("\n"))
  const root = builders(calls).find((c) => c.prompt.includes("Package Root"))
  assert(root.prompt.includes("go.mod: add module x (for a)") && root.prompt.includes("errors in the realization"), root.prompt)
  assert(!calls.some((c) => c.prompt.includes("Package e")))
  assert(builders(calls).every((c) => c.opts.isolation === "worktree"))
  assert.strictEqual(result.stopped, "spec-problems")
  assert.strictEqual(result.items, 2)
}

// Nothing gets done: three rounds without a shorter list, then stop.
{
  const s = base()
  s.stuck = true
  const { result } = await run(build, world(s))
  assert.strictEqual(result.stopped, "no-progress")
  assert.strictEqual(result.rounds, 3)
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
}
// A merge lost later reports the status of the last merge that returned, not of
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
  const reviewers = calls.filter((c) => c.opts.agentType === "markitect-reviewer")
  assert.strictEqual(reviewers.length, 5)
  assert(reviewers.every((c) => !c.opts.isolation))
  assert.strictEqual(result.stopped, "reviewed")
  const all = await run({ mode: "review", all: true }, world(base()))
  assert.strictEqual(all.calls.filter((c) => c.opts.agentType === "markitect-reviewer").length, 6)
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
  assert(calls.some((c) => c.prompt.includes("Git steps") && c.prompt.includes("markitect: hunt")))
  assert.strictEqual(result.stopped, "spec-problems")
}

// Cleanup: every Package with Definitions once, Root last, one round.
{
  const { result, calls } = await run({ mode: "cleanup", into: "main" }, world(base()))
  const strands = builders(calls).map((c) => c.prompt)
  assert.strictEqual(strands.length, 6)
  assert(strands.every((p) => p.startsWith("Clean up Package")))
  assert(strands[strands.length - 1].startsWith("Clean up Package Root"))
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
