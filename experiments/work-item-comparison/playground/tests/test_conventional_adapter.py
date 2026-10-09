"""Provider-free contract and setup fixtures for the Conventional adapter."""
from __future__ import annotations

from datetime import datetime, timedelta, timezone
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

PLAYGROUND = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(PLAYGROUND))
from conventional.adapter import ConventionalAdapter  # noqa: E402


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_json(path: Path, value: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + "\n", encoding="utf-8")


class FakeService:
    def __init__(self, config: dict):
        self.config = config
        self.calls = []
        self.run_folder = Path(config["audit"]) / "service-run"
        self.run_folder.mkdir(exist_ok=True)

    def _record(self, run_id):
        return self.run_folder, {"runId": run_id}

    def _admit(self):
        return Path("."), {**self.config, "cwd": self.config["cwd"]}, self.config["order"], {}

    def start(self, prompt):
        self.calls.append(("start", prompt))
        return {"state": "starting", "runId": "run-start"}

    def resume(self, run_id, prompt):
        self.calls.append(("resume", run_id, prompt))
        return {"state": "running", "runId": "run-next", "nativeSessionId": "native"}

    def status(self, run_id):
        return {"state": "completed", "runId": run_id, "taskAssessment": "NOT RUN",
                "humanAcceptance": "not established", "controllerOwnership": "this_process"}

    def cancel(self, run_id):
        return {"state": "running", "runId": run_id, "cancelRequested": True}

    def close(self):
        return {"closed": True}


class ConventionalAdapterTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.repo = self.root / "repo"
        self.audit = self.root / "audit"
        self.repo.mkdir()
        self.audit.mkdir()
        (self.repo / ".study").mkdir()
        (self.repo / ".study/run-id").write_text("fixture-run\n", encoding="ascii")
        write_json(self.audit / "run.json", {"method": "Conventional", "runId": "fixture-run",
                   "repoPath": str(self.repo), "auditPath": str(self.audit)})
        (self.repo / "AGENTS.md").write_text("# Shared public rules\n", encoding="utf-8")
        subprocess.run(["git", "-C", str(self.repo), "init", "-b", "main"], check=True, capture_output=True)
        subprocess.run(["git", "-C", str(self.repo), "-c", "user.name=Fixture", "-c",
                        "user.email=fixture@example.invalid", "add", "AGENTS.md", ".study/run-id"],
                       check=True, capture_output=True)
        subprocess.run(["git", "-C", str(self.repo), "-c", "user.name=Fixture", "-c",
                        "user.email=fixture@example.invalid", "commit", "-m", "seed"],
                       check=True, capture_output=True)
        self.config = {"schema": 1, "execution_authorized": True, "backend": "codex-cli",
                       "cwd": str(self.repo), "audit": str(self.audit), "model": "fixture-model",
                       "effort": "high", "runtimeOptions": {"sandbox": "workspace-write",
                       "approvalPolicy": "never", "memoryEnabled": False}}
        self.service = FakeService(self.config)
        self.adapter = ConventionalAdapter.__new__(ConventionalAdapter)
        self.adapter.config_path = self.root / "config.json"
        self.adapter._service = self.service
        self.adapter._runtime_probe = None
        self.adapter._config = self.config
        self.adapter._source_root = PLAYGROUND
        self.adapter._setup_context = None
        self.adapter._setup_receipt = None
        self.adapter._closed = False
        self.adapter._manifest_source_pins = None

    def setup_context(self):
        return {"schema": 1, "method": "Conventional", "case": "fixture", "station": "S1",
                "repoPath": str(self.repo), "auditPath": str(self.audit),
                "profile": {"model": "fixture-model", "effort": "high", "nativeBackend": "codex-cli"},
                "setupSourceRoot": str(PLAYGROUND / "public/conventional")}

    def test_descriptor_uses_stable_conventional_identity_and_manifest_pins(self):
        source = PLAYGROUND / "conventional/adapter.py"
        pins = {str(source.resolve()): digest(source)}
        self.adapter.bind_manifest_source_pins(pins)
        description = self.adapter.describe()
        self.assertEqual((description["schema"], description["id"], description["method"]),
                         (1, "conventional", "Conventional"))
        self.assertEqual(description["sourcePins"], pins)
        self.assertEqual(description["runtime"]["owner"], "adapter")

    def test_generic_manifest_loader_imports_pinned_conventional_package(self):
        source_root = PLAYGROUND / "conventional"
        sources = [source_root / "__init__.py", source_root / "adapter.py",
                   source_root / "service.py", source_root / "backends.py"]
        pins = {str(path.resolve()): digest(path) for path in sources}
        config_path = self.root / "generic-config.json"
        write_json(config_path, {"schema": 1, "execution_authorized": False,
                                 "backend": "codex-cli", "filePins": {
                                     str((source_root / "adapter.py").resolve()): digest(source_root / "adapter.py")}})
        manifest_path = self.root / "adapter-manifest.json"
        write_json(manifest_path, {"schema": 1, "factory": "conventional.adapter:ConventionalAdapter",
                                   "configPath": str(config_path.resolve()), "sourcePins": pins})
        script = (
            "import json,sys; from adapters.loader import load_adapter; "
            "a=load_adapter(sys.argv[1]); print(json.dumps(a.describe()))"
        )
        result = subprocess.run([sys.executable, "-B", "-c", script, str(manifest_path)],
                                cwd=str(PLAYGROUND), capture_output=True, timeout=30, check=False)
        self.assertEqual(result.returncode, 0, result.stderr.decode("utf-8", "replace"))
        description = json.loads(result.stdout.decode("utf-8"))
        self.assertEqual(description["id"], "conventional")
        self.assertEqual(description["sourcePins"], pins)

    def test_setup_installs_fragment_and_profile_with_guarded_commit_and_is_idempotent(self):
        first = self.adapter.setup(self.setup_context())
        self.assertEqual(first["state"], "ready", first)
        receipt = json.loads((self.audit / "adapter-setup.json").read_text(encoding="utf-8"))
        self.assertEqual(receipt["method"], "Conventional")
        self.assertTrue(receipt["setupCommit"])
        self.assertTrue((self.repo / ".study/runtime-profile.json").is_file())
        agents = (self.repo / "AGENTS.md").read_text(encoding="utf-8")
        self.assertIn("playground-adapter:conventional:v1", agents)
        second = self.adapter.setup(self.setup_context())
        self.assertEqual(second["state"], "ready", second)
        self.assertEqual(second["receipt"], receipt)

    def test_setup_refuses_profile_that_does_not_match_frozen_config(self):
        context = self.setup_context()
        context["profile"]["model"] = "different-model"
        result = self.adapter.setup(context)
        self.assertEqual(result["state"], "blocked")
        self.assertFalse((self.repo / ".study/runtime-profile.json").exists())

    def test_cli_readiness_does_not_claim_from_version_or_unauthorized_probe(self):
        self.config["order"] = {"runtimeChecksAuthorized": False}
        result = self.adapter.ensure_runtime()
        self.assertEqual(result["state"], "blocked")
        self.assertIn("does not authorize", result["reason"])

    def test_cli_no_model_probe_requires_exact_external_order_authorization(self):
        self.config["order"] = {"runtimeChecksAuthorized": True}
        observed = {"protocolReady": True, "toolReady": True, "sandboxReady": True,
                    "nativeTurnStarted": False}
        self.adapter._runtime_probe = lambda spec: observed
        result = self.adapter.ensure_runtime()
        self.assertEqual(result["state"], "ready")
        self.assertFalse(result["runtime"]["nativeTurnStarted"])

    def test_app_server_probe_uses_owned_stdio_initialize_and_starts_no_turn(self):
        fake = self.root / "fake_app_server.py"
        fake.write_text(
            "import json,sys\n"
            "for line in sys.stdin:\n"
            " m=json.loads(line)\n"
            " if m.get('method') == 'initialize':\n"
            "  print(json.dumps({'id':m['id'],'result':{'serverInfo':{'name':'fixture'},'capabilities':{}}}),flush=True)\n"
            "  break\n"
            "for line in sys.stdin:\n"
            " if json.loads(line).get('method') == 'initialized': break\n",
            encoding="utf-8")
        self.config["order"] = {"runtimeChecksAuthorized": True, "maxRuntimeStartupChecks": 4}
        spec = {"command": [sys.executable, str(fake)], "cwd": str(self.root), "model": "fixture-model",
                "effort": "high", "runtimeOptions": self.config["runtimeOptions"], "requestTimeoutSeconds": 3}
        result = self.adapter._probe_app_server(spec, self.config["order"],
                                                {"configSha256": "fixture-config", "orderSha256": "fixture-order",
                                                 "runId": "fixture-run"})
        self.assertTrue(result["protocolReady"], result)
        self.assertEqual(result["toolReady"], "unknown")
        self.assertFalse(result["nativeTurnStarted"])
        self.assertEqual(result["owner"], "adapter")
        self.assertTrue(result["processTerminalConfirmed"])
        self.assertFalse((self.audit / "adapter-runtime-checks/active.json").exists())
        checks = list((self.audit / "adapter-runtime-checks").glob("check-*/receipt.json"))
        self.assertEqual(len(checks), 1)
        receipt = json.loads(checks[0].read_text(encoding="utf-8"))
        self.assertEqual(receipt["state"], "finished")

    def test_cli_sandbox_probe_is_bounded_native_argv_and_records_startup_check(self):
        self.config["order"] = {"runtimeChecksAuthorized": True, "maxRuntimeStartupChecks": 4}
        options = {**self.config["runtimeOptions"], "windowsSandbox": "mxc"}
        spec = {"command": ["C:/native/codex.exe"], "cwd": str(self.repo), "runtimeOptions": options}

        class CompletedNativeSandbox:
            pid = 7241
            returncode = 0

            def communicate(self, timeout):
                return b"PLAYGROUND_WRITE_READ_OK\r\nPLAYGROUND_REPO_GIT_READ_OK\r\nPLAYGROUND_SANDBOX_PROBE_OK\r\n", b""

            def poll(self):
                return self.returncode

            def kill(self):
                raise AssertionError("successful fixture must not be killed")

        with patch("conventional.adapter.subprocess.Popen", return_value=CompletedNativeSandbox()) as popen:
            result = self.adapter._probe_native_sandbox(
                spec, self.config["order"],
                {"configSha256": "cfg", "orderSha256": "order", "runId": "fixture-run"},
                windows_host=True)
        self.assertTrue(result["sandboxReady"], result)
        self.assertTrue(result["writeReadReady"])
        self.assertTrue(result["gitCommandReady"])
        self.assertFalse(result["nativeTurnStarted"])
        argv = popen.call_args.args[0]
        self.assertEqual(argv[1:7], ["-c", 'windows.sandbox="mxc"', "sandbox",
                                     "--include-managed-config", "--permission-profile", ":workspace"])
        self.assertIn("git -C", " ".join(argv))
        self.assertIn("status --porcelain", " ".join(argv))
        self.assertIsNone(result["gitMetadataReady"])
        self.assertEqual(result["directMetadataWriteProbe"], "not performed")
        self.assertNotIn("metadata readiness", " ".join(argv))
        self.assertNotIn("playground-runtime-probe-", " ".join(argv).split("git -C",1)[1])
        self.assertNotIn("git init --quiet", " ".join(argv))
        self.assertNotIn("PLAYGROUND_REPO_GIT_WRITE_BLOCKED", " ".join(argv))
        check = next((self.audit / "adapter-runtime-checks").glob("check-*/receipt.json"))
        stored = json.loads(check.read_text(encoding="utf-8"))
        self.assertEqual(stored["state"], "finished")
        self.assertEqual(stored["observed"]["pid"], 7241)

    def test_scratch_success_cannot_claim_real_repository_git_readiness(self):
        self.config["order"]={"runtimeChecksAuthorized":True}
        basic={"sandboxReady":True,"writeReadReady":True,"gitCommandReady":True,"gitMetadataReady":False}
        with patch.object(self.adapter,"_probe_cli_version",return_value={"versionReady":True}), patch.object(self.adapter,"_probe_native_sandbox",return_value=basic):
            result=self.adapter.ensure_runtime()
        self.assertEqual(result["state"],"blocked")
        self.assertEqual(result["runtime"]["gitMergeReadiness"],"blocked protected metadata")
        self.assertFalse(result["runtime"]["actualGitApprovalObserved"])

    def test_scoped_broker_readiness_remains_conditional_before_real_approval(self):
        self.config["backend"]="codex-app-server"
        self.config["runtimeOptions"].update(approvalPolicy="on-request",scopedGitApproval=True)
        self.config["order"]={"runtimeChecksAuthorized":True}
        basic={"sandboxReady":True,"writeReadReady":True,"gitMetadataReady":False}
        with patch.object(self.adapter,"_probe_app_server",return_value={"protocolReady":True}), patch.object(self.adapter,"_probe_native_sandbox",return_value=basic):
            result=self.adapter.ensure_runtime()
        self.assertEqual(result["state"],"ready")
        self.assertIn("conditional",result["runtime"]["gitMergeReadiness"])
        self.assertFalse(result["runtime"]["actualGitApprovalObserved"])

    def test_workspace_readiness_requires_no_git_metadata_write_or_broker(self):
        self.config["backend"]="codex-app-server"
        self.config["order"]={"runtimeChecksAuthorized":True}
        self.adapter._setup_context={"completionTarget":"workspace_snapshot"}
        basic={"sandboxReady":True,"writeReadReady":True,"gitMetadataReady":None}
        with patch.object(self.adapter,"_probe_app_server",return_value={"protocolReady":True}), patch.object(self.adapter,"_probe_native_sandbox",return_value=basic):
            result=self.adapter.ensure_runtime()
        self.assertEqual(result["state"],"ready")
        self.assertEqual(result["runtime"]["gitMergeReadiness"],"not required for workspace_snapshot")
        self.assertFalse(result["runtime"]["scopedGitApprovalConfigured"])

    def test_lifecycle_facade_maps_service_without_task_scoring(self):
        started = self.adapter.start("ordinary prompt")
        self.assertEqual((started["state"], started["runId"]), ("accepted", "run-start"))
        resumed = self.adapter.resume("run-start", "next prompt")
        self.assertEqual((resumed["state"], resumed["runId"]), ("accepted", "run-next"))
        status = self.adapter.status("run-next")
        self.assertEqual(status["state"], "completed")
        self.assertEqual(status["taskAssessment"], "NOT RUN")
        self.assertIsNone(status["runtimeFailure"])
        self.assertEqual(self.adapter.cancel("run-next")["cancelRequested"], True)
        self.assertEqual(self.adapter.close()["state"], "closed")

    def test_status_classifies_only_exact_failure_in_the_requested_service_event_log(self):
        exact = b'{"sequence":1,"event":{"raw":"helper_unknown_error: setup refresh had errors"}}\n'
        self.service.run_folder.joinpath("events.jsonl").write_bytes(exact)
        result = self.adapter.status("run-owned")
        self.assertEqual(result["runtimeFailure"], {
            "kind": "native_tool_setup", "reason": "helper_unknown_error: setup refresh had errors"})
        other = self.root / "unrelated" / "events.jsonl"
        other.parent.mkdir()
        other.write_bytes(exact)
        self.service.run_folder.joinpath("events.jsonl").write_bytes(b'{"event":{"raw":"helper unknown error"}}\n')
        self.assertIsNone(self.adapter.status("run-owned")["runtimeFailure"])


if __name__ == "__main__":
    unittest.main()
