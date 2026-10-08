"""Archive completed own evidence; no candidate/test execution or repair."""
import datetime as dt, hashlib, json, pathlib, subprocess, zipfile
P=pathlib.Path(__file__).resolve().parent
E=pathlib.Path('C:/Users/Consiliari/Documents/Scientist-Native-v3-20261008/ephemeral-20261009')
S=E/'state'
def sha(p):return hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
def write(p,v):p.write_text(json.dumps(v,indent=2)+'\n',encoding='utf8')
def git(*a,cwd=None):return subprocess.check_output(['git',*a],cwd=cwd,text=True).strip()
def instant(t):return dt.datetime.fromisoformat(t.replace('Z','+00:00'))
now=dt.datetime.now(dt.timezone.utc);r=json.loads((S/'report.json').read_text());f=json.loads((P/'freeze.json').read_text());l=json.loads((P/'ledger.json').read_text());a=l['activations'][0]
assert len(l['activations'])==1 and a['status']=='completed'
assert not git('status','--porcelain',cwd=E/'repo')
assert git('rev-parse','HEAD',cwd=E/'repo')==f['candidate']==r['candidate']['commit']
assert git('rev-parse','HEAD^{tree}',cwd=E/'repo')==f['candidateTree']==r['candidate']['tree']
bindings=[]
for name,h in f['files'].items():
    p=pathlib.Path(name)
    if p.resolve()==(P/'ledger.json').resolve():
        blob=subprocess.check_output(['git','show',f['cleanScientistSource']+':'+p.relative_to(pathlib.Path.cwd()).as_posix()]);actual=hashlib.sha256(blob).hexdigest();kind='initial ledger Git blob'
    else:actual=sha(p);kind='immutable file'
    assert actual==h,name;bindings.append({'path':name,'sha256':actual,'kind':kind,'matches':True})
reportbindings=[]
mapping={'stage5ReleaseSha256':'stage-5/release.json','stage5ChecksSha256':'stage-5/checks.py','publicBriefSha256':'public/brief.md','publicArchitectureSha256':'public/architecture.md','holdoutPlanDigest':'oracle/private/holdout-plan.json'}
for k,name in mapping.items():
    actual=sha(E/'inputs'/name);assert actual==r['inputs'][k];reportbindings.append({'field':k,'sha256':actual,'matches':True})
tasks=sorted((E/'inputs/public/tasks').glob('0*.md'));assert len(tasks)==5
for p,h in zip(tasks,r['inputs']['task1to5Sha256']):assert sha(p)==h;reportbindings.append({'field':p.name,'sha256':h,'matches':True})
logbindings=[]
for item in r['commandsAndLogs']['logs']:
    p=S/item['path'];assert sha(p)==item['sha256'] and p.stat().st_size==item['bytes'];logbindings.append({'path':item['path'],'sha256':sha(p),'matches':True})
public=json.loads((S/'public-before-restart.checks.log').read_text());restart=json.loads((S/'public-after-restart.checks.log').read_text());assert public['status']==restart['status']=='passed';assert len(public['checks'])==46 and len(restart['checks'])==3
holdouts=[]
for case in ['order-normalization','validation-boundaries','idempotent-concurrency','terminal-transition-ordering','rule-change-entrypoint-parity']:
    q=json.loads((S/(case+'.log')).read_text());assert q['status']=='passed' and q['candidateCommit']==f['candidate'];holdouts.append({'case':case,'status':q['status'],'assertionCount':q['assertionCount']})
before=json.loads((S/'restart-before.log').read_text());after=json.loads((S/'restart-after.log').read_text());assert before['status']=='passed' and after['status']=='failed'
starts=json.loads((S/'owned-server-starts.json').read_text());assert len(starts)==12 and starts==r['serverStarts']['records']
for x in starts:
    assert x['urls']=='http://127.0.0.1:0' and x.get('stoppedUtc') and x.get('exitCode') is not None
    assert x['url'] in (S/(x['label']+'.server.log')).read_text()
    assert instant(x['stoppedUtc'])<instant(a['deadlineUtc'])
assert instant(r['finishedUtc'])<instant(a['deadlineUtc'])
source=pathlib.Path('C:/Users/Consiliari/.codex/sessions/2026/10/07/rollout-2026-10-07T15-45-00-01a1169c-3df6-7571-997d-48ab57365875.jsonl');calls=[];profile=None
for line in source.open(encoding='utf8'):
    try:d=json.loads(line)
    except ValueError:continue
    q=d.get('payload',{})
    if d.get('type')=='turn_context':profile={'timestamp':d.get('timestamp'),'requestedModel':q.get('model'),'requestedReasoningEffort':q.get('effort'),'servingModel':None}
    if d.get('type')=='response_item' and q.get('type')=='function_call' and any(q.get('name','').endswith(t) for t in ['spawn_agent','followup_task','interrupt_agent']):
        try:args=json.loads(q.get('arguments','{}'))
        except ValueError:continue
        if 'ephemeral_v3_independent_runtime' in str(args.get('task_name',args.get('target',''))):calls.append({'timestamp':d.get('timestamp'),'name':q['name'],'callId':q.get('call_id'),'arguments':args})
assert len(calls)==1 and calls[0]['name'].endswith('spawn_agent') and calls[0]['arguments']['fork_turns']=='none'
assert 'model' not in calls[0]['arguments'] and 'reasoning_effort' not in calls[0]['arguments'] and instant(a['reservedUtc'])<instant(calls[0]['timestamp'])
write(P/'native-call-receipts.json',calls);write(P/'parent-profile-observation.json',profile)
old=P.parent/'native-v3-conventional-gf-posthoc-validation-20261009';retained=[]
for path in git('ls-tree','-r','--name-only','cc8b8d63f58a309b5b22ec5cd3d776ce5f0dfd94','--',old.relative_to(pathlib.Path.cwd()).as_posix()).splitlines():
    b=subprocess.check_output(['git','show','cc8b8d63f58a309b5b22ec5cd3d776ce5f0dfd94:'+path]);assert pathlib.Path(path).read_bytes()==b;retained.append({'path':path,'sha256':hashlib.sha256(b).hexdigest()})
base=P.parent/'native-codex-exploratory-v3-20261008';o=json.loads((base/'freeze.json').read_text());pins=0
for name,h in o['files'].items():assert sha(name)==h;pins+=1
for c in o['cells']:
    for name,h in c['supportPins'].items():assert sha(name)==h;pins+=1
assert json.loads((base/'ledger.json').read_text())['trialActivations']==json.loads((old/'original-ledger-at-reconciliation.json').read_text())['trialActivations']
files=[]
with zipfile.ZipFile(P/'evidence.zip','w',zipfile.ZIP_DEFLATED) as z:
    for p in sorted(E.rglob('*')):
        if p.is_file() and not any(x in p.parts for x in ['.git','bin','obj','__pycache__']):
            name=p.relative_to(E).as_posix();z.write(p,name);files.append({'path':name,'sha256':sha(p),'bytes':p.stat().st_size})
with zipfile.ZipFile(P/'evidence.zip') as z:
    assert z.testzip() is None
    for x in files:assert hashlib.sha256(z.read(x['path'])).hexdigest()==x['sha256']
write(P/'archive-manifest.json',{'archiveSha256':sha(P/'evidence.zip'),'files':files,'exclusions':['.git object databases, generated bin/obj, __pycache__; unchanged candidate history retained in candidate.bundle']})
l.update(status='closed',outcome='partial_posthoc_runtime_evidence_with_oracle_failures_and_diagnostic_scope_deviation',closedUtc=now.isoformat(),actualActivations=1,remainingActivations=0,
    maxObservedConcurrent=1,ownServerStarts=12,ownedStoppedReceipts=12,reportedActorSeconds=r['elapsedSecondsFromReservation'],
    reservationToCompletionObservationSeconds=(instant(a['observedUtc'])-instant(a['reservedUtc'])).total_seconds(),elapsedFromIssueToSealSeconds=(now-instant(l['issuedUtc'])).total_seconds(),
    providerRequests=None,servingModel=None,tokens=None,cost=None,humanInterventionSeconds=None,originalOutcome='incomplete-at-limit/coordinator-interruption',task6='NOT RUN')
assert l['preparationElapsedFromIssueSeconds']<300 and l['reportedActorSeconds']<600 and l['elapsedFromIssueToSealSeconds']<1200
write(P/'ledger.json',l)
write(P/'final-readonly-validation.json',{'checkedUtc':now.isoformat(),'freezeBindings':bindings,'reportInputBindings':reportbindings,'logBindings':logbindings,'originalFreezeChecks':pins,'old30ReservationObjectsUnchanged':True,
    'priorSupplementFilesRetained':retained,'candidateClean':True,'publicChecksActualCount':46,'publicRestartCount':3,'passedHoldouts':holdouts,'restartBefore':before,'restartAfter':after,
    'atomicStock':'inapplicable/NOT RUN','diagnostic':'unplanned observational start12; no accepted test/score credit; see operator-errata.md','serverStarts':12,'stoppedOwnHandleReceipts':12,'preReservedFreshActor':True,'archiveMembersVerified':len(files)})
print(json.dumps({'sealedUtc':now.isoformat(),'freezeBindings':len(bindings),'reportBindings':len(reportbindings),'logBindings':len(logbindings),'originalPins':pins,'retainedPriorSupplementFiles':len(retained),'archiveMembers':len(files),'ledger':l},indent=2))
