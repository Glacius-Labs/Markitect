"""Read-only binding checks and byte-preserving supplement evidence closure."""
import collections, datetime as dt, hashlib, json, pathlib, subprocess, zipfile
ROOT=pathlib.Path(__file__).resolve().parent
BASE=ROOT.parent/'native-codex-exploratory-v3-20261008'
EXT=pathlib.Path('C:/Users/Consiliari/Documents/Scientist-Native-v3-20261008/posthoc-20261009')
def sha(p):return hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
def write(p,v):p.write_text(json.dumps(v,indent=2)+'\n',encoding='utf8')
def git(*args,repo=None):return subprocess.check_output(['git',*args],cwd=repo,text=True).strip()
def instant(s):return dt.datetime.fromisoformat(s.replace('Z','+00:00'))
now=dt.datetime.now(dt.timezone.utc)
l=json.loads((ROOT/'ledger.json').read_text()); assert len(l['activations'])==2 and all(x['status']=='completed' for x in l['activations'])
assert instant(l['activations'][0]['observedUtc'])<=instant(l['activations'][1]['reservedUtc'])
fr=json.loads((ROOT/'freeze.json').read_text()); assert len(fr['files'])==150
bindings=[]
for name,h in fr['files'].items():
    p=pathlib.Path(name)
    if p.resolve()==(ROOT/'ledger.json').resolve():
        blob=subprocess.check_output(['git','show',fr['scientistCleanSource']+':'+p.relative_to(pathlib.Path.cwd()).as_posix()]); actual=hashlib.sha256(blob).hexdigest();kind='initial ledger immutable Git blob'
    else:actual=sha(p);kind='immutable file'
    assert actual==h,(name,actual,h)
    bindings.append({'path':name,'sha256':actual,'kind':kind,'matches':True})
reports={role:json.loads((EXT/role/'state/report.json').read_text()) for role in ['public-review','independent-evaluation']}
checks=[]
def check(label,actual,expected):
    assert actual.lower()==expected.lower(),(label,actual,expected)
    checks.append({'binding':label,'matches':True,'actual':actual})
for role,r in reports.items():
    own=EXT/role; repo=own/'repo'; assert not git('status','--porcelain',repo=repo)
    check(role+'/head',git('rev-parse','HEAD',repo=repo),'d1e271b8472cc55565696b976a87ff968a4e8c7a')
    check(role+'/tree',git('rev-parse','HEAD^{tree}',repo=repo),'a07ff00a3d3182521d202c618410f9c0520ea85f')
    if role=='public-review':
        check(role+'/checks',r['bindings']['stage5ChecksSha256'],sha(own/'inputs/stage-5/checks.py'))
        for n,h in r['bindings']['publicTaskSha256'].items():check(role+'/'+n,h,sha(own/'inputs/stage-5'/n))
    else:
        mapping={'publicBriefSha256':'stage-5/brief.md','task5Sha256':'stage-5/05-change-limit.md','releasedChecksSha256':'stage-5/checks.py',
                 'rubricSha256':'public/rubric.json','holdoutProcedureSha256':'public/holdout-procedure.md','privatePlanSha256':'oracle/private/holdout-plan.json','privateRunnerSha256':'oracle/private/holdout_checks.py'}
        for k,n in mapping.items():check(role+'/'+k,r['bindings'][k],sha(own/'inputs'/n))
    for a in l['activations']:
        if a['role']==role:check(role+'/result',a['resultSha256'],sha(own/'state/report.json'))
    end=reports[role].get('finishedAtUtc',reports[role].get('reportedUtc'))
    assert instant(end)<instant(next(x for x in l['activations'] if x['role']==role)['deadlineUtc'])
source=pathlib.Path('C:/Users/Consiliari/.codex/sessions/2026/10/07/rollout-2026-10-07T15-45-00-01a1169c-3df6-7571-997d-48ab57365875.jsonl')
requests=[]
for line in source.open(encoding='utf8'):
    try:d=json.loads(line)
    except ValueError:continue
    q=d.get('payload',{})
    if d.get('type')=='response_item' and q.get('type')=='function_call' and any(q.get('name','').endswith(x) for x in ['spawn_agent','followup_task','interrupt_agent']):
        try:args=json.loads(q.get('arguments','{}'))
        except ValueError:continue
        if str(args.get('task_name',args.get('target',''))).startswith(('posthoc_v3_','/root/posthoc_v3_')):requests.append({'timestamp':d.get('timestamp'),'name':q['name'],'callId':q.get('call_id'),'arguments':args})
assert len(requests)==2 and all(x['name'].endswith('spawn_agent') and x['arguments']['fork_turns']=='none' and 'model' not in x['arguments'] and 'reasoning_effort' not in x['arguments'] for x in requests)
for a,q in zip(l['activations'],requests):assert instant(a['reservedUtc'])<instant(q['timestamp'])
write(ROOT/'native-call-receipts.json',requests)
old=json.loads((ROOT/'original-ledger-at-reconciliation.json').read_text());current=json.loads((BASE/'ledger.json').read_text());assert old['trialActivations']==current['trialActivations']
original=json.loads((BASE/'freeze.json').read_text()); original_checks=0
for name,h in original['files'].items():assert sha(name)==h;original_checks+=1
for c in original['cells']:
    for name,h in c['supportPins'].items():assert sha(name)==h;original_checks+=1
retained=[]
prefix=BASE.relative_to(pathlib.Path.cwd()).as_posix()+'/cells'
for path in git('ls-tree','-r','--name-only','b9128650b3a84cd30a84d3ce582ee66b243a544e','--',prefix).splitlines():
    blob=subprocess.check_output(['git','show','b9128650b3a84cd30a84d3ce582ee66b243a544e:'+path])
    actual=pathlib.Path(path).read_bytes();assert actual==blob,path
    retained.append({'path':path,'sha256':hashlib.sha256(actual).hexdigest()})
files=[]
with zipfile.ZipFile(ROOT/'evidence.zip','w',zipfile.ZIP_DEFLATED) as z:
    for p in sorted(EXT.rglob('*')):
        if p.is_file() and not any(x in p.parts for x in ['.git','bin','obj','__pycache__']):
            name=p.relative_to(EXT).as_posix();z.write(p,name);files.append({'path':name,'sha256':sha(p),'bytes':p.stat().st_size})
with zipfile.ZipFile(ROOT/'evidence.zip') as z:
    assert z.testzip() is None
    for f in files:assert hashlib.sha256(z.read(f['path'])).hexdigest()==f['sha256']
write(ROOT/'archive-manifest.json',{'files':files,'archiveSha256':sha(ROOT/'evidence.zip'),'exclusions':['.git object databases, generated bin/obj, Python bytecode caches; candidate history retained in original candidate.bundle'],'originalCandidateBundleSha256':sha(BASE/'cells/conventional-greenfield/candidate.bundle')})
l.update(status='closed',outcome='inconclusive_runtime_port_bind_failure',closedUtc=now.isoformat(),actualActivations=2,maxObservedConcurrent=1,
         preparationElapsedFromGrantIssueSeconds=(instant(l['activations'][0]['reservedUtc'])-instant(l['issuedUtc'])).total_seconds(),
         elapsedFromGrantIssueAtSealSeconds=(now-instant(l['issuedUtc'])).total_seconds(),remainingActualActivations=0,
         actorReportedElapsedSeconds=[85,95],conservativeReservationToCompletionObservationSeconds=[(instant(x['observedUtc'])-instant(x['reservedUtc'])).total_seconds() for x in l['activations']],
         humanInterventionSeconds=None,providerRequests=None,rawTokens=None,cost=None,task6='NOT RUN')
assert l['preparationElapsedFromGrantIssueSeconds']<=900 and l['elapsedFromGrantIssueAtSealSeconds']<=2700
write(ROOT/'ledger.json',l)
write(ROOT/'final-readonly-validation.json',{'checkedUtc':now.isoformat(),'supplementBindings':bindings,'reportedInputAndCandidateBindings':checks,'originalFreezePinChecks':original_checks,
    'old30ReservationObjectsUnchanged':True,'retainedOriginalEvidenceFiles':retained,'actualSupplementStarts':len(requests),'preReserved':True,'forkTurnsNone':True,'noModelOverrides':True,
    'newArchiveMembersVerified':len(files),'runtimeAssertions':'NOT RUN; no pass inferred','evaluationErrata':'see evaluator-boundaries-errata.md'})
print(json.dumps({'sealedUtc':now.isoformat(),'bindings':len(bindings),'reportBindings':len(checks),'originalPins':original_checks,'oldEvidenceFilesUnchanged':len(retained),'archiveMembers':len(files),'ledger':l},indent=2))
