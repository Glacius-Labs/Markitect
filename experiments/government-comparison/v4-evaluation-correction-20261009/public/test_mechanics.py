"""Focused offline fixtures for known DTO, deadline and own-handle risks."""
import ast, copy, io, json, pathlib, tempfile, unittest
from unittest.mock import patch
import dto_preflight as d
from deadlines import role_deadline
from owned_transport import Server
from release import release,staged_checks

class DTOTests(unittest.TestCase):
    def setUp(self):
        self.i={'apiVersion':d.API,'runId':'a'*32,'nonce':'b'*32,'inputDigest':'c'*64,'request':{'role':'executor','scopeIds':['scope-1'],'policyIds':['policy-1'],'artifacts':[{'path':'src/a.cs'}]}}
        self.r={k:self.i[k] for k in ['apiVersion','runId','nonce','inputDigest']}
        self.r.update(role='executor',outcome='proposed',candidateFiles=[{'path':'src/a.cs','mode':'0644','content':'// synthetic fixture'}],evidenceRefs=['scope-1','src/a.cs'],verifierObservations=[],uncertainty=[],usage=None)
    def check(self,r=None,i=None):return d.validate(json.dumps(i or self.i).encode(),json.dumps(r or self.r).encode(),['src/a.cs'])
    def test_valid_bytes_and_modes_no_semantic_acceptance(self):
        for mode in ['0644','0600','0755']:
            r=copy.deepcopy(self.r);r['candidateFiles'][0]['mode']=mode
            self.assertFalse(self.check(r)['semanticAcceptance'])
        r=copy.deepcopy(self.r);r['candidateFiles'][0]['mode']='100644'
        with self.assertRaises(d.Invalid):self.check(r)
    def test_nonce_and_evidence_never_filled(self):
        for key,value in [('nonce','d'*32),('evidenceRefs',['not-supplied'])]:
            r=copy.deepcopy(self.r);r[key]=value
            with self.assertRaises(d.Invalid):self.check(r)
    def test_alias_traversal_and_allowlist(self):
        for name in ['../a.cs','src/../a.cs','src\\a.cs','C:/a.cs','src/b.cs','src/.git/config','src/CON.txt','src/a.cs\n']:
            r=copy.deepcopy(self.r);r['candidateFiles'][0]['path']=name
            with self.assertRaises(d.Invalid):self.check(r)
        r=copy.deepcopy(self.r);r['candidateFiles'].append(copy.deepcopy(r['candidateFiles'][0]))
        with self.assertRaises(d.Invalid):self.check(r)
    def test_missing_oversize_foreign_role_and_unknown(self):
        for mutate in [lambda r:r.pop('uncertainty'),lambda r:r.update(evidenceRefs=['scope-1']*129),lambda r:r.update(candidateJson=None),lambda r:r.update(extra='bad')]:
            r=copy.deepcopy(self.r);mutate(r)
            with self.assertRaises(d.Invalid):self.check(r)
    def test_closed_json_and_opaque_inference_case(self):
        with self.assertRaises(d.Invalid):d.strict(b'{"runId":"x","RunID":"y"}')
        with self.assertRaises(d.Invalid):d.strict(b'{} {}')
        with self.assertRaises(d.Invalid):d.strict(b'{"x":NaN}')
        i=copy.deepcopy(self.i);i['request']['role']='infer';r=copy.deepcopy(self.r);r.update(role='infer',candidateFiles=[],candidateJson={'Name':1,'name':2})
        self.assertTrue(self.check(r,i)['productValidationRequired'])
    def test_usage_requires_real_receipt_shape(self):
        for usage in [{'source':'estimated','inputTokens':1},{'source':'provider-reported','inputTokens':True},{'source':'provider-reported'}]:
            r=copy.deepcopy(self.r);r['usage']=usage
            with self.assertRaises(d.Invalid):self.check(r)
    def test_atomic_publish_preserves_original_and_refuses_audited_root(self):
        with tempfile.TemporaryDirectory() as tmp:
            p=pathlib.Path(tmp);audit=p/'audit';audit.mkdir();inp=p/'inv.json';response=p/'raw.json';inp.write_bytes(json.dumps(self.i).encode());raw=json.dumps(self.r,indent=3).encode();response.write_bytes(raw)
            with self.assertRaises(d.Invalid):d.publish(inp,response,audit/'response.json',['src/a.cs'],[audit])
            with self.assertRaises(d.Invalid):d.publish(inp,response,p/'out.json',['src/a.cs'],[])
            d.publish(inp,response,p/'out.json',['src/a.cs'],[audit]);self.assertEqual((p/'out.json').read_bytes(),raw)
            with self.assertRaises(d.Invalid):d.publish(inp,response,p/'out.json',['src/a.cs'],[audit])

class DeadlineTests(unittest.TestCase):
    def args(self):return dict(now=1100,task_start=1000,trial_end=8200,grant_end=10000,runner_seconds=180,bridge_end=1700,role_seconds=580)
    def test_actual_runner_minimum_and_interrupt_margin(self):
        r=role_deadline(**self.args());self.assertEqual(r['effectiveEnd'],1280);self.assertEqual(r['interruptRequestBy'],1270)
    def test_implementation_cannot_consume_assessment_reserve(self):
        a=self.args();a.update(now=1800,bridge_end=2200);self.assertEqual(role_deadline(**a)['effectiveEnd'],1840)
        a['now']=1830
        with self.assertRaises(ValueError):role_deadline(**a)
    def test_assessment_late_or_nonfinite_refused(self):
        a=self.args();a.update(now=1920,bridge_end=2200)
        with self.assertRaises(ValueError):role_deadline(**a,assessment=True)
        a=self.args();a['runner_seconds']=float('inf')
        with self.assertRaises(ValueError):role_deadline(**a)

class FakeChild:
    pid=123
    def __init__(self,line,ended=False):self.stdout=io.StringIO(line);self.returncode=0 if ended else None;self.terminated=False
    def poll(self):return self.returncode
    def terminate(self):self.terminated=True;self.returncode=1
    def wait(self,timeout=None):return self.returncode
    def kill(self):self.returncode=1

class TransportTests(unittest.TestCase):
    def test_own_stdout_url_and_owned_cleanup_no_process_launched(self):
        with tempfile.TemporaryDirectory() as tmp:
            child=FakeChild('Now listening on: http://127.0.0.1:12345\n')
            with patch('owned_transport.subprocess.Popen',return_value=child) as popen:
                with Server('fixture.dll','fixture.db',tmp,'owned','2099-01-01T00:00:00Z') as s:self.assertEqual(s.url,'http://127.0.0.1:12345')
                self.assertTrue(child.terminated);self.assertEqual(popen.call_args.kwargs['env']['ASPNETCORE_URLS'],'http://127.0.0.1:0')
            records=json.loads((pathlib.Path(tmp)/'owned-server-starts.json').read_text());self.assertEqual(len(records),1);self.assertIn('stoppedUtc',records[0])
    def test_10013_and_missing_url_stop_without_retry(self):
        for line,ended in [('SocketException10013\n',False),('unrelated text\n',True)]:
            with tempfile.TemporaryDirectory() as tmp:
                child=FakeChild(line,ended)
                with patch('owned_transport.subprocess.Popen',return_value=child) as popen:
                    with self.assertRaises(RuntimeError):
                        with Server('fixture.dll','fixture.db',tmp,'owned','2099-01-01T00:00:00Z'):pass
                    self.assertEqual(popen.call_count,1)
    def test_start_cap_and_expired_deadline_never_spawn(self):
        with tempfile.TemporaryDirectory() as tmp:
            (pathlib.Path(tmp)/'owned-server-starts.json').write_text(json.dumps([{}]*12))
            with patch('owned_transport.subprocess.Popen') as popen:
                with self.assertRaises(AssertionError):
                    with Server('fixture.dll','fixture.db',tmp,'cap','2099-01-01T00:00:00Z'):pass
                with self.assertRaises(AssertionError):
                    with Server('fixture.dll','fixture.db',tmp,'expired','2000-01-01T00:00:00Z'):pass
                popen.assert_not_called()

class StagingTests(unittest.TestCase):
    def test_each_cutoff_excludes_future_behavior_and_cards(self):
        for condition in ['greenfield','brownfield']:
            for cutoff in range(1,7):
                source=staged_checks(cutoff,condition);compile(source,'released-checks','exec')
                if cutoff<2:self.assertNotIn('/inventory',source)
                if cutoff<3:self.assertNotIn('--verify-restart',source)
                if cutoff<4:self.assertNotIn('/cancel',source);self.assertNotIn('/fulfill',source)
                if cutoff<5:self.assertNotIn('new-limit-',source)
                if condition=='greenfield' and cutoff<5:self.assertNotIn('/legacy/orders',source)
                with tempfile.TemporaryDirectory() as tmp:
                    root=pathlib.Path(tmp);repo=root/'repo';repo.mkdir();out=root/'inputs'
                    result=release(cutoff,out,condition,repo)
                    self.assertEqual(len(list(out.glob('0*.md'))),cutoff)
                    self.assertEqual(result['throughTask'],cutoff)
                    self.assertFalse(any('private' in x['path'] for x in result['inputs']))
    def test_staging_refuses_actor_tree_and_bad_cutoff(self):
        with tempfile.TemporaryDirectory() as tmp:
            p=pathlib.Path(tmp)
            with self.assertRaises(ValueError):release(1,p/'inputs','greenfield',p)
        with self.assertRaises(ValueError):staged_checks(7,'greenfield')
