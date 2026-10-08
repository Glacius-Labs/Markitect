"""Synthetic mechanical fixtures only: no native Actors, product roles or servers."""
import concurrent.futures, copy, json, pathlib, tempfile, unittest
from unittest.mock import patch
import dispatch_gate as dg
import dto_preflight as dto
from native_boundary import prepare_spawn
from deadlines import role_deadline

def data(x):return json.dumps(x).encode()

class DTO(unittest.TestCase):
    def setUp(self):
        self.i={'apiVersion':dto.API,'runId':'a'*32,'nonce':'b'*32,'inputDigest':'sha256:'+'c'*64,
                'request':{'role':'executor','sourceRevision':'d'*40,'modelDigest':'sha256:'+'e'*64,'modulePin':'sha256:'+'f'*64,'projectionId':'projection-1','scopeIds':['scope'],'policyIds':['policy'],'artifacts':[{'path':'input.cs','mode':'0644','digest':'sha256:'+__import__('hashlib').sha256(b'input').hexdigest(),'content':'aW5wdXQ='}],'context':{'Name':1,'name':2}}}
        self.r={k:self.i[k] for k in ['apiVersion','runId','nonce','inputDigest']}
        self.r.update(role='executor',outcome='proposed',candidateFiles=[{'path':'src/Order.cs','mode':'0644','content':'ordinary semantic bytes'}],evidenceRefs=['scope'],verifierObservations=[],uncertainty=[])
    def check(self):return dto.validate(data(self.i),data(self.r),['src/Order.cs'])
    def test_context_case_sensitive_payload(self):self.assertEqual(self.check()['structuralPreflight'],'passed')
    def test_exact_context_duplicate_refused(self):
        with self.assertRaises(dto.Invalid):dto.strict(b'{"request":{"context":{"Name":1,"Name":2}}}')
    def test_protocol_alias_refused(self):
        with self.assertRaises(dto.Invalid):dto.strict(b'{"runId":"a","RunID":"b"}')
    def test_all_product_modes(self):
        for mode in ['0600','0644','0755']:
            with self.subTest(mode=mode):self.r['candidateFiles'][0]['mode']=mode;self.check()
    def test_git_mode_is_not_wire_mode(self):
        self.r['candidateFiles'][0]['mode']='100644'
        with self.assertRaises(dto.Invalid):self.check()
    def test_wrong_identity(self):
        self.r['nonce']='d'*32
        with self.assertRaises(dto.Invalid):self.check()
    def test_old_bare_digest_fixture_is_refused(self):
        self.i['inputDigest']=self.r['inputDigest']='c'*64
        with self.assertRaises(dto.Invalid):self.check()
    def test_closed_invocation_and_request(self):
        self.i['extra']=1
        with self.assertRaises(dto.Invalid):self.check()
        del self.i['extra'];self.i['request']['extra']=1
        with self.assertRaises(dto.Invalid):self.check()
    def test_unsupplied_ref(self):
        self.r['evidenceRefs']=['invented proof']
        with self.assertRaises(dto.Invalid):self.check()
    def test_inference_payload_is_opaque(self):
        self.i['request']['role']='infer';self.r.update(role='infer',candidateFiles=[],candidateJson={'Name':1,'name':2});self.check()
    def test_closed_response(self):
        self.r['newProtocolField']=True
        with self.assertRaises(dto.Invalid):self.check()
    def test_publish_preserves_raw_bytes_and_guard(self):
        with tempfile.TemporaryDirectory() as d:
            p=pathlib.Path(d);repo=p/'repo';repo.mkdir();outside=p/'state';outside.mkdir()
            i=p/'invocation.json';i.write_bytes(data(self.i));r=p/'raw.json';raw=data(self.r)+b'\n';r.write_bytes(raw)
            with self.assertRaises(dto.Invalid):dto.publish(i,r,repo/'response.json',['src/Order.cs'],[repo])
            out=outside/'response.json';dto.publish(i,r,out,['src/Order.cs'],[repo]);self.assertEqual(out.read_bytes(),raw)

class Clock:
    def __init__(self):self.utc=100.;self.mono=1000.
    def __call__(self):return self.utc,self.mono
    def advance(self,s):self.utc+=s;self.mono+=s

class Dispatch(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup);self.root=pathlib.Path(self.tmp.name)
        self.repo=self.root/'repo';self.repo.mkdir();self.clock=Clock();self.observed={'sha':'a'*40,'tree':'b'*40}
        self.mock=patch.object(dg,'snapshot',side_effect=lambda _:dict(self.observed));self.mock.start();self.addCleanup(self.mock.stop)
        self.grant={'kind':'mechanical-fixture','key':'fixture-no-native-authority','issued':90.,'end':20000.,'cells':['gf'],'executionAllowed':True,'admissionComplete':True,'maxActivations':72,'maxConcurrent':4,'profileSha256':'c'*64}
        self.g=dg.Gate(self.root,self.grant,self.clock);self.g.register_cell('gf',self.repo,'greenfield')
        self.prompt=self.root/'prompt.txt';self.prompt.write_text('mock fixture only; no native call')
        self.input=self.root/'release'/'01-task.md';self.input.parent.mkdir();self.input.write_text('only released task')
        self.begin(1)
    def begin(self,task,**kw):
        manifest=self.root/'release'/f'manifest-{task}.json'
        manifest.write_text(json.dumps({'throughTask':task,'condition':'greenfield','inputs':[{'path':self.input.name,'sha256':dg.digest(self.input)}]}))
        self.g.begin_task('gf',task,manifest,dg.digest(manifest),**kw)
    def reserve(self,role='primary',task=1,**kw):return self.g.reserve('gf',task,role,self.prompt,180,10000.,**kw)
    def authorize(self,a):return self.g.before_native(a['id'],a['promptSha256'],a['profileSha256'])
    def result(self,a,outcome='passed'):
        r={k:a.get(k) for k in ['id','grantKey','cell','task','role','promptSha256','candidateSha']}
        r.update(actorId='actor-'+a['id'],toolCallId='call-'+a['id'],outcome=outcome,independentAssessment=True)
        return r
    def start(self,a):self.authorize(a);self.g.record_start(a['id'],'call-'+a['id'],'actor-'+a['id'])
    def finish_task(self,task):
        self.g.freeze_candidate('gf',task,self.observed['sha'],self.observed['tree'])
        a=self.reserve('independent-assessment',task);self.start(a);self.g.record_result(a['id'],self.result(a));self.g.close_task('gf',task,a['id'])
    def test_reservation_precedes_mock_boundary(self):
        a=self.reserve();state=json.loads((self.root/'dispatch.json').read_text());self.assertEqual(state['actions'][a['id']]['status'],'reserved')
        boundary=self.authorize(a);self.assertFalse(boundary['nativeExecutionAuthorized']);self.assertTrue(boundary['mechanicalMockOnly'])
    def test_exact_native_argument_preparation_remains_mock(self):
        a=self.reserve();boundary=prepare_spawn(self.g,a['id'],a['promptSha256'],a['profileSha256'])
        self.assertFalse(boundary['mayExecute']);self.assertEqual(boundary['toolArguments']['message'],self.prompt.read_text())
        self.assertEqual(boundary['toolArguments']['fork_turns'],'none');self.assertNotIn('model',boundary['toolArguments']);self.assertNotIn('reasoning_effort',boundary['toolArguments'])
    def test_missing_or_consumed_reservation(self):
        with self.assertRaises(KeyError):self.g.before_native('missing','x','y')
        a=self.reserve();self.authorize(a)
        with self.assertRaises(dg.Refused):self.authorize(a)
    def test_runner_min_and_stop_receipt(self):
        a=self.reserve();self.assertEqual(a['effectiveEnd'],280.);self.start(a);self.clock.advance(170)
        due=self.g.stop_due();self.assertEqual(len(due),1);self.assertIsNone(due[0]['providerStopped']);self.assertIsNone(due[0]['osStopped'])
        self.g.record_stop_request(a['id'],'mock-interrupt',{'status':'mock-request-only'})
        self.g.mark_terminal_incomplete(a['id'],[])
        self.assertEqual(json.loads(self.g.path.read_text())['actions'][a['id']]['status'],'interrupted')
    def test_late_reservation(self):
        self.clock.advance(830)
        with self.assertRaises(ValueError):self.reserve()
    def test_expired_unstarted_reservation_has_no_fake_actor_stop(self):
        self.reserve();self.clock.advance(171);due=self.g.stop_due()
        self.assertEqual(due[0]['localAction'],'mark_terminal_incomplete');self.assertIsNone(due[0]['tool']);self.assertIsNone(due[0]['actorId'])
    def test_late_start_receipt_refused(self):
        a=self.reserve();self.authorize(a);self.clock.advance(171)
        with self.assertRaises(dg.Refused):self.g.record_start(a['id'],'mock-call','mock-actor')
    def test_monotonic_boundary_survives_backward_utc(self):
        a=self.reserve();self.clock.mono+=171;self.clock.utc-=5
        with self.assertRaises(dg.Refused):self.authorize(a)
    def test_monotonic_phase_min_after_clock_adjustment(self):
        self.clock.utc+=700;self.clock.mono+=720
        a=self.reserve();self.assertEqual(a['effectiveEnd'],920.)
    def test_external_grant_dict_cannot_extend_bound_grant(self):
        self.grant['end']=90000.;self.assertEqual(self.g.grant['end'],20000.)
    def test_missing_state_with_journal_cannot_refill_quota(self):
        self.reserve();self.g.path.unlink()
        with self.assertRaises(dg.Refused):self.reserve()
    def test_phase_requires_frozen_candidate(self):
        with self.assertRaises(dg.Refused):self.reserve('independent-assessment')
        self.g.freeze_candidate('gf',1,self.observed['sha'],self.observed['tree'])
        with self.assertRaises(dg.Refused):self.reserve('primary')
    def test_freeze_refuses_active_implementation(self):
        self.reserve()
        with self.assertRaises(dg.Refused):self.g.freeze_candidate('gf',1,self.observed['sha'],self.observed['tree'])
    def test_assessment_candidate_mutation_before_call(self):
        self.g.freeze_candidate('gf',1,self.observed['sha'],self.observed['tree']);a=self.reserve('independent-assessment');self.observed['sha']='d'*40
        with self.assertRaises(dg.Refused):self.authorize(a)
    def test_assessment_candidate_mutation_before_result(self):
        self.g.freeze_candidate('gf',1,self.observed['sha'],self.observed['tree']);a=self.reserve('independent-assessment');self.start(a);self.observed['tree']='d'*40
        with self.assertRaises(dg.Refused):self.g.record_result(a['id'],self.result(a))
    def test_result_identity(self):
        a=self.reserve();self.start(a);r=self.result(a);r['actorId']='other'
        with self.assertRaises(dg.Refused):self.g.record_result(a['id'],r)
    def test_fresh_actor_identity(self):
        a=self.reserve();self.start(a);self.g.record_result(a['id'],self.result(a));b=self.reserve();self.authorize(b)
        with self.assertRaises(dg.Refused):self.g.record_start(b['id'],'different-call','actor-'+a['id'])
    def test_staged_input_tamper(self):
        a=self.reserve();self.input.write_text('changed')
        with self.assertRaises(dg.Refused):self.authorize(a)
    def test_runtime_input_tamper(self):
        f=self.root/'runtime.bin';f.write_bytes(b'bound');a=self.reserve(input_files=[f]);f.write_bytes(b'changed')
        with self.assertRaises(dg.Refused):self.authorize(a)
    def test_audited_root_collision(self):
        with self.assertRaises(dg.Refused):self.reserve(audited_roots=[self.root])
        self.assertEqual(json.loads(self.g.path.read_text())['actions'],{})
    def test_serialized_total_quota_across_gate_instances(self):
        self.grant['maxActivations']=1
        # State digest represents the frozen grant, so create a separate root for this fixture.
        q=self.root/'quota';q.mkdir();repo=q/'repo';repo.mkdir();prompt=q/'prompt';prompt.write_text('mock')
        inp=q/'task';inp.write_text('released');manifest=q/'manifest';manifest.write_text(json.dumps({'throughTask':1,'condition':'greenfield','inputs':[{'path':'task','sha256':dg.digest(inp)}]}))
        one=dg.Gate(q,self.grant,self.clock);one.register_cell('gf',repo,'greenfield');one.begin_task('gf',1,manifest,dg.digest(manifest));two=dg.Gate(q,self.grant,self.clock)
        def attempt(g):
            try:g.reserve('gf',1,'primary',prompt,180,10000);return True
            except dg.Refused:return False
        with concurrent.futures.ThreadPoolExecutor(2) as pool:out=list(pool.map(attempt,[one,two]))
        self.assertEqual(sorted(out),[False,True]);self.assertEqual(len(json.loads(one.path.read_text())['actions']),1)
    def test_maximum_two_active_cells(self):
        root=self.root/'multiple';root.mkdir();grant=copy.deepcopy(self.grant);grant['cells']=['one','two','three'];g=dg.Gate(root,grant,self.clock)
        prompt=root/'prompt';prompt.write_text('mock only');f=root/'task';f.write_text('released');m=root/'manifest';m.write_text(json.dumps({'throughTask':1,'condition':'greenfield','inputs':[{'path':'task','sha256':dg.digest(f)}]}))
        for cell in grant['cells']:
            repo=root/cell;repo.mkdir();g.register_cell(cell,repo,'greenfield');g.begin_task(cell,1,m,dg.digest(m))
        for cell in ['one','two']:g.reserve(cell,1,'primary',prompt,180,10000)
        with self.assertRaises(dg.Refused):g.reserve('three',1,'primary',prompt,180,10000)
    def test_concurrency_and_no_quota_refund(self):
        actions=[self.reserve() for _ in range(4)]
        with self.assertRaises(dg.Refused):self.reserve()
        self.g.mark_terminal_incomplete(actions[0]['id'],[]);self.reserve();self.assertEqual(len(json.loads(self.g.path.read_text())['actions']),5)
    def test_task_twelve_activation_cap(self):
        for _ in range(12):
            a=self.reserve();self.g.mark_terminal_incomplete(a['id'],[])
        with self.assertRaises(dg.Refused):self.reserve()
    def test_no_task_deadline_reset(self):
        with self.assertRaises(dg.Refused):self.begin(1)
    def test_next_task_requires_pass(self):
        with self.assertRaises(dg.Refused):self.begin(2)
    def test_task6_requires_bound_independent_equivalence(self):
        self.finish_task(1)
        for task in range(2,6):self.begin(task);self.finish_task(task)
        with self.assertRaises(dg.Refused):self.begin(6)
        receipt=self.root/'equivalence.json';receipt.write_text(json.dumps({'passed':True,'peerEquivalent':True,'candidateSha':self.observed['sha'],'invariantId':'opaque-private-id','peerCandidateShas':['e'*40],'independentEvaluatorActorId':'separate-evaluator-fixture'}))
        self.begin(6,equivalence={'path':str(receipt),'sha256':dg.digest(receipt)})
    def test_current_grant_cannot_authorize_actors(self):
        p=pathlib.Path(__file__).resolve().parents[1]/'evidence/authorization-grant.json';real=json.loads(p.read_text())
        self.assertEqual(real['actualActorActivations'],0)
        self.grant['executionAllowed']=False
        # Reopening durable state with changed authorization is rejected, rather than resetting it.
        g=dg.Gate(self.root,self.grant,self.clock)
        with self.assertRaises(dg.Refused):g.reserve('gf',1,'primary',self.prompt,180,10000)
        zero=copy.deepcopy(self.grant);zero.update(key=real['key'],maxActivations=real['actualActorActivations'],executionAllowed=False,admissionComplete=False,cells=[])
        fresh=dg.Gate(self.root/'real-zero-config',zero,self.clock)
        with self.assertRaisesRegex(dg.Refused,'no execution grant'):fresh.reserve('unallocated',1,'primary',self.prompt,180,10000)

class Deadlines(unittest.TestCase):
    def test_bridge_cannot_extend_actual_runner(self):
        d=role_deadline(now=100,task_start=100,trial_end=7300,grant_end=9000,runner_seconds=180,bridge_end=680,role_seconds=600)
        self.assertEqual(d['effectiveEnd'],280)
    def test_assessment_reserve(self):
        with self.assertRaises(ValueError):role_deadline(now=1001,task_start=100,trial_end=7300,grant_end=9000,runner_seconds=600,bridge_end=3000,role_seconds=600,assessment=True)

if __name__=='__main__':unittest.main(verbosity=2)
