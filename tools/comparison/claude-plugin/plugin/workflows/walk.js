export const meta = {
  name: 'walk',
  description: 'Markitect Apply (arm P): Assessment, Implement and Verify through the compiled Manager tree',
  whenToUse: 'Invoked by /markitect:apply after mkp.py prepare passed. args: the prepare record {runId, base, mkp, tree} plus {maxCorrections, maxReturns, maxQuestions, maxAgents}.',
  phases: [{ title: 'Assessment' }, { title: 'Implement' }, { title: 'Verify' }],
}

// The script owns order and limits; Managers own meaning. Every agent() call
// is a fresh context in its own git worktree on its own branch. Nothing here
// reads files: the prepare record supplies the tree, and branches carry the
// work. Publication does not trust this script's result: the host records
// every judgment from the agents' own transcripts (mkp.py record).
const A = typeof args === 'string' ? JSON.parse(args) : args
if (!A || !A.runId || !A.base || !A.mkp || !A.tree) throw new Error('apply needs the prepare record {runId, base, mkp, tree}')
const RUN = A.runId
const BASE = A.base
const MAX_CORRECTIONS = A.maxCorrections ?? 2
const MAX_RETURNS = A.maxReturns ?? 2
const MAX_QUESTIONS = A.maxQuestions ?? 3
const MAX_AGENTS = A.maxAgents ?? 150
const MKP = `python "${A.mkp}"`
const SHA = '^[0-9a-f]{40}$'

let agents = 0
let exhausted = false
let branches = 0
const log_ = []
function call(prompt, opts) {
  if (agents >= MAX_AGENTS) {
    exhausted = true
    throw new Error(`agent allowance ${MAX_AGENTS} used up`)
  }
  agents++
  return agent(prompt, { isolation: 'worktree', ...opts })
}
// Every ending names its stop; an exhausted allowance is never hidden behind
// a failed agent (concept §3.6).
function done(result) {
  return { ...result, agents, log: log_, ...(exhausted ? { allowance: `agent allowance ${MAX_AGENTS} used up` } : {}) }
}
const slug = p => (p === '.' ? '_root' : p.replace(/\//g, '--'))
const name = p => (p === '.' ? 'Root' : p)
// A git branch can be checked out in only one worktree, and Claude Code keeps
// every worktree that committed: each agent therefore gets a branch of its own.
const branch = (round, n, kind) => `mk/${RUN}/i${round}/${slug(n.pkg)}/${kind}${++branches}`
;(function walk(n, parent) { n.parent = parent; n.children.forEach(c => walk(c, n)) })(A.tree, null)
// The Package that owns a Definition named "package/ID".
const owner = f => (f.definition && f.definition.includes('/') ? f.definition.slice(0, f.definition.lastIndexOf('/')) : null)
const within = (pkg, n) => pkg === n.pkg || (n.pkg !== '.' && pkg !== null && pkg.startsWith(n.pkg + '/'))

const FINDING = { type: 'object', required: ['text'], properties: {
  definition: { type: 'string', description: 'Package/ID of the Definition concerned, when there is one' },
  text: { type: 'string' } } }
const JUDGMENT = { type: 'object', required: ['decision', 'findings', 'summary'], properties: {
  decision: { enum: ['approve', 'reject', 'model_blocker'] },
  findings: { type: 'array', items: FINDING },
  summary: { type: 'string', description: 'What you examined and why you decided so; optional observations go here' } } }
const HANDOFF = { type: 'object', required: ['status', 'account'], properties: {
  status: { enum: ['submitted', 'question', 'model_blocker'] },
  commit: { type: 'string', pattern: SHA, description: 'Full 40-character SHA of your committed handoff; required for submitted' },
  questions: { type: 'array', items: { type: 'string' } },
  findings: { type: 'array', items: FINDING },
  briefs: { type: 'array', description: 'One delegation message per direct child', items: { type: 'object', required: ['pkg', 'text'],
    properties: { pkg: { type: 'string' }, text: { type: 'string' } } } },
  account: { type: 'string', description: 'Short account of what you changed and checked' } } }
const REVIEW = { type: 'object', required: ['status', 'accepted', 'returned', 'account'], properties: {
  status: { enum: ['submitted', 'model_blocker'] },
  accepted: { type: 'array', items: { type: 'string' } },
  returned: { type: 'array', items: { type: 'object', required: ['pkg', 'findings'],
    properties: { pkg: { type: 'string' }, findings: { type: 'array', items: FINDING } } } },
  commit: { type: 'string', pattern: SHA, description: 'Full 40-character SHA of your integrated state; required when nothing is returned' },
  findings: { type: 'array', items: FINDING },
  account: { type: 'string' } } }
const ANSWER = { type: 'object', required: ['escalate'], properties: {
  escalate: { type: 'boolean', description: 'true when the answer is outside your authority' },
  answer: { type: 'string' } } }

// The first line is parsed by the host's record hook: keep its shape.
function header(mandate, round, n) {
  const kids = n.children.map(c => c.pkg)
  return `Markitect ${mandate} round ${round}, run ${RUN}. You are the Manager of ${name(n.pkg)} (Package "${n.pkg}").\n` +
    `Your ownership areas: ${JSON.stringify(n.areas)}. Your parent: ${n.parent ? name(n.parent.pkg) : 'none (you are Root)'}. ` +
    `Your direct children: ${kids.length ? JSON.stringify(kids) : 'none'}.\nBound input of this command: ${BASE}.\n` +
    `Your checked context: ${MKP} context ${n.pkg}\nThe deterministic check of your working state: ${MKP} check\n`
}

// Assessment and Verify: children judge first, then the parent judges its own
// area, the interfaces and the children's judgments. A child's reject or
// model_blocker stays in force above it (concept §13.3).
async function judge(mandate, round, n, candidate) {
  const kids = await parallel(n.children.map(c => () => judge(mandate, round, c, candidate)))
  if (kids.some(k => !k)) return null
  const own = await call(header(mandate, round, n) +
    `Check out the fixed candidate detached first: git checkout --detach ${candidate}\n` +
    (mandate === 'Verify' ? `Its changes against the bound input: git diff ${BASE} ${candidate}\n` : '') +
    (kids.length ? `Your children's judgments: ${JSON.stringify(kids.map(k => ({ pkg: k.pkg, decision: k.decision, summary: k.summary, findings: k.findings })))}\n` : '') +
    `Judge ${mandate === 'Assessment' ? 'model readiness for your area' : 'the candidate for your area'} and answer with your decision.`,
    { agentType: 'markitect:verifier', schema: JUDGMENT, phase: mandate, label: `${mandate.toLowerCase()}:${n.pkg}:${round}` })
  if (!own) return null
  const decisions = [own.decision, ...kids.map(k => k.decision)]
  const decision = decisions.includes('model_blocker') ? 'model_blocker' : decisions.includes('reject') ? 'reject' : 'approve'
  log_.push({ mandate, round, pkg: n.pkg, own: own.decision, decision })
  return { pkg: n.pkg, decision, summary: own.summary, own: own.decision,
           findings: [...own.findings.map(f => ({ manager: n.pkg, ...f })), ...kids.flatMap(k => k.findings)] }
}

// A question climbs the ancestors until one answers within its authority.
// Root escalating ends the Apply as a model blocker: nobody outside the Run is asked.
async function answerUp(n, round, questions, step) {
  for (let a = n.parent; a; a = a.parent) {
    const r = await call(header('Implement', round, a) +
      `Your descendant ${name(n.pkg)} asks: ${JSON.stringify(questions)}\n` +
      `Answer within your authority and the checked model, or escalate when the answer is outside it. Change nothing.`,
      { agentType: 'markitect:verifier', schema: ANSWER, phase: 'Implement', label: `answer:${a.pkg}:${n.pkg}:${step}` })
    if (!r) return null
    if (!r.escalate) return `${name(a.pkg)}: ${r.answer}`
  }
  return null
}

// A Manager's own work: everything for a leaf; own obligations, shared
// prerequisites and delegation briefs for a parent. Questions pause the work;
// the answer starts a fresh agent from the Manager's own last commit.
async function work(n, round, from, brief, findings) {
  const answers = []
  for (let q = 0; q <= MAX_QUESTIONS; q++) {
    const b = branch(round, n, 'w')
    const h = await call(header('Implement', round, n) +
      `Start: git checkout -B ${b} ${from}\n` +
      (brief ? `Delegation from your parent: ${brief}\n` : '') +
      (findings.length ? `Findings you must resolve: ${JSON.stringify(findings)}\n` : '') +
      (answers.length ? `Answers to your earlier questions: ${JSON.stringify(answers)}\n` : '') +
      (n.children.length
        ? `Realize your own Definitions and prepare the shared prerequisites your children need, within your areas; do not realize their areas. ` +
          `Then write one delegation brief per direct child; pass on every Finding that concerns a child's area.\n`
        : `Realize your area.\n`) +
      `Commit everything on ${b} and hand off with status submitted and the full 40-character SHA from git rev-parse HEAD. ` +
      `Ask (status question) only when the checked model and your context cannot answer it.`,
      { agentType: 'markitect:manager', schema: HANDOFF, phase: 'Implement', label: `work:${n.pkg}:i${round}:${branches}` })
    if (!h) return null
    if (h.status === 'submitted' && !h.commit) return { pkg: n.pkg, status: 'incomplete', account: 'submitted without a commit' }
    if (h.status !== 'question') return { ...h, pkg: n.pkg }
    if (q === MAX_QUESTIONS) break
    const answer = await answerUp(n, round, h.questions || [], `${round}.${q}`)
    if (answer === null) return { pkg: n.pkg, status: 'model_blocker', findings: (h.questions || []).map(text => ({ manager: n.pkg, text })), account: 'question unanswered up to Root' }
    answers.push({ questions: h.questions, answer })
    if (h.commit) from = h.commit
  }
  return { pkg: n.pkg, status: 'incomplete', account: `question limit ${MAX_QUESTIONS} reached` }
}

async function implement(n, round, from, brief, findings) {
  // Root receives every Finding and delegates through its briefs. Below Root a
  // Manager receives the Findings on Definitions of its own subtree, and those
  // its own subtree reported without naming a Definition.
  const mine = n.parent ? findings.filter(f => within(owner(f), n) || (!owner(f) && within(f.manager || null, n))) : findings
  const own = await work(n, round, from, brief, mine)
  if (!own || own.status !== 'submitted' || !n.children.length) return own
  const briefs = Object.fromEntries((own.briefs || []).map(b => [b.pkg, b.text]))
  const handoffs = {}
  const first = await parallel(n.children.map(c => () => implement(c, round, own.commit, briefs[c.pkg] || '', mine)))
  n.children.forEach((c, i) => { handoffs[c.pkg] = first[i] })
  const children = n.children.map(c => c.pkg)
  let base = own.commit
  let undecided = []
  for (let r = 0; ; r++) {
    const bad = Object.entries(handoffs).find(([, h]) => !h || h.status !== 'submitted')
    if (bad) return { pkg: n.pkg, status: (bad[1] && bad[1].status) || 'incomplete', findings: (bad[1] && bad[1].findings) || [], account: `child ${bad[0]}: ${bad[1] ? bad[1].account : 'no handoff'}` }
    const b = branch(round, n, 'n')
    const rev = await call(header('Implement', round, n) +
      `Integrate on a new branch: git checkout -B ${b} ${base}\n` +
      `Your children's handoffs: ${JSON.stringify(Object.values(handoffs).map(h => ({ pkg: h.pkg, commit: h.commit, account: h.account })))}\n` +
      (undecided.length ? `Your last review decided nothing about ${JSON.stringify(undecided)}: accept or return every child.\n` : '') +
      `Review each handoff by its actual diff (git diff ${base}...COMMIT) and its host check note (git notes --ref=markitect-check show COMMIT). ` +
      `Merge every handoff you accept (git merge --no-ff COMMIT), resolve conflicts, align the interfaces and run the check. ` +
      `List every child either in accepted or in returned. Return a handoff with concrete Findings when it needs a correction; do not repair a child's area yourself. ` +
      `When nothing is returned, commit the integrated state on ${b} and answer submitted with its full 40-character SHA.`,
      { agentType: 'markitect:manager', schema: REVIEW, phase: 'Implement', label: `integrate:${n.pkg}:i${round}:r${r}` })
    if (!rev) return { pkg: n.pkg, status: 'incomplete', account: 'integration agent failed' }
    if (rev.status === 'model_blocker') return { pkg: n.pkg, status: 'model_blocker', findings: rev.findings || [], account: rev.account }
    const returned = rev.returned.filter(x => children.includes(x.pkg))
    undecided = children.filter(c => !rev.accepted.includes(c) && !returned.some(x => x.pkg === c))
    if (!returned.length && !undecided.length) {
      if (!rev.commit) return { pkg: n.pkg, status: 'incomplete', account: 'integration submitted without a commit' }
      return { pkg: n.pkg, status: 'submitted', commit: rev.commit, account: rev.account }
    }
    if (r >= MAX_RETURNS) return { pkg: n.pkg, status: 'incomplete', account: `return limit ${MAX_RETURNS} reached` }
    if (rev.commit) base = rev.commit
    const again = await parallel(returned.map(x => () => {
      const c = n.children.find(k => k.pkg === x.pkg)
      return implement(c, round, handoffs[x.pkg].commit, `Correction from your parent. Merge my current base ${base} first.`,
        x.findings.map(f => ({ manager: c.pkg, ...f })))
    }))
    returned.forEach((x, i) => { handoffs[x.pkg] = again[i] })
  }
}

async function apply() {
  phase('Assessment')
  const assessment = await judge('Assessment', 0, A.tree, BASE)
  if (!assessment) return done({ outcome: 'incomplete', at: 'Assessment' })
  if (assessment.decision !== 'approve') return done({ outcome: assessment.decision, at: 'Assessment', assessment })

  let candidate = BASE
  let findings = []
  let verify = null
  const rounds = []
  for (let round = 1; round <= MAX_CORRECTIONS + 1; round++) {
    phase('Implement')
    const impl = await implement(A.tree, round, candidate, '', findings)
    rounds.push({ round, implement: impl && { status: impl.status, commit: impl.commit, account: impl.account } })
    if (!impl || impl.status !== 'submitted') return done({ outcome: (impl && impl.status) || 'incomplete', at: 'Implement', round, impl, assessment, rounds })
    candidate = impl.commit
    phase('Verify')
    verify = await judge('Verify', round, A.tree, candidate)
    rounds[rounds.length - 1].verify = verify && { decision: verify.decision, findings: verify.findings }
    if (!verify) return done({ outcome: 'incomplete', at: 'Verify', round, candidate, rounds })
    if (verify.decision === 'approve') return done({ outcome: 'approved', round, candidate, assessment, verify, rounds })
    if (verify.decision === 'model_blocker') return done({ outcome: 'model_blocker', at: 'Verify', candidate, verify, rounds })
    // DEC-073: only Findings that name a Definition reach the next Implement.
    findings = verify.findings.filter(f => f.definition)
    if (!findings.length) return done({ outcome: 'reject', at: 'Verify', candidate, verify, rounds, reason: 'no Finding names a Definition' })
  }
  return done({ outcome: 'reject', at: 'corrections exhausted', candidate, verify, rounds })
}

try {
  return await apply()
} catch (e) {
  return done({ outcome: 'incomplete', error: String(e && e.message || e) })
}
