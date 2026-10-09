"""Bounded ordinary argv correction preparation; zero model or product calls."""
import hashlib, importlib.util, json, os, shutil, subprocess, sys, time
from datetime import datetime, timezone
from pathlib import Path

HERE = Path(__file__).resolve().parent
PACKET = HERE.parents[1]
REPO = PACKET.parents[2]
BASE = '7cd94977274fc6ea86e469ba6b849c3e92f09154'
GRANT = Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/conventional-explicit-workspace-entry-fix-grant-20261009.json')
RESERVATION = GRANT.with_name('conventional-roombook-explicit-workspace-trial-reservation-20261009.json')
DEST = Path('C:/Users/Consiliari/Documents/Luna-Work-Item-Nests-20261009')
EXE = Path('C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe')

def utc(): return datetime.now(timezone.utc).isoformat()
def sha(p): return hashlib.sha256(Path(p).read_bytes()).hexdigest()
def write(name, value): (HERE/name).write_text(json.dumps(value,indent=2,ensure_ascii=False)+'\n',encoding='utf-8',newline='\n')
def git(*args): return subprocess.check_output(['git','-C',str(REPO),*args],timeout=30).decode().strip()

if (HERE/'batch-reservation.json').exists(): raise SystemExit('Already reserved; no retry')
assert git('rev-parse','HEAD') == BASE
assert sha(GRANT) == '53da947ddc53f8b177e39fd3eff21d54ba8864d629aa9d31adb96a3a128f77cf'
assert sha(RESERVATION) == '728482a9c43e44568697a4b20edc0f8dc8a3fd851d922ab178f13fdac84ce431'
grant=json.loads(GRANT.read_text(encoding='utf-8-sig'))
assert datetime.now(timezone.utc) < datetime.fromisoformat(grant['absoluteNotAfterUtc'].replace('Z','+00:00'))
shutil.copyfile(GRANT,HERE/'fix-grant.json'); shutil.copyfile(RESERVATION,HERE/'trial-reservation.json')
changed={'public/nest.py','tests/test_nest.py'}
original={p.relative_to(PACKET).as_posix():sha(p) for p in PACKET.rglob('*') if p.is_file() and HERE not in p.parents and p.relative_to(PACKET).as_posix() not in changed and '__pycache__' not in p.parts}
spec=importlib.util.spec_from_file_location('nest',PACKET/'public/nest.py')
nest=importlib.util.module_from_spec(spec); spec.loader.exec_module(nest)
cell=DEST/'cells/roombook-conventional-explicit-workspace'
state=DEST/'state/roombook-conventional-explicit-workspace'
home=DEST/'homes/roombook-conventional-explicit-workspace'
assert not home.exists(); home.mkdir()
shutil.copyfile(PACKET/'public/native/config.toml',home/'config.toml')
prepared=nest.cell('roombook','conventional',DEST/'seeds/roombook',cell,state)
old=json.loads((PACKET/'evidence/conventional-roombook-current-native-preparation-20261009/bindings.json').read_text(encoding='utf-8-sig'))
files=nest.inventory(cell)
assert files == old['preparedFileHashes']
assert prepared['preparedTree'] == old['preparedTree'] == '8cea62c20a64d4571017818c7e73855b1ead5676'
assert prepared['seedCommit'] == old['seedCommit'] == '71acaae99c826d0dce0ade81a5a84ab0aee6cfef'
assert sha(EXE) == '3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68'
env=os.environ.copy(); env['CODEX_HOME']=str(home); env['PYTHONDONTWRITEBYTECODE']='1'
argv=[str(EXE),'exec','resume','--help']; start=time.monotonic(); began=utc()
meta=subprocess.run(argv,env=env,cwd=cell,capture_output=True,timeout=15)
(HERE/'resume-help.stdout.txt').write_bytes(meta.stdout); (HERE/'resume-help.stderr.txt').write_bytes(meta.stderr)
write('metadata-receipt.json',{'argv':argv,'startedUtc':began,'endedUtc':utc(),'wallSeconds':time.monotonic()-start,'exitCode':meta.returncode,'modelCalls':0})
assert meta.returncode == 0
assert b'Usage:' in meta.stdout and b'resume' in meta.stdout
write('batch-reservation.json',{'grantId':grant['grantId'],'reservedUtc':utc(),'batch':1,'maxWallSeconds':180,'scope':'Existing eight fixtures plus one initial/resume/assessment argv regression','sourceHashes':{p:sha(PACKET/p) for p in changed},'modelCalls':0,'reviewerCalls':0,'actualStarts':0})
argv=[sys.executable,'-B','-m','unittest','discover','-s',str(PACKET/'tests'),'-p','test_nest.py','-v']
start=time.monotonic(); began=utc()
with (HERE/'mechanical.stdout.txt').open('wb') as out,(HERE/'mechanical.stderr.txt').open('wb') as err:
 proc=subprocess.Popen(argv,cwd=REPO,env=env,stdout=out,stderr=err,creationflags=subprocess.CREATE_NEW_PROCESS_GROUP if os.name=='nt' else 0,start_new_session=os.name!='nt')
 stop=None
 try: proc.wait(timeout=150)
 except subprocess.TimeoutExpired: stop=nest.stop_owned(proc)
elapsed=time.monotonic()-start
unchanged=all(sha(PACKET/p)==h for p,h in original.items())
passed=proc.returncode==0 and stop is None and unchanged and elapsed<=180
write('mechanical-receipt.json',{'status':'PASS' if passed else 'FAIL','argv':argv,'pid':proc.pid,'startedUtc':began,'endedUtc':utc(),'wallSeconds':elapsed,'exitCode':proc.returncode,'stop':stop,'batch':1,'fixtures':9,'originalFilesUnchangedExceptTwoAllowedFiles':unchanged,'originalUnchangedFileCount':len(original),'modelCalls':0,'providerCalls':0,'reviewerCalls':0,'actualStarts':0})
assert passed, 'Terminal mechanical failure; no retry'
bindings={'grantId':grant['grantId'],'sourceBasis':BASE,'runnerSha256':sha(PACKET/'public/nest.py'),'testSha256':sha(PACKET/'tests/test_nest.py'),'repo':str(cell),'stateDir':str(state),'nativeHome':str(home),'preparedCommit':prepared['preparedCommit'],'preparedTree':prepared['preparedTree'],'seedCommit':prepared['seedCommit'],'mainCommit':nest.git(cell,'rev-parse','main'),'dirtyStatus':nest.git(cell,'status','--porcelain'),'preparedFileHashes':files,'sameNinePreparedInputBytes':True,'nativeExecutable':str(EXE),'nativeExecutableSha256':sha(EXE),'nativeVersionBasis':'Previously pinned 0.162.0-alpha.2; unchanged executable hash; no extra version call','nativeHomeConfigSha256':sha(home/'config.toml'),'projectConfigSha256':sha(cell/'.codex/config.toml'),'model':'gpt-6-luna','reasoning':'high','requestedSandbox':'workspace-write','approvalPolicy':'never','maxThreads':4,'maxDepth':2,'nativeHomeEntries':[p.name for p in sorted(home.iterdir())],'accountMaterialProvisioned':False,'effectiveRuntimeSandbox':'NOT RUN; prospective explicit selector only','actualStarts':0,'entries':{name:nest.command(EXE,state/'evaluation/repo' if assessment else cell,HERE/'last-message.txt',session,assessment) for name,session,assessment in [('initial',None,False),('resume','EXPLICIT_OWN_SESSION_ID',False),('assessment',None,True)]}}
write('bindings.json',bindings)
proposal=json.loads((PACKET/'evidence/conventional-roombook-current-native-preparation-20261009/activation-proposal.json').read_text(encoding='utf-8-sig'))
proposal.update(grantId='conventional-roombook-explicit-workspace-trial-20261009',executionAuthorized=False,preparedCommit=prepared['preparedCommit'],nativeHome=str(home),nativeHomeConfigSha256=bindings['nativeHomeConfigSha256'],sourceBasis=BASE,runnerSha256=bindings['runnerSha256'],normalEntry='Corrected nest.py drive with explicit normal --sandbox workspace-write; Root exact activation required',nativeConfigurationReviewed=False,freshContextConfirmed=False)
write('activation-proposal.json',proposal)
write('original-file-preservation.json',{'scope':'All preexisting packet working bytes except public/nest.py and tests/test_nest.py; no credential inventory','sha256':original,'unchanged':unchanged})
write('continuation-anchor.json',{'status':'PREPARED / actual disabled / await Overseer activation','grantId':grant['grantId'],'newActualStarts':0,'oldGlobalStarts':2,'mechanicalBatches':1,'newModelActors':0,'newReviewerActors':0,'sourceBasis':BASE,'absoluteActualEndUtc':'2026-10-09T09:12:43Z','mustHaveFull7200SecondsBeforeStart':True,'completionCallback':'pending once after private commit/push'})
print(json.dumps({'status':'PASS','mechanicalSeconds':elapsed,'newPreparedCommit':prepared['preparedCommit'],'sameTree':prepared['preparedTree'],'inputPaths':len(files),'modelCalls':0,'actualStarts':0},indent=2))
