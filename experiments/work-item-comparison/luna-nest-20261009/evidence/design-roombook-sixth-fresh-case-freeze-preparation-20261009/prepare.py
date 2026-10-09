"""One bounded fresh file/Git configuration batch. No models/tests/product calls."""
from pathlib import Path
import hashlib,json,shutil,subprocess,sys,time
from datetime import datetime,timezone

E=Path(__file__).resolve().parent
PUBLIC=E.parent.parent/'public'
sys.path.insert(0,str(PUBLIC))
import nest,app_ledger
started=datetime.now(timezone.utc); clock=time.monotonic(); commands=[]
def sha(p): return hashlib.sha256(Path(p).read_bytes()).hexdigest()
def write(p,v):
    p=Path(p);p.parent.mkdir(parents=True,exist_ok=True)
    p.write_text(json.dumps(v,ensure_ascii=True,indent=2)+'\n',encoding='utf-8')
def git(repo,*args):
    argv=['git','-C',str(repo),*args];commands.append(argv)
    return subprocess.check_output(argv,timeout=45).decode('utf-8').strip()
def copy(src,dst):
    dst=Path(dst);dst.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(src,dst)
    assert sha(src)==sha(dst)
def inventory(root):
    return [{'path':p.relative_to(root).as_posix(),'sha256':sha(p),'bytes':p.stat().st_size}
            for p in sorted(root.rglob('*')) if p.is_file() and '.git' not in p.relative_to(root).parts]

grants=Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government')
prep=nest.read(grants/'design-roombook-sixth-fresh-case-freeze-preparation-20261009-grant.json')
reservation=nest.read(grants/'design-roombook-sixth-actual-reservation-20261009-grant.json')
assert datetime.now(timezone.utc)<app_ledger.date(prep['notAfterUtc'])
assert not prep['executionAuthorized'] and not reservation['executionAuthorized']
for name in ['design-roombook-sixth-fresh-case-freeze-preparation-20261009-grant.json','design-roombook-sixth-actual-reservation-20261009-grant.json']:
    copy(grants/name,E/name)
repo=Path(prep['freshRepo']);state=Path(prep['freshState'])
correction=sys.argv[1:]==['--correction-once']
assert not state.exists(),'State must be fresh; no ledger reset/reuse'
assert (repo.exists() if correction else not repo.exists()),'Only exact own partial preparation may be corrected once'
seed=Path('C:/Users/Consiliari/Documents/Luna-Work-Item-Nests-20261009/seeds/roombook')
assert git(seed,'rev-parse','HEAD')==prep['originalSeed']
assert not git(seed,'status','--porcelain')
seed_inventory=inventory(seed)
if correction:
    assert git(repo,'rev-parse','HEAD')==prep['originalSeed']
    assert git(repo,'branch','--show-current')=='codex/design-roombook-prepared'
    assert not git(repo,'remote')
else:
    argv=['git','-c','core.autocrlf=false','clone','--no-hardlinks',str(seed),str(repo)];commands.append(argv)
    subprocess.run(argv,check=True,capture_output=True,timeout=60)
    git(repo,'remote','remove','origin');git(repo,'config','core.autocrlf','false')
    assert git(repo,'branch','--show-current')=='main'
    git(repo,'checkout','-b','codex/design-roombook-prepared')
normalized=[]
for row in seed_inventory:
    source=seed/row['path'];target=repo/row['path']
    if sha(target)!=row['sha256']:
        assert correction and source.read_bytes().replace(b'\r\n',b'\n')==target.read_bytes().replace(b'\r\n',b'\n'),row['path']
        normalized.append(row['path']);copy(source,target)
    assert sha(target)==row['sha256']
copy(PUBLIC/'native/config.toml',repo/'.codex/config.toml')
stations=nest.read(repo/'STATIONS.json')['stations'];nest.write(repo/'.study/station.json',stations[0])
common=inventory(repo);assert len(common)==9,common
write(E/'common-nine-inputs.json',common)

previous=E.parent/'matched-design-inner-role-accounting-binding-20261009'
old=previous/'private-owner-installation';new=previous/'private-owner-replacement'
owner=nest.read(old/'handoff.json');replacement=nest.read(new/'handoff.json')
installed=Path(owner['installation']['repo'])
assert replacement['source']['sha']==prep['productSource']
assert sha(new/'runtime-assets/runner.py')=='1284ea4e187cf68ab8a9c15062366cadc750d63a9deb5b833214ec05bcc56ba6'
assert sha(new/'markitect-bcd614a3.exe')=='abf877849379900cf2f1cf6c19f3d5d1e67a134592ac810cedc437191e9a5f29'
model_rows=[]
for row in owner['model']['installedModel']:
    original=PUBLIC/row['publicPath'];source=installed/row['installedPath'];target=repo/row['installedPath']
    assert sha(original)==row['originalSha256'] and sha(source)==row['installedSha256']
    copy(source,target);copy(original,E/'original-model'/row['installedPath'])
    model_rows.append(row)
assert len(model_rows)==10
for row in owner['installation']['files']:
    name=row['path']
    if name in ['AGENTS.md','.markitect/runtime.yaml'] or any(name==r['installedPath'] for r in model_rows): continue
    assert sha(installed/name)==row['sha256'];copy(installed/name,repo/name)
common_agents=(repo/'AGENTS.md').read_bytes();owner_agents=(installed/'AGENTS.md').read_bytes()
assert sha(installed/'AGENTS.md')==next(r['sha256'] for r in owner['installation']['files'] if r['path']=='AGENTS.md')
(repo/'AGENTS.md').write_bytes(common_agents+b'\n'+owner_agents)
assert (repo/'AGENTS.md').read_bytes().startswith(common_agents)
assets=state/'runtime-assets';assets.mkdir(parents=True)
for name in ['inner_runtime_budget.py','app_ledger.py','nest.py']: copy(PUBLIC/name,assets/name)
copy(new/'runtime-assets/runner.py',assets/'runner.py');copy(new/'markitect-bcd614a3.exe',assets/'markitect.exe')
assert sha(assets/'inner_runtime_budget.py')==reservation['wrapperSha256']
python=Path('C:/Python313/python.exe')
native=Path('C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe')
assert sha(python)=='d87063e5597f257004c731b66c59c56c91038861c6877b1a3dca6b8c4e919125'
assert sha(native)=='3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68'
command=[str(python),str(assets/'runner.py'),'--model','gpt-6-luna','--codex-executable',str(native),'--codex-version','0.162.0-alpha.2']
pins=[{'path':str(p),'sha256':sha(p)} for p in [python,assets/'runner.py',native,assets/'inner_runtime_budget.py',assets/'app_ledger.py',assets/'nest.py']]
binding={'executionAuthorized':False,'stateDir':str(state),'grantPath':str(grants/'design-roombook-sixth-actual-reservation-20261009-grant.json'),
         'grantSha256':sha(grants/'design-roombook-sixth-actual-reservation-20261009-grant.json'),'phase':'implementation',
         'parentReservation':None,'parentHandle':None,'maxOwnedWallSeconds':300,'cleanupReserveSeconds':10,
         'command':command,'runtimePins':pins,'activation':'Root enabled grant and actual recorded parent required; no fabricated handle'}
binding_path=state/'inner-binding.json';write(binding_path,binding)
base=json.loads((old/'runtime-setup-preview.json').read_text(encoding='utf-8-sig'))
# Exact owner replacement runtime is the template; only resource prefix/paths/pins change.
import re
text=(new/'runtime-replacement.yaml').read_text(encoding='utf-8')
text=text.replace(str(Path(replacement['runtime']['adapterPath'])),str(assets/'runner.py'))
text=text.replace(str(Path(replacement['runtime']['adapterPath'])).replace('\\','/'),str(assets/'runner.py'))
args_prefix=['-B',str(assets/'inner_runtime_budget.py'),'--binding',str(binding_path),'--binding-sha256',sha(binding_path),'--',str(python)]
lines=text.splitlines();result=[]
for line in lines:
    if line.strip()=='args:':
        result.append(line);indent=' '*(len(line)-len(line.lstrip())+4)
        result.extend(indent+'- '+json.dumps(a) for a in args_prefix)
    elif line.strip()=='runtimeFiles:':
        result.append(line);indent=' '*(len(line)-len(line.lstrip())+4)
        for p in [assets/'inner_runtime_budget.py',assets/'app_ledger.py',assets/'nest.py']:
            result += [indent+'- path: '+json.dumps(str(p)),indent+'  mode: "0644"',indent+'  digest: sha256:'+sha(p)]
    else: result.append(line)
runtime='\n'.join(result)+'\n'
assert runtime.count(str(assets/'runner.py'))==4 and '0ceb15e6' not in runtime
assert runtime.count('--binding-sha256')==2 and runtime.count(sha(binding_path))==2
assert runtime.count('1284ea4e187cf68ab8a9c15062366cadc750d63a9deb5b833214ec05bcc56ba6')==2
runtime_path=repo/'.markitect/runtime.yaml';runtime_path.write_text(runtime,encoding='utf-8')
resource_notes=f'''# Declared local tool and resource bindings

Genuine source-bound Markitect executable: `{assets / 'markitect.exe'}`.
Selected model and normal native instructions remain in this repository.
Product runtime is `.markitect/runtime.yaml`; raw genuine adapter argv and inherited profile remain owned by the product.
This preparation is disabled. Do not invoke product runtime until Root has recorded your actual outer handle in the own ledger and sealed the matching enabled resource binding.
Own resource state: `{state}`. No semantic help or private evaluation is supplied by resource orchestration.
All outer, inner, helper and final roles share 72 starts, four active roles and depth two; station/phase deadlines never extend.
Serving identity, tokens and cost remain unknown without actual receipts.
'''
(repo/'TOOLCHAIN.md').write_text(resource_notes,encoding='utf-8')
for row in common:
    if row['path']!='AGENTS.md': assert sha(repo/row['path'])==row['sha256']
assert not (repo/'app.py').exists() and not (repo/'tests').exists()
git(repo,'add','.')
# Required immediate cached check before every commit.
git(repo,'diff','--cached','--check')
git(repo,'-c','user.name=Study Seed','-c','user.email=study@example.invalid','commit','-m','Prepare original Roombook model and disabled genuine resource runtime')
prepared=git(repo,'rev-parse','HEAD');tree=git(repo,'rev-parse','HEAD^{tree}')
git(repo,'branch','-f','main',prepared);git(repo,'checkout','main')
assert not git(repo,'status','--porcelain')
record={'case':'roombook','arm':'markitect','repo':str(repo),'stateDir':str(state),'seedCommit':prep['originalSeed'],
        'preparedCommit':prepared,'preparedTree':tree,'createdUtc':nest.utc(),'status':'prepared','sessions':[],
        'interventions':[],'productReadiness':None,'initialSetup':[],'trialGrant':None,'sessionId':None,
        'deadlineUtc':None,'implementationDeadlineUtc':None,'evaluationDeadlineUtc':None,'tokens':None,'cost':None,
        'servingModel':None,'stationIndex':0,'stations':stations,'snapshots':[],'freeze':None}
nest.write(state/'state.json',record);app_ledger.initialize(state)
assert not nest.read(state/'app-ledger.json')['entries']
copy(state/'state.json',E/'initial-state.json');copy(state/'app-ledger.json',E/'initial-app-ledger.json')
copy(binding_path,E/'disabled-inner-binding.json');copy(runtime_path,E/'disabled-runtime.yaml')
for p in sorted(repo.rglob('*')):
    if p.is_file() and '.git' not in p.relative_to(repo).parts: copy(p,E/'prepared-case-files'/p.relative_to(repo))
bundle=E/'prepared-case.bundle';git(repo,'bundle','create',str(bundle),'--all')
write(E/'model-migration-provenance.json',{'originalTen':model_rows,'ownerMigration':owner['model']['migration'],'commonAgentsPrefixSha256':hashlib.sha256(common_agents).hexdigest(),'appendedOwnerAgentsSha256':hashlib.sha256(owner_agents).hexdigest()})
write(E/'runtime-inventory.json',{'productSource':replacement['source']['sha'],'binary':{'path':str(assets/'markitect.exe'),'sha256':sha(assets/'markitect.exe')},'pins':pins,'bindingSha256':sha(binding_path),'runtimeSha256':sha(runtime_path),'executionAuthorized':False})
write(E/'setup-terminal.json',{'status':'fresh clean prepared main; actual disabled','startedUtc':started.isoformat(),'endedUtc':nest.utc(),
 'freshSetupSeconds':(datetime.now(timezone.utc)-app_ledger.date(prep['issuedUtc'])).total_seconds(),'successfulBatchSeconds':time.monotonic()-clock,'notAfterUtc':prep['notAfterUtc'],'sourceBaseline':prep['source'],'preparedCommit':prepared,'preparedTree':tree,
 'initialStation':stations[0]['id'],'ledgerEntries':0,'originalSeed':prep['originalSeed'],'conventionalPreparedTreeReference':prep['conventionalPreparedTreeReference'],
 'commonNineCount':len(common),'commonAgents':'Original bytes preserved as prefix; exact genuine owner instructions appended.',
 'topLevelLocalCommandBatchesIncludingClosurePlanned':8,'preparationBatchGitSubcommands':commands,'productSetupOrConfigCliInvocations':0,
 'configurationMethod':'Fresh file configuration from exact owner package with explicit byte/hash guards; no product CLI executed.',
 'setupCorrectionBatches':int(correction),'restoredSeedLineEndingsOnly':normalized,'modelActorsReviewersTestsBuildsProviderTransport':0,'knownOwnJobs':'All subprocess Git metadata/file setup jobs completed; no other own jobs created.',
 'oldSetupSecondsSeparatelyRetained':1385.854584,'servingModel':None,'tokens':None,'cost':None,'humanTime':None,
 'limitations':['No runtime/YAML compatibility execution or provider test; final pipe-drain path remains NOT RUN.','Owner native full delivery remains unproved; this is exploratory preparation only.','No actual handle/reservation or enabled grant fabricated; Root activation must gate outer actor before any inner call.']})
write(E/'retained-file-manifest.json',[{'path':p.relative_to(E).as_posix(),'sha256':sha(p),'bytes':p.stat().st_size}
 for p in sorted(E.rglob('*')) if p.is_file() and p.name!='retained-file-manifest.json'])
print(json.dumps({'preparedCommit':prepared,'preparedTree':tree,'freshSetupSeconds':time.monotonic()-clock,'files':len(inventory(repo)),'runtimeSha256':sha(runtime_path),'bindingSha256':sha(binding_path)}))
