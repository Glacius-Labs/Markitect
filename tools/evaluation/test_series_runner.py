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
# and writes the run folder the suite would; `judge`, `judge-chain`, `review` and `report` write their outputs, a judgment
# invalid while FAKE_DIR/invalid-judgments counts down. Every call goes to FAKE_DIR/calls.jsonl with whether the engine
# lock existed. While the lock exists, a launch is refused as the suite refuses it.
FAKE_SUITE = textwrap.dedent(r'''
    import json, os, pathlib, sys, time
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
        if outcome == "crash":
            sys.exit(1)
        run = out if command == "run" else out / "step-01" / "run"
        (run / "variant").mkdir(parents=True)
        record = {"conditions": "native" if "--native" in args else "sandbox", "validForCounting": True, "delivered": True,
                  "exitCode": 0, "timedOut": False, "model": opt("--model"), "modelMatches": True, "reported": {}, "sandbox": {},
                  "result": {"changedPaths": ["realization/x.go"]}}
        (run / "variant" / "run.json").write_text("{}")
        if outcome == "could-not-run":
            record.update(delivered=False, exitCode=2, reported={"status": "could-not-run"})
        elif outcome == "undelivered":
            record.update(delivered=False, exitCode=1)
        elif outcome == "five-hour":
            (run / "variant" / "stream.jsonl").write_text(json.dumps({"type": "rate_limit_event", "rate_limit_info": {
                "status": "allowed", "unifiedWindows": {"five_hour": {"utilization": 0.9, "resetsAt": int(time.time()) + 300}}}}) + "\n")
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
        if "--verify" in args:
            (run / "variant" / "results").mkdir()
            state = "incomplete" if outcome == "incomplete-verify" else "verified"
            (run / "variant" / "results" / "v.json").write_text(json.dumps({"format": 1, "command": "verify", "state": state}))
        if command == "chain":
            (out / "step-01" / "task").mkdir(parents=True)
            (out / "step-01" / "task" / "start.bundle").write_text("start")
            (run / "variant" / "result.bundle").write_text("result")
            (out / "chain.json").write_text(json.dumps({"steps": [{"step": 1, "delivered": record["delivered"]}]}))
            state = pathlib.Path(opt("--state"))
            with open(state / "log.txt", "a") as f:
                f.write(out.name + "\n")
    elif command == "task":
        pathlib.Path(opt("--out")).mkdir(parents=True)
        (pathlib.Path(opt("--out")) / "task.json").write_text(json.dumps({"start": opt("--start"), "model": opt("--model")}))
    elif command == "judge":
        invalid = d / "invalid-judgments"
        left = int(invalid.read_text()) if invalid.exists() else 0
        invalid.write_text(str(max(0, left - 1)))
        always = (d / "invalid-for").read_text() if (d / "invalid-for").exists() else None  # a run its judge never judges
        pathlib.Path(opt("--out")).mkdir(parents=True)
        bad = left > 0 or bool(always and always in opt("--run").replace("\\", "/"))
        (pathlib.Path(opt("--out")) / "judgment.json").write_text(json.dumps({"valid": not bad}))
    elif command == "judge-chain":
        out = pathlib.Path(opt("--out"))
        out.mkdir(parents=True)
        rows = json.loads((pathlib.Path(opt("--chain")) / "chain.json").read_text())["steps"]
        for i, row in enumerate(rows, 1):  # as the suite does: the delivered steps only
            if row["delivered"]:
                (out / f"step-{i:02d}").mkdir()
                (out / f"step-{i:02d}" / "judgment.json").write_text(json.dumps({"valid": True}))
        (out / "chain-judgment.json").write_text("{}")
    elif command == "review":
        pathlib.Path(opt("--out")).mkdir(parents=True)
        (pathlib.Path(opt("--out")) / "review.json").write_text(json.dumps({"valid": True}))
    elif command == "report":
        pathlib.Path(opt("--out")).mkdir(parents=True)
        (pathlib.Path(opt("--out")) / "report.md").write_text("report")
''')

# A fake engine for the runner's own calls: every image asked for is there, and nothing is left.
FAKE_ENGINE = textwrap.dedent(r'''
    import sys
    args = sys.argv[1:]
    if args[:2] == ["image", "inspect"]:
        print(args[-1])
''')

# A fake `claude -p`: like the CLI near expiry, it refreshes the login in FAKE_LOGIN when its access token expires within
# five minutes, unless FAKE_REFRESH says it fails. Every call goes to FAKE_CLAUDE_CALLS, with whether the lock was held.
FAKE_CLAUDE = textwrap.dedent(r'''
    import json, os, time
    with open(os.environ["FAKE_CLAUDE_CALLS"], "a") as f:
        f.write("call locked\n" if os.path.exists(os.environ["FAKE_LOCK"]) else "call\n")
    path = os.environ["FAKE_LOGIN"]
    data = json.load(open(path))
    oauth = data["claudeAiOauth"]
    if oauth["expiresAt"] / 1000 < time.time() + 300 and os.environ.get("FAKE_REFRESH") != "fails":
        oauth.update(accessToken="sk-ant-oat01-" + "c" * 48, expiresAt=int((time.time() + 8 * 3600) * 1000))
        json.dump(data, open(path, "w"))
    print("OK")
''')


class Quiet:
    def __init__(self, reasons=()):
        self.answers = list(reasons)

    def reasons(self):
        return self.answers.pop(0) if self.answers else []


class AnyLogin:
    def ensure(self, seconds):
        pass


def unit(uid, arm, kind="fixed", step="C1", **extra):
    out = {"id": uid, "arm": arm, "kind": kind, "series": "gs", "repetition": 1, **extra}
    if kind == "fixed":
        out["step"] = step
    return out


class Fixture(unittest.TestCase):
    def setUp(self):
        self.tmp = pathlib.Path(tempfile.mkdtemp())
        self.work, self.prep, self.fake, self.bin = self.tmp / "work", self.tmp / "prepared", self.tmp / "fake", self.tmp / "bin"
        for folder in (self.fake, self.bin):
            folder.mkdir()
        (self.fake / "suite.py").write_text(FAKE_SUITE)
        (self.fake / "engine.py").write_text(FAKE_ENGINE)
        (self.bin / "markitect-remote").write_text("remote")
        self.lock = self.tmp / "NATIVE-RUN.lock"
        self.login = self.tmp / "credentials.json"
        self.login.write_text(json.dumps({"claudeAiOauth": {"accessToken": ACCESS, "refreshToken": REFRESH,
                                                            "expiresAt": int((time.time() + 8 * 3600) * 1000)}}))
        for folder in ("tasks/C1", "tasks/C2", "plants/p1/task", "steps/W0", "steps/C1", "steps/C2", "judges/judge",
                       "judges/plants/p1/judge"):
            (self.prep / folder).mkdir(parents=True)
        (self.prep / "judges" / "plants" / "p1" / "judge" / "plant.json").write_text(json.dumps({"format": 1, "kind": "defect"}))
        env = {"FAKE_DIR": str(self.fake), "FAKE_LOCK": str(self.lock), "FAKE_TOKEN": ACCESS, "FAKE_WORK": str(self.work)}
        self.addCleanup(lambda: [os.environ.pop(k, None) for k in env])
        os.environ.update(env)
        self.slept = []

    def config(self, units, protocol="counted", **series):
        gs = {"role": "confirmatory", "deadlineSeconds": 60, "steps": ["W0", "C1", "C2"], "fixed": ["C1", "C2"],
              "plants": {"p1": {"step": "C1"}}, "prepared": str(self.prep), **series}
        return {"work": str(self.work), "rev": "0" * 40, "bin": str(self.bin), "managerImage": "sha256:m", "sandboxImage": "sha256:s",
                "lock": str(self.lock), "hidden": [str(self.tmp / "absent")], "login": str(self.login), "model": MODEL, "docker": "docker",
                "engine": [sys.executable, "-I", str(self.fake / "engine.py")], "protocol": protocol, "series": {"gs": gs},
                "relaunches": 2, "launchBound": 50, "contrasts": series_runner.CONTRASTS,
                "reviews": {"repetition": 1, "chains": True} if protocol == "counted" else {}, "units": units}

    def runner(self, config, sleep=None, guard=None):
        def default_sleep(seconds):
            self.slept.append(seconds)
        return series_runner.Runner(config, suite=self.fake / "suite.py", guard=guard or Quiet(), login=AnyLogin(),
                                    sleep=sleep or default_sleep)

    def scenario(self, outcomes):
        (self.fake / "scenario.json").write_text(json.dumps(outcomes))

    def calls(self, command=None):
        calls = [json.loads(x) for x in (self.fake / "calls.jsonl").read_text().splitlines()] if (self.fake / "calls.jsonl").exists() else []
        return [c for c in calls if command is None or c["args"][0] == command]

    def arg(self, call, name):
        return call["args"][call["args"].index(name) + 1]

    def ledger(self):
        return json.loads((self.work / "ledger.json").read_text())

    def events(self, kind):
        return [e for e in map(json.loads, (self.work / "series.jsonl").read_text().splitlines()) if e["event"] == kind]

    def stop_reason(self):
        return (self.work / "STOP").read_text()


class PlanTest(unittest.TestCase):
    def prepared(self):
        spec = pathlib.Path(tempfile.mkdtemp()) / "new.json"
        spec.write_text(json.dumps({
            "name": "new", "deadlineSeconds": 3600, "steps": ["W0"] + [f"C{i}" for i in range(1, 8)], "fixed": [f"C{i}" for i in range(1, 8)],
            "plants": {f"p{i}": {"step": f"C{i + 1}"} for i in range(1, 4)}, "inputs": str(spec.parent / "inputs"),
            "judges": str(spec.parent / "judges")}))
        return spec

    def test_counted_series_per_platform(self):
        config = series_runner.plan("counted", {"work": "w"}, [self.prepared()])
        units = config["units"]
        # 222 launches and one verify launch per chain and arm, 8 in all.
        self.assertEqual(series_runner.summary(units)["launches"], 230)
        self.assertEqual(series_runner.summary(units)["byArm"], {"M": 115, "K": 115})
        self.assertEqual(config["launchBound"], 345)
        self.assertTrue(all(u.get("verify") for u in units if u["kind"] == "chain"))
        self.assertEqual([(c["endpoint"], c.get("minimumDifference"), c.get("blockBy")) for c in config["contrasts"]],
                         [("casesPassedShare", 0.05, "change"), ("casesFailed", 1, "change"), ("defectsLeft", 1, None),
                          ("ownCasesPassedShare", None, "change")])
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
        self.assertEqual(series_runner.summary(units), {"launches": 28, "byArm": {"M": 14, "K": 14}, "bySeries": {
            "library fixed": 6, "makerspace chain": 20, "makerspace verify": 2}})
        self.assertEqual([u["id"] for u in units][:6], ["library-fixed-M2-gate-M", "library-fixed-M2-gate-K", "library-fixed-M3-gate-K",
                                                        "library-fixed-M3-gate-M", "library-fixed-M4-gate-M", "library-fixed-M4-gate-K"])
        with_new = series_runner.plan("gate", {}, [self.prepared()])
        self.assertEqual((series_runner.summary(with_new["units"])["launches"], with_new["launchBound"]), (32, 48))
        self.assertIn("new-fixed-C1-gate-K", [u["id"] for u in with_new["units"]])


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
        env = {"FAKE_LOGIN": str(self.path), "FAKE_CLAUDE_CALLS": str(self.calls), "FAKE_LOCK": str(self.lock)}
        self.addCleanup(lambda: [os.environ.pop(k, None) for k in (*env, "FAKE_REFRESH")])
        os.environ.update(env)
        self.events, self.slept = [], []

    def login(self, minutes_left, refresh_until=None, sleep=None, halted=lambda: False):
        oauth = {"accessToken": ACCESS, "refreshToken": REFRESH, "expiresAt": int((time.time() + minutes_left * 60) * 1000),
                 "scopes": ["user:inference"]}
        if refresh_until is not None:
            oauth["refreshTokenExpiresAt"] = int((time.time() + refresh_until * 60) * 1000)
        self.path.write_text(json.dumps({"claudeAiOauth": oauth}))
        return series_runner.Login(self.path, self.lock, MODEL, lambda **e: self.events.append(e),
                                   claude=[sys.executable, "-I", str(self.tmp / "claude.py")], sleep=sleep or self.slept.append,
                                   halted=halted)

    def test_a_login_that_covers_the_launch_is_left_alone(self):
        self.login(120).ensure(2700)
        self.assertFalse(self.calls.exists())

    def test_refreshed_under_the_lock_before_a_launch_it_cannot_cover(self):
        login = self.login(60)
        login.ensure(2700)
        self.assertEqual(self.calls.read_text(), "call locked\n")
        self.assertFalse(self.lock.exists())
        self.assertGreater(login.left(), 7 * 3600)
        self.assertEqual(self.events[-1]["event"], "login-refresh")

    def test_marking_expired_changes_no_token(self):
        login = self.login(60)
        login.set_expiry(int(time.time() * 1000))
        oauth = json.loads(self.path.read_text())["claudeAiOauth"]
        self.assertEqual((oauth["accessToken"], oauth["refreshToken"], oauth["scopes"]), (ACCESS, REFRESH, ["user:inference"]))
        self.assertLess(abs(oauth["expiresAt"] / 1000 - time.time()), 5)
        self.assertEqual([p.name for p in self.tmp.iterdir() if p.name.startswith("credentials")], ["credentials.json"])

    def test_a_login_another_process_changed_is_not_overwritten(self):
        login = self.login(60)
        stale = self.path.read_bytes()
        self.path.write_text(json.dumps({"claudeAiOauth": {"accessToken": "sk-ant-oat01-" + "d" * 48, "expiresAt": 1}}))
        self.assertFalse(login.replace(stale, {"claudeAiOauth": {"accessToken": ACCESS, "expiresAt": 2}}))
        self.assertEqual(json.loads(self.path.read_text())["claudeAiOauth"]["expiresAt"], 1)

    def test_waits_while_a_run_holds_the_lock(self):
        self.lock.write_text("{}")
        login = self.login(60, sleep=lambda s: self.lock.unlink())  # the other holder's run ends
        login.ensure(2700)
        self.assertEqual([e["event"] for e in self.events], ["login-refresh-waits", "login-refresh"])
        self.assertEqual(self.calls.read_text(), "call locked\n")
        self.assertFalse(self.lock.exists())

    def test_a_stop_ends_the_wait_for_the_lock(self):
        self.lock.write_text("{}")
        with self.assertRaises(series_runner.Halt):
            self.login(60, halted=lambda: True).ensure(2700)
        self.assertTrue(self.lock.exists())
        self.assertFalse(self.calls.exists())

    def test_a_lock_that_cannot_be_created_stops_after_a_few_tries(self):
        from unittest import mock
        login = self.login(60)
        with mock.patch.object(series_runner.os, "open", side_effect=PermissionError("denied")):
            with self.assertRaisesRegex(series_runner.Stop, "cannot be created"):
                login.ensure(2700)
        self.assertFalse(self.calls.exists())

    def test_a_failed_refresh_stops_and_puts_the_expiry_back(self):
        os.environ["FAKE_REFRESH"] = "fails"
        login = self.login(60)
        expiry = login.oauth()["expiresAt"]
        with self.assertRaisesRegex(series_runner.Stop, "could not be refreshed"):
            login.ensure(2700)
        self.assertEqual(login.oauth()["expiresAt"], expiry)
        self.assertFalse(self.lock.exists())

    def test_an_expired_refresh_token_stops_without_a_refresh(self):
        with self.assertRaisesRegex(series_runner.Stop, "log in again"):
            self.login(60, refresh_until=0).ensure(2700)
        self.assertFalse(self.calls.exists())
        self.login(60, refresh_until=30).ensure(2700)  # valid now: the refresh gives it a new one
        self.assertEqual(self.calls.read_text(), "call locked\n")


class SeriesTest(Fixture):
    def test_each_launch_is_classed_with_its_arms_signatures(self):
        self.scenario({"m": ["could-not-run", "result"], "k": ["could-not-run"]})
        self.runner(self.config([unit("m", "M"), unit("k", "K")])).run()
        attempts = self.ledger()["attempts"]
        self.assertEqual([(a["unit"], a["class"], a["kind"], a["answer"]) for a in attempts],
                         [("m", "void", "variant runtime", "again"), ("m", "result", None, "done"), ("k", "result", None, "done")])
        self.assertEqual([self.arg(c, "--run").replace("\\", "/").split("/runs/")[1] for c in self.calls("judge")], ["m/a2"])
        self.assertTrue((self.work / "done.json").exists())
        self.assertFalse((self.work / "runner.lock").exists())

    def test_each_arm_gets_its_conditions_and_the_series_login(self):
        self.scenario({"m": ["result"], "k": ["result"]})
        self.runner(self.config([unit("m", "M"), unit("k", "K")])).run()
        m, k = self.calls("run")
        self.assertEqual((self.arg(m, "--native"), self.arg(m, "--login"), self.arg(m, "--hidden-absent")),
                         ("sha256:s", str(self.login), str(self.tmp / "absent")))
        self.assertTrue(self.arg(m, "--markitect").endswith("markitect.exe" if os.name == "nt" else "markitect"))
        self.assertEqual((self.arg(k, "--sandbox"), self.arg(k, "--login")), ("sha256:s", str(self.login)))
        self.assertEqual(k["args"][k["args"].index("--") + 1:], ["python3", "-I", "-B", "/variant/claude_code.py", "--effort", "high"])
        # K gets the task without the registry pages; M works from them.
        self.assertEqual(("--without-pages" in k["args"][:k["args"].index("--")], "--without-pages" in m["args"]), (True, False))
        # K's own failure is void, as M's are.
        signatures = json.loads((self.work / "signatures.json").read_text())["variants"]
        self.assertEqual([s["pattern"] for s in signatures["K"]], ["^error$"])

    def test_a_busy_lock_is_waited_for_and_never_held_by_the_runner(self):
        self.scenario({"m": ["result"]})
        self.lock.write_text('{"holder": "another session"}')
        released = []

        def sleep(seconds):
            if self.lock.exists():
                self.lock.unlink()
                released.append(seconds)
        self.runner(self.config([unit("m", "M")]), sleep=sleep).run()
        self.assertEqual(released, [60])
        self.assertEqual([c["lockHeld"] for c in self.calls("run")], [True, False])
        self.assertTrue(all(not c["lockHeld"] for c in self.calls() if c["args"][0] != "run"))
        self.assertEqual(len(self.ledger()["attempts"]), 1)
        self.assertEqual(len(self.events("lock-busy")), 1)

    def test_a_wait_for_a_quiet_engine_is_recorded_and_a_stop_ends_it(self):
        self.scenario({"a": ["result"], "b": ["result"]})
        config = self.config([unit("a", "M"), unit("b", "K", step="C2")])
        self.runner(config, guard=Quiet([["ryuk (started)"], []])).run()
        self.assertEqual([w["containers"] for w in self.ledger()["waits"]], [["ryuk (started)"]])
        self.assertIn("ended", self.ledger()["waits"][0])

    def test_a_stop_during_a_wait_ends_the_window_before_the_launch(self):
        self.scenario({"a": ["result"]})

        def sleep(seconds):
            (self.work / "STOP").write_text("the owner needs the engine\n")
        self.runner(self.config([unit("a", "M")]), sleep=sleep, guard=Quiet([["ryuk (running)"]] * 5)).run()
        self.assertEqual(self.calls("run"), [])
        self.assertEqual(self.stop_reason(), "the owner needs the engine\n")
        self.assertEqual(len(self.events("halt")), 1)
        self.assertFalse((self.work / "done.json").exists())

    def test_other_works_containers_are_relaunched_once_then_stop(self):
        self.scenario({"a": ["foreign", "result"], "b": ["foreign", "foreign"]})
        self.runner(self.config([unit("a", "M"), unit("b", "M", step="C2")])).run()
        self.assertEqual([(a["unit"], a["class"], a["foreign"]) for a in self.ledger()["attempts"]],
                         [("a", "invalid", True), ("a", "result", False), ("b", "invalid", True), ("b", "invalid", True)])
        self.assertIn("an invalid launch", self.stop_reason())
        self.assertFalse((self.work / "done.json").exists())

    def test_any_other_invalid_launch_stops_and_is_launched_again_once_after(self):
        self.scenario({"a": ["own-escape", "result"], "b": ["own-escape", "own-escape"], "c": ["result"]})
        config = self.config([unit("a", "M"), unit("b", "K", step="C2"), unit("c", "M", step="C2")])
        self.runner(config).run()
        self.assertEqual(len(self.ledger()["attempts"]), 1)
        self.assertIn("markitect-run-x", self.stop_reason())
        for _ in range(3):  # Build looked each time and found no change needed
            (self.work / "STOP").unlink()
            self.runner(config).run()
        self.assertEqual([(a["unit"], a["class"]) for a in self.ledger()["attempts"]],
                         [("a", "invalid"), ("a", "result"), ("b", "invalid"), ("b", "invalid"), ("c", "result")])
        self.assertTrue((self.work / "done.json").exists())

    def test_a_void_share_above_one_in_four_stops_and_its_launch_goes_on_after(self):
        self.scenario({"a": ["could-not-run"] * 3, "b": ["could-not-run"] * 3, "c": ["could-not-run", "could-not-run", "result"]})
        config = self.config([unit(u, "M") for u in ("a", "b", "c")])
        self.runner(config).run()
        self.assertEqual(len(self.ledger()["attempts"]), 8)
        self.assertIn("8 of 8 launches void", self.stop_reason())
        (self.work / "STOP").unlink()
        self.runner({**config, "relaunches": 2}).run()  # Build looked; the void launch of c still has a relaunch left
        self.assertEqual([a["class"] for a in self.ledger()["attempts"] if a["unit"] == "c"], ["void", "void", "result"])

    def test_the_stability_gate_stops_at_its_first_launch_that_is_not_a_result(self):
        for outcome in ("could-not-run", "foreign"):
            with self.subTest(outcome=outcome):
                self.setUp()
                self.scenario({"a": [outcome], "b": ["result"]})
                self.runner(self.config([unit("a", "M"), unit("b", "K")], protocol="gate")).run()
                self.assertEqual(len(self.ledger()["attempts"]), 1)
                self.assertIn("stability gate", self.stop_reason())

    def test_the_stability_gate_writes_its_verdict(self):
        self.scenario({"a": ["result"], "ch/W0": ["result"], "ch/C1": ["result"], "ch": ["result"]})
        config = self.config([unit("a", "M"), unit("ch", "M", kind="chain", steps=["W0", "C1"], verify=True)], protocol="gate")
        config["series"]["gs"]["role"] = "public"
        config["series"]["makerspace"] = config["series"]["gs"]
        config["units"][1]["series"] = "makerspace"
        self.runner(config).run()
        verdict = json.loads((self.work / "gate.json").read_text())
        self.assertEqual(verdict["checks"], {"everyLaunchAResult": True, "everyResultJudged": True, "everyChainJudged": True,
                                             "everyChainVerified": True, "noCredential": True, "nothingLeft": True, "mLandedTheFirstChainStep": True})
        self.assertTrue(verdict["passes"])

    def test_a_credential_in_the_evidence_stops(self):
        self.scenario({"a": ["leak"], "b": ["result"]})
        self.runner(self.config([unit("a", "M"), unit("b", "K")])).run()
        self.assertIn("leak.txt", self.stop_reason())
        self.assertEqual(len(self.ledger()["attempts"]), 1)

    def test_an_invalid_judgment_is_judged_again_once_then_stops(self):
        (self.fake / "invalid-judgments").write_text("1")
        self.scenario({"a": ["result"], "b": ["result"]})
        self.runner(self.config([unit("a", "M"), unit("b", "K", step="C2")])).run()
        self.assertEqual([e.get("judgment") for e in self.ledger()["launches"]], ["judgments/a-a1-j2", "judgments/b-a1"])
        (self.fake / "invalid-judgments").write_text("2")
        self.setUp()
        (self.fake / "invalid-judgments").write_text("2")
        self.scenario({"a": ["result"], "b": ["result"]})
        self.runner(self.config([unit("a", "M"), unit("b", "K", step="C2")])).run()
        self.assertIn("no valid judgment of a-a1 twice", self.stop_reason())
        self.assertEqual([p["id"] for p in self.ledger()["pending"]], ["a-a1"])
        self.assertEqual(len(self.ledger()["failures"]), 2)
        (self.work / "STOP").unlink()  # the judge was fixed: the next window judges it in a fresh folder first
        self.runner(self.config([unit("a", "M"), unit("b", "K", step="C2")])).run()
        self.assertEqual([e.get("judgment") for e in self.ledger()["launches"]], ["judgments/a-a1-j3", "judgments/b-a1"])
        self.assertEqual(self.ledger()["pending"], [])

    def test_a_judge_that_keeps_failing_stops_only_the_window_of_its_launch(self):
        (self.fake / "invalid-for").write_text("runs/a/")
        self.scenario({"a": ["result"], "b": ["result"]})
        config = self.config([unit("a", "M"), unit("b", "K", step="C2")])
        self.runner(config).run()
        self.assertIn("no valid judgment of a-a1 twice", self.stop_reason())
        (self.work / "STOP").unlink()
        self.runner(config).run()  # the next window goes on; the end tries the judge once more
        self.assertEqual(len(self.calls("run")), 2)
        done = json.loads((self.work / "done.json").read_text())
        self.assertEqual([(p["id"], p["failed"]) for p in done["pending"]], [("a-a1", True)])
        self.assertEqual(sorted(p.name for p in (self.work / "judgments").iterdir()), ["a-a1", "a-a1-j2", "a-a1-j3", "a-a1-j4", "b-a1"])

    def test_a_judgment_the_runner_did_not_finish_stays_pending_and_is_made_in_the_next_window(self):
        self.scenario({"a": ["result"], "b": ["result"]})
        config = self.config([unit("a", "M"), unit("b", "K", step="C2")])
        runner = self.runner(config)
        plain = runner.suite

        def killed_at_the_first_judgment(name, args):
            if args[0] == "judge" and not self.calls("judge"):
                raise RuntimeError("killed")
            return plain(name, args)
        runner.suite = killed_at_the_first_judgment
        runner.run()
        self.assertIn("the runner failed", self.stop_reason())
        self.assertEqual([(e["id"], e.get("judgment")) for e in self.ledger()["launches"]], [("a-a1", None)])
        self.assertEqual([p["id"] for p in self.ledger()["pending"]], ["a-a1"])
        (self.work / "STOP").unlink()
        self.runner(config).run()
        self.assertEqual([e.get("judgment") for e in self.ledger()["launches"]], ["judgments/a-a1", "judgments/b-a1"])
        self.assertEqual(self.ledger()["pending"], [])

    def test_a_five_hour_wait_outlasts_a_stop(self):
        self.scenario({"a": ["five-hour"], "b": ["result"]})
        config = self.config([unit("a", "M"), unit("b", "K", step="C2")])

        def sleep(seconds):
            (self.work / "STOP").write_text("window ends\n")
        self.runner(config, sleep=sleep).run()
        self.assertGreater(self.ledger()["quotaWaitUntil"], time.time())
        (self.work / "STOP").unlink()
        self.runner(config).run()
        self.assertGreater(sum(self.slept), 200)
        self.assertIsNone(self.ledger()["quotaWaitUntil"])
        self.assertEqual(len(self.calls("run")), 2)

    def test_resumes_after_a_stop_without_repeating_what_counted(self):
        self.scenario({"a": ["quota"], "b": ["stop-file"], "c": ["result"]})
        config = self.config([unit("a", "M"), unit("b", "K"), unit("c", "M", step="C2")])
        self.runner(config).run()
        self.assertIn("quota", self.stop_reason())
        (self.work / "STOP").unlink()
        self.runner(config).run()
        self.assertEqual(self.stop_reason(), "window ends\n")
        (self.work / "STOP").unlink()
        self.runner(config).run()
        self.assertEqual([a["unit"] for a in self.ledger()["attempts"]], ["a", "b", "c"])
        self.assertEqual(self.ledger()["done"], ["a", "b", "c"])
        self.assertEqual(len(self.ledger()["windows"]), 3)
        self.assertTrue((self.work / "done.json").exists())
        with self.assertRaises(SystemExit):
            self.runner(config).run()

    def test_a_launch_the_runner_did_not_live_to_record_is_recorded_and_relaunched(self):
        (self.work / "runs" / "a" / "a1").mkdir(parents=True)  # the suite died with the runner and wrote no record
        self.scenario({"a": ["result"]})
        self.runner(self.config([unit("a", "M")])).run()
        self.assertEqual([(a["run"], a["class"], a["exit"]) for a in self.ledger()["attempts"]],
                         [("runs/a/a1", "void", None), ("runs/a/a2", "result", 0)])

    def test_one_runner_per_work_folder(self):
        self.work.mkdir()
        (self.work / "runner.lock").write_text("{}")
        with self.assertRaisesRegex(SystemExit, "another runner"):
            self.runner(self.config([unit("a", "M")])).run()

    def test_a_frozen_binary_that_changes_between_windows_stops(self):
        self.scenario({"a": ["stop-file"], "b": ["result"]})
        config = self.config([unit("a", "M"), unit("b", "K")])
        self.runner(config).run()
        (self.work / "STOP").unlink()
        (self.bin / "markitect-remote").write_text("rebuilt")
        self.runner(config).run()
        self.assertIn("frozen bin changed", self.stop_reason())
        self.assertEqual(len(self.ledger()["attempts"]), 1)

    def test_a_chain_relaunches_a_void_step_from_its_state_and_is_judged_whole(self):
        self.scenario({"ch/W0": ["result"], "ch/C1": ["could-not-run", "result"], "ch/C2": ["result"]})
        self.runner(self.config([unit("ch", "M", kind="chain", steps=["W0", "C1", "C2"])])).run()
        self.assertEqual((self.work / "runs" / "ch" / "state" / "log.txt").read_text().split(), ["W0-a1", "C1-a2", "C2-a1"])
        chain = self.calls("chain")
        self.assertNotIn("--start", chain[0]["args"])
        self.assertTrue(self.arg(chain[2], "--start").replace("\\", "/").endswith("ch/W0-a1/step-01/run/variant/result.bundle"))
        self.assertEqual(self.arg(self.calls("judge-chain")[0], "--steps"), "W0,C1,C2")
        ledger = self.ledger()
        self.assertEqual([c["id"] for c in ledger["chains"]], ["ch"])
        self.assertEqual([bool(e.get("judgment")) for e in ledger["launches"]], [True, False, True, True])
        self.assertEqual([bool(e.get("review")) for e in ledger["launches"]], [False, False, False, True])
        self.assertEqual([self.arg(c, "--out").replace("\\", "/").split("/report/")[1] for c in self.calls("report")],
                         ["all", "gs", "gs-fixed", "gs-chains"])
        self.assertEqual(json.loads((self.work / "ledger-gs-chains.json").read_text())["chains"][0]["id"], "ch")
        self.assertEqual(json.loads((self.work / "ledger-gs-fixed.json").read_text())["launches"], [])

    def test_a_chains_end_state_is_verified_by_the_2_0_host_whichever_arm_made_it(self):
        self.scenario({"ch/W0": ["result"], "ch/C1": ["result"], "ch": ["could-not-run", "result"]})
        self.runner(self.config([unit("ch", "K", kind="chain", steps=["W0", "C1"], verify=True)])).run()
        task = self.calls("task")[0]
        self.assertTrue(self.arg(task, "--start").replace("\\", "/").endswith("runs/ch/C1-a1/step-01/run/variant/result.bundle"))
        self.assertEqual(pathlib.Path(self.arg(task, "--model")), self.prep / "steps" / "C1" / "model")
        verify = [c for c in self.calls("run") if "--verify" in c["args"]]
        self.assertEqual(len(verify), 2)  # the void one is relaunched
        self.assertIn("--native", verify[0]["args"])
        self.assertTrue(any(x.endswith("markitect_arm.py") for x in verify[0]["args"]))
        ledger = self.ledger()
        self.assertEqual([(a["arm"], a["class"]) for a in ledger["attempts"] if a["step"] == "verify"], [("V", "void"), ("V", "result")])
        self.assertEqual(ledger["verifies"], {"ch": "runs/ch/verify-a2/variant/results/v.json"})
        self.assertEqual(ledger["chains"][0]["verifyFull"], "runs/ch/verify-a2/variant/results/v.json")
        self.assertNotIn("verify", [e["step"] for e in ledger["launches"]])  # not a launch of either arm in the report

    def test_a_gate_start_of_the_new_project_is_verified_after_its_launch(self):
        self.scenario({"k": ["result", "result"]})  # the launch, then its verify launch
        self.runner(self.config([unit("k", "K", step="C2", verify=True)], protocol="gate")).run()
        task = self.calls("task")[0]
        self.assertTrue(self.arg(task, "--start").replace("\\", "/").endswith("runs/k/a1/variant/result.bundle"))
        self.assertEqual(pathlib.Path(self.arg(task, "--model")), self.prep / "steps" / "C2" / "model")
        self.assertEqual([c["args"][0] for c in self.calls() if c["args"][0] == "run"], ["run", "run"])
        self.assertEqual(self.ledger()["verifies"], {"k": "runs/k/verify-a1/variant/results/v.json"})
        self.assertTrue(json.loads((self.work / "gate.json").read_text())["checks"]["everyChainVerified"])

    def test_the_end_state_verified_is_the_last_delivered_steps_against_its_own_model(self):
        self.scenario({"ch/W0": ["result"], "ch/C1": ["undelivered"], "ch": ["result"]})
        self.runner(self.config([unit("ch", "M", kind="chain", steps=["W0", "C1"], verify=True)])).run()
        task = self.calls("task")[0]
        self.assertTrue(self.arg(task, "--start").replace("\\", "/").endswith("runs/ch/W0-a1/step-01/run/variant/result.bundle"))
        self.assertEqual(pathlib.Path(self.arg(task, "--model")), self.prep / "steps" / "W0" / "model")
        self.assertTrue((self.work / "verify-tasks" / "ch").is_dir())
        self.assertFalse((self.work / "verify-tasks" / "ch.partial").exists())

    def test_a_chain_that_delivered_nothing_has_no_end_state_to_verify(self):
        self.scenario({"ch/W0": ["undelivered"]})
        self.runner(self.config([unit("ch", "K", kind="chain", steps=["W0"], verify=True)])).run()
        self.assertEqual(self.calls("task"), [])
        self.assertEqual(self.ledger()["verifies"], {})
        self.assertEqual(len(self.events("verify-skipped")), 1)

    def test_a_verify_that_did_not_check_every_item_fails_the_gate(self):
        self.scenario({"ch/W0": ["result"], "ch": ["incomplete-verify"]})
        config = self.config([unit("ch", "M", kind="chain", steps=["W0"], verify=True)], protocol="gate")
        config["series"]["gs"]["role"] = "public"
        self.runner(config).run()
        verdict = json.loads((self.work / "gate.json").read_text())
        self.assertFalse(verdict["checks"]["everyChainVerified"])
        self.assertFalse(verdict["passes"])
        self.assertIn("ch", self.ledger()["verifies"])  # filed, and the report shows its state

    def test_a_chain_with_an_undelivered_step_is_judged_on_its_delivered_ones(self):
        self.scenario({"ch/W0": ["undelivered"], "ch/C1": ["result"]})
        self.runner(self.config([unit("ch", "K", kind="chain", steps=["W0", "C1"])])).run()
        self.assertTrue(self.arg(self.calls("chain")[1], "--start").replace("\\", "/").endswith("ch/W0-a1/step-01/task/start.bundle"))
        ledger = self.ledger()
        self.assertEqual([c["id"] for c in ledger["chains"]], ["ch"])
        self.assertEqual([e.get("judgment") for e in ledger["launches"]], [None, "judgments/ch-chain/step-02"])
        self.assertTrue((self.work / "done.json").exists())

    def test_a_chain_resumes_at_its_next_step_and_ends_at_an_invalid_one(self):
        self.scenario({"ch/W0": ["stop-file"], "ch/C1": ["own-escape"]})
        config = self.config([unit("ch", "K", kind="chain", steps=["W0", "C1", "C2"])])
        self.runner(config).run()
        (self.work / "STOP").unlink()
        self.runner(config).run()
        self.assertIn("an invalid launch", self.stop_reason())
        (self.work / "STOP").unlink()
        self.runner(config).run()
        self.assertEqual([self.arg(c, "--steps") for c in self.calls("judge-chain")], ["W0"])
        self.assertEqual(self.ledger()["progress"]["ch"]["kept"], [["W0", "runs/ch/W0-a1"]])
        self.assertTrue((self.work / "done.json").exists())

    def test_a_hidden_judge_waits_off_the_machine(self):
        judges = self.tmp / "hidden-judges"
        self.scenario({"a": ["result"], "ch/W0": ["result"], "p": ["result"]})
        config = self.config([unit("a", "M"), {"id": "p", "arm": "K", "kind": "plant", "series": "gs", "plant": "p1", "repetition": 1},
                              unit("ch", "K", kind="chain", steps=["W0"])], judges=str(judges))
        self.runner(config).run()
        arm = self.calls("run")[0]["args"]
        self.assertIn(str(judges), [arm[i + 1] for i, x in enumerate(arm) if x == "--hidden-absent"])
        self.assertEqual([p["kind"] for p in self.ledger()["pending"]], ["launch", "launch", "chain"])
        self.assertEqual(self.calls("judge") + self.calls("judge-chain"), [])
        self.assertNotIn("planted", self.ledger()["tasks"]["plant-p1"])
        # Nothing of a series judged later is in a report while it waits: the whole report leaves it out.
        self.assertEqual([self.arg(c, "--out").replace("\\", "/").split("/report/")[1] for c in self.calls("report")], ["all"])
        self.assertEqual(json.loads((self.work / "ledger-all.json").read_text())["launches"], [])
        (judges / "judge").mkdir(parents=True)
        (judges / "plants" / "p1" / "judge").mkdir(parents=True)
        (judges / "plants" / "p1" / "judge" / "plant.json").write_text(json.dumps({"format": 1, "kind": "gap"}))
        self.runner(config).judge()
        ledger = self.ledger()
        self.assertEqual(ledger["pending"], [])
        self.assertEqual([bool(e.get("judgment")) for e in ledger["launches"]], [True, True, True])
        self.assertEqual(ledger["tasks"]["plant-p1"]["planted"], "gap")
        self.assertEqual(len(self.calls("review")), 2)
        self.assertEqual(ledger["released"], {"gs": True})
        self.assertEqual([self.arg(c, "--out").replace("\\", "/").split("/report/")[1] for c in self.calls("report")][1:],
                         ["all", "gs", "gs-fixed", "gs-chains"])

    def test_a_series_judged_later_is_released_only_when_all_of_it_ran(self):
        judges = self.tmp / "hidden-judges"
        self.scenario({"a": ["stop-file"], "b": ["result"]})
        config = self.config([unit("a", "M"), unit("b", "K", step="C2")], judges=str(judges))
        self.runner(config).run()
        (judges / "judge").mkdir(parents=True)
        self.runner(config).judge()
        self.assertEqual(self.ledger().get("released", {}), {})
        self.assertEqual(self.ledger()["pending"], [])

    def test_a_plant_runs_from_its_task_and_is_judged_at_its_step(self):
        self.scenario({"p": ["result"]})
        self.runner(self.config([{"id": "p", "arm": "K", "kind": "plant", "series": "gs", "plant": "p1", "repetition": 1}])).run()
        self.assertEqual(pathlib.Path(self.arg(self.calls("run")[0], "--task")), self.prep / "plants" / "p1" / "task")
        judge = self.calls("judge")[0]
        self.assertEqual((pathlib.Path(self.arg(judge, "--judge")), self.arg(judge, "--step")),
                         (self.prep / "judges" / "plants" / "p1" / "judge", "C1"))
        self.assertEqual(self.ledger()["tasks"]["plant-p1"]["planted"], "defect")

    def test_hidden_material_on_the_machine_stops_before_a_launch(self):
        (self.tmp / "absent").mkdir()
        self.scenario({"a": ["result"]})
        self.runner(self.config([unit("a", "M")])).run()
        self.assertIn("hidden material", self.stop_reason())
        self.assertEqual(self.calls("run"), [])


if __name__ == "__main__":
    unittest.main()
