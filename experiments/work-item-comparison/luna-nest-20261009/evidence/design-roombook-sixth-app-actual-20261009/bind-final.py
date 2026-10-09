from pathlib import Path
import sys,json,shutil
E=Path(__file__).resolve().parent;PUBLIC=E.parent.parent/'public'
sys.path.insert(0,str(PUBLIC));import nest,app_ledger
grant=Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/design-roombook-sixth-enabled-grant-20261009.json')
g=nest.read(grant);state=Path(g['stateDir']);repo=Path(g['repo']);handle='/root/design_roombook_sixth_independent_final'
assert nest.sha(E/'final-message.txt')=='3f9fed314dc0ae8cf8d648c35f3db0601673b3c20476816132f3031b2239b404'
assert nest.read(state/'final-startup-gate.json')['released'] is False
assert nest.read(state/'startup-gate.json')['released'] is False
receipt={'actualReturnedToolHandle':handle,'actualToolResult':{'task_name':handle},'toolReturnRecordedUtc':nest.utc(),
         'toolInvocationUtc':None,'actualModelStartUtc':None,'requestedModel':'gpt-6-luna','reasoning':'high','forkTurns':'none',
         'actualServingModel':None,'usage':None,'cost':None,'messageSha256':nest.sha(E/'final-message.txt')}
app_ledger.record(state,2,'started',handle,receipt)
binding=nest.read(state/'inner-binding.json');oldsha=nest.sha(state/'inner-binding.json')
for pin in binding['runtimePins']:assert nest.sha(pin['path'])==pin['sha256']
runtime=repo/'.markitect/runtime.yaml';before=runtime.read_bytes();assert before.count(oldsha.encode())==2
(E/'final-runtime-before.yaml').write_bytes(before);shutil.copyfile(state/'inner-binding.json',E/'final-inner-binding-before.json')
binding.update(parentReservation=2,parentHandle=handle,phase='assessment')
nest.write(state/'inner-binding.json',binding);newsha=nest.sha(state/'inner-binding.json')
after=before.replace(oldsha.encode(),newsha.encode());assert after.count(newsha.encode())==2
assert after.replace(newsha.encode(),oldsha.encode())==before
runtime.write_bytes(after);(E/'final-runtime-enabled.yaml').write_bytes(after)
shutil.copyfile(state/'inner-binding.json',E/'final-inner-binding-enabled.json')
frozen=nest.read(state/'state.json')['freeze'];gate={'released':True,'reservation':2,'ownHandle':handle,
 'phase':'assessment','stationIndex':0,'bindingSha256':newsha,'runtimeSha256':nest.sha(runtime),
 'grantSha256':nest.sha(grant),'releasedUtc':nest.utc(),'frozenCandidateUntouched':True}
nest.write(E/'final-tool-return.json',receipt)
nest.write(E/'final-resource-activation.json',{'gate':gate,'originalMutableCaseRuntimeOnly':True,
 'frozenMainCandidate':frozen['evaluationRepo'],'frozenMainCommit':frozen['mainCommit'],
 'runtimeBeforeSha256':nest.sha(E/'final-runtime-before.yaml'),'runtimeAfterSha256':nest.sha(runtime),
 'bindingBeforeSha256':oldsha,'bindingAfterSha256':newsha,'candidateFilesNotChanged':True})
nest.write(state/'final-startup-gate.json',gate);shutil.copyfile(state/'final-startup-gate.json',E/'final-startup-gate-released.json')
print(json.dumps({'handle':handle,'releasedUtc':gate['releasedUtc'],'bindingSha256':newsha,'runtimeSha256':nest.sha(runtime)}))
