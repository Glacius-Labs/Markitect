"""Offline final-assessor independence and frozen-copy boundaries."""
from datetime import datetime,timedelta,timezone
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
import final_assessor

class FinalAssessorTests(unittest.TestCase):
    def test_fresh_read_only_session_sees_exact_frozen_copy(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);candidate=root/"frozen";candidate.mkdir();(candidate/"README.md").write_bytes(b"public requirements\n")
            audit=root/"audit";audit.mkdir()
            choice={"case":"readinglog","config":{"model":"gpt-6-luna","effort":"high","runtimeOptions":{"sandbox":"workspace-write","approvalPolicy":"never","memoryEnabled":False,"windowsSandbox":"mxc"}}}
            with patch("conventional.backends.run",return_value={"state":"completed"}) as native:
                result=final_assessor.assess({"id":"fixture"},choice,Path(sys.executable),candidate,audit,datetime.now(timezone.utc)+timedelta(seconds=30))
            spec,prompt,resume,emit,cancel=native.call_args.args
            self.assertIsNone(resume)
            self.assertIn("agents.enabled=false",spec["command"])
            self.assertIn("features.multi_agent=false",spec["command"])
            self.assertEqual(spec["runtimeOptions"]["sandbox"],"read-only")
            self.assertEqual(spec["runtimeOptions"]["nativeMaxConcurrentAgents"],1)
            self.assertEqual((Path(spec["cwd"])/"README.md").read_bytes(),b"public requirements\n")
            self.assertNotIn(str(audit),prompt)
            self.assertTrue(result["candidateFilesUnchanged"])
            self.assertFalse((candidate/".git").exists())
    def test_expired_case_cannot_start_final_model(self):
        with patch("conventional.backends.run") as native:
            result=final_assessor.assess({}, {}, None,None,None,datetime.now(timezone.utc)-timedelta(seconds=1))
        self.assertEqual(result["state"],"NOT RUN");native.assert_not_called()
