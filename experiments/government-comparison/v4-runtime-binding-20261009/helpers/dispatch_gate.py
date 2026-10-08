"""Durable call-boundary admission, not a native timer or OS/provider sandbox.

The coordinator must call before_native immediately before its actual tool call,
then bind the actual start/result/stop receipts. This module invokes no tool.
"""
import contextlib, hashlib, json, os, pathlib, subprocess, time, uuid
from deadlines import role_deadline
class Refused(ValueError):pass
def require(ok,why):
    if not ok:raise Refused(why)
def digest(path):return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()
def snapshot(repo):
    def run(*args):return subprocess.check_output(['git','-C',str(repo),*args],text=True,timeout=10).strip()
    require(not run('status','--porcelain'),'candidate dirty')
    return {'sha':run('rev-parse','HEAD'),'tree':run('rev-parse','HEAD^{tree}')}
class Gate:
    def __init__(self,directory,grant,clock=None):
        self.root=pathlib.Path(directory).resolve();self.root.mkdir(parents=True,exist_ok=True)
        self.path=self.root/'dispatch.json';self.lock=self.root/'dispatch.lock';self.events=self.root/'events.jsonl';self.grant=json.loads(json.dumps(grant));self.clock=clock or (lambda:(time.time(),time.monotonic()))
        self.grant_digest=hashlib.sha256(json.dumps(self.grant,sort_keys=True,separators=(',',':')).encode()).hexdigest()
    @contextlib.contextmanager
    def transaction(self):
        end=time.monotonic()+2
        while True:
            try:fd=os.open(self.lock,os.O_CREAT|os.O_EXCL|os.O_WRONLY,0o600);break
            except FileExistsError:
                if time.monotonic()>=end:raise Refused('shared counter lock unavailable; no stale-lock takeover')
                time.sleep(.01)
        try:
            os.write(fd,str(os.getpid()).encode());os.close(fd)
            if self.path.exists():state=json.loads(self.path.read_text());require(state['grantDigest']==self.grant_digest,'grant/config changed')
            else:
                require(not self.events.exists(),'missing state with retained journal; no quota reset')
                state={'grantDigest':self.grant_digest,'cells':{},'actions':{}}
            if self.events.exists():
                for line in self.events.read_text().splitlines():
                    entry=json.loads(line)
                    if entry['event']=='reservation-before-native-call':require(entry['action']['id'] in state['actions'],'reservation journal/state mismatch; no quota reset')
            yield state
            temporary=self.path.with_suffix('.tmp');temporary.write_text(json.dumps(state,indent=2)+'\n',encoding='utf8');temporary.replace(self.path)
        finally:self.lock.unlink(missing_ok=True)
    def live(self):
        utc,mono=self.clock();require(self.grant['issued']<=utc<self.grant['end'],'grant outside actual time window');return utc,mono
    def event(self,value):
        with self.events.open('a',encoding='utf8') as f:f.write(json.dumps(value,sort_keys=True)+'\n')
    def own(self,path):
        p=pathlib.Path(path).resolve();require(p.is_relative_to(self.root),'path outside declared own dispatcher root');return p
    def register_cell(self,cell,repo,condition):
        with self.transaction() as s:
            now,mono=self.live();require(cell in self.grant['cells'],'unallocated cell');require(cell not in s['cells'],'cell already registered')
            repo=self.own(repo);require(repo.is_dir(),'missing own repository');require(condition in {'greenfield','brownfield'},'condition')
            s['cells'][cell]={'repo':str(repo),'condition':condition,'started':now,'startedMono':mono,'end':min(now+7200,self.grant['end']),'endMono':mono+min(7200,self.grant['end']-now),'tasks':{}}
    def begin_task(self,cell,task,release_manifest,expected_digest,equivalence=None):
        with self.transaction() as s:
            now,mono=self.live();c=s['cells'][cell];require(type(task) is int and 1<=task<=6,'task range')
            require(now<c['end'] and c['startedMono']<=mono<c['endMono'],'trial exhausted or monotonic clock reset')
            require(str(task) not in c['tasks'],'task already begun; deadline cannot reset')
            if task>1:require(c['tasks'].get(str(task-1),{}).get('outcome')=='passed','previous task not passed')
            if task==6:
                previous=c['tasks']['5'];require(isinstance(equivalence,dict),'Task6 independent equivalence missing')
                receipt=self.own(equivalence.get('path',''));require(receipt.is_file() and digest(receipt)==equivalence.get('sha256'),'Task6 equivalence receipt binding missing')
                eq=json.loads(receipt.read_text());require(eq.get('passed') is True and eq.get('peerEquivalent') is True and eq.get('candidateSha')==previous['candidate']['sha'],'Task6 independent equivalence missing')
                require(bool(eq.get('invariantId')) and bool(eq.get('peerCandidateShas')) and bool(eq.get('independentEvaluatorActorId')),'Task6 peer/invariant/evaluator identity missing')
                require(not any(a.get('actorId')==eq['independentEvaluatorActorId'] for a in s['actions'].values()),'equivalence evaluator is not independent of trial Actors')
            manifest=self.own(release_manifest);require(not manifest.is_relative_to(pathlib.Path(c['repo'])),'release metadata must stay outside Actor repository');require(digest(manifest)==expected_digest,'release manifest changed')
            r=json.loads(manifest.read_text());require(r['throughTask']==task and r['condition']==c['condition'],'wrong task/condition release')
            inputs=[]
            for entry in r['inputs']:
                p=(manifest.parent/entry['path']).resolve();require(p.is_relative_to(manifest.parent),'release path escape');require(digest(p)==entry['sha256'],'released input changed')
                if p.name[:2].isdigit():require(int(p.name[:2])<=task,'future card exposed')
                inputs.append({'path':str(p),'sha256':entry['sha256']})
            if task==6:inputs.append({'path':str(receipt),'sha256':equivalence['sha256']})
            c['tasks'][str(task)]={'start':now,'startMono':mono,'end':min(now+1200,c['end']),'endMono':min(mono+1200,c['endMono']),'phase':'implementation','manifest':str(manifest),'manifestDigest':expected_digest,'inputs':inputs,'candidate':None}
    def freeze_candidate(self,cell,task,expected_sha,expected_tree):
        with self.transaction() as s:
            now,mono=self.live();c=s['cells'][cell];t=c['tasks'][str(task)];require(t['phase']=='implementation','freeze requires implementation phase');require(now<=t['start']+840 and now<t['end']-300 and 0<=mono-t['startMono']<=840 and mono<t['endMono']-300,'assessment reserve unavailable')
            require(not any(a['cell']==cell and a['task']==task and a['status'] in {'reserved','authorized','running','stop-requested'} for a in s['actions'].values()),'implementation actions still active')
            observed=snapshot(c['repo']);require(observed=={'sha':expected_sha,'tree':expected_tree},'candidate identity mismatch')
            t['candidate']={**observed,'frozen':now};t['phase']='assessment'
    def reserve(self,cell,task,role,prompt,runner_seconds,bridge_end,role_seconds=600,audited_roots=(),input_files=()):
        with self.transaction() as s:
            now,mono=self.live()
            require(self.grant.get('executionAllowed') is True,'no execution grant')
            require(self.grant.get('admissionComplete') is True,'candidate/runtime/profile admission incomplete')
            c=s['cells'][cell];t=c['tasks'][str(task)]
            actions=list(s['actions'].values());require(len(actions)<self.grant.get('maxActivations',0),'total activation quota')
            require(sum(x['status'] in {'reserved','authorized','running','stop-requested'} for x in actions)<self.grant['maxConcurrent'],'shared concurrency quota')
            active_cells={x['cell'] for x in actions if x['status'] in {'reserved','authorized','running','stop-requested'}}
            require(cell in active_cells or len(active_cells)<self.grant.get('maxActiveCells',2),'shared primary-cell concurrency quota')
            require(sum(x['cell']==cell for x in actions)<72,'cell quota');require(sum(x['cell']==cell and x['task']==task for x in actions)<12,'task quota')
            require(c['startedMono']<=mono<c['endMono'] and t['startMono']<=mono<t['endMono'],'monotonic trial/task cap')
            assessment=role=='independent-assessment';require(role in {'primary','native-role','public-review','independent-assessment'},'role not admitted')
            require(t['phase']==('assessment' if assessment else 'implementation'),'wrong phase')
            if assessment:require(t['candidate'] is not None,'assessment requires frozen candidate')
            limits=role_deadline(now=now,task_start=t['start'],trial_end=c['end'],grant_end=self.grant['end'],runner_seconds=runner_seconds,bridge_end=bridge_end,role_seconds=role_seconds,assessment=assessment)
            if assessment:require(t['endMono']-mono>=300,'monotonic assessment reserve unavailable')
            phase_mono=t['endMono']-30 if assessment else min(t['startMono']+840,t['endMono']-360)
            limits['effectiveEnd']=min(limits['effectiveEnd'],now+phase_mono-mono)
            require(limits['effectiveEnd']-now>15,'no monotonic launch window');limits['interruptRequestBy']=limits['effectiveEnd']-10
            prompt=self.own(prompt);require(prompt.is_file(),'missing own prompt')
            a_id=uuid.uuid4().hex;state=self.root/'actors'/a_id
            roots=[pathlib.Path(x).resolve() for x in audited_roots];roots.append(pathlib.Path(c['repo']).resolve())
            require(all(not state.is_relative_to(x) for x in roots),'Actor state intersects audit/repository root')
            files=[{'path':str(pathlib.Path(x).resolve()),'sha256':digest(x)} for x in input_files]
            require(all(not state.is_relative_to(pathlib.Path(x['path'])) for x in files),'state intersects invocation/runtime input')
            state.mkdir(parents=True,exist_ok=False)
            a={'id':a_id,'grantKey':self.grant['key'],'grantDigest':self.grant_digest,'cell':cell,'task':task,'role':role,'status':'reserved','reserved':now,'reservedMono':mono,
               'prompt':str(prompt),'promptSha256':digest(prompt),'stateDirectory':str(state),'output':str(state/'response.json'),'logs':str(state/'logs'),'auditedRoots':[str(x) for x in roots],
               'inputFiles':files,'candidateSha':t['candidate']['sha'] if assessment else None,'profileSha256':self.grant['profileSha256'],'forkTurns':'none','modelOverride':None,'reasoningOverride':None,
               'effectiveEnd':limits['effectiveEnd'],'interruptRequestBy':limits['interruptRequestBy'],'monoEnd':mono+limits['effectiveEnd']-now,'usage':None}
            s['actions'][a_id]=a;self.event({'event':'reservation-before-native-call','action':a});return dict(a)
    def before_native(self,action_id,expected_prompt_sha,profile_sha):
        with self.transaction() as s:
            now,mono=self.live();a=s['actions'][action_id];c=s['cells'][a['cell']];t=c['tasks'][str(a['task'])]
            require(a['status']=='reserved','missing/consumed reservation');require(now<a['interruptRequestBy'] and mono<a['monoEnd']-10,'late native call')
            require(t['phase']==('assessment' if a['role']=='independent-assessment' else 'implementation'),'phase changed after reservation')
            require(profile_sha==a['profileSha256'] and expected_prompt_sha==a['promptSha256']==digest(a['prompt']),'prompt/profile binding changed')
            require(digest(t['manifest'])==t['manifestDigest'],'stage manifest changed')
            for item in t['inputs']+a['inputFiles']:require(digest(item['path'])==item['sha256'],'staged/runtime input changed')
            self.own(a['stateDirectory']);require(all(not pathlib.Path(a['stateDirectory']).resolve().is_relative_to(pathlib.Path(x)) for x in a['auditedRoots']),'own output/audit conflict')
            if a['role']=='independent-assessment':require(snapshot(c['repo'])=={k:t['candidate'][k] for k in ['sha','tree']},'frozen candidate changed')
            a['status']='authorized';a['authorized']=now;mock=self.grant.get('kind')=='mechanical-fixture'
            result={**a,'nativeExecutionAuthorized':not mock,'mechanicalMockOnly':mock};self.event({'event':'before-native-boundary','action':result});return result
    def record_start(self,action_id,tool_call_id,actor_id):
        with self.transaction() as s:
            now,mono=self.live();a=s['actions'][action_id];require(a['status']=='authorized','start without authorization');require(now<a['interruptRequestBy'] and mono<a['monoEnd']-10,'start after interruption boundary')
            require(bool(tool_call_id) and bool(actor_id),'missing actual call/Actor identity')
            require(not any(x.get('actorId')==actor_id or x.get('toolCallId')==tool_call_id for x in s['actions'].values()),'Actor/call must be fresh and unique')
            a.update(status='running',toolCallId=tool_call_id,actorId=actor_id,started=now);self.event({'event':'start-receipt','action':dict(a)})
    def record_result(self,action_id,result):
        with self.transaction() as s:
            now,mono=self.clock();a=s['actions'][action_id];require(a['status']=='running','result without running reservation')
            for key in ['id','grantKey','cell','task','role','promptSha256','actorId','toolCallId','candidateSha']:require(result.get(key)==a.get(key),'result identity mismatch: '+key)
            require(now<=a['effectiveEnd'] and mono<=a['monoEnd'],'late result; incomplete, not pass')
            require(result.get('outcome') in {'passed','failed','incomplete'},'result outcome missing')
            if a['role']=='independent-assessment':
                require(result.get('independentAssessment') is True,'independent assessment receipt missing')
                c=s['cells'][a['cell']];t=c['tasks'][str(a['task'])]
                require(snapshot(c['repo'])=={k:t['candidate'][k] for k in ['sha','tree']},'candidate changed during assessment')
            a.update(status='completed',ended=now,result=result);self.event({'event':'result-receipt','action':dict(a)})
    def close_task(self,cell,task,assessment_id):
        with self.transaction() as s:
            a=s['actions'][assessment_id];t=s['cells'][cell]['tasks'][str(task)]
            require(a['cell']==cell and a['task']==task and a['role']=='independent-assessment' and a['status']=='completed','missing exact completed assessment')
            require(t['phase']=='assessment' and a['candidateSha']==t['candidate']['sha'],'assessment candidate differs or task closed')
            require(snapshot(s['cells'][cell]['repo'])=={k:t['candidate'][k] for k in ['sha','tree']},'candidate changed before closure')
            t.update(phase='closed',outcome=a['result']['outcome'],assessment=assessment_id)
    def stop_due(self):
        with self.transaction() as s:
            now,mono=self.clock();due=[]
            for a in s['actions'].values():
                if a['status'] in {'reserved','authorized','running'} and (now>=a['interruptRequestBy'] or mono>=a['monoEnd']-10):
                    running=a['status']=='running'
                    due.append({'actionId':a['id'],'actorId':a.get('actorId'),'tool':'collaboration.interrupt_agent' if running else None,'localAction':None if running else 'mark_terminal_incomplete','requestBy':a['interruptRequestBy'],'deadline':a['effectiveEnd'],'reason':'effective deadline' if running else 'expired unstarted reservation','osStopped':None,'providerStopped':None})
            return due
    def record_stop_request(self,action_id,tool_call_id,stop_result):
        with self.transaction() as s:
            now,mono=self.clock();a=s['actions'][action_id];require(a['status'] in {'authorized','running'},'stop status')
            a.update(status='stop-requested',stopRequestUtc=now,stopToolCallId=tool_call_id,stopReceipt=stop_result,stopRequestAfterDeadline=now>a['effectiveEnd'],osStopped=None,providerStopped=None)
            self.event({'event':'stop-request-receipt','action':dict(a)})
    def mark_terminal_incomplete(self,action_id,own_process_receipts):
        with self.transaction() as s:
            a=s['actions'][action_id];require(a['status'] in {'reserved','authorized','running','stop-requested'},'terminal status')
            a.update(status='interrupted',outcome='incomplete',ownProcessReceipts=own_process_receipts,usage=None)
            self.event({'event':'terminal-incomplete-no-refund','action':dict(a)})
