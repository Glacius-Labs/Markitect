export const meta = {
  name: 'apply',
  description: 'Markitect Apply (arm P): Assessment, Implement and Verify through the compiled Manager tree',
  whenToUse: 'Invoked by /markitect:apply after mkp.py prepare passed. args: the prepare record {runId, base, mkp, tree} plus {maxCorrections, maxReturns, maxQuestions, maxAgents}.',
  phases: [{ title: 'Assessment' }, { title: 'Implement' }, { title: 'Verify' }],
}

// The script owns order and limits; Managers own meaning. Every agent() call
// is a fresh context in its own git worktree. Nothing here reads files: the
// prepare record supplies the tree, and branches carry the work.
const A = typeof args === 'string' ? JSON.parse(args) : args
if (!A || !A.runId || !A.base || !A.mkp || !A.tree) throw new Error('apply needs the prepare record {runId, base, mkp, tree}')
const RUN = A.runId
const BASE = A.base
const MAX_CORRECTIONS = A.maxCorrections ?? 2
const MAX_RETURNS = A.maxReturns ?? 2
const MAX_QUESTIONS = A.maxQuestions ?? 3
const MAX_AGENTS = A.maxAgents ?? 150
const MKP = `python "${A.mkp}"`

let agents = 0
let exhausted = false
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
const all = []
;(function walk(n, parent) { n.parent = parent; all.push(n); n.children.forEach(c => walk(c, n)) })(A.tree, null)

const FINDING = { type: 'object', required: ['text'], properties: {
  definition: { type: 'string', description: 'Package/ID of the Definition concerned, when there is one' },
  text: { type: 'string' } } }
const JUDGMENT = { type: 'object', required: ['decision', 'findings', 'summary'], properties: {
  decision: { enum: ['approve', 'reject', 'model_blocker'] },
  findings: { type: 'array', items: FINDING },
  summary: { type: 'string', description: 'What you examined and why you decided so; optional observations go here' } } }
const HANDOFF = { type: 'object', required: ['status', 'account'], properties: {
  status: { enum: ['submitted', 'question', 'model_blocker'] },
  commit: { type: 'string', description: 'Full SHA of your committed handoff; required for submitted' },
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
  commit: { type: 'string', description: 'Full SHA of your integrated state; required when nothing is returned' },
  findings: { type: 'array', items: FINDING },
  account: { type: 'string' } } }
const ANSWER = { type: 'object', required: ['escalate'], properties: {
  escalate: { type: 'boolean', description: 'true when the answer is outside your authority' },
  answer: { type: 'string' } } }

function tools(pkg) {
  return `Your checked context: ${MKP} context ${pkg}\nThe deterministic check of your working state: ${MKP} check\n`
}
function header(mandate, n) {
  const kids = n.children.map(c => c.pkg)
  return `Markitect ${mandate}, run ${RUN}. You are the Manager of ${name(n.pkg)} (Package "${n.pkg}").\n` +
    `Your ownership areas: ${JSON.stringify(n.areas)}. Your parent: ${n.parent ? name(n.parent.pkg) : 'none (you are Root)'}. ` +
    `Your direct children: ${kids.length ? JSON.stringify(kids) : 'none'}.\nBound input of this command: ${BASE}.\n` + tools(n.pkg)
}

// Assessment and Verify: children judge first, then the parent judges its own
// area, the interfaces and the children's judgments. A child's reject or
// model_blocker stays in force above it (concept §13.3).
async function judge(mandate, n, candidate) {
  const kids = await parallel(n.children.map(c => () => judge(mandate, c, candidate)))
  if (kids.some(k => !k)) return null
  const own = await call(header(mandate, n) +
    `Check out the fixed candidate detached: git checkout --detach ${candidate}\n` +
    (mandate === 'Verify' ? `Its changes against the bound input: git diff ${BASE} ${candidate}\n` : '') +
    (kids.length ? `Your children's judgments: ${JSON.stringify(kids.map(k => ({ pkg: k.pkg, decision: k.decision, summary: k.summary, findings: k.findings })))}\n` : '') +
    `Judge ${mandate === 'Assessment' ? 'model readiness for your area' : 'the candidate for your area'} and answer with your decision.`,
    { agentType: 'markitect:verifier', schema: JUDGMENT, phase: mandate, label: `${mandate.toLowerCase()}:${n.pkg}` })
  if (!own) return null
  const decisions = [own.decision, ...kids.map(k => k.decision)]
  const decision = decisions.includes('model_blocker') ? 'model_blocker' : decisions.includes('reject') ? 'reject' : 'approve'
  log_.push({ mandate, pkg: n.pkg, own: own.decision, decision })
  return { pkg: n.pkg, decision, summary: own.summary, own: own.decision,
           findings: [...own.findings.map(f => ({ manager: n.pkg, ...f })), ...kids.flatMap(k => k.findings)] }
}

// A question climbs the ancestors until one answers within its authority.
// Root escalating ends the Apply as a model blocker: nobody outside the Run is asked.
async function answerUp(n, questions, round) {
  for (let a = n.parent; a; a = a.parent) {
    const r = await call(header('Implement', a) +
      `Your descendant ${name(n.pkg)} asks: ${JSON.stringify(questions)}\n` +
      `Answer within your authority and the checked model, or escalate when the answer is outside it. Change nothing.`,
      { agentType: 'markitect:verifier', schema: ANSWER, phase: 'Implement', label: `answer:${a.pkg}:${n.pkg}:${round}` })
    if (!r) return null
    if (!r.escalate) return `${name(a.pkg)}: ${r.answer}`
  }
  return null
}

// A Manager's own work: everything for a leaf; own obligations, shared
// prerequisites and delegation briefs for a parent. Questions pause the work;
// the answer starts a fresh agent on the Manager's own branch.
async function work(n, round, from, brief, findings) {
  const branch = `mk/${RUN}/i${round}/${slug(n.pkg)}`
  const answers = []
  for (let q = 0; q <= MAX_QUESTIONS; q++) {
    const h = await call(header('Implement', n) +
      `Start: git checkout -B ${branch} ${from}\n` +
      (brief ? `Delegation from your parent: ${brief}\n` : '') +
      (findings.length ? `Findings you must resolve: ${JSON.stringify(findings)}\n` : '') +
      (answers.length ? `Answers to your earlier questions: ${JSON.stringify(answers)}\n` : '') +
      (n.children.length
        ? `Realize your own Definitions and prepare the shared prerequisites your children need, within your areas; do not realize their areas. ` +
          `Then write one delegation brief per direct child.\n`
        : `Realize your area.\n`) +
      `Commit everything on ${branch} and hand off with status submitted and the full commit SHA. ` +
      `Ask (status question) only when the checked model and your context cannot answer it.`,
      { agentType: 'markitect:manager', schema: HANDOFF, phase: 'Implement', label: `work:${n.pkg}:i${round}${q ? ':q' + q : ''}` })
    if (!h || h.status !== 'question') return h && { ...h, pkg: n.pkg }
    if (q === MAX_QUESTIONS) break
    const answer = await answerUp(n, h.questions || [], `${round}.${q}`)
    if (answer === null) return { pkg: n.pkg, status: 'model_blocker', findings: (h.questions || []).map(text => ({ manager: n.pkg, text })), account: 'question unanswered up to Root' }
    answers.push({ questions: h.questions, answer })
    if (h.commit) from = h.commit
  }
  return { pkg: n.pkg, status: 'incomplete', account: `question limit ${MAX_QUESTIONS} reached` }
}

async function implement(n, round, from, brief, findings) {
  // Root receives every Finding and delegates through its briefs; below Root a
  // Manager also sees the Findings its own subtree reported.
  const mine = n.parent ? findings.filter(f => f.manager === n.pkg || !f.manager) : findings
  const own = await work(n, round, from, brief, mine)
  if (!own || own.status !== 'submitted' || !n.children.length) return own
  const briefs = Object.fromEntries((own.briefs || []).map(b => [b.pkg, b.text]))
  const below = c => findings.filter(f => f.manager === c.pkg || (f.manager || '').startsWith(c.pkg + '/'))
  const handoffs = {}
  const first = await parallel(n.children.map(c => () => implement(c, round, own.commit, briefs[c.pkg] || '', below(c))))
  n.children.forEach((c, i) => { handoffs[c.pkg] = first[i] })
  let base = own.commit
  for (let r = 0; ; r++) {
    const bad = Object.entries(handoffs).find(([, h]) => !h || h.status !== 'submitted')
    if (bad) return { pkg: n.pkg, status: (bad[1] && bad[1].status) || 'incomplete', findings: (bad[1] && bad[1].findings) || [], account: `child ${bad[0]}: ${bad[1] ? bad[1].account : 'no handoff'}` }
    const rev = await call(header('Implement', n) +
      `Continue on your branch: git checkout -B mk/${RUN}/i${round}/${slug(n.pkg)} ${base}\n` +
      `Your children's handoffs: ${JSON.stringify(Object.values(handoffs).map(h => ({ pkg: h.pkg, commit: h.commit, account: h.account })))}\n` +
      `Review each handoff by its actual diff (git diff ${base}...COMMIT) and its host check note (git notes --ref=markitect-check show COMMIT). ` +
      `Merge every handoff you accept (git merge --no-ff COMMIT), resolve conflicts, align the interfaces and run the check. ` +
      `Return a handoff with concrete Findings when it needs a correction; do not repair a child's area yourself. ` +
      `When nothing is returned, commit the integrated state and answer submitted with its full SHA.`,
      { agentType: 'markitect:manager', schema: REVIEW, phase: 'Implement', label: `integrate:${n.pkg}:i${round}:r${r}` })
    if (!rev) return { pkg: n.pkg, status: 'incomplete', account: 'integration agent failed' }
    if (rev.status === 'model_blocker') return { pkg: n.pkg, status: 'model_blocker', findings: rev.findings || [], account: rev.account }
    if (!rev.returned.length) return { pkg: n.pkg, status: 'submitted', commit: rev.commit, account: rev.account }
    if (r >= MAX_RETURNS) return { pkg: n.pkg, status: 'incomplete', account: `return limit ${MAX_RETURNS} reached` }
    if (rev.commit) base = rev.commit
    const again = await parallel(rev.returned.map(x => () => {
      const c = n.children.find(k => k.pkg === x.pkg)
      if (!c) return Promise.resolve(null)
      return implement(c, round, handoffs[x.pkg].commit, `Correction from your parent. Merge my current base ${base} first.`, x.findings.map(f => ({ manager: c.pkg, ...f })))
    }))
    rev.returned.forEach((x, i) => { handoffs[x.pkg] = again[i] })
  }
}

phase('Assessment')
const assessment = await judge('Assessment', A.tree, BASE)
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
  verify = await judge('Verify', A.tree, candidate)
  rounds[rounds.length - 1].verify = verify && { decision: verify.decision, findings: verify.findings }
  if (!verify) return done({ outcome: 'incomplete', at: 'Verify', round, candidate, rounds })
  if (verify.decision === 'approve') return done({ outcome: 'approved', candidate, assessment, verify, rounds })
  if (verify.decision === 'model_blocker') return done({ outcome: 'model_blocker', at: 'Verify', candidate, verify, rounds })
  // DEC-073: only Findings that name a Definition reach the next Implement.
  findings = verify.findings.filter(f => f.definition)
  if (!findings.length) return done({ outcome: 'reject', at: 'Verify', candidate, verify, rounds, reason: 'no Finding names a Definition' })
}
return done({ outcome: 'reject', at: 'corrections exhausted', candidate, verify, rounds })
