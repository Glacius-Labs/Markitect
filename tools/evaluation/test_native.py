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
    state = json.load(open(path)) if os.path.exists(path) else {"containers": {}, "networks": {}, "volumes": {}, "seq": 0}
    def save():
        json.dump(state, open(path, "w"))
    # Networks have IDs of their own, nNNN-NAME, so that a name used twice names two networks.
    def netid(name):
        return state["networks"][name]["id"] if name in state["networks"] else name
    def network(ref):
        return next((n for n, item in state["networks"].items() if ref in (n, item["id"])), None)
    def event(kind, action, ident, **attrs):
        with open(log, "a") as f:
            # As Docker 29 writes them: the object's ID in Actor.ID only.
            f.write(json.dumps({"Type": kind, "Action": action, "Actor": {"ID": ident, "Attributes": attrs}}) + "\n")
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
        event("network", "connect", netid(net), name=net, container=name)
    if args[:2] == ["image", "inspect"]:
        print(os.environ["FAKE_IMAGE"])
    elif args[0] == "ps":
        print("\n".join(labelled("containers", opt("--filter")[len("label="):])))
    elif args[:2] == ["network", "ls"]:
        print("\n".join(labelled("networks", opt("--filter")[len("label="):])))
    elif args[:2] == ["network", "create"]:
        options = dict(o.partition("=")[::2] for o in (args[i + 1] for i, a in enumerate(args) if a == "--opt"))
        state["seq"] += 1
        state["networks"][args[-1]] = {"label": opt("--label", ""), "internal": "--internal" in args, "options": options,
                                       "id": f"n{state['seq']:03d}-{args[-1]}"}; save()
        event("network", "create", netid(args[-1]), name=args[-1])
    elif args[:2] == ["network", "connect"]:
        event("network", "connect", netid(args[2]), name=args[2], container=args[3])
    elif args[:2] == ["network", "inspect"]:
        name = network(args[-1])
        if name is None:
            sys.exit(1)
        if "Internal" in (opt("--format") or ""):
            if os.environ.get("FAKE_UNINSPECTABLE") == name:
                sys.exit("no such network")
            net = state["networks"][name]
            print(json.dumps(net["internal"]), json.dumps(net["options"]))
    elif args[:2] == ["network", "rm"]:
        name = network(args[2])
        if any(c["network"] == name for c in state["containers"].values()):
            sys.exit("network has active endpoints")
        state["networks"].pop(name, None); save()
    elif args[:2] == ["volume", "create"]:
        state["volumes"][args[-1]] = {}; save()
        event("volume", "create", args[-1], driver="local")
    elif args[:2] == ["volume", "inspect"]:
        sys.exit(0 if args[-1] in state["volumes"] else 1)
    elif args[:2] == ["volume", "rm"]:
        state["volumes"].pop(args[-1], None); save()
    elif args[0] == "create":
        make(opt("--name"), opt("--network", "bridge"), opt("--label"))
    elif args[0] == "start":
        event("container", "start", args[1], name=args[1], **labels(args[1]))
    elif args[0] == "kill":
        pass
    elif args[0] == "inspect" and opt("--format") == "{{.Id}}":
        if args[-1] not in state["containers"]:
            sys.exit(1)
        print(args[-1])
    elif args[0] == "inspect" and "NetworkSettings" in opt("--format"):
        print(json.dumps({n: {"IPAddress": "10.9.0.2"} for n in state["networks"]}))
    elif args[0] == "inspect" and ".State.Running" in opt("--format"):
        print("true")
    elif args[0] == "logs":
        print(json.dumps({"ready": True, "port": 443}))
    elif args[:2] == ["rm", "-f"]:
        state["containers"].pop(args[2], None); save()
    elif args[0] == "events":
        if os.environ.get("FAKE_EVENTS") == "dead":
            sys.exit("cannot connect to the engine")
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


# A native arm whose agent, like the 2.0 host's, joins the evaluation network and a network of the arm's own, created
# with ARM_NETWORK_FLAGS, as a Run's remote and agents share one.
OWN_NETWORK = textwrap.dedent('''
    import json, os, subprocess
    engine = json.loads(os.environ["EVALUATION_ENGINE"])
    e = lambda *a: subprocess.run([*engine, *a], check=True, capture_output=True)
    if os.environ.get("ARM_RECREATE"):
        # An internal network first, then another of the same name that is not.
        e("network", "create", "--internal", "arm-run-net")
        e("network", "rm", "arm-run-net")
    if os.environ.get("ARM_VOLUME"):
        e("volume", "create", "arm-left-volume")
    e("network", "create", *os.environ["ARM_NETWORK_FLAGS"].split(), "arm-run-net")
    e("create", "--name", "arm-agent", "--label", os.environ["EVALUATION_LABEL"], "--network", os.environ["EVALUATION_NETWORK"],
      os.environ["FAKE_IMAGE"])
    e("network", "connect", "arm-run-net", "arm-agent")
    e("start", "arm-agent")
    e("rm", "-f", "arm-agent")
    e("network", "rm", "arm-run-net")
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
        # The gateway is the suite's, known by its ID, and still there for its log.
        self.assertNotIn(box["runId"] + "-gateway", [c["name"] for c in box["armContainers"]])
        self.assertIn("allowed", box["egress"])
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

    def test_without_an_event_record_the_check_fails_closed(self):
        os.environ["FAKE_EVENTS"] = "dead"
        record, out = self.run_mode("change")
        self.assertFalse(record["validForCounting"])
        self.assertIn("event record", record["sandbox"]["error"])
        self.assertEqual(classify.classify(out)["class"], "void")

    def test_an_internal_network_of_the_arm_is_allowed_and_any_other_is_an_escape(self):
        (self.tmp / "own.py").write_text(OWN_NETWORK)
        for flags, valid in (("--internal --opt com.docker.network.bridge.gateway_mode_ipv4=isolated", True), ("--driver bridge", False)):
            with self.subTest(flags=flags):
                os.environ["ARM_NETWORK_FLAGS"] = flags
                record, out = self.run_mode("own-" + str(valid), [sys.executable, "-B", str(self.tmp / "own.py")])
                box = record["sandbox"]
                self.assertEqual(record["validForCounting"], valid, box.get("escapes"))
                self.assertEqual([(n["name"], n["internal"]) for n in box["armNetworks"]], [("arm-run-net", valid)])
                self.assertEqual([c["networks"] for c in box["armContainers"]], [sorted([box["network"], "arm-run-net"])])
                self.assertEqual(classify.classify(out)["class"], "result" if valid else "invalid")
                if valid:
                    self.assertEqual(box["armNetworks"][0]["gatewayMode"], "isolated")

    def test_a_network_is_judged_by_its_id_and_fails_closed(self):
        (self.tmp / "own.py").write_text(OWN_NETWORK)
        variant = [sys.executable, "-B", str(self.tmp / "own.py")]
        os.environ["ARM_NETWORK_FLAGS"] = "--driver bridge"
        os.environ["ARM_RECREATE"] = "1"
        record, _ = self.run_mode("recreated", variant)
        # The first may be gone before it is inspected (None); either way the second, of the same name, is its own network.
        networks = record["sandbox"]["armNetworks"]
        self.assertEqual(([n["name"] for n in networks], networks[0]["internal"] in (True, None), networks[1]["internal"]),
                         (["arm-run-net", "arm-run-net"], True, False))
        self.assertNotEqual(networks[0]["id"], networks[1]["id"])
        self.assertEqual([c["name"] for c in record["sandbox"]["escapes"]], ["arm-agent"])
        del os.environ["ARM_RECREATE"]
        os.environ.update(ARM_NETWORK_FLAGS="--internal", FAKE_UNINSPECTABLE="arm-run-net")
        record, out = self.run_mode("uninspectable", variant)
        self.assertEqual([n["internal"] for n in record["sandbox"]["armNetworks"]], [None])
        self.assertEqual(classify.classify(out)["class"], "invalid")

    def test_a_volume_the_arm_leaves_is_residue(self):
        (self.tmp / "own.py").write_text(OWN_NETWORK)
        os.environ.update(ARM_NETWORK_FLAGS="--internal", ARM_VOLUME="1")
        record, out = self.run_mode("volume", [sys.executable, "-B", str(self.tmp / "own.py")])
        self.assertIn("volume arm-left-volume", record["sandbox"]["residue"])
        self.assertEqual(json.loads((self.tmp / "engine.json").read_text())["volumes"], {})
        self.assertEqual(classify.classify(out)["class"], "void")

    def test_a_reader_that_stops_fails_the_network_check(self):
        lines = [{"Type": "network", "Action": "create", "Actor": {"ID": "n1", "Attributes": {"name": "run-internal"}}},
                 {"Type": "network", "Action": "create", "Actor": {"ID": "n2", "Attributes": {"name": "other"}}}]
        script = "import json\nfor x in %r: print(json.dumps(x), flush=True)\nimport time\ntime.sleep(5)\n" % (lines,)

        def broken(*_):
            raise RuntimeError("the engine call broke")
        events = native.Events([sys.executable, "-c", script], broken, "0", "run-internal", self.tmp)
        time.sleep(2)
        self.assertIn("the engine call broke", events.stop())

    def test_a_copied_gateway_label_does_not_hide_an_escape(self):
        (self.tmp / "copycat.py").write_text(textwrap.dedent('''
            import json, os, subprocess, sys
            engine = json.loads(os.environ["EVALUATION_ENGINE"])
            run_id = os.environ["EVALUATION_LABEL"].rpartition("=")[2]
            subprocess.run([*engine, "run", "--name", "copycat", "--label", "markitect.evaluation=" + run_id, os.environ["FAKE_IMAGE"],
                            sys.executable, "-c", "pass"], check=True)
            subprocess.run([*engine, "rm", "-f", "copycat"], check=True)
        '''))
        record, out = self.run_mode("copy", [sys.executable, "-B", str(self.tmp / "copycat.py")])
        self.assertEqual([c["name"] for c in record["sandbox"]["escapes"]], ["copycat"])
        self.assertEqual(classify.classify(out)["class"], "invalid")

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
