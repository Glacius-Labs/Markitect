"""Focused offline App reservation checks; never calls native agent tools."""
import sys, tempfile, unittest
from pathlib import Path
from datetime import datetime, timedelta, timezone
from unittest.mock import patch
sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'public'))
import app_ledger as app
import nest

class AppReservations(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory(); self.state=Path(self.tmp.name)
        self.at=datetime(2026,10,9,6,45,tzinfo=timezone.utc)
        nest.write(self.state/'state.json',{'preparedCommit':'prepared','repo':'OWN_CASE','stationIndex':0})
        self.grant=self.state/'grant.json'
        nest.write(self.grant,{'executionAuthorized':True,'model':'gpt-6-luna','reasoning':'high',
                             'preparedCommit':'prepared','repo':'OWN_CASE','notAfterUtc':(self.at+timedelta(seconds=7400)).isoformat()})
        app.initialize(self.state)
        self.git=patch.object(nest,'git',side_effect=lambda repo,*args:'prepared' if args==('rev-parse','HEAD') else '')
        self.git.start()
    def tearDown(self): self.git.stop(); self.tmp.cleanup()
    def reserve(self,kind='implementation',**kw): return app.reserve(self.state,self.grant,kind,at=self.at,**kw)
    def test_disabled_full_window_and_grant_drift(self):
        g=nest.read(self.grant); g['executionAuthorized']=False; nest.write(self.grant,g)
        with self.assertRaises(ValueError): self.reserve()
        g['executionAuthorized']=True; g['notAfterUtc']=(self.at+timedelta(seconds=7199)).isoformat(); nest.write(self.grant,g)
        with self.assertRaises(ValueError): self.reserve()
        g['notAfterUtc']=(self.at+timedelta(seconds=7400)).isoformat(); nest.write(self.grant,g); self.reserve()
        g['extra']='drift'; nest.write(self.grant,g)
        with self.assertRaises(ValueError): self.reserve('helper')
    def test_attempts_handles_and_station_ceiling(self):
        a=self.reserve(); app.record(self.state,a['reservation'],'returned',handle='OWN')
        with self.assertRaises(ValueError): app.reserve(self.state,self.grant,'helper',at=self.at+timedelta(seconds=1330))
        b=self.reserve(continuation_handle='OWN'); app.record(self.state,b['reservation'],'call-failed')
        with self.assertRaises(ValueError): self.reserve()
        self.assertEqual(len(nest.read(self.state/'app-ledger.json')['entries']),2)
        with self.assertRaises(ValueError): self.reserve('helper',continuation_handle='FOREIGN')
    def test_cooperative_concurrency_depth_and_total_cap(self):
        a=self.reserve()
        with self.assertRaises(ValueError): self.reserve('helper',depth=3,parent=a['reservation'])
        with self.assertRaises(ValueError): self.reserve('helper',depth=2,parent=999)
        for _ in range(3): self.reserve('helper',depth=2,parent=a['reservation'])
        with self.assertRaises(ValueError): self.reserve('helper',depth=2,parent=a['reservation'])
        l=nest.read(self.state/'app-ledger.json'); l['entries']=[dict(l['entries'][0],reservation=i+1,status='call-failed') for i in range(72)]; nest.write(self.state/'app-ledger.json',l)
        with self.assertRaises(ValueError): self.reserve('helper')
    def test_terminal_policy_freeze_final_and_phase_deadline(self):
        a=self.reserve(); app.record(self.state,a['reservation'],'returned',handle='OWN')
        app.close_implementation(self.state,'policy blocker')
        with self.assertRaises(ValueError): self.reserve()
        s=nest.read(self.state/'state.json'); s['freeze']={'evaluationRepo':'OWN_FROZEN_MAIN'}; nest.write(self.state/'state.json',s)
        final=self.reserve('assessment'); self.assertEqual(app.date(final['deadlineUtc']),self.at+timedelta(seconds=1200))
        helper=self.reserve('assessment-helper',depth=2,parent=final['reservation'])
        app.record(self.state,helper['reservation'],'returned',handle='FINAL_HELPER')
        app.record(self.state,final['reservation'],'returned',handle='ASSESSOR')
        with self.assertRaises(ValueError): self.reserve('assessment')
        app.close(self.state,'NOT RUN')
        with self.assertRaises(ValueError): self.reserve('helper')

if __name__=='__main__': unittest.main()
