"""Offline tests of the evaluation sandbox: a fake engine, the egress gateway on loopback and the escape probe's checks."""
import json
import os
import pathlib
import socket
import ssl
import subprocess
import sys
import tempfile
import textwrap
import threading
import time
import unittest

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
sys.path.insert(0, str(HERE / "sandbox"))
import egress  # noqa: E402
import escape_probe  # noqa: E402
import sandbox  # noqa: E402
import suite  # noqa: E402

MODEL = "claude:claude-sonnet-5-5"
IMAGE = "sha256:" + "a" * 64
GATEWAY_IP = "10.9.0.2"

# A fake engine. It keeps its containers and networks in FAKE_DIR/engine.json, logs every call and runs a
# sandbox's command on the host with each mount target replaced by its source. FAKE_NOT_READY keeps the
# gateway from reporting ready; FAKE_KEEP makes removals fail silently, leaving residue; FAKE_RUN_FAIL fails a run
# before its container starts, as a bad flag or mount would; FAKE_GATEWAY_DEAD reports the gateway stopped. A run
# also checks that FAKE_LOCK, when set, is held while it runs.
FAKE_ENGINE = textwrap.dedent(r'''
    import json, os, subprocess, sys
    args, d = sys.argv[1:], os.environ["FAKE_DIR"]
    path = os.path.join(d, "engine.json")
    state = json.load(open(path)) if os.path.exists(path) else {"containers": {}, "networks": {}}
    def save():
        json.dump(state, open(path, "w"))
    with open(os.path.join(d, "calls.jsonl"), "a") as f:
        f.write(json.dumps(args) + "\n")
    def opt(name):
        return args[args.index(name) + 1]
    def labelled(kind, flt):
        key, _, value = flt.partition("=")
        return [n for n, labels in state[kind].items() for k, v in [labels.split("=", 1)] if k == key and (not value or v == value)]
    if args[:2] == ["image", "inspect"]:
        print(os.environ["FAKE_IMAGE"])
    elif args[0] == "ps":
        print("\n".join(labelled("containers", opt("--filter")[len("label="):])))
    elif args[:2] == ["network", "ls"]:
        print("\n".join(labelled("networks", opt("--filter")[len("label="):])))
    elif args[:2] == ["network", "create"]:
        state["networks"][args[-1]] = opt("--label"); save()
    elif args[:2] == ["network", "rm"]:
        if not os.environ.get("FAKE_KEEP"):
            state["networks"].pop(args[2], None); save()
    elif args[0] in ("network", "start", "kill"):
        pass
    elif args[0] == "create":
        state["containers"][opt("--name")] = opt("--label"); save()
    elif args[0] == "inspect" and "NetworkSettings" in opt("--format"):
        print(json.dumps({n: {"IPAddress": "10.9.0.2"} for n in state["networks"]}))
    elif args[0] == "inspect" and ".State.Running" in opt("--format"):
        print("false" if os.environ.get("FAKE_GATEWAY_DEAD") else "true")
    elif args[0] == "inspect":
        if args[-1] not in state["containers"]:
            sys.exit("no such container")
        started = args[-1] in state.get("exit", {})
        print(json.dumps({"ExitCode": state.get("exit", {}).get(args[-1]) or 0, "OOMKilled": False, "Running": False,
                          "StartedAt": "2026-10-28T00:00:00Z" if started else "0001-01-01T00:00:00Z"}))
    elif args[0] == "logs":
        if not os.environ.get("FAKE_NOT_READY"):
            for e in ({"ready": True, "port": 443}, {"sni": "api.anthropic.com", "allowed": True},
                      {"sni": "github.com", "allowed": False}, {"sni": None, "allowed": False}):
                print(json.dumps(e))
    elif args[:2] == ["rm", "-f"]:
        if not os.environ.get("FAKE_KEEP"):
            state["containers"].pop(args[2], None); save()
    elif args[0] == "run":
        name = opt("--name")
        state["containers"][name] = opt("--label"); save()
        if os.environ.get("FAKE_RUN_FAIL"):
            sys.exit(125)
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
            return a
        env = {k: host(v) for k, v in env.items()}
        if env.get("FAKE_LOCK") and not os.path.exists(env["FAKE_LOCK"]):
            sys.exit("the engine lock is not held")
        state = json.load(open(path)); state.setdefault("exit", {})[name] = None; save()
        code = subprocess.run([host(a) for a in args[i + 1:]], env=env).returncode
        state = json.load(open(path)); state.setdefault("exit", {})[name] = code; save()
        sys.exit(code)
    else:
        sys.exit("fake engine: unknown call " + " ".join(args))
''')

# A variant for the sandbox: reads the login the suite mounted, then behaves as the first line of task.md says:
# "ok", "leak" (commits the access token), "sleep", or "state" (records itself in STATE_DIR).
FAKE_VARIANT = textwrap.dedent('''
    import json, os, pathlib, subprocess, sys, time
    task, out = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
    state = pathlib.Path(sys.argv[3]) if len(sys.argv) > 3 else None
    login = json.loads(pathlib.Path(os.environ["EVALUATION_LOGIN"]).read_text())["claudeAiOauth"]
    assert "refreshToken" not in login, "refresh token reached the sandbox"
    mode = (task / "task.md").read_text().splitlines()[0]
    spec = json.loads((task / "task.json").read_text())
    if mode == "sleep":
        time.sleep(60)
    if state:
        with open(state / "seen.txt", "a") as f:
            f.write(mode + "\\n")
    repo = out / "repo"
    env = dict(os.environ, GIT_AUTHOR_NAME="v", GIT_AUTHOR_EMAIL="v@x", GIT_COMMITTER_NAME="v", GIT_COMMITTER_EMAIL="v@x")
    g = lambda *a: subprocess.run(["git", "-C", str(repo), *a], check=True, capture_output=True, env=env)
    subprocess.run(["git", "clone", "-q", str(task / "start.bundle"), str(repo)], check=True)
    g("checkout", "-q", "-B", "work", spec["start"]["commit"])
    with open(repo / "work.txt", "a") as f:
        f.write((login["accessToken"] if mode == "leak" else mode) + "\\n")
    g("add", "-A")
    g("-c", "commit.gpgsign=false", "commit", "-q", "-m", "work")
    g("branch", "-f", "result", "HEAD")
    g("bundle", "create", str((out / "result.bundle").resolve()), "refs/heads/result")
    import shutil; shutil.rmtree(repo, ignore_errors=True)
    (out / "run.json").write_text(json.dumps({"provider": "claude", "model": os.environ["EVALUATION_MODEL"].split(":", 1)[1]}))
''')


def calls(tmp):
    return [json.loads(line) for line in (tmp / "calls.jsonl").read_text().splitlines()]


class SandboxRunTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)
        self.engine = self.tmp / "engine.py"
        self.engine.write_text(FAKE_ENGINE)
        self.code = self.tmp / "variant-code"
        self.code.mkdir()
        (self.code / "fake.py").write_text(FAKE_VARIANT)
        self.variant = [sys.executable, "-B", "/variant/fake.py"]
        self.login = self.tmp / "credentials.json"
        self.login.write_text(json.dumps({"claudeAiOauth": {"accessToken": "sk-access-SECRET", "refreshToken": "sk-refresh-SECRET",
                                                            "expiresAt": int((time.time() + 7200) * 1000)}}))
        self.series = sandbox.load_series(sandbox.DEFAULT_SERIES)
        self._env = dict(os.environ)
        os.environ.update(FAKE_DIR=str(self.tmp), FAKE_IMAGE=IMAGE)

    def tearDown(self):
        os.environ.clear()
        os.environ.update(self._env)
        self._tmp.cleanup()

    def box(self):
        return sandbox.Sandbox([sys.executable, "-I", str(self.engine)], IMAGE, self.series, self.code, self.login, "claude")

    def task(self, mode, deadline=120):
        (self.tmp / f"{mode}.md").write_text(mode + "\n")
        out = self.tmp / f"task-{mode}"
        suite.make_task(self.tmp / f"{mode}.md", None, None, deadline, out)
        return out

    def test_sandboxed_run_delivers_and_cleans_up(self):
        record = suite.run(self.task("ok"), self.variant, self.tmp / "run", MODEL, sandbox=self.box())
        box = record["sandbox"]
        self.assertTrue(record["delivered"], (record["resultError"], box.get("error")))
        self.assertEqual((record["conditions"], record["validForCounting"], record["modelMatches"]), ("sandbox", True, True))
        self.assertEqual(box["residue"], [])
        self.assertEqual(box["quarantine"], [])
        self.assertEqual(box["container"], {"exitCode": 0, "oomKilled": False})
        self.assertEqual(box["egress"], {"allowed": {"api.anthropic.com": 1}, "refused": {"github.com": 1, "(none)": 1}})
        self.assertTrue((self.tmp / "run" / "egress.jsonl").is_file())
        self.assertEqual(box["limits"], self.series["limits"])
        self.assertEqual(json.loads((self.tmp / "run" / "suite.json").read_text())["sandbox"]["limits"], self.series["limits"])

    def test_sandbox_argv_mounts_limits_and_network(self):
        record = suite.run(self.task("ok"), self.variant, self.tmp / "run", MODEL, sandbox=self.box())
        argv, run_id = record["sandbox"]["argv"], record["sandbox"]["runId"]
        mounts = sorted((argv[i + 1] for i, a in enumerate(argv) if a == "--mount"), key=lambda m: m.split(",dst=")[1])
        self.assertEqual([m.split(",dst=")[1] for m in mounts], ["/login/credentials.json,readonly", "/out", "/task,readonly",
                                                                  "/variant,readonly"])
        self.assertTrue(all(m.startswith("type=bind,") for m in mounts))
        self.assertNotIn("-v", argv)
        self.assertFalse(any("docker.sock" in a for c in calls(self.tmp) for a in c))
        for flag in ("--read-only", "--init", "no-new-privileges", f"--add-host=api.anthropic.com:{GATEWAY_IP}"):
            self.assertIn(flag, argv)
        self.assertEqual(argv[argv.index("--cap-drop") + 1], "ALL")
        self.assertEqual(argv[argv.index("--network") + 1], f"{run_id}-internal")
        limits = self.series["limits"]
        self.assertEqual([argv[argv.index(f) + 1] for f in ("--cpus", "--memory", "--memory-swap", "--pids-limit")],
                         [limits["cpus"], str(limits["memoryBytes"]), str(limits["memoryBytes"]), str(limits["pids"])])
        self.assertIn(f"/tmp:rw,exec,nosuid,nodev,mode=1777,size={limits['tmpBytes']}", argv)
        self.assertEqual(argv[argv.index(IMAGE) + 1:], [*self.variant, "/task", "/out"])
        self.assertIn(f"EVALUATION_MODEL={MODEL}", argv)
        if os.name != "nt":
            self.assertEqual(argv[argv.index("--user") + 1], f"{os.getuid()}:{os.getgid()}")
        # The internal network has no route out; the gateway alone joins both networks.
        log = calls(self.tmp)
        self.assertIn(["network", "create", "--internal", "--opt", "com.docker.network.bridge.gateway_mode_ipv4=isolated",
                       "--label", f"markitect.evaluation={run_id}", f"{run_id}-internal"], log)
        self.assertEqual(argv[argv.index("--dns") + 1], sandbox.NO_DNS)
        self.assertIn(["network", "create", "--label", f"markitect.evaluation={run_id}", f"{run_id}-egress"], log)
        self.assertIn(["network", "connect", f"{run_id}-internal", f"{run_id}-gateway"], log)
        create = next(c for c in log if c[0] == "create")
        self.assertEqual(create[create.index("--network") + 1], f"{run_id}-egress")
        self.assertEqual(create[create.index(IMAGE) + 1:], ["-I", "-B", "/opt/evaluation/egress.py", "--allow", "api.anthropic.com"])
        # Cleanup: both containers, then both networks, then a census by the run's label.
        removals = [c for c in log if c[:2] in (["rm", "-f"], ["network", "rm"])]
        self.assertEqual(removals, [["rm", "-f", f"{run_id}-sandbox"], ["rm", "-f", f"{run_id}-gateway"],
                                    ["network", "rm", f"{run_id}-egress"], ["network", "rm", f"{run_id}-internal"]])
        self.assertEqual(log[-1][:3], ["network", "ls", "--filter"])
        # The login copy is gone after the run.
        login = next(m for m in argv if "dst=/login" in m).split(",")[1][len("src="):]
        self.assertFalse(pathlib.Path(login).exists())

    def test_deadline_kills_and_removes_the_sandbox(self):
        record = suite.run(self.task("sleep", deadline=3), self.variant, self.tmp / "run", MODEL, sandbox=self.box())
        self.assertTrue(record["timedOut"])
        self.assertFalse(record["delivered"])
        steps = [s["step"] for s in record["sandbox"]["steps"]]
        self.assertLess(steps.index("sandbox-kill"), steps.index("remove-sandbox"))
        self.assertEqual(record["sandbox"]["residue"], [])

    def test_a_leaked_login_is_quarantined(self):
        record = suite.run(self.task("leak"), self.variant, self.tmp / "run", MODEL, sandbox=self.box())
        self.assertIn("variant/result.bundle", record["sandbox"]["quarantine"])
        self.assertFalse(record["delivered"])
        for p in (self.tmp / "run").rglob("*"):
            if p.is_file():
                self.assertNotIn(b"SECRET", p.read_bytes(), p)

    def test_a_gateway_that_never_starts_voids_the_run_and_cleans_up(self):
        os.environ["FAKE_NOT_READY"] = "1"
        ready, sandbox.READY_SECONDS = sandbox.READY_SECONDS, 1
        try:
            record = suite.run(self.task("ok"), self.variant, self.tmp / "run", MODEL, sandbox=self.box())
        finally:
            sandbox.READY_SECONDS = ready
        self.assertIn("did not become ready", record["sandbox"]["error"])
        self.assertFalse(record["validForCounting"])
        self.assertFalse(record["delivered"])
        self.assertFalse(any(c[0] == "run" for c in calls(self.tmp)))
        self.assertEqual(record["sandbox"]["residue"], [])

    def test_a_container_that_never_started_is_not_valid(self):
        os.environ["FAKE_RUN_FAIL"] = "1"
        record = suite.run(self.task("ok"), self.variant, self.tmp / "run", MODEL, sandbox=self.box())
        self.assertEqual(record["exitCode"], 125)
        self.assertIn("never started", record["sandbox"]["error"])
        self.assertFalse(record["validForCounting"])
        self.assertEqual(record["sandbox"]["residue"], [])

    def test_a_gateway_that_dies_during_the_run_voids_it(self):
        os.environ["FAKE_GATEWAY_DEAD"] = "1"
        record = suite.run(self.task("ok"), self.variant, self.tmp / "run", MODEL, sandbox=self.box())
        self.assertIn("gateway stopped", record["sandbox"]["error"])
        self.assertFalse(record["validForCounting"])

    def test_links_left_in_the_output_are_removed_unread(self):
        out = self.tmp / "links"
        out.mkdir()
        (out / "target.json").write_text("{}")
        try:
            (out / "run.json").symlink_to(out / "target.json")
        except OSError:
            self.skipTest("no symlinks here")
        self.assertEqual(sandbox.files_containing(out, [b"SECRET"]), [out / "run.json"])
        self.assertIsNone(suite.reported(out / "run.json"))

    def test_residue_makes_the_run_invalid(self):
        os.environ["FAKE_KEEP"] = "1"
        record = suite.run(self.task("ok"), self.variant, self.tmp / "run", MODEL, sandbox=self.box())
        self.assertTrue(record["delivered"])
        self.assertEqual(len(record["sandbox"]["residue"]), 4)
        self.assertFalse(record["validForCounting"])

    def test_leftovers_of_another_run_refuse(self):
        (self.tmp / "engine.json").write_text(json.dumps({"containers": {"eval-old-sandbox": "markitect.evaluation=eval-old"}, "networks": {}}))
        record = suite.run(self.task("ok"), self.variant, self.tmp / "run", MODEL, sandbox=self.box())
        self.assertIn("eval-old-sandbox", record["sandbox"]["error"])
        self.assertFalse(any(c[0] == "run" for c in calls(self.tmp)))

    def test_chain_stops_when_the_sandbox_fails(self):
        os.environ["FAKE_RUN_FAIL"] = "1"
        steps = []
        for i in (1, 2):
            d = self.tmp / "steps" / str(i)
            d.mkdir(parents=True)
            (d / "task.md").write_text("ok\n")
            steps.append(d)
        summary = suite.chain(steps, None, 120, self.variant, self.tmp / "chain", MODEL, self.box())
        self.assertEqual(len(summary["steps"]), 1)
        self.assertEqual(summary["stopped"]["step"], 1)
        self.assertIn("never started", summary["stopped"]["error"])

    def test_a_login_written_to_state_is_quarantined(self):
        state = self.tmp / "state"
        state.mkdir()
        (state / "copy.txt").write_text("sk-access-SECRET")
        record = suite.run(self.task("ok"), self.variant, self.tmp / "run", MODEL, state, sandbox=self.box())
        self.assertEqual(record["sandbox"]["quarantine"], ["state/copy.txt"])
        self.assertFalse((state / "copy.txt").exists())

    def test_chain_in_the_sandbox_keeps_state(self):
        steps = []
        for i, mode in enumerate(("state", "state"), 1):
            d = self.tmp / "steps" / str(i)
            d.mkdir(parents=True)
            (d / "task.md").write_text(mode + "\n")
            steps.append(d)
        summary = suite.chain(steps, None, 120, self.variant, self.tmp / "chain", MODEL, self.box())
        self.assertEqual(summary["conditions"], "sandbox")
        self.assertEqual([(s["delivered"], s["validForCounting"]) for s in summary["steps"]], [(True, True)] * 2)
        self.assertEqual((self.tmp / "chain" / "state" / "seen.txt").read_text().split(), ["state", "state"])
        record = json.loads((self.tmp / "chain" / "step-02" / "run" / "suite.json").read_text())
        self.assertIn("/state", record["sandbox"]["argv"][-3:])

    def test_host_mode_is_not_valid_for_counting(self):
        os.environ["EVALUATION_LOGIN"] = str(self.login)  # the host variant reads it directly
        record = suite.run(self.task("ok"), [sys.executable, "-B", str(self.code / "fake.py")], self.tmp / "run", MODEL)
        self.assertEqual((record["conditions"], record["validForCounting"]), ("host", False))
        self.assertNotIn("sandbox", record)

    def test_cli_holds_the_lock_and_refuses_sandbox_options_without_sandbox(self):
        lock = self.tmp / "NATIVE-RUN.lock"
        engine = self.tmp / ("engine.cmd" if os.name == "nt" else "engine.sh")
        if os.name == "nt":
            engine.write_text(f'@"{sys.executable}" -I "{self.engine}" %*\n')
        else:
            engine.write_text(f'#!/bin/sh\nexec "{sys.executable}" -I "{self.engine}" "$@"\n')
            engine.chmod(0o755)
        task = self.task("ok")
        common = ["run", "--task", str(task), "--model", MODEL]
        lock.write_text("{}")
        busy = suite.main([*common, "--out", str(self.tmp / "busy"), "--sandbox", IMAGE, "--variant-code", str(self.code),
                           "--login", str(self.login), "--engine", str(engine), "--lock", str(lock), "--", *self.variant])
        self.assertEqual(busy, 2)
        self.assertFalse((self.tmp / "busy").exists())
        lock.unlink()
        os.environ["FAKE_LOCK"] = str(lock)
        self.assertEqual(suite.main([*common, "--out", str(self.tmp / "bad"), "--lock", str(lock), "--", *self.variant]), 2)
        self.assertFalse((self.tmp / "bad").exists())
        code = suite.main([*common, "--out", str(self.tmp / "cli"), "--sandbox", IMAGE, "--variant-code", str(self.code),
                           "--login", str(self.login), "--engine", str(engine), "--lock", str(lock), "--", *self.variant])
        self.assertEqual(code, 0)
        self.assertTrue(json.loads((self.tmp / "cli" / "suite.json").read_text())["validForCounting"])
        self.assertFalse(lock.exists())

    def test_configuration_is_checked_before_anything_runs(self):
        with self.assertRaisesRegex(sandbox.SandboxError, "image ID"):
            sandbox.Sandbox(["docker"], "markitect-eval-sandbox:latest", self.series, self.code, self.login, "claude")
        with self.assertRaisesRegex(sandbox.SandboxError, "no egress for codex"):
            sandbox.Sandbox(["docker"], IMAGE, self.series, self.code, self.login, "codex")
        bad = self.tmp / "series.json"
        for limits, message in (({"cpus": "8", "memoryBytes": 1024, "pids": 10, "tmpBytes": 2048}, "cannot exceed"),
                                ({"cpus": "0", "memoryBytes": 1024, "pids": 10, "tmpBytes": 512}, "positive")):
            bad.write_text(json.dumps({"limits": limits, "egress": {}}))
            with self.assertRaisesRegex(sandbox.SandboxError, message):
                sandbox.load_series(bad)
        bad.write_text(json.dumps({"limits": self.series["limits"], "egress": {"claude": ["*.anthropic.com"]}}))
        with self.assertRaisesRegex(sandbox.SandboxError, "host names"):
            sandbox.load_series(bad)
        copy, secrets = sandbox.access_login("claude", self.login.read_bytes(), 600)
        self.assertNotIn(b"refresh", copy)
        self.assertEqual(len(secrets), 2)
        with self.assertRaisesRegex(sandbox.SandboxError, "expires"):
            sandbox.access_login("claude", self.login.read_bytes(), 10 * 3600)
        with self.assertRaisesRegex(sandbox.SandboxError, "comma"):
            sandbox.host_path(self.tmp / "a,b")
        if os.name == "nt":
            self.assertEqual(sandbox.host_path(r"C:\Users\x"), "C:/Users/x")
            with self.assertRaisesRegex(sandbox.SandboxError, "drive"):
                sandbox.host_path(r"\\server\share\x")


def client_hello(name):
    """The first TLS record a client sends for the server name."""
    context = ssl.create_default_context()
    incoming, outgoing = ssl.MemoryBIO(), ssl.MemoryBIO()
    conn = context.wrap_bio(incoming, outgoing, server_hostname=name)
    try:
        conn.do_handshake()
    except ssl.SSLWantReadError:
        pass
    return outgoing.read()


class EgressGatewayTest(unittest.TestCase):
    def test_server_name_is_read_from_a_client_hello(self):
        self.assertEqual(egress.server_name(client_hello("API.Anthropic.com")), "api.anthropic.com")
        self.assertIsNone(egress.server_name(b"GET / HTTP/1.1\r\n\r\n"))
        self.assertIsNone(egress.server_name(client_hello("github.com")[:40]))

    def test_gateway_forwards_allowed_names_only(self):
        # An echo server stands in for the provider; the gateway forwards to it as "localhost".
        upstream = socket.socket()
        upstream.bind(("127.0.0.1", 0))
        upstream.listen(4)

        def echo():
            while True:
                try:
                    conn, _ = upstream.accept()
                except OSError:
                    return
                with conn:
                    while data := conn.recv(65536):
                        conn.sendall(data)
        threading.Thread(target=echo, daemon=True).start()
        servers = []
        lines = []
        old_log, egress.log = egress.log, lines.append
        try:
            threading.Thread(target=egress.serve, args=({"localhost"}, 0, upstream.getsockname()[1], servers.append), daemon=True).start()
            deadline = time.monotonic() + 5
            while not servers and time.monotonic() < deadline:
                time.sleep(0.05)
            port = servers[0].getsockname()[1]

            def exchange(name):
                hello = client_hello(name)
                with socket.create_connection(("127.0.0.1", port), timeout=5) as s:
                    s.sendall(hello)
                    s.shutdown(socket.SHUT_WR)
                    data = b""
                    while chunk := s.recv(65536):
                        data += chunk
                return hello, data
            hello, back = exchange("localhost")
            self.assertEqual(back, hello)
            self.assertEqual(exchange("github.com")[1], b"")
            with socket.create_connection(("127.0.0.1", port), timeout=5) as s:
                s.sendall(b"GET / HTTP/1.1\r\nHost: github.com\r\n\r\n")
                s.shutdown(socket.SHUT_WR)
                self.assertEqual(s.recv(100), b"")
            deadline = time.monotonic() + 5
            while len([e for e in lines if "sni" in e]) < 3 and time.monotonic() < deadline:
                time.sleep(0.05)
        finally:
            egress.log = old_log
            for s in servers:
                s.close()
            upstream.close()
        entries = {e["sni"]: e for e in lines if "sni" in e}
        self.assertTrue(entries["localhost"]["allowed"])
        self.assertEqual(entries["localhost"]["bytesDown"], len(hello))
        self.assertFalse(entries["github.com"]["allowed"])
        self.assertFalse(entries[None]["allowed"])


class EscapeProbeTest(unittest.TestCase):
    MOUNTINFO = textwrap.dedent("""\
        1 0 0:1 / / ro - overlay overlay ro
        2 1 0:2 / /proc rw - proc proc rw
        3 1 0:3 / /task ro - fakeowner grpcfuse ro
        4 1 0:3 / /login/credentials.json ro - fakeowner grpcfuse ro
        5 1 0:4 / /tmp rw - tmpfs tmpfs rw
        6 1 0:5 / /etc/hosts rw - ext4 /dev/sda1 rw
        7 1 0:6 / /sbin/docker-init ro - ext4 /dev/sda1 ro
        10 1 0:6 / /usr/sbin/docker-init ro - ext4 /dev/sda1 ro
        8 1 0:7 / /mnt/host\\040c rw - 9p drvfs rw
        9 1 0:8 / /var/run/docker.sock rw - tmpfs tmpfs rw
        """)

    def test_foreign_mounts_are_found(self):
        self.assertEqual(escape_probe.foreign_mounts(self.MOUNTINFO), ["/mnt/host c", "/var/run/docker.sock"])

    def test_repository_markers_are_found_outside_the_suite_mounts(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            for p in ("host/repo/.git/HEAD", "host/repo/AGENTS.md", "task/.git/HEAD", "work/plain.txt"):
                (root / p).parent.mkdir(parents=True, exist_ok=True)
                (root / p).write_text("x")
            found = escape_probe.markers(str(root), (str(root / "task"),))
            self.assertEqual([pathlib.Path(f).relative_to(root).as_posix() for f in found], ["host/repo/.git", "host/repo/AGENTS.md"])

    def test_route_hosts_name_the_gateway_and_each_subnet_host_side(self):
        head = "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n"
        bridge = (head + "eth0\t00000000\t010011AC\t0003\t0\t0\t0\t00000000\t0\t0\t0\n"
                  "eth0\t000011AC\t00000000\t0001\t0\t0\t0\t0000FFFF\t0\t0\t0\n")
        self.assertEqual(escape_probe.route_hosts(bridge), ["172.17.0.1", "172.17.0.1"])
        internal = head + "eth0\t000016AC\t00000000\t0001\t0\t0\t0\t0000FFFF\t0\t0\t0\n"
        self.assertEqual(escape_probe.route_hosts(internal), ["172.22.0.1"])

    def test_tcp_reaches_a_listener_and_not_a_closed_port(self):
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            listener.listen(1)
            self.assertTrue(escape_probe.tcp("127.0.0.1", listener.getsockname()[1])["reached"])
            closed = listener.getsockname()[1]
        self.assertFalse(escape_probe.tcp("127.0.0.1", closed, 2)["reached"])

    def test_probe_report_is_contained_only_when_every_escape_is_blocked(self):
        blocked = [{"target": "x", "reached": False}]
        results = {"hostFiles": blocked, "github": blocked + [{"target": "dns", "reached": False, "informational": True}],
                   "engine": blocked}
        with tempfile.TemporaryDirectory() as tmp:
            old = (escape_probe.host_files, escape_probe.github, escape_probe.engine, escape_probe.tls, escape_probe.resolve)
            escape_probe.host_files = lambda: results["hostFiles"]
            escape_probe.github = lambda gw: results["github"]
            escape_probe.engine = lambda gw: results["engine"]
            escape_probe.tls = lambda *a, **k: {"target": "endpoint", "reached": True}
            escape_probe.resolve = lambda name: [GATEWAY_IP]
            try:
                self.assertEqual(escape_probe.main(["/task", tmp]), 0)
                record = json.loads((pathlib.Path(tmp) / "run.json").read_text())
                self.assertEqual((record["status"], record["egressWorks"]), ("contained", True))
                self.assertEqual(record["probe"]["allowedEndpoint"]["gateway"], GATEWAY_IP)
                results["engine"] = [{"target": "/var/run/docker.sock", "reached": True}]
                self.assertEqual(escape_probe.main(["/task", tmp]), 1)
                record = json.loads((pathlib.Path(tmp) / "run.json").read_text())
                self.assertEqual(record["status"], "escaped")
                self.assertFalse(record["probe"]["engine"]["blocked"])
            finally:
                (escape_probe.host_files, escape_probe.github, escape_probe.engine, escape_probe.tls, escape_probe.resolve) = old


class SandboxImageTest(unittest.TestCase):
    def test_image_pins_the_runtime_versions(self):
        recipe = (HERE / "sandbox" / "Dockerfile").read_text()
        runtime = (HERE.parents[1] / "packaging" / "runtime" / "Dockerfile").read_text()
        for line in runtime.splitlines():
            if line.startswith(("FROM golang:", "FROM node:", "ARG CODEX_VERSION=", "ARG CLAUDE_CODE_VERSION=")):
                self.assertIn(line.split(" AS ")[0], recipe)
        self.assertIn("USER 1000:1000", recipe)
        self.assertIn("GOPROXY=off", recipe)


if __name__ == "__main__":
    unittest.main()
