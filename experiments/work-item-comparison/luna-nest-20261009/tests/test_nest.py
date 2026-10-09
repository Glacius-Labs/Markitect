"""Mechanical tests only: fake local processes, Git/file fixtures, no model/product invocation."""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import tomllib
import unittest
from unittest.mock import patch
from datetime import datetime, timedelta, timezone

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("nest", ROOT / "public" / "nest.py")
nest = importlib.util.module_from_spec(spec)
spec.loader.exec_module(nest)


class Mechanics(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="luna-nest-mechanical-")
        self.base = Path(self.tmp.name)
        self.seed = self.base / "seed"
        self.repo = self.base / "repo"
        self.state = self.base / "state"
        nest.seed("readinglog", self.seed)
        nest.cell("readinglog", "conventional", self.seed, self.repo, self.state)
        self.home = self.base / "native-home"
        self.home.mkdir()
        (self.home / "config.toml").write_bytes((ROOT / "public/native/config.toml").read_bytes())
        self.grant = self.base / "grant.json"
        self.authorize()

    def tearDown(self):
        self.tmp.cleanup()

    def authorize(self):
        state = nest.read(self.state / "state.json")
        nest.write(self.grant, {"executionAuthorized": True, "cell": "readinglog/conventional",
                   "model": "gpt-6-luna", "reasoning": "high", "preparedCommit": state["preparedCommit"],
                   "executableSha256": nest.sha(sys.executable), "nativeHome": str(self.home),
                   "nativeHomeConfigSha256": nest.sha(self.home / "config.toml"),
                   "maxCellWallSeconds": 7200, "maxImplementationSessions": 8,
                   "nativeConfigurationReviewed": True, "freshContextConfirmed": True,
                   "notAfterUtc": (datetime.now(timezone.utc) + timedelta(seconds=7400)).isoformat()})

    def test_shared_seed_parity_and_non_solution_greenfield(self):
        other = self.base / "markitect"
        otherstate = self.base / "markitect-state"
        nest.cell("readinglog", "markitect", self.seed, other, otherstate)
        for name in ("README.md", "BACKLOG.md", "STATIONS.json", "QUALITY.md", "app.py", "tests/test_baseline.py"):
            self.assertEqual((self.repo / name).read_bytes(), (other / name).read_bytes())
        gf = self.base / "gf"
        nest.seed("roombook", gf)
        self.assertFalse((gf / "app.py").exists())
        stations = nest.read(gf / "STATIONS.json")["stations"]
        self.assertEqual([len(s["items"]) for s in stations], [1, 3, 7, 1])
        self.assertTrue(stations[2]["requiresTeam"])

    def test_lock_and_exact_grants(self):
        with nest.locked(self.state):
            with self.assertRaises(FileExistsError):
                with nest.locked(self.state):
                    pass
        state = nest.read(self.state / "state.json")
        grant = nest.read(self.grant)
        grant["executionAuthorized"] = False
        with self.assertRaises(ValueError):
            nest.admission(state, grant, sys.executable, "implementation")
        grant["executionAuthorized"] = True
        grant["preparedCommit"] = "wrong"
        with self.assertRaises(ValueError):
            nest.admission(state, grant, sys.executable, "implementation")
        state["status"] = "running"
        with self.assertRaises(ValueError):
            nest.admission(state, nest.read(self.grant), sys.executable, "implementation")

    def test_actual_raw_event_parser_and_missing_usage(self):
        events = self.base / "events"
        events.write_text('{"type":"thread.started","thread_id":"owned"}\n'
                          '{"type":"turn.completed","usage":{"input_tokens":7,"output_tokens":3}}\n'
                          '{"type":"turn.completed"}\ninvalid\n', encoding="utf-8")
        got = nest.parse_events(events)
        self.assertEqual(got["sessionId"], "owned")
        self.assertEqual(got["completedTurns"][0]["usage"]["input_tokens"], 7)
        self.assertIsNone(got["completedTurns"][1]["usage"])
        self.assertEqual(got["invalidLines"], 1)

    def test_fake_local_process_capture_resume_and_grant_no_refill(self):
        fake = self.base / "fake.py"
        fake.write_text('import json,os,sys\n'
                        'sys.stdin.read()\n'
                        'print(json.dumps({"type":"thread.started","thread_id":"fixture-own"}))\n'
                        'print(json.dumps({"type":"turn.completed","usage":{"input_tokens":7,"output_tokens":3}}))\n',
                        encoding="utf-8")
        with patch.object(nest, "command", return_value=[sys.executable, "-B", str(fake)]):
            result = nest.run(self.state, self.grant, sys.executable)
            self.assertEqual(result["status"], "returned")
            self.assertEqual(nest.read(self.state / "state.json")["sessionId"], "fixture-own")
            second = nest.run(self.state, self.grant, sys.executable)
            self.assertEqual(second["status"], "returned")
            with self.assertRaises(ValueError):
                nest.run(self.state, self.grant, sys.executable)
        grant = nest.read(self.grant)
        grant["anythingChanged"] = True
        nest.write(self.grant, grant)
        with self.assertRaisesRegex(ValueError, "grant changed"):
            nest.run(self.state, self.grant, sys.executable)
        self.assertEqual(nest.command("codex", self.repo, "last", "fixture-own")[1:3], ["exec", "resume"])

    def test_snapshot_retains_dirty_separate_from_main_and_no_replay(self):
        (self.repo / "unfinished.txt").write_text("retained", encoding="utf-8")
        shot = nest.snapshot(self.state)
        self.assertIn("unfinished.txt", shot["rawFiles"])
        self.assertFalse((Path(shot["evaluationRepo"]) / "unfinished.txt").exists())
        self.assertTrue(nest.advance(self.state))
        self.assertEqual(nest.read(self.repo / ".study/station.json")["id"], "S2")
        final = nest.freeze(self.state)
        self.assertFalse(final["mergedAndClean"])
        with self.assertRaises(ValueError):
            nest.freeze(self.state)
        with self.assertRaises(ValueError):
            nest.admission(nest.read(self.state / "state.json"), nest.read(self.grant), sys.executable, "implementation")

    def test_fixed_dispatch_all_stations_without_code_advice(self):
        def fake_run(state_dir, grant, executable, phase="implementation"):
            state = nest.read(Path(state_dir) / "state.json")
            if phase == "implementation":
                station = state["stations"][state["stationIndex"]]
                nest.write(Path(state["repo"]) / ".study/completion.json", {"station": station["id"], "status": "complete"})
                nest.commit(state["repo"], "Mechanical fixture station")
                state["sessions"].append({"phase": phase, "stationIndex": state["stationIndex"]})
                state["status"] = "returned"
                nest.write(Path(state_dir) / "state.json", state)
            return {"status": "returned", "events": {"errors": []}}
        with patch.object(nest, "run_locked", side_effect=fake_run):
            result = nest.drive(self.state, self.grant, sys.executable)
        state = nest.read(self.state / "state.json")
        self.assertEqual([s["station"] for s in state["snapshots"]], ["S1", "S2", "S3", "S4"])
        self.assertEqual(result["assessment"]["status"], "returned")
        self.assertEqual(len(state["sessions"]), 4)

    def test_bounded_owned_non_model_child_stop(self):
        proc = subprocess.Popen([sys.executable, "-B", "-c", "import time;time.sleep(30)"],
                                creationflags=subprocess.CREATE_NEW_PROCESS_GROUP if os.name == "nt" else 0,
                                start_new_session=os.name != "nt")
        try:
            stopped = nest.stop_owned(proc)
            self.assertTrue(stopped.get("localExitConfirmed", stopped.get("alreadyExited", False)))
            self.assertIsNotNone(proc.poll())
        finally:
            if proc.poll() is None:
                proc.kill()
                proc.wait(timeout=5)

    def test_native_template_is_luna_high_and_memory_disabled(self):
        config = tomllib.loads((ROOT / "public/native/config.toml").read_text(encoding="utf-8-sig"))
        self.assertEqual(config["model"], "gpt-6-luna")
        self.assertEqual(config["model_reasoning_effort"], "high")
        self.assertFalse(config["features"]["memories"])
        self.assertNotIn("default_subagent_model", config["agents"])
        launcher = (ROOT / "public/conventional/oldschool/Invoke-NestTurn.ps1").read_text(encoding="utf-8-sig")
        self.assertNotIn("WaitForExit()", launcher)
        self.assertIn("$activeSeconds = $seconds - 30", launcher)


if __name__ == "__main__":
    unittest.main()
