"""Offline final-assessor independence and frozen-copy boundaries."""
from datetime import datetime,timedelta,timezone
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
import final_assessor

class FinalAssessorTests(unittest.TestCase):
    @staticmethod
    def _write_initial_public(audit, contents):
        initial=audit/"initial-public";initial.mkdir(parents=True)
        for name,body in contents.items():
            path=initial/name;path.parent.mkdir(parents=True,exist_ok=True);path.write_bytes(body)
        manifest={str(path.relative_to(initial)).replace("\\","/"):final_assessor.hashlib.sha256(path.read_bytes()).hexdigest()
                  for path in sorted(initial.rglob("*")) if path.is_file()}
        binding={"schema":1,"kind":"prepared_public_workspace","fileManifest":manifest,
                 "fileManifestSha256":final_assessor.hashlib.sha256(
                     (json.dumps(manifest,sort_keys=True,separators=(",",":"))+"\n").encode("utf-8")).hexdigest()}
        (audit/"initial-public-binding.json").write_text(json.dumps(binding),encoding="utf-8")
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
    def test_workspace_snapshot_uses_declared_appserver_readonly_and_exact_candidate_pins(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            audit=root/"audit";audit.mkdir()
            self._write_initial_public(audit,{"README.md":b"frozen initial requirements\n",
                                              "QUALITY.md":b"frozen checks\n"})
            frozen=audit/"final-freeze";frozen.mkdir()
            (frozen/"snapshot.json").write_text(json.dumps({"completionTarget":"workspace_snapshot",
                "assessmentCandidate":{"path":"../workspace-snapshot","manifest":{
                    "app.py":final_assessor.hashlib.sha256(b"# fixture application\n").hexdigest(),
                    "AGENTS.md":final_assessor.hashlib.sha256(b"public rules\n").hexdigest()}}}),encoding="utf-8")
            candidate=frozen/"workspace-snapshot"
            candidate.mkdir()
            (candidate/"app.py").write_bytes(b"# fixture application\n")
            (candidate/"AGENTS.md").write_bytes(b"public rules\n")
            frozen_record=json.loads((frozen/"snapshot.json").read_text(encoding="utf-8"))
            frozen_record["assessmentCandidate"]["path"]="workspace-snapshot"
            (frozen/"snapshot.json").write_text(json.dumps(frozen_record),encoding="utf-8")
            choice={"case":"readinglog","config":{"backend":"codex-app-server","model":"gpt-6-luna","effort":"high",
                    "runtimeOptions":{"sandbox":"workspace-write","approvalPolicy":"on-request","memoryEnabled":False,
                    "windowsSandbox":"mxc","scopedGitApproval":True,"gitApprovalShell":{"fixture":True},
                    "allowLoginShell":False}}}
            with patch("conventional.backends.run",return_value={"state":"completed"}) as native:
                result=final_assessor.assess({"id":"snapshot-fixture"},choice,Path(sys.executable),candidate,audit,
                                             datetime.now(timezone.utc)+timedelta(seconds=30),
                                             completion_target="workspace_snapshot")
            spec,prompt,resume,emit,cancel=native.call_args.args
            binding=json.loads((audit/"independent-final-assessment"/"binding.json").read_text(encoding="utf-8"))
            self.assertEqual(spec["backend"],"codex-app-server")
            self.assertIsNone(resume)
            self.assertIn("agents.enabled=false",spec["command"])
            self.assertIn("features.multi_agent=false",spec["command"])
            self.assertEqual(spec["runtimeOptions"]["sandbox"],"read-only")
            self.assertEqual(spec["runtimeOptions"]["approvalPolicy"],"never")
            self.assertFalse(spec["runtimeOptions"]["scopedGitApproval"])
            self.assertNotIn("gitApprovalShell",spec["runtimeOptions"])
            assessed=Path(spec["cwd"])
            self.assertEqual((assessed/"app.py").read_bytes(),b"# fixture application\n")
            self.assertEqual(binding["completionTarget"],"workspace_snapshot")
            self.assertEqual(binding["assessmentCandidate"]["manifest"],binding["candidateFilePins"])
            self.assertIn("README.md",binding["frozenInitialPublicRequirements"]["manifest"])
            self.assertTrue(result["frozenInitialPublicRequirementsUnchanged"])
            self.assertTrue(result["initialPublicWorkspaceBindingUnchanged"])
            self.assertTrue(result["candidateFilesUnchanged"])
            self.assertFalse((assessed/".git").exists())
    def test_workspace_snapshot_final_assessment_never_falls_back_from_appserver(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);candidate=root/"candidate";candidate.mkdir();(candidate/"README.md").write_text("candidate")
            audit=root/"audit";audit.mkdir()
            choice={"case":"readinglog","config":{"backend":"codex-cli","model":"gpt-6-luna","effort":"high",
                    "runtimeOptions":{"sandbox":"workspace-write","approvalPolicy":"never","memoryEnabled":False}}}
            with patch("conventional.backends.run") as native:
                with self.assertRaisesRegex(ValueError,"declared AppServer backend"):
                    final_assessor.assess({"id":"snapshot-fixture"},choice,Path(sys.executable),candidate,audit,
                                          datetime.now(timezone.utc)+timedelta(seconds=30),
                                          completion_target="workspace_snapshot")
            native.assert_not_called()
    def test_workspace_snapshot_mutation_during_assessment_is_a_failure(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);audit=root/"audit";audit.mkdir()
            self._write_initial_public(audit,{"README.md":b"initial public requirements"})
            frozen=audit/"final-freeze";candidate=frozen/"worktree";candidate.mkdir(parents=True)
            (candidate/"app.py").write_text("original")
            bound={"app.py":final_assessor.hashlib.sha256(b"original").hexdigest()}
            (frozen/"snapshot.json").write_text(json.dumps({"completionTarget":"workspace_snapshot",
                "assessmentCandidate":{"path":"worktree","manifest":bound}}))
            choice={"case":"readinglog","config":{"backend":"codex-app-server","model":"gpt-6-luna","effort":"high",
                    "runtimeOptions":{"sandbox":"workspace-write","approvalPolicy":"on-request","memoryEnabled":False}}}
            def mutate(spec,prompt,resume,emit,cancel):
                (Path(spec["cwd"])/"app.py").write_text("modified")
                return {"state":"completed"}
            with patch("conventional.backends.run",side_effect=mutate):
                result=final_assessor.assess({"id":"snapshot-fixture"},choice,Path(sys.executable),candidate,audit,
                    datetime.now(timezone.utc)+timedelta(seconds=30),completion_target="workspace_snapshot")
            self.assertEqual(result["nativeState"],"completed")
            self.assertEqual(result["state"],"failed")
            self.assertFalse(result["candidateFilesUnchanged"])
    def test_expired_case_cannot_start_final_model(self):
        with patch("conventional.backends.run") as native:
            result=final_assessor.assess({}, {}, None,None,None,datetime.now(timezone.utc)-timedelta(seconds=1))
        self.assertEqual(result["state"],"NOT RUN");native.assert_not_called()
