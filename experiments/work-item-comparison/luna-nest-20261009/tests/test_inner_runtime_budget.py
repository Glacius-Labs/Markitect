"""Six targeted offline checks. Children are Python byte/process fixtures only."""
from pathlib import Path
import hashlib
import json
import os
import sys
import tempfile
import unittest
from unittest.mock import patch
from datetime import timedelta

PUBLIC=Path(__file__).resolve().parents[1]/'public'
sys.path.insert(0,str(PUBLIC))
import nest
import app_ledger
import inner_runtime_budget as gate

FAKE = '''import sys,subprocess,time
raw=sys.stdin.buffer.read()
mode=sys.argv[1]
if mode=='tree':
 p=subprocess.Popen([sys.executable,'-B','-c','import time;time.sleep(30)'])
 print(p.pid,flush=True);time.sleep(30)
else:
 sys.stdout.buffer.write(raw);sys.stderr.buffer.write(b'fixture-stderr\\n')
 sys.exit(7 if mode=='fail' else 0)
'''


class MechanicalChecks(unittest.TestCase):
    def fixture(self, mode='echo'):
        root=Path(tempfile.mkdtemp(prefix='markitect-inner-offline-')).resolve()
        self.assertTrue(root.is_relative_to(Path(tempfile.gettempdir()).resolve()))
        state=root/'state';state.mkdir();repo=root/'repo';repo.mkdir()
        child=root/'fake_bytes.py';child.write_text(FAKE,encoding='utf-8')
        grant=root/'grant.json';start=app_ledger.now()
        nest.write(grant,{'executionAuthorized':True,'model':'gpt-6-luna','reasoning':'high',
                         'repo':str(repo),'preparedCommit':'OFFLINE-FAKE-NOT-A-CASE',
                         'notAfterUtc':(start+timedelta(seconds=10000)).isoformat()})
        parent={'reservation':1,'kind':'implementation','depth':1,'parent':None,'stationIndex':0,
                'status':'started','handle':'OFFLINE-OWN-PARENT','reservedUtc':start.isoformat(),
                'deadlineUtc':(start+timedelta(seconds=1350)).isoformat()}
        nest.write(state/'state.json',{'repo':str(repo),'preparedCommit':'OFFLINE-FAKE-NOT-A-CASE',
                                     'stationIndex':0,'status':'running','freeze':None})
        nest.write(state/'app-ledger.json',{'status':'active','startedUtc':start.isoformat(),
                                          'grantSha256':nest.sha(grant),'implementationClosed':False,
                                          'entries':[parent]})
        command=[sys.executable,'-B',str(child),mode]
        binding={'executionAuthorized':True,'stateDir':str(state),'grantPath':str(grant),
                 'grantSha256':nest.sha(grant),'phase':'implementation','parentReservation':1,
                 'parentHandle':'OFFLINE-OWN-PARENT','command':command,'maxOwnedWallSeconds':30,
                 'cleanupReserveSeconds':5,'runtimePins':[{'path':sys.executable,'sha256':nest.sha(sys.executable)},
                                                        {'path':str(child),'sha256':nest.sha(child)}]}
        return binding,state

    def ledger(self,state): return nest.read(state/'app-ledger.json')

    def setUp(self):
        self.env=patch.dict(os.environ,{'MARKITECT_AGENT_CONFIG_JSON':json.dumps({
            'model':'gpt-6-luna','modelOptions':{'model_reasoning_effort':'high'},'providerVersion':'OFFLINE'})})
        self.env.start()

    def tearDown(self): self.env.stop()

    def test_1_opaque_bytes_and_failed_requests(self):
        for mode,expected,status in [('echo',0,'returned'),('fail',7,'call-failed')]:
            b,s=self.fixture(mode);raw=b'  opaque UTF8 \xc3\xbc\x00\n'
            code,out,err,r=gate.run(b,raw,b['command'])
            self.assertEqual((code,out,err),(expected,raw,b'fixture-stderr\n'))
            self.assertEqual(self.ledger(s)['entries'][-1]['status'],status)
            self.assertEqual(len(self.ledger(s)['entries']),2)
            self.assertTrue(r['stop']['localTreeExitConfirmed'])
            self.assertEqual(r['argv'],b['command'])

    def test_2_shared_caps_closed_and_unknown_admission(self):
        for issue in ['72','four','closed','unknown']:
            b,s=self.fixture();l=self.ledger(s)
            if issue=='72':
                l['entries'] += [dict(l['entries'][0],reservation=n,status='returned',depth=2,kind='helper') for n in range(2,73)]
            elif issue=='four':
                l['entries'] += [dict(l['entries'][0],reservation=n,depth=2,kind='helper') for n in range(2,5)]
            elif issue=='closed': l['status']='closed'
            else: l['entries'].append(dict(l['entries'][0],reservation=2,status='unknown-stop',depth=2,kind='helper'))
            nest.write(s/'app-ledger.json',l)
            with self.assertRaises(ValueError): gate.run(b,b'x',b['command'])
            self.assertEqual(len(self.ledger(s)['entries']),len(l['entries']))

    def test_3_exact_parent_station_phase_and_final_clock(self):
        b,s=self.fixture();l=self.ledger(s);l['entries'][0]['status']='returned';nest.write(s/'app-ledger.json',l)
        with self.assertRaises(ValueError): gate.run(b,b'x',b['command'])
        b,s=self.fixture();state=nest.read(s/'state.json');state['stationIndex']=1;nest.write(s/'state.json',state)
        with self.assertRaises(ValueError): gate.run(b,b'x',b['command'])
        b,s=self.fixture();l=self.ledger(s);now=app_ledger.now();l['startedUtc']=(now-timedelta(seconds=5500)).isoformat()
        l['entries'][0].update(kind='assessment',reservedUtc=(now-timedelta(seconds=1150)).isoformat(),
                               deadlineUtc=(now+timedelta(seconds=45)).isoformat());l['implementationClosed']=True
        nest.write(s/'app-ledger.json',l);state=nest.read(s/'state.json');state['freeze']={'offline':True};nest.write(s/'state.json',state)
        b['phase']='assessment';code,_,_,r=gate.run(b,b'final-bytes',b['command'])
        self.assertEqual(code,0);self.assertEqual(self.ledger(s)['entries'][-1]['kind'],'assessment-helper')
        self.assertLessEqual(app_ledger.date(r['effectiveDeadlineUtc']),app_ledger.date(l['entries'][0]['deadlineUtc']))

    def test_4_deadline_kills_known_descendant_tree(self):
        b,s=self.fixture();b['cleanupReserveSeconds']=.2
        with self.assertRaises(ValueError): gate.run(b,b'no-model',b['command'])
        self.assertEqual(self.ledger(s)['entries'][-1]['status'],'call-failed')
        b,s=self.fixture();b['runtimePins']=b['runtimePins'][1:]
        with self.assertRaises(ValueError): gate.run(b,b'no-model',b['command'])
        self.assertEqual(self.ledger(s)['entries'][-1]['status'],'call-failed')
        b,s=self.fixture('tree');b.update(maxOwnedWallSeconds=6,cleanupReserveSeconds=5)
        code,out,_,r=gate.run(b,b'no-model',b['command'])
        self.assertEqual(code,124);self.assertTrue(out.strip().isdigit())
        self.assertTrue(r['stop']['localTreeExitConfirmed'])
        self.assertLessEqual(app_ledger.date(r['endedUtc']),app_ledger.date(r['effectiveDeadlineUtc']))
        self.assertEqual(self.ledger(s)['entries'][-1]['status'],'interrupted')

    def test_5_disabled_drift_and_profile_fail_closed(self):
        b,s=self.fixture();b['executionAuthorized']=False
        with self.assertRaises(ValueError): gate.run(b,b'x',b['command'])
        self.assertEqual(len(self.ledger(s)['entries']),1)
        b,s=self.fixture();b['runtimePins'][-1]['sha256']=gate.OLD_ADAPTER
        with self.assertRaises(ValueError): gate.run(b,b'x',b['command'])
        self.assertEqual(self.ledger(s)['entries'][-1]['status'],'call-failed')
        b,s=self.fixture()
        with patch.dict(os.environ,{'MARKITECT_AGENT_CONFIG_JSON':'{"model":"wrong"}'}):
            with self.assertRaises(ValueError): gate.run(b,b'x',b['command'])
        self.assertEqual(self.ledger(s)['entries'][-1]['status'],'call-failed')

    def test_6_unconfirmed_cleanup_blocks_next_request(self):
        b,s=self.fixture()
        with patch.object(gate.OwnedTree,'active',return_value=True),patch.object(gate.OwnedTree,'stop',return_value={'localTreeExitConfirmed':False,'offlineInjected':True}):
            code,_,_,_=gate.run(b,b'x',b['command'])
        self.assertEqual(code,125);self.assertEqual(self.ledger(s)['entries'][-1]['status'],'unknown-stop')
        with self.assertRaises(ValueError): gate.run(b,b'x',b['command'])


if __name__=='__main__': unittest.main(verbosity=2)
