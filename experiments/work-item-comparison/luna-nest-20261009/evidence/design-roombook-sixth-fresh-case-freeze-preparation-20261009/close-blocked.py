"""Readonly case capture after consumed correction; no further case/state edits."""
from pathlib import Path
import json,hashlib,shutil,subprocess
from datetime import datetime,timezone

E=Path(__file__).resolve().parent
ROOT=E.parent.parent/'public'
repo=Path('C:/Users/Consiliari/Documents/Luna-Work-Item-Nests-20261009/cells/roombook-design-app-native')
state=Path('C:/Users/Consiliari/Documents/Luna-Work-Item-Nests-20261009/state/roombook-design-app-native')
def sha(p):return hashlib.sha256(Path(p).read_bytes()).hexdigest()
def write(name,v):(E/name).write_text(json.dumps(v,ensure_ascii=True,indent=2)+'\n',encoding='utf-8')
def command(*args):
    p=subprocess.run(['git','-C',str(repo),*args],capture_output=True,timeout=45)
    return {'argv':['git','-C',str(repo),*args],'exitCode':p.returncode,'stdout':p.stdout.decode('utf-8'),'stderr':p.stderr.decode('utf-8')}

cached=command('diff','--cached','--check');write('blocked-cached-check.json',cached)
status=command('status','--porcelain');write('blocked-case-status.json',status)
head=command('rev-parse','HEAD')['stdout'].strip();tree=command('rev-parse','HEAD^{tree}')['stdout'].strip()
assert head=='71acaae99c826d0dce0ade81a5a84ab0aee6cfef'
for name,args in [('staged-case.patch',['diff','--cached','--binary']),('unstaged-case.patch',['diff','--binary'])]:
    p=subprocess.run(['git','-C',str(repo),*args],capture_output=True,timeout=45);assert p.returncode==0;(E/name).write_bytes(p.stdout)
subprocess.run(['git','-C',str(repo),'bundle','create',str(E/'partial-case-seed.bundle'),'--all'],check=True,capture_output=True,timeout=45)
case=[]
for p in sorted(repo.rglob('*')):
    if p.is_file() and '.git' not in p.relative_to(repo).parts:
        relative=p.relative_to(repo);target=E/'partial-case-working-files'/relative;target.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(p,target)
        assert sha(p)==sha(target);case.append({'path':relative.as_posix(),'sha256':sha(p),'bytes':p.stat().st_size})
assets=[]
for p in sorted(state.rglob('*')):
    if p.is_file():
        relative=p.relative_to(state);target=E/'partial-state-files'/relative;target.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(p,target)
        assert sha(p)==sha(target);assets.append({'path':relative.as_posix(),'sha256':sha(p),'bytes':p.stat().st_size})
binding=json.loads((state/'inner-binding.json').read_text(encoding='utf-8'))
assert binding['executionAuthorized'] is False and binding['parentHandle'] is None and binding['parentReservation'] is None
assert not (state/'state.json').exists() and not (state/'app-ledger.json').exists()
assert not (repo/'app.py').exists() and not (repo/'tests').exists()
models=json.loads((E.parent/'matched-design-inner-role-accounting-binding-20261009/private-owner-installation/handoff.json').read_text(encoding='utf-8-sig'))
for row in models['model']['installedModel']:assert sha(repo/row['installedPath'])==row['installedSha256']
write('model-migration-provenance.json',{'originalTen':models['model']['installedModel'],'ownerMigration':models['model']['migration'],'commonAgentsPrefixPreserved':(repo/'AGENTS.md').read_bytes().startswith((ROOT/'common/AGENTS.md').read_bytes())})
write('partial-case-inventory.json',case);write('partial-state-inventory.json',assets)
grant=json.loads((E/'design-roombook-sixth-fresh-case-freeze-preparation-20261009-grant.json').read_text(encoding='utf-8'))
now=datetime.now(timezone.utc)
write('blocked-terminal.json',{'status':'BLOCKED before prepared commit; no actual admission','closedUtc':now.isoformat(),
 'prepCostSecondsFromIssuedThroughClosure':(now-datetime.fromisoformat(grant['issuedUtc'].replace('Z','+00:00'))).total_seconds(),
 'notAfterUtc':grant['notAfterUtc'],'partialRepo':str(repo),'partialState':str(state),'caseHead':head,'caseHeadTree':tree,
 'preparedCommit':None,'preparedTree':None,'initialStation':json.loads((repo/'.study/station.json').read_text(encoding='utf-8'))['id'],
 'stateJson':'NOT CREATED','appLedger':'NOT CREATED; no entries or actual reservation created','runtimeBindingAuthorized':False,
 'parentHandle':None,'parentReservation':None,'originalSeed':grant['originalSeed'],'conventionalPreparedTreeReference':grant['conventionalPreparedTreeReference'],
 'topLevelLocalCommandBatchesIncludingClosurePlanned':8,'setupBatchesAttempted':2,'setupCorrectionBatchesConsumed':1,
 'metadataGitSubcommands':'Multiple Git read/config/clone/stage operations inside bounded batches; not eight individual executables. Exact successful correction command sequence is in prepare.py; first source pin unavailable.',
 'providerProductTransportActualCalls':0,'modelsReviewersTestsBuilds':0,'productSetupConfigCliInvocations':0,
 'blocker':cached,'knownOwnJobs':'All own Git subprocesses terminal. No actor/provider/product jobs created; no global inventory.',
 'oldSetupSecondsSeparatelyRetained':1385.854584,'servingModel':None,'tokens':None,'cost':None,'humanTime':None,
 'limitations':['No second correction performed, no whitespace suppression or commit bypass.','Normal case commit check failed; immutable prepared case and empty initialized ledger requirements unmet.','Resource runtime/YAML compatibility unexecuted; final pipe-drain path NOT RUN.','Root must decide disposition; no automatic grant/refill/retry.']})
write('retained-file-manifest.json',[{'path':p.relative_to(E).as_posix(),'sha256':sha(p),'bytes':p.stat().st_size} for p in sorted(E.rglob('*')) if p.is_file() and p.name!='retained-file-manifest.json'])
print(json.dumps({'caseHead':head,'cachedCheckExit':cached['exitCode'],'cachedCheckOutput':cached['stdout'],'files':len(case),'closedUtc':now.isoformat()}))
