from pathlib import Path
import sys,json,shutil
E=Path(__file__).resolve().parent;PUBLIC=E.parent.parent/'public'
sys.path.insert(0,str(PUBLIC));import nest,app_ledger
grant=Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/design-roombook-sixth-enabled-grant-20261009.json')
g=nest.read(grant);state=Path(g['stateDir']);repo=Path(g['repo'])
nest.write(state/'startup-gate.json',{'released':False,'reservation':1,'ownHandle':'/root/design_roombook_sixth_primary',
                                  'phase':'implementation','closedUtc':nest.utc(),'reason':'Reported genuine product-policy blocker; no continuation'})
completion=nest.read(repo/'.study/completion.json');assert completion['station']=='S1' and completion['status']=='blocked'
receipt={'terminalToolResultReceived':True,'observedUtc':nest.utc(),'reportedMainCommit':'ffcf65173c030e8718ab7098c8b461dfe03b0395',
 'reportedFeatureHead':'8fe3df7e8f5c74173071793a8e399a364c2c5e4e','reportedOwnJobsAllTerminal':True,
 'reportedProductBlocker':'project briefings/readiness: briefing bundle is invalid: incomplete definition identity',
 'actualModelStartUtc':None,'reportedRequestedModel':'gpt-6-luna','reasoning':'high','actualServingModel':None,'usage':None,'cost':None,
 'reportedExposure':'Own public project files/rules, own grant/gate; no foreign case/chat/private eval/memory/global inventory/credentials known; no OS isolation attestation'}
nest.write(E/'primary-terminal-resource-receipt.json',receipt)
app_ledger.close_implementation(state,'Genuine product-policy blocker at S1; no continuation or alternate transport')
app_ledger.record(state,1,'returned','/root/design_roombook_sixth_primary',receipt)
assert len(nest.read(state/'app-ledger.json')['entries'])==1
shot=nest.snapshot(state);freeze=nest.freeze(state)
nest.write(E/'S1-snapshot.json',shot);nest.write(E/'freeze.json',freeze)
nest.write(E/'implementation-closed-ledger.json',nest.read(state/'app-ledger.json'))
copy=E/'reported-completion.json';shutil.copyfile(repo/'.study/completion.json',copy)
entry=app_ledger.reserve(state,grant,'assessment');nest.write(E/'final-reservation.json',entry)
nest.write(state/'final-startup-gate.json',{'released':False,'reservation':entry['reservation'],'ownHandle':None,'phase':'assessment'})
prompt=(PUBLIC/'evaluation/assessor-prompt.txt').read_text(encoding='utf-8')
assert nest.sha(PUBLIC/'evaluation/assessor-prompt.txt')=='7eec156128339344121c1040c3f6b76f1cc5545c239eb99b6462420163a72741'
message=prompt+f'''
Execution/resource bindings only:
Own frozen main candidate: {freeze['evaluationRepo']}
Own retained station S1 main: {shot['evaluationRepo']}
Own resource state: {state}
Immutable enabled grant: {grant}; SHA256 {nest.sha(grant)}.
Current assessment reservation: {entry['reservation']}; deadline {entry['deadlineUtc']}.
Requested gpt-6-luna/high/fork_turns=none. One fresh final assessor only; any executing own helper must reserve in this same app_ledger as assessment-helper depth2 before spawn, same profile, shared72/four/depth2. No extra role/probe is supplied by missing receipts.
Before any assessment tool work, wait on {state/'final-startup-gate.json'} until released=true with this real own handle and reservation. No candidate changes/repairs/commits or product/provider compatibility probes. Own temporary tests may test existing public candidate behavior; do not create a substitute application for missing candidate code.
No implementer transcript, previous evaluation, foreign case/code/chat/memory/private holdouts or global agent/process inventory. Use only the named own immutable main snapshots/public requirements and own temporary files; do not modify retained candidate bytes. Inherited system/developer/workspace context is not proven sanitized or isolated; general context labels alone do not establish forbidden exposure, known concrete forbidden content requires stop and unresolved case-specific exposure stays uncertain.
Provide your normal readable report in an own external file under {state/'assessment'} and return its absolute path/digest. Record actual commands, candidate hashes before/after, requested/observed profile, own known shell/process jobs terminal status. Serving/tokens/cost/human acceptance remain unknown unless genuine receipts exist. Stage outcomes may be PASS/FAIL/NOT RUN/unknown; do not infer success from transport. Do not advise or message the implementer.
'''
(E/'final-message.txt').write_bytes(message.encode('utf-8'))
nest.write(E/'final-call-intent.json',{'tool':'collaboration.spawn_agent','task_name':'design_roombook_sixth_independent_final',
 'fork_turns':'none','model':'gpt-6-luna','reasoning_effort':'high','messageSha256':nest.sha(E/'final-message.txt'),
 'reservation':entry['reservation'],'toolInvocationUtc':None,'actualModelStartUtc':None,'handle':None})
print(json.dumps({'freeze':freeze,'snapshot':{k:shot[k] for k in ['station','mainCommit','headCommit','evaluationRepo']},
 'entry':entry,'messageSha256':nest.sha(E/'final-message.txt'),'message':message},ensure_ascii=True))
