// Runs plugin/workflows/apply.js with scripted agents instead of Claude, so the
// orchestration can be tested without a provider: node simulate.js SCENARIO.json
// The scenario holds the workflow args and rules; every agent call is answered
// by the first rule whose label prefix matches. The output lists every call.
const fs = require('fs')
const path = require('path')

const scenario = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'))
let source = fs.readFileSync(path.join(__dirname, 'plugin', 'workflows', 'apply.js'), 'utf8')
source = source.replace('export const meta', 'const meta')

const calls = []
let commits = 0
const used = {}
function respond(prompt, opts) {
  const label = opts.label
  calls.push({ label, agentType: opts.agentType, isolation: opts.isolation })
  for (const rule of scenario.rules) {
    if (!label.startsWith(rule.label)) continue
    used[rule.label] = (used[rule.label] || 0) + 1
    if (rule.times && used[rule.label] > rule.times) continue
    const value = JSON.parse(JSON.stringify(rule.answer))
    if (value.commit === 'NEW') value.commit = 'c' + String(++commits).padStart(39, '0')
    return value
  }
  throw new Error('no rule for ' + label)
}
const agent = async (prompt, opts) => respond(prompt, opts)
const parallel = async thunks => Promise.all(thunks.map(t => t().catch(() => null)))
const phase = () => {}
const log = () => {}

const run = new Function('args', 'agent', 'parallel', 'phase', 'log', 'return (async () => {' + source + '\n})()')
run(scenario.args, agent, parallel, phase, log).then(
  result => process.stdout.write(JSON.stringify({ result, calls }) + '\n'),
  error => process.stdout.write(JSON.stringify({ error: String(error), calls }) + '\n'))
