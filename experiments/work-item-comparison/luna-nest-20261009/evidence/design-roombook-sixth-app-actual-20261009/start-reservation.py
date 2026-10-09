from pathlib import Path
import sys,json,shutil,hashlib
E=Path(__file__).resolve().parent;PUBLIC=E.parent.parent/'public'
sys.path.insert(0,str(PUBLIC));import nest,app_ledger
grant=Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/design-roombook-sixth-enabled-grant-20261009.json')
assert nest.sha(grant)=='6ffb11f2c31b043f283c15484840ee4630a17d968c7b6b37743b969ca6055811'
g=nest.read(grant);state=Path(g['stateDir']);repo=Path(g['repo'])
assert nest.sha(g['rootActivationReceipt'])==g['rootActivationReceiptSha256']
assert nest.sha(PUBLIC/'app_ledger.py')==g['appLedgerSourceSha256'] and nest.sha(PUBLIC/'nest.py')==g['nestSourceSha256']
assert nest.read(state/'app-ledger.json')['entries']==[]
for src,name in [(grant,'enabled-grant.json'),(Path(g['rootActivationReceipt']),'root-activation.json')]:shutil.copyfile(src,E/name)
nest.write(state/'startup-gate.json',{'released':False,'ownHandle':None,'reservation':None,'phase':'implementation','grantSha256':nest.sha(grant)})
entry=app_ledger.reserve(state,grant,'implementation')
nest.write(E/'first-reservation.json',entry)
message=nest.PROMPT+f'''

Execution/resource bindings only:
Own isolated project: {repo}
Own resource state: {state}
Immutable enabled grant: {grant}
Enabled grant SHA256: {nest.sha(grant)}
Current outer reservation: {entry['reservation']}; phase implementation; S1.
Requested model gpt-6-luna, reasoning high, fresh fork_turns=none. All own helpers/reviewers must use the same profile and this case's existing app-ledger reservations before spawning. Shared maximum72 starts including all outer/inner/helper/final roles, four active roles total, depth2; at most8outer implementation entries/two per station; original station and phase deadlines never extend. Current outer deadline: {entry['deadlineUtc']}.
Read own repo AGENTS.md and TOOLCHAIN.md, own public task files and genuine declared local product installation. Work only in this project and its own temporary files. Normal native App/tools/protections remain active. Use only legitimate history from this own task. Do not read other cells/chats/past case solutions/private evaluation materials or enumerate global agents/processes. General system/developer/workspace instructions remain inherited; this is not proof of isolation. Report concrete forbidden foreign exposure and stop; unresolved case-specific exposure remains uncertain and must not be used as advice.
STARTUP RESOURCE GATE: before any product invocation, wait on {state/'startup-gate.json'} until released=true and its real ownHandle/current reservation match the resource binding. While closed do no product/provider compatibility probe or alternative transport. Wait in intervals at most10seconds, inside this outer deadline. The resource coordinator records your actual returned handle, enables the exact inner binding and installs both runtime binding digest arguments before releasing this gate. This release carries resource metadata only, no task advice. Failed or unknown own cleanup stops future admission; no deadline/quota reset. Actual product roles keep genuine source invocation/response handling and are counted by the existing external wrapper in this same ledger.
All existing common rules and completion handoff format remain in the own repository. Do not install packages/change global settings or bypass protections. Respect the immutable enabled grant and report a real policy/resource blocker rather than changing transport. Serving model/tokens/cost stay unknown without actual receipts.
'''
(E/'primary-message.txt').write_bytes(message.encode('utf-8'))
nest.write(E/'primary-call-intent.json',{'tool':'collaboration.spawn_agent','task_name':'design_roombook_sixth_primary','fork_turns':'none','model':'gpt-6-luna','reasoning_effort':'high','messageSha256':nest.sha(E/'primary-message.txt'),'messageBytes':len(message.encode('utf-8')),'reservation':entry['reservation'],'reservationUtc':entry['reservedUtc'],'toolInvocationUtc':None,'actualModelStartUtc':None,'handle':None})
print(json.dumps({'entry':entry,'messageSha256':nest.sha(E/'primary-message.txt'),'message':message},ensure_ascii=False))
