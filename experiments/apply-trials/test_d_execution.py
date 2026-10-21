"""Deterministic local controls; no container engine or real provider is invoked.

    python -B -m unittest discover -s experiments/apply-trials -p test_d_execution.py
"""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

import d_series
from materialize import git, commit
import plan_arm


def series(root, order=None, seconds=60, starts=12):
    workspace = root / "writer"; workspace.mkdir()
    (workspace / "plan.md").write_text("# Requirements\n\nKeep conforming delivery unchanged.\n")
    git(workspace, "init", "-q"); commit(workspace, "Accepted requirements")
    head = git(workspace, "rev-parse", "HEAD").strip()
    manifest = root / "manifest.json"
    data = {"seriesId": "local-control", "order": order or [{"scenario": "A2", "arm": "plan"}],
            "scenarios": {"A2": {"arms": {"plan": {"head": head, "repo": str(workspace),
                "files": {"plan.md": d_series.sha(workspace / "plan.md")}}}}}}
    d_series.write_new(manifest, data)
    authorization = root / "authorization.json"
    d_series.write_new(authorization, {"seriesId": data["seriesId"], "manifestSha256": d_series.sha(manifest),
        "authorizationReference": "local simulated control only", "mode": "simulation",
        "maxStarts": starts, "maxSeconds": seconds})
    ledger = d_series.initialize(manifest, root / "ledgers", authorization)
    return workspace, ledger, manifest, authorization


def profile():
    return {"formatVersion": 1, "mode": "provider", "protocol": "codex-exec-jsonl", "provider": "codex",
            "model": "frozen-model", "effort": "high", "engineExecutable": sys.executable,
            "engineSha256": d_series.sha(sys.executable), "imageID": "sha256:" + "a" * 64,
            "network": "none", "singleAgentTurnBoundVerified": True,
            "argv": ["/usr/local/bin/codex", "exec", "--json", "--model", "{model}", "-c", 'model_reasoning_effort="{effort}"',
                     "--disable", "multi_agent", "--disable", "apps", "--disable", "memories", "-"],
            "limits": {"milliCPUs": 2000, "memoryBytes": 4096, "processes": 32, "maxOutputBytes": 65536}}


class DExecution(unittest.TestCase):
    def test_local_success_failure_timeout_and_missing_executable_are_separate_starts(self):
        commands = [("success", [sys.executable, "-B", "-c", "print('First completion')"], "completed", None),
                    ("failure", [sys.executable, "-B", "-c", "raise SystemExit(7)"], "error", None),
                    ("timeout", [sys.executable, "-B", "-c", "import time; time.sleep(30)"], "timeout", 0.1),
                    ("missing", ["definitely-missing-program-2754"], "error", None)]
        for name, command, status, limit in commands:
            with self.subTest(name=name), tempfile.TemporaryDirectory() as folder:
                root = Path(folder); workspace, ledger, _, _ = series(root)
                result = plan_arm.run_simulation(ledger, "A2", workspace, command, root / "output", limit)
                self.assertEqual(result["status"], status, result)
                self.assertEqual(result["mode"], "simulation")
                self.assertNotEqual(result["providerEvidence"], "transport-reported")
                rows = d_series.records(ledger)
                self.assertEqual([r["kind"] for r in rows], ["series", "start", "end"])
                self.assertGreater(rows[-1]["chargedSeconds"], 0)
                with self.assertRaises(ValueError): d_series.reserve(ledger, "A2", "plan")

    def test_pending_abandonment_exhaustion_and_no_reset(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder); workspace, ledger, manifest, auth = series(root,
                [{"scenario": "A2", "arm": "plan"}, {"scenario": "ARTIFACTS", "arm": "direct"}], seconds=3)
            start = d_series.reserve(ledger, "A2", "plan")
            with self.assertRaises(ValueError): d_series.reserve(ledger, "ARTIFACTS", "direct")
            end = d_series.finish(ledger, start["attempt"])
            self.assertEqual(end["chargedSeconds"], 3)
            self.assertEqual(end["status"], "abandoned")
            with self.assertRaises(ValueError): d_series.reserve(ledger, "ARTIFACTS", "direct")
            with self.assertRaises(FileExistsError): d_series.initialize(manifest, root / "ledgers", auth)

    def test_manifest_and_result_binding_and_concurrent_writer_refuse(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder); _, ledger, manifest, _ = series(root)
            with d_series.locked(ledger):
                with self.assertRaises(FileExistsError): d_series.reserve(ledger, "A2", "plan")
            start = d_series.reserve(ledger, "A2", "plan")
            wrong = root / "wrong.json"
            d_series.write_new(wrong, {"attempt": "other", "scenario": "A2", "mode": "simulation"})
            with self.assertRaises(ValueError): d_series.finish(ledger, start["attempt"], wrong)
            manifest.write_text("{}")
            with self.assertRaises(ValueError): d_series.records(ledger)

    def test_protected_plan_and_commit_changes_are_not_a_success(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder); workspace, ledger, _, _ = series(root)
            script = "from pathlib import Path; Path('plan.md').write_text('changed'); print('Done')"
            result = plan_arm.run_simulation(ledger, "A2", workspace, [sys.executable, "-B", "-c", script], root / "output")
            self.assertEqual(result["status"], "error")
            self.assertIn("authoritative plan.md", result["error"])

    def test_existing_output_is_preserved_and_failure_is_recorded_elsewhere(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder); workspace, ledger, _, _ = series(root)
            output = root / "output"; output.mkdir(); (output / "kept.txt").write_text("keep")
            result = plan_arm.run_simulation(ledger, "A2", workspace, [sys.executable, "-c", "print('Done')"], output)
            self.assertEqual(result["status"], "error")
            self.assertEqual(sorted(p.name for p in output.iterdir()), ["kept.txt"])
            self.assertTrue((ledger / (result["attempt"] + ".json")).is_file())

    def test_output_is_bounded_and_recorded_as_incomplete(self):
        with tempfile.TemporaryDirectory() as folder:
            result = plan_arm.capture([sys.executable, "-B", "-c", "print('x' * 200000)"], folder, "prompt", 5, 1024)
            self.assertEqual(result["status"], "output_limit")
            self.assertLessEqual(len(result["stdout"]) + len(result["stderr"]), 1024)

    def test_native_provider_records_require_actual_completion_and_remain_simulated_in_controls(self):
        p = profile()
        events = [{"type": "thread.started", "thread_id": "s1"}, {"type": "turn.started"},
                  {"type": "item.completed", "item": {"type": "agent_message", "text": "Done"}},
                  {"type": "turn.completed", "usage": {"input_tokens": 12, "output_tokens": 5}}]
        raw = "\n".join(map(json.dumps, events))
        result = plan_arm.provider_reply(raw, p, 400)
        self.assertEqual(result["turnUnit"], "codex-input-turn")
        self.assertEqual(result["sessionId"], "s1")
        for bad in (events[:-1], events + [{"type": "error"}], events + [events[-1]]):
            with self.assertRaises(ValueError): plan_arm.provider_reply("\n".join(map(json.dumps, bad)), p, 400)
        p["protocol"] = "claude-print-json"
        claude = {"type": "result", "subtype": "success", "session_id": "s2", "result": "Done", "num_turns": 3,
                  "total_cost_usd": 1.25}
        self.assertEqual(plan_arm.provider_reply(json.dumps(claude), p, 400)["turns"], 3)
        with self.assertRaises(ValueError): plan_arm.provider_reply(json.dumps(dict(claude, is_error=True)), p, 400)
        with self.assertRaises(ValueError): plan_arm.provider_reply(json.dumps(claude), p, 2)

    def test_only_frozen_packet_and_individual_auth_files_enter_container(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder); workspace = root / "writer"; workspace.mkdir(); (workspace / "plan.md").write_text("requirements")
            p = profile(); path = root / "profile.json"; d_series.write_new(path, p)
            self.assertEqual(plan_arm.profile_data(path)["model"], "frozen-model")
            for feature in ("multi_agent", "apps", "memories"):
                bad = copy.deepcopy(p); index = bad["argv"].index(feature)
                del bad["argv"][index - 1:index + 1]; path.write_text(json.dumps(bad))
                with self.assertRaises(ValueError): plan_arm.profile_data(path)
            path.write_text(json.dumps(p))
            argv = plan_arm.create_command(p, workspace, "owned-attempt", 400)
            mounts = [argv[i + 1] for i, a in enumerate(argv) if a == "--mount"]
            self.assertEqual(len(mounts), 2)
            self.assertIn("target=/work/plan.md,readonly", mounts[1])
            self.assertIn("--pids-limit", argv)
            self.assertNotIn("--privileged", argv)
            p["authFiles"] = [{"source": str(root), "target": "/run/provider-auth/x"}]
            with self.assertRaises(ValueError): plan_arm.create_command(p, workspace, "owned-attempt", 400)
            p = profile(); p["imageID"] = "latest"; path.write_text(json.dumps(p))
            with self.assertRaises(ValueError): plan_arm.profile_data(path)

    def test_simulation_cannot_be_published_as_provider_evidence(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder); workspace, ledger, _, _ = series(root)
            # Even a fake provider-looking reply runs only through the simulation entry point.
            command = [sys.executable, "-B", "-c", "print('{\"protocol\":\"plan-agent/v1\",\"simulation\":false}')"]
            result = plan_arm.run_simulation(ledger, "A2", workspace, command, root / "output")
            self.assertEqual(result["mode"], "simulation")
            self.assertEqual(result["providerEvidence"], "simulation")
            self.assertIsNone(result["turns"])
            self.assertIsNone(result["containers"])

    def test_auth_mount_traversal_and_option_injection_refuse(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder); auth = root / "credential"; auth.write_text("synthetic")
            p = profile()
            for target in ("/run/provider-auth/../../work/secret", "/run/provider-auth/./x",
                           "/run/provider-auth//x", "/run/provider-auth/x,target=/work", "/run/provider-auth/x\\y"):
                p["authFiles"] = [{"source": str(auth), "target": target}]
                with self.subTest(target=target), self.assertRaises(ValueError):
                    plan_arm.create_command(p, root / "writer", "owned", 400)
            p["authFiles"] = [{"source": str(auth), "target": "/run/provider-auth/auth.json"}]
            self.assertIn("target=/run/provider-auth/auth.json,readonly",
                          " ".join(plan_arm.create_command(p, root / "writer", "owned", 400)))

    def test_claude_profile_excludes_subagent_tools_and_requires_native_limits(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "profile.json"; p = profile()
            p.update(protocol="claude-print-json", provider="claude", argv=["/usr/local/bin/claude", "-p",
                "--output-format", "json", "--model", "{model}", "--effort", "{effort}",
                "--max-turns", "{maxTurns}", "--tools", "Read,Write,Edit,Glob,Grep,Bash"])
            path.write_text(json.dumps(p)); plan_arm.profile_data(path)
            for argv in (p["argv"][:-2], p["argv"][:-1] + ["Read,Agent"],
                         p["argv"] + ["--resume", "old"], [a.replace("--max-turns", "--other") for a in p["argv"]]):
                path.write_text(json.dumps(dict(p, argv=argv)))
                with self.assertRaises(ValueError): plan_arm.profile_data(path)

    def test_provider_binding_checks_actual_runtime_and_ordinary_contents(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder); p = profile(); ordinary = root / "plan.json"; ordinary.write_text(json.dumps(p))
            runtime = root / "runtime.json"
            settings = {"apiVersion": "markitect.runtime/v1", "agentProfiles": {"project-agent": {"codexAppServer": {
                "imageID": p["imageID"], "session": {"model": p["model"], "effort": p["effort"], "cpus": 2,
                "memoryBytes": 4096, "pids": 32}}}}}
            runtime.write_text(json.dumps(settings)); run = root / "run.json"; run.write_text("{}")
            common = {k: p[k] for k in ("provider", "model", "effort", "imageID")}
            common["resources"] = {k: p["limits"][k] for k in ("milliCPUs", "memoryBytes", "processes")}
            bindings = {arm: dict(common, profilePath=str(ordinary if arm == "plan" else runtime),
                profileSha256=d_series.sha(ordinary if arm == "plan" else runtime), productBinary=sys.executable,
                productBinarySha256=d_series.sha(sys.executable), productRevision="a" * 40,
                runConfigPath=str(run), runConfigSha256=d_series.sha(run)) for arm in ("direct", "staged", "plan")}
            d_series.validate_bindings(bindings)
            bindings["staged"]["productRevision"] = "b" * 40
            with self.assertRaisesRegex(ValueError, "same product revision"):
                d_series.validate_bindings(bindings)
            bindings["staged"]["productRevision"] = "a" * 40
            settings["agentProfiles"]["project-agent"]["codexAppServer"]["session"]["model"] = "different"
            runtime.write_text(json.dumps(settings))
            for arm in ("direct", "staged"): bindings[arm]["profileSha256"] = d_series.sha(runtime)
            with self.assertRaisesRegex(ValueError, "Runtime profile contents"):
                d_series.validate_bindings(bindings)
            settings["agentProfiles"]["project-agent"]["codexAppServer"]["session"]["model"] = p["model"]
            runtime.write_text(json.dumps(settings))
            for arm in ("direct", "staged"): bindings[arm]["profileSha256"] = d_series.sha(runtime)
            p["limits"]["processes"] = 33; ordinary.write_text(json.dumps(p)); bindings["plan"]["profileSha256"] = d_series.sha(ordinary)
            with self.assertRaisesRegex(ValueError, "ordinary profile contents"):
                d_series.validate_bindings(bindings)

    def test_measurement_keeps_unknown_cost_and_container_counts_unknown(self):
        report = {"run": {"turns": [{"id": "t1"}], "usage": {"inputTokens": 15, "costUSD": 0}}}
        result = d_series.measure(report)
        self.assertEqual(result["turns"], 1)
        self.assertIsNone(result["containers"])
        self.assertIsNone(result["costUSD"])
        self.assertIn("containers", result["unknowns"])
        self.assertEqual(result["evidenceMode"], "unclassified")


if __name__ == "__main__":
    unittest.main()
