"""One-shot evidence closure and separate authorized posthoc input preparation."""
import datetime as dt
import hashlib
import json
import pathlib
import shutil
import subprocess
import zipfile

ROOT = pathlib.Path(__file__).resolve().parent
WORK = ROOT.parents[3]
BASE = pathlib.Path('C:/Users/Consiliari/Documents/Scientist-Native-v3-20261008')
SUP = ROOT.parent / 'native-v3-conventional-gf-posthoc-validation-20261009'
EXT = BASE / 'posthoc-20261009'
REPO = BASE / 'study/cells/conventional-greenfield'
CANDIDATE = 'd1e271b8472cc55565696b976a87ff968a4e8c7a'

def sha(p): return hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
def write(p, value):
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(value, indent=2, ensure_ascii=False)+'\n', encoding='utf8')
def git(*args, repo=REPO): return subprocess.check_output(['git', '-C', str(repo), *args], text=True).strip()

assert git('rev-parse','HEAD') == CANDIDATE and not git('status','--porcelain')
assert not EXT.exists() and not SUP.exists(), 'one-shot only'
SUP.mkdir(); EXT.mkdir()
coord = pathlib.Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/coordination-state.json')
grant = next(x for x in json.loads(coord.read_text(encoding='utf-8-sig'))['threads'] if x['name']=='Scientist')['evidence']['nativeV3PosthocValidationGrant']
write(SUP/'authorization-grant.json', grant)
now = dt.datetime.now(dt.timezone.utc).isoformat()
requests = json.loads((ROOT/'native-spawn-requests.json').read_text())
assert not any(x['name'].endswith('followup_task') and 'public_review_5' in x['arguments'].get('target','') for x in requests)
state = BASE/'support/conventional-greenfield/state'
assert not list(state.glob('*re-review*'))
ledger = json.loads((ROOT/'ledger.json').read_text())
shutil.copyfile(ROOT/'ledger.json', SUP/'original-ledger-at-reconciliation.json')
cell = next(x for x in ledger['cells'] if x['arm']=='conventional' and x['condition']=='greenfield')
cell.update(status='closed', outcome='incomplete-at-limit/coordinator-interruption', closedUtc=now,
            remainingTasks='6 NOT RUN', acceptedFinalStudyCandidateSha=None)
cell['tasks']['5'].update(status='closed', outcome='incomplete-at-limit/coordinator-interruption', candidateSha=CANDIDATE,
                         closureObservedUtc=now, independentRereview='NOT RUN; activation-0030 reserved but no launch receipt')
write(ROOT/'ledger.json', ledger)
write(ROOT/'cells/conventional-greenfield/closure-reconciliation.json', {
    'observedUtc':now, 'originalTask5DeadlineUtc':'2026-10-08T22:35:08.316849Z',
    'activation0030':'reserved but not launched; original reservation bytes and usage retained',
    'receiptCount':len(requests), 'lastActualActivationCallUtc':requests[-1]['timestamp'],
    'originalReviewerStatus':'completed original activation-0028; no re-review turn',
    'candidateSha':CANDIDATE, 'tree':git('rev-parse','HEAD^{tree}'), 'clean':True,
    'outcome':'incomplete-at-limit/coordinator-interruption', 'task6':'NOT RUN',
    'cause':'Coordinator ended turn with generic wait after context compaction before reserved followup launch; no retrospective extension.',
    'nativeUsage':None, 'processCleanup':'Prior repair report records both owned services stopped; no global inventory or foreign termination during reconciliation.'})
arc=ROOT/'cells/conventional-greenfield'; arc.mkdir(exist_ok=True)
git('bundle','create',str((arc/'candidate.bundle').resolve()),'--all')
manifest=[]
with zipfile.ZipFile(arc/'evidence.zip','w',zipfile.ZIP_DEFLATED) as z:
    for p in sorted((BASE/'support/conventional-greenfield').rglob('*')):
        if p.is_file() and p.suffix.lower() not in {'.db','.sqlite','.sqlite3','.pyc'} and not any(x in p.parts for x in ['bin','obj','__pycache__']):
            name=str(p.relative_to(BASE/'support/conventional-greenfield')).replace('\\','/')
            z.write(p,name); manifest.append({'path':name,'sha256':sha(p),'bytes':p.stat().st_size})
write(arc/'archive-manifest.json', {'candidate':CANDIDATE,'tree':git('rev-parse','HEAD^{tree}'),'files':manifest,
    'artifacts':{n:sha(arc/n) for n in ['candidate.bundle','evidence.zip']},
    'exclusions':['generated bin/obj/__pycache__, Python bytecode, transient DB files; no claims from excluded database bytes']})
config=json.loads((ROOT/'native-configuration.json').read_text())
config['provenance']='Parent turn_context 2026-10-08T22:51:49.068Z checked: gpt-6.1-sol/high, danger-full-access, never; fork_turns=none and no overrides.'
write(SUP/'native-configuration.json',config)
for role,port in [('public-review',62581),('independent-evaluation',62582)]:
    own=EXT/role; own.mkdir(); (own/'state').mkdir()
    subprocess.run(['git','clone','--no-hardlinks',str((arc/'candidate.bundle').resolve()),str(own/'repo')],check=True,capture_output=True)
    git('checkout','--detach',CANDIDATE,repo=own/'repo')
    inp=own/'inputs'; inp.mkdir()
    for n in range(1,6): shutil.copytree(BASE/f'support/conventional-greenfield/inputs/stage-{n}',inp/f'stage-{n}')
    public=inp/'public'; public.mkdir(); (public/'tasks').mkdir()
    for n in ['brief.md','architecture.md','rubric.json','holdout-procedure.md']:
        shutil.copyfile(WORK/'experiments/government-comparison/public'/n,public/n)
    for p in (inp/'stage-5').glob('0*.md'): shutil.copyfile(p,public/'tasks'/p.name)
    write(inp/'resource-lease.json',{'role':role,'port':port,'url':f'http://127.0.0.1:{port}',
        'repository':str(own/'repo'),'state':str(own/'state'),'processOwnership':'Only actual child PID/Popen/exec session handles created by this Actor. No global inventory; unavailable port ends affected check.',
        'OSIsolation':False,'subspawns':0,'taskCutoff':5})
    shutil.copyfile(SUP/'native-configuration.json',inp/'native-configuration.json')
    if role=='independent-evaluation':
        private=inp/'oracle/private'; private.mkdir(parents=True)
        for n in ['holdout-plan.json','holdout_checks.py']:
            shutil.copyfile(WORK/'experiments/government-comparison/.study-data/private'/n,private/n)
        shutil.copytree(public,inp/'oracle/public')
        evidence=inp/'method-evidence'; evidence.mkdir()
        for p in state.iterdir():
            if p.is_file() and p.suffix.lower() in {'.json','.md','.txt','.py','.log','.ps1'}: shutil.copyfile(p,evidence/p.name)
        write(evidence/'own-cell-ledger.json',{'cell':cell,'activations':[x for x in ledger['trialActivations'] if x['trialId']==cell['trialId']],
            'originalTimingOutcome':cell['outcome'],'posthocMayNotChangeOriginalOutcome':True})
write(SUP/'ledger.json',{'key':grant['key'],'issuedUtc':grant['issuedUtc'],'notAfterUtc':grant['notAfterUtc'],
    'preparationStartedUtc':now,'preparationActors':0,'maxActivations':2,'maxConcurrent':1,'perActorWallSeconds':600,
    'activations':[],'usage':None,'originalStudyOutcome':'incomplete-at-limit/coordinator-interruption','task6':'NOT RUN'})
print(json.dumps({'preparedUtc':now,'supplement':str(SUP.resolve()),'external':str(EXT),'candidate':CANDIDATE,'archiveFiles':len(manifest)},indent=2))
