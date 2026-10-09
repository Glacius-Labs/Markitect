"""One explicitly granted formatting batch; no product/model/test execution."""
from pathlib import Path
import hashlib,json,shutil,subprocess,sys,difflib
from datetime import datetime,timezone

E=Path(__file__).resolve().parent;PUBLIC=E.parent.parent/'public'
sys.path.insert(0,str(PUBLIC));import nest,app_ledger
prior=E.parent/'design-roombook-sixth-fresh-case-freeze-preparation-20261009'
issued=Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/design-roombook-sixth-format-only-preparation-correction-20261009-grant.json')
calls=[]
def sha(p):return hashlib.sha256(Path(p).read_bytes()).hexdigest()
def write(name,value):
    p=E/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_bytes((json.dumps(value,ensure_ascii=True,indent=2)+'\n').encode('utf-8'))
def copy(src,dst):
    dst=Path(dst);dst.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(src,dst);assert sha(src)==sha(dst)
def git(repo,*args):
    argv=['git','-C',str(repo),*args];p=subprocess.run(argv,capture_output=True,timeout=45)
    calls.append({'argv':argv,'exitCode':p.returncode,'stdout':p.stdout.decode('utf-8'),'stderr':p.stderr.decode('utf-8')})
    if p.returncode:raise RuntimeError('Next failure; no further correction: '+json.dumps(calls[-1]))
    return p.stdout.decode('utf-8').strip()
def inventory(root):
    return [{'path':p.relative_to(root).as_posix(),'sha256':sha(p),'bytes':p.stat().st_size}
            for p in sorted(root.rglob('*')) if p.is_file() and '.git' not in p.relative_to(root).parts]
def normalized(raw):
    return [line.rstrip(b' \t') for line in raw.replace(b'\r\n',b'\n').split(b'\n')]
def meaningful(raw):
    lines=normalized(raw)
    while lines and lines[-1]==b'':lines.pop()
    return lines
started=datetime.now(timezone.utc);copy(issued,E/'grant.json')
assert sha(issued)=='48c1d7f93bafb131c5ced1d4db6ab7176aac93e6016443ba38bd0686c687c7dc'
grant=nest.read(issued);repo=Path(grant['caseRepo']);state=Path(grant['stateDirectory'])
assert started<app_ledger.date(grant['notAfterUtc']) and grant['executionAuthorized'] is False
assert git(repo,'rev-parse','HEAD')=='71acaae99c826d0dce0ade81a5a84ab0aee6cfef'
assert git(repo,'branch','--show-current')=='codex/design-roombook-prepared'
assert not (state/'state.json').exists() and not (state/'app-ledger.json').exists()
common=nest.read(prior/'common-nine-inputs.json');original_agents=(PUBLIC/'common/AGENTS.md').read_bytes()
assert hashlib.sha256(original_agents).hexdigest()==next(r['sha256'] for r in common if r['path']=='AGENTS.md')
for row in common:
    if row['path']=='AGENTS.md':assert (repo/row['path']).read_bytes().startswith(original_agents)
    else:assert sha(repo/row['path'])==row['sha256'],row['path']
models=nest.read(prior/'model-migration-provenance.json')
for row in models['originalTen']:assert sha(repo/row['installedPath'])==row['installedSha256']
binding=nest.read(state/'inner-binding.json')
assert binding['executionAuthorized'] is False and binding['parentHandle'] is None and binding['parentReservation'] is None
for pin in binding['runtimePins']:assert sha(pin['path'])==pin['sha256']
files=['.markitect/runtime.yaml','TOOLCHAIN.md','.agents/skills/markitect-model-first/SKILL.md',
       '.claude/skills/markitect-model-first/SKILL.md','CLAUDE.md','docs/markitect/project.md','AGENTS.md']
before={name:(repo/name).read_bytes() for name in files}
for name,raw in before.items():
    assert raw==(prior/'partial-case-working-files'/name).read_bytes(),name
    copy(repo/name,E/'before'/name)
changes=[]
for name,raw in before.items():
    if name in ['.markitect/runtime.yaml','TOOLCHAIN.md']:
        result=raw.replace(b'\r\n',b'\n')
    elif name=='AGENTS.md':
        assert raw.startswith(original_agents)
        result=original_agents+b'\n'.join(meaningful(raw[len(original_agents):]))+b'\n'
        assert result.startswith(original_agents)
    else:result=b'\n'.join(meaningful(raw))+b'\n'
    assert meaningful(raw)==meaningful(result),name
    (repo/name).write_bytes(result);copy(repo/name,E/'after'/name)
    changes.append({'path':name,'beforeSha256':hashlib.sha256(raw).hexdigest(),'afterSha256':sha(repo/name),
                    'nonWhitespaceContentPreserved':True,'commonPrefixPreserved':name!='AGENTS.md' or result.startswith(original_agents)})
    delta=''.join(difflib.unified_diff(raw.decode('utf-8').splitlines(keepends=True),result.decode('utf-8').splitlines(keepends=True),fromfile='before/'+name,tofile='after/'+name))
    p=E/'format-deltas'/Path(name+'.diff');p.parent.mkdir(parents=True,exist_ok=True);p.write_bytes(delta.encode('utf-8'))
write('format-provenance.json',changes)
for row in models['originalTen']:assert sha(repo/row['installedPath'])==row['installedSha256']
for row in common:
    if row['path']!='AGENTS.md':assert sha(repo/row['path'])==row['sha256']
assert not (repo/'app.py').exists() and not (repo/'tests').exists()
try:
    git(repo,'add','.')
    # Required immediate gate; no command between this check and commit.
    git(repo,'diff','--cached','--check')
    git(repo,'-c','user.name=Study Seed','-c','user.email=study@example.invalid','commit','-m','Prepare original Roombook model and disabled genuine runtime with format-only disposition')
    prepared=git(repo,'rev-parse','HEAD');tree=git(repo,'rev-parse','HEAD^{tree}')
    git(repo,'branch','-f','main',prepared);git(repo,'checkout','main')
    assert not git(repo,'status','--porcelain')
    stations=nest.read(repo/'STATIONS.json')['stations'];assert nest.read(repo/'.study/station.json')==stations[0]
    record={'case':'roombook','arm':'markitect','repo':str(repo),'stateDir':str(state),'seedCommit':'71acaae99c826d0dce0ade81a5a84ab0aee6cfef',
       'preparedCommit':prepared,'preparedTree':tree,'createdUtc':nest.utc(),'status':'prepared','sessions':[],'interventions':[],
       'productReadiness':None,'initialSetup':[],'trialGrant':None,'sessionId':None,'deadlineUtc':None,
       'implementationDeadlineUtc':None,'evaluationDeadlineUtc':None,'tokens':None,'cost':None,'servingModel':None,
       'stationIndex':0,'stations':stations,'snapshots':[],'freeze':None}
    nest.write(state/'state.json',record);app_ledger.initialize(state)
    assert nest.read(state/'app-ledger.json')['entries']==[] and nest.read(state/'app-ledger.json')['startedUtc'] is None
    case_inventory=inventory(repo)
    for row in case_inventory:
        raw=subprocess.check_output(['git','-C',str(repo),'show','HEAD:'+row['path']],timeout=45)
        assert hashlib.sha256(raw).hexdigest()==row['sha256'],row['path']
        copy(repo/row['path'],E/'prepared-case-files'/row['path'])
    for p in sorted(state.rglob('*')):
        if p.is_file():copy(p,E/'initial-state-files'/p.relative_to(state))
    copy(prior/'activation-plan.md',E/'activation-plan.md');copy(prior/'common-nine-inputs.json',E/'common-nine-inputs.json')
    copy(prior/'model-migration-provenance.json',E/'model-migration-provenance.json')
    write('prepared-case-inventory.json',case_inventory);write('initial-state-inventory.json',inventory(state))
    git(repo,'bundle','create',str(E/'prepared-case.bundle'),'--all')
    terminal={'status':'clean prepared main; DISABLED pending separate Root admission','startedUtc':started.isoformat(),'closedUtc':nest.utc(),
      'grantEndUtc':grant['notAfterUtc'],'sourceBaseline':grant['sourceBaseline'],'preparedCommit':prepared,'preparedTree':tree,
      'repo':str(repo),'stateDir':str(state),'station':'S1','stationIndex':0,'ledgerEntries':0,'ledgerStartedUtc':None,
      'bindingAuthorized':False,'parentHandle':None,'parentReservation':None,'grantPathInDisabledBinding':binding['grantPath'],
      'bindingSha256':sha(state/'inner-binding.json'),'runtimeSha256':sha(repo/'.markitect/runtime.yaml'),
      'wrapperSha256':sha(state/'runtime-assets/inner_runtime_budget.py'),'adapterSha256':sha(state/'runtime-assets/runner.py'),
      'binarySha256':sha(state/'runtime-assets/markitect.exe'),'requestedModel':'gpt-6-luna','reasoning':'high',
      'servingModel':None,'tokens':None,'cost':None,'humanTime':None,'actorsReviewersTestsBuildsProductProviderTransportCalls':0,
      'mechanicalCorrectionBatches':1,'topLevelLocalBatchesIncludingClosurePlanned':4,
      'rawGitWorkingCaseBindings':len(case_inventory),'nestedGitSubcommands':calls,
      'rawGitCaseBlobReads':len(case_inventory),'knownOwnJobs':'All Git subprocesses terminal; no actors/product/provider jobs created.',
      'priorPreparationCostSecondsThroughCallback':635,'oldOwnerSetupSecondsSeparatelyRetained':1385.854584,
      'freshFormatCostSecondsThroughFreeze':(datetime.now(timezone.utc)-app_ledger.date(grant['issuedUtc'])).total_seconds(),
      'limits':['Format-only genuine generated-view delta retained; ten canonical model bytes untouched.','No runtime YAML/adapter compatibility product execution or tests.','Final wrapper pipe-drain exceptional path remains NOT RUN/not exact-final independent review.','No native full-delivery readiness/main/human acceptance established.','Separate immutable Root enabled grant/full7200s and actual own parent binding required.','Cooperative shared72/four/depth2, original5400/1200/600; no OS/hard-global/escaped-provider containment.']}
except BaseException as error:
    write('terminal-next-failure.json',{'status':'STOPPED on next failure; no further correction','error':str(error),'utc':nest.utc(),'commands':calls})
    raise
write('terminal.json',terminal)
write('retained-file-manifest.json',[{'path':p.relative_to(E).as_posix(),'bytes':p.stat().st_size,'sha256':sha(p)} for p in sorted(E.rglob('*')) if p.is_file() and p.name!='retained-file-manifest.json'])
print(json.dumps({'preparedCommit':prepared,'preparedTree':tree,'caseBindings':len(case_inventory),'runtimeSha256':terminal['runtimeSha256'],'bindingSha256':terminal['bindingSha256'],'closedUtc':terminal['closedUtc']}))
