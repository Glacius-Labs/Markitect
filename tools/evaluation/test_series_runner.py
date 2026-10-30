"""Offline tests of the series runner: its plans, the guard, the login refresh, and whole series against a fake suite."""
import json
import os
import pathlib
import sys
import tempfile
import textwrap
import time
import unittest

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import series_runner  # noqa: E402

MODEL = "claude:claude-sonnet-5-5"
ACCESS, REFRESH = "sk-ant-oat01-" + "a" * 48, "sk-ant-ort01-" + "b" * 48

# A fake suite.py. Each `run` or `chain` launch takes the next outcome its unit (or unit/step) has in FAKE_DIR/scenario.json
# and writes the run folder the suite would; `judge`, `judge-chain`, `review` and `report` write their outputs. Every call
# goes to FAKE_DIR/calls.jsonl with whether the engine lock existed. While the lock exists, a launch is refused as the
# suite refuses it.
FAKE_SUITE = textwrap.dedent(r'''
    import json, os, pathlib, sys
    args, d = sys.argv[1:], pathlib.Path(os.environ["FAKE_DIR"])
    lock = pathlib.Path(os.environ["FAKE_LOCK"])
    opt = lambda name: args[args.index(name) + 1] if name in args else None
    with open(d / "calls.jsonl", "a") as f:
        f.write(json.dumps({"args": args, "lockHeld": lock.exists()}) + "\n")
    command = args[0]
    if command in ("run", "chain"):
        if lock.exists():
            sys.stderr.write("the engine is busy: another holder\n")
            sys.exit(2)
        out = pathlib.Path(opt("--out"))
        key = out.parent.name if command == "run" else out.parent.name + "/" + out.name.rsplit("-a", 1)[0]
        counts = json.loads((d / "counts.json").read_text()) if (d / "counts.json").exists() else {}
        outcome = json.loads((d / "scenario.json").read_text())[key][counts.get(key, 0)]
        counts[key] = counts.get(key, 0) + 1
        (d / "counts.json").write_text(json.dumps(counts))
        run = out if command == "run" else out / "step-01" / "run"
        (run / "variant").mkdir(parents=True)
        record = {"conditions": "native" if "--native" in args else "sandbox", "validForCounting": True, "delivered": True,
                  "exitCode": 0, "timedOut": False, "model": opt("--model"), "modelMatches": True, "reported": {}, "sandbox": {}}
        (run / "variant" / "run.json").write_text("{}")
        if outcome == "could-not-run":
            record.update(delivered=False, exitCode=2, reported={"status": "could-not-run"})
        elif outcome == "foreign":
            record["sandbox"]["escapes"] = [{"name": "ryuk-1", "id": "c0ffee"}]
            (run / "engine-events.jsonl").write_text(json.dumps({"Type": "container", "Action": "start", "Actor": {
                "ID": "c0ffee99", "Attributes": {"name": "ryuk-1", "org.testcontainers": "true"}}}) + "\n")
        elif outcome == "own-escape":
            record["sandbox"]["escapes"] = [{"name": "markitect-run-x", "id": "beef"}]
            (run / "engine-events.jsonl").write_text(json.dumps({"Type": "container", "Action": "start", "Actor": {
                "ID": "beef01", "Attributes": {"name": "markitect-run-x", "markitect.run": "x"}}}) + "\n")
        elif outcome == "leak":
            (run / "variant" / "leak.txt").write_text(os.environ["FAKE_TOKEN"])
        elif outcome == "quota":
            (run / "variant" / "stream.jsonl").write_text(json.dumps({"type": "rate_limit_event", "rate_limit_info": {
                "status": "allowed", "unifiedWindows": {"seven_day": {"utilization": 0.95}}}}) + "\n")
        elif outcome == "stop-file":
            (pathlib.Path(os.environ["FAKE_WORK"]) / "STOP").write_text("window ends\n")
        (run / "suite.json").write_text(json.dumps(record))
        if command == "chain":
            (out / "step-01" / "task").mkdir(parents=True)
            (out / "step-01" / "task" / "start.bundle").write_text("start")
            (run / "variant" / "result.bundle").write_text("result")
            (out / "chain.json").write_text(json.dumps({"steps": [{"step": 1}]}))
            state = pathlib.Path(opt("--state"))
            with open(state / "log.txt", "a") as f:
                f.write(out.name + "\n")
    elif command == "judge":
        pathlib.Path(opt("--out")).mkdir(parents=True)
        (pathlib.Path(opt("--out")) / "judgment.json").write_text("{}")
    elif command == "judge-chain":
        out = pathlib.Path(opt("--out"))
        for i, _ in enumerate(opt("--steps").split(","), 1):
            (out / f"step-{i:02d}").mkdir(parents=True)
            (out / f"step-{i:02d}" / "judgment.json").write_text("{}")
        (out / "chain-judgment.json").write_text("{}")
    elif command == "review":
        pathlib.Path(opt("--out")).mkdir(parents=True)
        (pathlib.Path(opt("--out")) / "review.json").write_text("{}")
    elif command == "report":
        pathlib.Path(opt("--out")).mkdir(parents=True)
        (pathlib.Path(opt("--out")) / "report.md").write_text("report")
''')

# A fake `claude -p`: like the CLI near expiry, it refreshes the login in FAKE_LOGIN when its access token expires within
# five minutes, unless FAKE_REFRESH says it fails; every call goes to FAKE_CLAUDE_CALLS.
FAKE_CLAUDE = textwrap.dedent(r'''
    import json, os, time
    with open(os.environ["FAKE_CLAUDE_CALLS"], "a") as f:
        f.write("call\n")
    path = os.environ["FAKE_LOGIN"]
    data = json.load(open(path))
    oauth = data["claudeAiOauth"]
    if oauth["expiresAt"] / 1000 < time.time() + 300 and os.environ.get("FAKE_REFRESH") != "fails":
        oauth.update(accessToken="sk-ant-oat01-" + "c" * 48, expiresAt=int((time.time() + 8 * 3600) * 1000))
        json.dump(data, open(path, "w"))
    print("OK")
''')


class Quiet:
    def reasons(self):
        return []


class AnyLogin:
    def ensure(self, deadline):
        pass


def unit(uid, arm, kind="fixed", step="C1", **extra):
    out = {"id": uid, "arm": arm, "kind": kind, "series": "gs", "repetition": 1, **extra}
    if kind == "fixed":
        out["step"] = step
    return out


class Fixture(unittest.TestCase):
    def setUp(self):
        self.tmp = pathlib.Path(tempfile.mkdtemp())
        self.work, self.prep, self.fake = self.tmp / "work", self.tmp / "prepared", self.tmp / "fake"
        self.fake.mkdir()
        (self.fake / "suite.py").write_text(FAKE_SUITE)
        self.lock = self.tmp / "NATIVE-RUN.lock"
        self.login = self.tmp / "credentials.json"
        self.login.write_text(json.dumps({"claudeAiOauth": {"accessToken": ACCESS, "refreshToken": REFRESH,
                                                            "expiresAt": int((time.time() + 8 * 3600) * 1000)}}))
        for folder in ("tasks/fixed-C1", "tasks/fixed-C2", "tasks/plant-p1", "steps/W0", "steps/C1", "steps/C2", "judges/series",
                       "judges/plant-p1"):
            (self.prep / folder).mkdir(parents=True)
        env = {"FAKE_DIR": str(self.fake), "FAKE_LOCK": str(self.lock), "FAKE_TOKEN": ACCESS, "FAKE_WORK": str(self.work)}
        self.addCleanup(lambda: [os.environ.pop(k, None) for k in env])
        os.environ.update(env)
        self.slept = []

    def config(self, units, **series):
        gs = {"role": "confirmatory", "deadlineSeconds": 60, "steps": ["W0", "C1", "C2"], "fixed": ["C1", "C2"],
              "plants": {"p1": {"step": "C1", "kind": "defect"}}, "prepared": str(self.prep), **series}
        return {"work": str(self.work), "repo": str(self.tmp), "rev": "0" * 40, "bin": str(self.tmp), "managerImage": "sha256:m",
                "sandboxImage": "sha256:s", "lock": str(self.lock), "hidden": [str(self.tmp / "absent")], "login": str(self.login),
                "model": MODEL, "docker": "docker", "protocol": "counted", "series": {"gs": gs}, "relaunches": 2, "launchBound": 50,
                "minimumDifference": 0.05, "reviews": {"repetition": 1, "chains": True}, "units": units}

    def runner(self, config, sleep=None):
        def default_sleep(seconds):
            self.slept.append(seconds)
        return series_runner.Runner(config, suite=self.fake / "suite.py", guard=Quiet(), login=AnyLogin(), sleep=sleep or default_sleep)

    def scenario(self, outcomes):
        (self.fake / "scenario.json").write_text(json.dumps(outcomes))

    def calls(self, command=None):
        calls = [json.loads(x) for x in (self.fake / "calls.jsonl").read_text().splitlines()]
        return [c for c in calls if command is None or c["args"][0] == command]

    def ledger(self):
        return json.loads((self.work / "ledger.json").read_text())

    def events(self, kind):
        return [e for e in map(json.loads, (self.work / "series.jsonl").read_text().splitlines()) if e["event"] == kind]


class PlanTest(unittest.TestCase):
    def prepared(self):
        folder = pathlib.Path(tempfile.mkdtemp())
        (folder / "series.json").write_text(json.dumps({
            "name": "new", "deadlineSeconds": 3600, "steps": ["W0"] + [f"C{i}" for i in range(1, 8)], "fixed": [f"C{i}" for i in range(1, 8)],
            "plants": {f"p{i}": {"step": f"C{i + 1}", "kind": "defect"} for i in range(1, 4)}, "judges": str(folder / "judges")}))
        return folder

    def test_counted_series_per_platform(self):
        config = series_runner.plan("counted", {"work": "w"}, [self.prepared()])
        units = config["units"]
        self.assertEqual(series_runner.summary(units)["launches"], 222)
        self.assertEqual(series_runner.summary(units)["byArm"], {"M": 111, "K": 111})
        self.assertEqual(config["launchBound"], 333)
        # Every fixed start and plant runs as often as its role says, for both arms; the chains come last.
        reps = {}
        for u in units:
            reps.setdefault((u["series"], u["kind"], u.get("step") or u.get("plant")), []).append((u["repetition"], u["arm"]))
        self.assertEqual(len(reps[("new", "fixed", "C1")]), 10)
        self.assertEqual(len(reps[("new", "plant", "p1")]), 6)
        self.assertEqual(len(reps[("library", "fixed", "M2")]), 4)
        self.assertEqual(len(reps[("makerspace", "plant", "makerspace-c5-threshold-gap")]), 4)
        kinds = [u["kind"] for u in units]
        self.assertEqual(kinds.index("chain"), len(kinds) - 8)
        self.assertEqual([u["id"] for u in units if u["kind"] == "chain"],
                         ["library-chain-r1-M", "library-chain-r1-K", "makerspace-chain-r1-K", "makerspace-chain-r1-M",
                          "new-chain-r1-M", "new-chain-r1-K", "new-chain-r2-K", "new-chain-r2-M"])
        # The first arm alternates per task and per repetition; the task order turns per repetition.
        self.assertEqual([arm for r, arm in reps[("new", "fixed", "C1")][::2]], ["M", "K", "M", "K", "M"])
        self.assertEqual([arm for r, arm in reps[("new", "fixed", "C2")][::2]], ["K", "M", "K", "M", "K"])
        self.assertNotEqual(units[0]["id"].rsplit("-", 2)[0], [u for u in units if u["repetition"] == 2][0]["id"].rsplit("-", 2)[0])

    def test_stability_gate(self):
        units = series_runner.plan("gate", {})["units"]
        self.assertEqual(series_runner.summary(units), {"launches": 26, "byArm": {"M": 13, "K": 13}, "bySeries": {
            "library fixed": 6, "makerspace chain": 20}})
        self.assertEqual([u["id"] for u in units][:6], ["library-fixed-M2-gate-M", "library-fixed-M2-gate-K", "library-fixed-M3-gate-K",
                                                        "library-fixed-M3-gate-M", "library-fixed-M4-gate-M", "library-fixed-M4-gate-K"])
        with self.assertRaises(SystemExit):
            series_runner.plan("gate", {}, [self.prepared()])


class GuardTest(unittest.TestCase):
    def test_foreign_containers(self):
        self.assertEqual(series_runner.foreign_running("ryuk\torg.testcontainers=true\nmarkitect-run-1\tmarkitect.run=r1\nbare\t\n"),
                         ["ryuk", "bare"])

    def test_quiet_after_ten_minutes_without_a_foreign_start(self):
        clock = [1000.0]

        class Engine(series_runner.Guard):
            def running(self):
                return []
        guard = Engine(["docker"], clock=lambda: clock[0])
        self.assertEqual(guard.reasons(), ["the engine not yet watched for ten minutes"])
        guard.since = clock[0]
        clock[0] += 300
        guard.seen(json.dumps({"name": "markitect-run-r1-root", "markitect.run": "r1"}))
        guard.seen(json.dumps({"name": "ryuk", "org.testcontainers": "true"}))
        clock[0] += 301
        self.assertEqual(guard.reasons(), ["ryuk (started)"])
        clock[0] += 600
        self.assertEqual(guard.reasons(), [])


class LoginTest(unittest.TestCase):
    def setUp(self):
        self.tmp = pathlib.Path(tempfile.mkdtemp())
        self.path, self.lock, self.calls = self.tmp / "credentials.json", self.tmp / "NATIVE-RUN.lock", self.tmp / "calls.txt"
        (self.tmp / "claude.py").write_text(FAKE_CLAUDE)
        env = {"FAKE_LOGIN": str(self.path), "FAKE_CLAUDE_CALLS": str(self.calls)}
        self.addCleanup(lambda: [os.environ.pop(k, None) for k in (*env, "FAKE_REFRESH")])
        os.environ.update(env)
        self.events, self.slept = [], []

    def login(self, minutes_left, refresh_until=None, sleep=None):
        oauth = {"accessToken": ACCESS, "refreshToken": REFRESH, "expiresAt": int((time.time() + minutes_left * 60) * 1000),
                 "scopes": ["user:inference"]}
        if refresh_until is not None:
            oauth["refreshTokenExpiresAt"] = int((time.time() + refresh_until * 60) * 1000)
        self.path.write_text(json.dumps({"claudeAiOauth": oauth}))
        return series_runner.Login(self.path, self.lock, MODEL, lambda **e: self.events.append(e),
                                   claude=[sys.executable, "-I", str(self.tmp / "claude.py")], sleep=sleep or self.slept.append)

    def test_a_login_that_covers_the_launch_is_left_alone(self):
        self.login(120).ensure(2700)
        self.assertFalse(self.calls.exists())

    def test_refreshed_before_a_launch_it_cannot_cover(self):
        login = self.login(60)
        login.ensure(2700)
        self.assertEqual(self.calls.read_text(), "call\n")
        self.assertGreater(login.left(), 7 * 3600)
        self.assertEqual(self.events[-1]["event"], "login-refresh")

    def test_marking_expired_changes_no_token(self):
        login = self.login(60)
        login.expire()
        oauth = json.loads(self.path.read_text())["claudeAiOauth"]
        self.assertEqual((oauth["accessToken"], oauth["refreshToken"], oauth["scopes"]), (ACCESS, REFRESH, ["user:inference"]))
        self.assertLess(abs(oauth["expiresAt"] / 1000 - time.time()), 5)
        self.assertEqual([p.name for p in self.tmp.iterdir() if p.name.startswith("credentials")], ["credentials.json"])

    def test_waits_while_a_run_holds_the_lock(self):
        self.lock.write_text("{}")
        login = self.login(60, sleep=lambda s: self.lock.unlink())
        login.ensure(2700)
        self.assertEqual([e["event"] for e in self.events], ["login-refresh-waits", "login-refresh"])
        self.assertFalse(self.lock.exists())

    def test_a_failed_refresh_stops(self):
        os.environ["FAKE_REFRESH"] = "fails"
        with self.assertRaisesRegex(series_runner.Stop, "could not be refreshed"):
            self.login(60).ensure(2700)

    def test_a_refresh_token_that_ends_first_stops_without_a_refresh(self):
        with self.assertRaisesRegex(series_runner.Stop, "log in again"):
            self.login(60, refresh_until=30).ensure(2700)
        self.assertFalse(self.calls.exists())


class SeriesTest(Fixture):
    def test_each_launch_is_classed_with_its_arms_signatures(self):
        self.scenario({"m": ["could-not-run", "result"], "k": ["could-not-run"]})
        self.runner(self.config([unit("m", "M"), unit("k", "K")])).run()
        attempts = self.ledger()["attempts"]
        self.assertEqual([(a["unit"], a["class"], a["kind"]) for a in attempts],
                         [("m", "void", "variant runtime"), ("m", "result", None), ("k", "result", None)])
        self.assertEqual([c["args"][c["args"].index("--run") + 1].replace("\\", "/").split("/runs/")[1] for c in self.calls("judge")], ["m/a2"])
        self.assertTrue((self.work / "done.json").exists())

    def test_the_runner_never_holds_the_engine_lock(self):
        self.scenario({"m": ["result"]})
        self.lock.write_text('{"holder": "another session"}')
        released = []

        def sleep(seconds):
            if self.lock.exists():
                self.lock.unlink()
                released.append(seconds)
        self.runner(self.config([unit("m", "M")]), sleep=sleep).run()
        self.assertEqual(released, [120])
        launches = self.calls("run")
        self.assertEqual([c["lockHeld"] for c in launches], [True, False])
        self.assertTrue(all(not c["lockHeld"] for c in self.calls() if c["args"][0] != "run"))
        self.assertEqual(len(self.ledger()["attempts"]), 1)
        self.assertEqual(len(self.events("lock-busy")), 1)
        for c in self.calls():
            self.assertEqual(c["args"][c["args"].index("--lock") + 1] if "--lock" in c["args"] else str(self.lock), str(self.lock))

    def test_other_works_containers_are_relaunched_once_then_stop(self):
        self.scenario({"a": ["foreign", "result"], "b": ["foreign", "foreign"]})
        self.runner(self.config([unit("a", "M"), unit("b", "M", step="C2")])).run()
        self.assertEqual([(a["unit"], a["class"], a["foreign"]) for a in self.ledger()["attempts"]],
                         [("a", "invalid", True), ("a", "result", False), ("b", "invalid", True), ("b", "invalid", True)])
        self.assertIn("an invalid launch", (self.work / "STOP").read_text())
        self.assertFalse((self.work / "done.json").exists())

    def test_any_other_invalid_launch_stops(self):
        self.scenario({"a": ["own-escape"], "b": ["result"]})
        self.runner(self.config([unit("a", "M"), unit("b", "K")])).run()
        self.assertEqual(len(self.ledger()["attempts"]), 1)
        self.assertIn("markitect-run-x", (self.work / "STOP").read_text())

    def test_a_void_share_above_one_in_four_stops(self):
        self.scenario({u: ["could-not-run"] * 3 for u in ("a", "b", "c")})
        self.runner(self.config([unit(u, "M") for u in ("a", "b", "c")])).run()
        self.assertEqual(len(self.ledger()["attempts"]), 8)
        self.assertIn("8 of 8 launches void", (self.work / "STOP").read_text())

    def test_a_credential_in_the_evidence_stops(self):
        self.scenario({"a": ["leak"], "b": ["result"]})
        self.runner(self.config([unit("a", "M"), unit("b", "K")])).run()
        self.assertIn("leak.txt", (self.work / "STOP").read_text())
        self.assertEqual(len(self.ledger()["attempts"]), 1)

    def test_resumes_after_a_stop_without_repeating_what_counted(self):
        self.scenario({"a": ["quota"], "b": ["stop-file"], "c": ["result"]})
        config = self.config([unit("a", "M"), unit("b", "K"), unit("c", "M", step="C2")])
        self.runner(config).run()
        self.assertIn("quota", (self.work / "STOP").read_text())
        (self.work / "STOP").unlink()
        self.runner(config).run()
        self.assertEqual((self.work / "STOP").read_text(), "window ends\n")
        (self.work / "STOP").unlink()
        self.runner(config).run()
        self.assertEqual([a["unit"] for a in self.ledger()["attempts"]], ["a", "b", "c"])
        self.assertEqual(self.ledger()["done"], ["a", "b", "c"])
        self.assertEqual(len(self.ledger()["windows"]), 3)
        self.assertTrue((self.work / "done.json").exists())
        with self.assertRaises(SystemExit):
            self.runner(config).run()

    def test_a_chain_relaunches_a_void_step_from_its_state_and_is_judged_whole(self):
        self.scenario({"ch/W0": ["result"], "ch/C1": ["could-not-run", "result"], "ch/C2": ["result"]})
        self.runner(self.config([unit("ch", "M", kind="chain", steps=["W0", "C1", "C2"])])).run()
        self.assertEqual((self.work / "runs" / "ch" / "state" / "log.txt").read_text().split(), ["W0-a1", "C1-a2", "C2-a1"])
        chain = self.calls("chain")
        self.assertNotIn("--start", chain[0]["args"])
        self.assertTrue(chain[2]["args"][chain[2]["args"].index("--start") + 1].replace("\\", "/").endswith("ch/W0-a1/step-01/run/variant/result.bundle"))
        judged = self.calls("judge-chain")[0]["args"]
        self.assertEqual(judged[judged.index("--steps") + 1], "W0,C1,C2")
        ledger = self.ledger()
        self.assertEqual([c["id"] for c in ledger["chains"]], ["ch"])
        self.assertEqual([bool(e.get("judgment")) for e in ledger["launches"]], [True, False, True, True])
        self.assertEqual([bool(e.get("review")) for e in ledger["launches"]], [False, False, False, True])

    def test_a_hidden_judge_waits_off_the_machine(self):
        judges = self.tmp / "hidden-judges"
        self.scenario({"a": ["result"], "ch/W0": ["result"]})
        config = self.config([unit("a", "M"), unit("ch", "K", kind="chain", steps=["W0"])], judges=str(judges))
        runner = self.runner(config)
        runner.run()
        arm = self.calls("run")[0]["args"]
        self.assertIn(str(judges), [arm[i + 1] for i, x in enumerate(arm) if x == "--hidden-absent"])
        self.assertEqual([p["kind"] for p in self.ledger()["pending"]], ["launch", "chain"])
        self.assertEqual(self.calls("judge") + self.calls("judge-chain"), [])
        (judges / "series").mkdir(parents=True)
        self.runner(config).judge()
        ledger = self.ledger()
        self.assertEqual(ledger["pending"], [])
        self.assertEqual([bool(e.get("judgment")) for e in ledger["launches"]], [True, True])
        self.assertEqual(len(self.calls("review")), 2)

    def test_hidden_material_on_the_machine_stops_before_a_launch(self):
        (self.tmp / "absent").mkdir()
        self.scenario({"a": ["result"]})
        self.runner(self.config([unit("a", "M")])).run()
        self.assertIn("hidden material", (self.work / "STOP").read_text())
        self.assertEqual(self.calls("run"), [])


if __name__ == "__main__":
    unittest.main()
