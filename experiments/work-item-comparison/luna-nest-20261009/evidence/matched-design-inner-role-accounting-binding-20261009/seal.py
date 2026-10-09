"""File/hash evidence closure only; does not start adapters, models or tests."""
from pathlib import Path
import hashlib,json,zipfile
from datetime import datetime,timezone

ROOT=Path(__file__).resolve().parent
PUBLIC=ROOT.parent.parent/'public'
def sha(path): return hashlib.sha256(Path(path).read_bytes()).hexdigest()
def write(name,value): (ROOT/name).write_text(json.dumps(value,ensure_ascii=True,indent=2)+'\n',encoding='utf-8')

replacement=ROOT/'private-owner-replacement'
new=json.loads((replacement/'handoff.json').read_text(encoding='utf-8-sig'))
assert sha(replacement/'handoff.json')=='2b5615e9231208990f8a4748bb6d022dda9b2d5201b9e851f205574e469f1c76'
assert sha(replacement/'native-policy-replacement-bcd614a3.zip')=='1eb592b4560204ed799eeaae4386c559a483cb1e1a27aecce2a2857e458f933a'
for item in new['files']: assert sha(replacement/item['path'])==item['sha256'],item['path']
old=ROOT/'private-owner-installation'
assert sha(old/'handoff.json')=='0cf49b54ae019ab2ab2cb6dd3e863c995823a49137a375a9e5e4547ee8aa4cb2'
assert sha(old/'roombook-model-installation-2b7b22ba.zip')=='1f29af68c6ec2b295c6930c76512deac93a3c565dd6f1fbef1539db5a78918b0'
archives=[]
for archive in [old/'roombook-model-installation-2b7b22ba.zip',replacement/'native-policy-replacement-bcd614a3.zip']:
    with zipfile.ZipFile(archive) as z:
        archives.append({'archive':str(archive.relative_to(ROOT)),'sha256':sha(archive),'entries':[
            {'name':i.filename,'bytes':i.file_size,'sha256':hashlib.sha256(z.read(i)).hexdigest()}
            for i in z.infolist() if not i.is_dir()]})
write('archive-payload-manifest.json',archives)
baseline={'app_ledger.py':'3b3633202954804e983e53e939d852f58ab45a9b3a1d1e232954c6e7e3c75d20',
          'nest.py':'45241de9e157394cf2dd83412ebd3d86ce83764f1b791447030e5d82f320d635',
          'app-entry.md':'63dff8fddec0562fd7929e9d09b49443e8a42c21d4d3ce787518aed437e09655',
          'oldschool-app-dispatch.md':'be8e2a003c561ad7e9d0cad809ea365e2c1b8535b7dbbf42f2954ede931d3751'}
for name,digest in baseline.items(): assert sha(PUBLIC/name)==digest,name
terminal={
 'closedUtc':datetime.now(timezone.utc).isoformat(),'status':'qualified source package; actual admission disabled',
 'grantSha256':sha(ROOT/'grant.json'),'grantEndUtc':'2026-10-09T10:12:18Z',
 'sourceBaseline':'9e2bbeaa9965bf6fb4d5d95c0812165a069f1191','sharedFramework':'2037bfa65e7e05ee004921d282893d4a6c8e899a',
 'offlineChecksConsumed':8,'offlineAttemptResults':['check1 FAIL','checks2-6 PASS','check7 exit preservation PASS','check8 reserve/pin rejection and descendant timeout PASS'],
 'sourceReviewerStarts':1,'reviewerHandle':'/root/inner_runtime_source_review','reviewerModelRequested':'gpt-6-luna','reviewerReasoningRequested':'high','reviewerForkTurns':'none',
 'reviewerMaxSeconds':420,'reviewerObservedIntervalUtc':['2026-10-09T10:02:39Z','2026-10-09T10:05:18Z'],
 'reviewerExactStartEnd':None,'actualServingModel':None,'usage':None,'cost':None,
 'providerProductTrialTransportStarts':0,'builds':0,'installs':0,'actualSixthAllocation':None,
 'reviewVerdict':'QUALIFIED; reviewed source differs from final source',
 'unrunFinalDelta':'After check8: pipe-drain TimeoutExpired directly records unknown-stop rather than triggering a second cleanup pass. NOT RUN; no remaining test quota; not independently re-reviewed.',
 'implementation':'Thin external reservation and owned-clock/process guard only; raw target invocation/response unchanged; Product Host owns semantics.',
 'unchangedPublicSourceHashes':baseline,
 'finalWrapperSha256':sha(PUBLIC/'inner_runtime_budget.py'),
 'finalTestsSha256':sha(ROOT.parent.parent/'tests/test_inner_runtime_budget.py'),
 'templateExecutionAuthorized':json.loads((PUBLIC/'inner-runtime-budget.example.json').read_text(encoding='utf-8'))['executionAuthorized'],
 'oldSetupSecondsSeparatelyCharged':1385.854584,'studySetup900SecondsPass':False,
 'replacementSourceSha':new['source']['sha'],'replacementActivated':False,
 'remaining':'Owner sealed runtime activation and hosted gates, final exceptional-path validation, separate Root readiness/actual admission, fresh case/common freeze. No product readiness or human acceptance established.',
 'containment':'Only own Windows Job Object/process group. No escaped-provider containment, hard App global enforcement or OS isolation.'}
assert terminal['templateExecutionAuthorized'] is False
write('terminal.json',terminal)
write('retained-file-manifest.json',[{'path':str(p.relative_to(ROOT)).replace('\\','/'),'bytes':p.stat().st_size,'sha256':sha(p)}
     for p in sorted(ROOT.rglob('*')) if p.is_file() and p.name!='retained-file-manifest.json'])
print(json.dumps({'files':len(json.loads((ROOT/'retained-file-manifest.json').read_text(encoding='utf-8'))),'finalWrapperSha256':terminal['finalWrapperSha256'],'finalTestsSha256':terminal['finalTestsSha256'],'closedUtc':terminal['closedUtc']}))
