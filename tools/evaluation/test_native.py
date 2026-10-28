"""Offline tests of the native arm mode: a fake engine that records events, and the scripted stand-in arm."""
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
import classify  # noqa: E402
import native  # noqa: E402
import sandbox  # noqa: E402
import suite  # noqa: E402

MODEL = "claude:claude-sonnet-5-5"
IMAGE = "sha256:" + "e" * 64

# A fake engine with events. State and events live in FAKE_DIR; container IDs are their names. `run` executes the
# command on the host with each mount target replaced by its source and python3 by this Python; the container
# stays until `rm -f`. `create` and `start` make a container that stays running. `events` prints every event so far
# and then follows the log until it is stopped.
FAKE_ENGINE = textwrap.dedent(r'''
    import json, os, subprocess, sys, time
    args, d = sys.argv[1:], os.environ["FAKE_DIR"]
    path, log = os.path.join(d, "engine.json"), os.path.join(d, "events.jsonl")
    state = json.load(open(path)) if os.path.exists(path) else {"containers": {}, "networks": {}}
    def save():
        json.dump(state, open(path, "w"))
    def event(kind, action, ident, **attrs):
        with open(log, "a") as f:
            f.write(json.dumps({"Type": kind, "Action": action, "id": ident, "Actor": {"ID": ident, "Attributes": attrs}}) + "\n")
    def labels(name):
        k, _, v = state["containers"][name]["label"].partition("=")
        return {k: v} if k else {}
    with open(os.path.join(d, "calls.jsonl"), "a") as f:
        f.write(json.dumps(args) + "\n")
    def opt(name, default=None):
        return args[args.index(name) + 1] if name in args else default
    def labelled(kind, flt):
        key, _, value = flt.partition("=")
        found = []
        for n, item in state[kind].items():
            k, _, v = item.get("label", "").partition("=")
            if k == key and (not value or v == value):
                found.append(n)
        return found
    def make(name, net, label):
        state["containers"][name] = {"label": label or "", "network": net}; save()
        event("container", "create", name, name=name, **labels(name))
        event("network", "connect", net, name=net, container=name)
    if args[:2] == ["image", "inspect"]:
        print(os.environ["FAKE_IMAGE"])
    elif args[0] == "ps":
        print("\n".join(labelled("containers", opt("--filter")[len("label="):])))
    elif args[:2] == ["network", "ls"]:
        print("\n".join(labelled("networks", opt("--filter")[len("label="):])))
    elif args[:2] == ["network", "create"]:
        state["networks"][args[-1]] = {"label": opt("--label", "")}; save()
        event("network", "create", args[-1], name=args[-1])
    elif args[:2] == ["network", "connect"]:
        event("network", "connect", args[2], name=args[2], container=args[3])
    elif args[:2] == ["network", "inspect"]:
        sys.exit(0 if args[-1] in state["networks"] else 1)
    elif args[:2] == ["network", "rm"]:
        if any(c["network"] == args[2] for c in state["containers"].values()):
            sys.exit("network has active endpoints")
        state["networks"].pop(args[2], None); save()
    elif args[0] == "create":
        make(opt("--name"), opt("--network", "bridge"), opt("--label"))
    elif args[0] == "start":
        event("container", "start", args[1], name=args[1], **labels(args[1]))
    elif args[0] == "kill":
        pass
    elif args[0] == "inspect" and opt("--format") == "{{.Id}}":
        sys.exit(0 if args[-1] in state["containers"] else 1)
    elif args[0] == "inspect" and "NetworkSettings" in opt("--format"):
        print(json.dumps({n: {"IPAddress": "10.9.0.2"} for n in state["networks"]}))
    elif args[0] == "inspect" and ".State.Running" in opt("--format"):
        print("true")
    elif args[0] == "logs":
        print(json.dumps({"ready": True, "port": 443}))
    elif args[:2] == ["rm", "-f"]:
        state["containers"].pop(args[2], None); save()
    elif args[0] == "events":
        seen = 0
        while True:
            lines = open(log).read().splitlines() if os.path.exists(log) else []
            for line in lines[seen:]:
                print(line, flush=True)
            seen = len(lines)
            time.sleep(0.1)
    elif args[0] == "run":
        name = opt("--name")
        make(name, opt("--network", "bridge"), opt("--label"))
        event("container", "start", name, name=name, **labels(name))
        mounts, env, i = [], dict(os.environ), 1
        while args[i] != os.environ["FAKE_IMAGE"]:
            if args[i] == "--mount":
                fields = dict(f.partition("=")[::2] for f in args[i + 1].split(","))
                mounts.append((fields["dst"], fields["src"]))
            if args[i] == "-e":
                k, _, v = args[i + 1].partition("=")
                env[k] = v
            i += 1 if args[i].startswith("--add-host=") or args[i] in ("--init", "--read-only") else 2
        mounts.sort(key=lambda m: -len(m[0]))
        def host(a):
            for dst, src in mounts:
                if a == dst or a.startswith(dst + "/"):
                    return src + a[len(dst):]
            return sys.executable if a == "python3" else a
        code = subprocess.run([host(a) for a in args[i + 1:]], env={k: host(v) for k, v in env.items()}).returncode
        event("container", "die", name, name=name)
        sys.exit(code)
    else:
        sys.exit("fake engine: unknown call " + " ".join(args))
''')

# A native arm that starts one container and then sleeps past the deadline.
SLEEPER = textwrap.dedent('''
    import json, os, subprocess, sys, time
    engine = json.loads(os.environ["EVALUATION_ENGINE"])
    subprocess.run([*engine, "create", "--name", "slow-run", "--label", os.environ["EVALUATION_LABEL"], "--network",
                    os.environ["EVALUATION_NETWORK"], os.environ["FAKE_IMAGE"]], check=True)
    subprocess.run([*engine, "start", "slow-run"], check=True)
    time.sleep(60)
''')


class NativeTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)
        (self.tmp / "engine.py").write_text(FAKE_ENGINE)
        self.engine = [sys.executable, "-I", str(self.tmp / "engine.py")]
        self._env = dict(os.environ)
        os.environ.update(FAKE_DIR=str(self.tmp), FAKE_IMAGE=IMAGE)
        login = {"claudeAiOauth": {"accessToken": "sk-ant-oat-native-secret", "refreshToken": "sk-ant-ort-native-refresh",
                                   "expiresAt": int((time.time() + 7200) * 1000)}}
        (self.tmp / "login.json").write_text(json.dumps(login))
        self.hidden = self.tmp / "hidden-judges"
        self.native = native.Native(self.engine, IMAGE, sandbox.load_series(sandbox.DEFAULT_SERIES), self.tmp / "login.json", "claude",
                                    [self.hidden])
        self.standin = [sys.executable, "-I", "-B", str(HERE / "native_standin.py"), "--image", IMAGE]

    def tearDown(self):
        os.environ.clear()
        os.environ.update(self._env)
        self._tmp.cleanup()

    def task(self, mode, deadline=600):
        (self.tmp / (mode + ".md")).write_text(mode + "\n")
        suite.make_task(self.tmp / (mode + ".md"), None, None, deadline, self.tmp / ("task-" + mode))
        return self.tmp / ("task-" + mode)

    def run_mode(self, mode, variant=None, deadline=600):
        out = self.tmp / ("run-" + mode)
        return suite.run(self.task(mode, deadline), variant or self.standin, out, MODEL, sandbox=self.native), out

    def test_a_native_arm_delivers_with_its_containers_on_the_network(self):
        record, out = self.run_mode("change")
        box = record["sandbox"]
        self.assertTrue(record["delivered"], record)
        self.assertEqual((record["conditions"], record["validForCounting"]), ("native", True))
        self.assertEqual([c["networks"] for c in box["armContainers"]], [[box["network"]]])
        self.assertEqual((box["escapes"], box["residue"]), ([], []))
        self.assertEqual(box["variantEnv"]["EVALUATION_NETWORK"], box["network"])
        self.assertEqual(box["variantEnv"]["EVALUATION_HOSTS"], "api.anthropic.com:10.9.0.2")
        self.assertIn("A\tSTANDIN.md", record["result"]["changedPaths"])
        self.assertTrue((out / "engine-events.jsonl").is_file())
        self.assertEqual(classify.classify(out)["class"], "result")
        run = next(c for c in map(json.loads, (self.tmp / "calls.jsonl").read_text().splitlines()) if c[0] == "run")
        for flag in ("--read-only", "--init"):
            self.assertIn(flag, run)
        self.assertEqual(run[run.index("--dns") + 1], sandbox.NO_DNS)
        self.assertNotIn("sk-ant-ort-native-refresh", (self.tmp / "calls.jsonl").read_text())

    def test_a_run_that_does_not_land_delivers_its_start(self):
        record, _ = self.run_mode("nothing")
        self.assertTrue(record["delivered"])
        self.assertEqual(record["result"]["changedPaths"], [])

    def test_a_container_off_the_network_is_an_escape(self):
        record, out = self.run_mode("escape")
        self.assertFalse(record["validForCounting"])
        self.assertEqual([c["networks"] for c in record["sandbox"]["escapes"]], [["bridge"]])
        result = classify.classify(out)
        self.assertEqual(result["class"], "invalid")
        self.assertIn("outside the evaluation network", json.dumps(result["evidence"]))

    def test_a_container_the_arm_leaves_is_residue_and_removed(self):
        record, out = self.run_mode("leave")
        self.assertFalse(record["validForCounting"])
        self.assertTrue(any(r.startswith("standin-") for r in record["sandbox"]["residue"]), record["sandbox"]["residue"])
        self.assertEqual(json.loads((self.tmp / "engine.json").read_text())["containers"], {})
        self.assertEqual(classify.classify(out)["class"], "void")

    def test_at_the_deadline_the_suite_stops_what_the_arm_left(self):
        (self.tmp / "sleeper.py").write_text(SLEEPER)
        record, out = self.run_mode("sleep", [sys.executable, "-B", str(self.tmp / "sleeper.py")], deadline=4)
        self.assertTrue(record["timedOut"])
        self.assertEqual(record["sandbox"]["armStoppedAtDeadline"], ["slow-run"])
        self.assertEqual(record["sandbox"]["residue"], [])
        self.assertTrue(record["validForCounting"])
        self.assertEqual(classify.classify(out)["class"], "result")

    def test_hidden_material_on_the_machine_refuses_the_run(self):
        self.hidden.mkdir()
        record, out = self.run_mode("change")
        self.assertIn("hidden material", record["sandbox"]["error"])
        self.assertFalse(record["validForCounting"])
        self.assertFalse((self.tmp / "calls.jsonl").exists() and any(
            c[0] == "run" for c in map(json.loads, (self.tmp / "calls.jsonl").read_text().splitlines())))

    def test_the_login_copy_is_access_only_and_gone_after_the_run(self):
        seen = self.tmp / "seen.json"
        (self.tmp / "peek.py").write_text(textwrap.dedent(f'''
            import os, pathlib, shutil, sys
            shutil.copyfile(os.environ["EVALUATION_LOGIN"], {str(seen)!r})
            pathlib.Path({str(self.tmp / "login-path.txt")!r}).write_text(os.environ["EVALUATION_LOGIN"])
        '''))
        self.run_mode("peek", [sys.executable, "-B", str(self.tmp / "peek.py")])
        self.assertNotIn("refreshToken", seen.read_text())
        self.assertFalse(pathlib.Path((self.tmp / "login-path.txt").read_text()).exists())

    def test_a_chain_runs_natively(self):
        steps = []
        for name in ("one", "two"):
            (self.tmp / "steps" / name).mkdir(parents=True)
            (self.tmp / "steps" / name / "task.md").write_text("change\n")
            steps.append(self.tmp / "steps" / name)
        summary = suite.chain(steps, None, 600, self.standin, self.tmp / "chain", MODEL, self.native)
        self.assertEqual([s["validForCounting"] for s in summary["steps"]], [True, True])
        self.assertEqual(summary["conditions"], "native")

    def test_the_cli_checks_the_native_options(self):
        base = ["run", "--task", str(self.task("change")), "--out", str(self.tmp / "cli"), "--model", MODEL, "--native", IMAGE]
        for extra, message in (([], "--hidden-absent"), (["--hidden-absent", "x", "--variant-code", "y"], "--variant-code"),
                               (["--hidden-absent", "x", "--sandbox", IMAGE], "exclude")):
            with self.subTest(extra=extra):
                err = self.tmp / "err.txt"
                with open(err, "w") as f:
                    saved, sys.stderr = sys.stderr, f
                    try:
                        self.assertEqual(suite.main([*base, *extra, "--", "x"]), 2)
                    finally:
                        sys.stderr = saved
                self.assertIn(message, err.read_text())


if __name__ == "__main__":
    unittest.main()
