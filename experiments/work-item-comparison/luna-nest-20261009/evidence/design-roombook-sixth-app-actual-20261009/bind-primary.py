"""Resource-only real-handle binding after the actual spawn returned."""
from pathlib import Path
import sys,json,shutil
E=Path(__file__).resolve().parent;PUBLIC=E.parent.parent/'public'
sys.path.insert(0,str(PUBLIC));import nest,app_ledger
handle='/root/design_roombook_sixth_primary'
grant=Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/design-roombook-sixth-enabled-grant-20261009.json')
g=nest.read(grant);state=Path(g['stateDir']);repo=Path(g['repo'])
assert nest.sha(grant)=='6ffb11f2c31b043f283c15484840ee4630a17d968c7b6b37743b969ca6055811'
assert nest.sha(E/'primary-message.txt')=='68030b756fa06eec79c15732c407b7080e1ef32bc559a2e4845ec602ef0dcf2f'
before=nest.read(state/'inner-binding.json');oldsha=nest.sha(state/'inner-binding.json')
assert before['executionAuthorized'] is False and nest.read(state/'startup-gate.json')['released'] is False
assert nest.read(state/'state.json')['stationIndex']==0
for pin in before['runtimePins']:assert nest.sha(pin['path'])==pin['sha256']
receipt={'actualReturnedToolHandle':handle,'actualToolResult':{'task_name':handle},'toolReturnRecordedUtc':nest.utc(),
 'toolInvocationUtc':None,'actualModelStartUtc':None,'requestedModel':'gpt-6-luna','reasoning':'high','forkTurns':'none',
 'servingModel':None,'usage':None,'cost':None,'literalMessageSha256':nest.sha(E/'primary-message.txt')}
app_ledger.record(state,1,'started',handle,receipt)
shutil.copyfile(state/'inner-binding.json',E/'inner-binding-before.json')
runtime=repo/'.markitect/runtime.yaml';raw=runtime.read_bytes();assert raw.count(oldsha.encode())==2
(E/'runtime-before.yaml').write_bytes(raw)
binding=dict(before,executionAuthorized=True,grantPath=str(grant),grantSha256=nest.sha(grant),parentReservation=1,parentHandle=handle,phase='implementation')
binding['activation']='Actual returned own handle recorded in shared ledger; resource-only activation inside original cell clock'
nest.write(state/'inner-binding.json',binding);newsha=nest.sha(state/'inner-binding.json')
updated=raw.replace(oldsha.encode(),newsha.encode());assert updated.count(newsha.encode())==2
assert updated.replace(newsha.encode(),oldsha.encode())==raw
runtime.write_bytes(updated)
shutil.copyfile(state/'inner-binding.json',E/'inner-binding-enabled.json');(E/'runtime-enabled.yaml').write_bytes(updated)
entry=nest.read(state/'app-ledger.json')['entries'][0]
assert entry['handle']==handle and entry['status']=='started' and entry['stationIndex']==0
assert app_ledger.now()<app_ledger.date(entry['deadlineUtc'])
gate={'released':True,'reservation':1,'ownHandle':handle,'phase':'implementation','stationIndex':0,
      'grantSha256':nest.sha(grant),'bindingSha256':newsha,'runtimeSha256':nest.sha(runtime),'releasedUtc':nest.utc()}
nest.write(E/'primary-tool-return.json',receipt)
nest.write(E/'resource-activation.json',{'parentEntry':entry,'gate':gate,'runtimeBeforeSha256':nest.sha(E/'runtime-before.yaml'),
 'runtimeAfterSha256':nest.sha(runtime),'bindingBeforeSha256':oldsha,'bindingAfterSha256':newsha,
 'delta':'Only enabled grant/actual own parent fields in binding, both binding-sha256 argv in runtime; no target argv/prompt/model/DTO change',
 'caseClockStartedUtc':entry['reservedUtc'],'modelStartUtc':None,'runtimeMutationInsideCellClock':True})
nest.write(state/'startup-gate.json',gate);shutil.copyfile(state/'startup-gate.json',E/'startup-gate-released.json')
nest.write(E/'start-ledger.json',nest.read(state/'app-ledger.json'))
print(json.dumps({'handle':handle,'reservationUtc':entry['reservedUtc'],'toolReturnRecordedUtc':receipt['toolReturnRecordedUtc'],
 'gateReleasedUtc':gate['releasedUtc'],'bindingSha256':newsha,'runtimeSha256':nest.sha(runtime),'outerDeadlineUtc':entry['deadlineUtc']}))
