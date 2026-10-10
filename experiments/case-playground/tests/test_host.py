import contextlib
import csv
import io
import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from playground import host

def station_plan(case: str, sizes: tuple[int, ...]) -> str:
    ids = iter(f"X{i:02d}" for i in range(1, 100))
    return json.dumps({"schema": 1, "case": case, "stations": [
        {"id": f"S{n}", "items": [next(ids) for _ in range(size)]} for n, size in enumerate(sizes, 1)]})


MANIFEST = {
    "schema": 1, "id": "fake-roombook-001", "case": "roombook", "method": "conventional",
    "agent": {"kind": "fake", "codexVersion": "0.162.0", "model": "gpt-6-luna", "effort": "high",
              "maxSubagents": 3},
    "limits": {"stationSeconds": 60, "totalSeconds": 300},
    "container": {"cpus": 2, "memory": "4g", "pidsLimit": 512},
}


def quiet():
    stack = contextlib.ExitStack()
    stack.enter_context(contextlib.redirect_stdout(io.StringIO()))
    stack.enter_context(contextlib.redirect_stderr(io.StringIO()))
    return stack


class FakeProc:
    def wait(self, timeout=None):
        return 0

    def poll(self):
        return 0

    def kill(self):
        pass


class FakeDocker:
    """Stands in for subprocess.run/Popen inside host; records every argv."""

    def __init__(self, wait_effect=None, run_fails=False, existing=False, wait_fails=False, inspect_error=None):
        self.calls = []
        self.inspect_error = inspect_error
        self.wait_effect = wait_effect
        self.run_fails = run_fails
        self.wait_fails = wait_fails
        self.container = "stopped" if existing else None  # None, "running" or "stopped"

    def run(self, cmd, **kwargs):
        self.calls.append(list(cmd))
        code, out, err = 0, "", ""
        if cmd[:2] == ["docker", "version"]:
            out = "29.4.1"
        elif cmd[:3] == ["docker", "image", "inspect"]:
            out = "sha256:feed"
        elif cmd[:3] == ["docker", "container", "inspect"]:
            if self.inspect_error:
                code, err = 1, self.inspect_error
            elif self.container is None:
                code, err = 1, f"Error: No such container: {cmd[-1]}"
            else:
                out = str(self.container == "running").lower()
        elif cmd[:2] == ["docker", "run"]:
            if self.run_fails:
                code, err = 125, "conflict"
            else:
                out, self.container = "cid", "running"
        elif cmd[:2] == ["docker", "wait"]:
            if self.wait_effect:
                raise self.wait_effect
            if self.wait_fails:
                code, err = 1, "error during connect: pipe closed"
            else:
                out, self.container = "1\n", "stopped"
        elif cmd[:2] == ["docker", "kill"]:
            self.container = "stopped" if self.container else None
        elif cmd[:2] == ["docker", "rm"]:
            self.container = None
        elif cmd[:2] == ["docker", "ps"]:
            out = "c1\nc2\n"
        return subprocess.CompletedProcess(cmd, code, stdout=out, stderr=err)

    def popen(self, cmd, **kwargs):
        self.calls.append(list(cmd))
        return FakeProc()

    def commands(self):
        return [" ".join(call[:2]) for call in self.calls]


class HostTestBase(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.base = Path(self.temp.name) / "Glacius Labs"
        root = self.base / "root"
        for name in ("playground/__pycache__", "cases/common", "cases/roombook", "cases/readinglog",
                     "methods/conventional", "methods/markitect", "tests", "container"):
            (root / name).mkdir(parents=True)
        (root / "cases" / "task-prompt.txt").write_text("prompt\n", encoding="utf-8")
        (root / "methods" / "conventional" / "AGENTS.fragment.md").write_text("fragment\n", encoding="utf-8")
        (root / "methods" / "markitect" / "README.md").write_text("notes for people\n", encoding="utf-8")
        (root / "playground" / "runner.py").write_text("", encoding="utf-8")
        (root / "playground" / "__pycache__" / "x.pyc").write_bytes(b"")
        (root / "tests" / "fake_agent.py").write_text("", encoding="utf-8")
        (root / "tests" / "fake_claude.py").write_text("", encoding="utf-8")
        (root / "cases" / "roombook" / "STATIONS.json").write_text(station_plan("roombook", (1, 3, 7, 1)),
                                                                   encoding="utf-8")
        self.manifest_path = self.base / "manifest.json"
        self.manifest_path.write_text(json.dumps(MANIFEST), encoding="utf-8")
        self.out = self.base / "runs" / "one, two"
        patcher = mock.patch.object(host, "ROOT", root)
        patcher.start()
        self.addCleanup(patcher.stop)
        self.addCleanup(self.temp.cleanup)

    def write_manifest(self, name: str, **changes) -> Path:
        data = json.loads(json.dumps(MANIFEST))
        for key, value in changes.items():
            target = data["agent"] if key in ("kind", "claudeVersion", "model") else data
            target[key] = value
        path = self.base / name
        path.write_text(json.dumps(data), encoding="utf-8")
        return path

    def run_host(self, docker, *extra, manifest=None):
        argv = ["run", "--manifest", str(manifest or self.manifest_path), "--out", str(self.out), *extra]
        with mock.patch.object(host.subprocess, "run", docker.run), \
                mock.patch.object(host.subprocess, "Popen", docker.popen), quiet():
            return host.main(argv)

    def host_record(self):
        return json.loads((self.out / "host.json").read_text(encoding="utf-8"))


class MountTests(unittest.TestCase):
    def test_mount_quotes_commas_and_keeps_spaces(self):
        source = Path("D:/Data/Some One/Glacius Labs/out, 1/inputs")
        value = host._mount(source, "/in", readonly=True)
        self.assertEqual(next(csv.reader([value])),
                         ["type=bind", f"source={source}", "target=/in", "readonly"])
        self.assertEqual(host._mount(Path("/a b"), "/out"), f"type=bind,source={Path('/a b')},target=/out")

    def test_docker_run_argv(self):
        argv = host.docker_run_argv(
            {**MANIFEST, "agent": {**MANIFEST["agent"], "kind": "codex"}}, name="mpg-x", image="img:1",
            image_id="sha256:1", inputs=Path("/h/Glacius Labs/in"), results=Path("/h/Glacius Labs/res"),
            auth=Path("/h/auth.json"), markitect={"commit": "c" * 40, "sha256": "abc"})
        self.assertEqual(argv[:8], ["docker", "run", "--detach", "--name", "mpg-x", "--label",
                                    "markitect-playground=1", "--init"])
        self.assertIn("seccomp=unconfined", argv)
        self.assertNotIn("--privileged", argv)
        self.assertNotIn("-v", argv)
        mounts = [argv[i + 1] for i, arg in enumerate(argv) if arg == "--mount"]
        self.assertEqual(mounts, [
            f"type=bind,source={Path('/h/Glacius Labs/in')},target=/in,readonly",
            f"type=bind,source={Path('/h/Glacius Labs/res')},target=/out",
            f"type=bind,source={Path('/h/auth.json')},target=/run/secrets/codex-auth.json,readonly"])
        for flag, value in (("--cpus", "2"), ("--memory", "4g"), ("--pids-limit", "512"),
                            ("--workdir", "/in")):
            self.assertEqual(argv[argv.index(flag) + 1], value)
        envs = [argv[i + 1] for i, arg in enumerate(argv) if arg == "--env"]
        self.assertEqual(envs, ["MPG_IMAGE_ID=sha256:1", "MPG_MARKITECT_COMMIT=" + "c" * 40,
                                "MPG_MARKITECT_SHA256=abc"])
        self.assertEqual(argv[-9:], ["img:1", "python3", "-m", "playground", "run", "--manifest",
                                     "/in/manifest.json", "--out", "/out"])

    def test_no_auth_mount_for_fake(self):
        argv = host.docker_run_argv(MANIFEST, name="n", image="i", image_id="d", inputs=Path("/i"),
                                    results=Path("/r"), auth=None, markitect=None)
        self.assertEqual(sum(arg == "--mount" for arg in argv), 2)
        self.assertFalse(any(arg.startswith("MPG_MARKITECT") for arg in argv))


class RunTests(HostTestBase):
    def test_completed_run_stages_inputs_and_removes_container(self):
        docker = FakeDocker()
        self.assertEqual(self.run_host(docker), 1)  # container exit code is passed through
        record = self.host_record()
        self.assertEqual(record["status"], "completed")
        self.assertEqual(record["containerExitCode"], 1)
        self.assertEqual(record["image"], {"tag": "markitect-playground:codex-0.162.0-claude-2.1.296",
                                           "id": "sha256:feed"})
        self.assertEqual(record["stations"], 4)
        self.assertEqual(record["secrets"], {"codexAuth": False, "claudeToken": False})
        self.assertEqual(record["dockerVersion"], "29.4.1")
        inputs = self.out / "inputs"
        self.assertTrue((inputs / "manifest.json").is_file())
        self.assertTrue((inputs / "tests" / "fake_agent.py").is_file())
        self.assertTrue((inputs / "playground" / "runner.py").is_file())
        self.assertFalse((inputs / "playground" / "__pycache__").exists())
        self.assertTrue((inputs / "cases" / "common").is_dir())
        self.assertTrue((inputs / "cases" / "roombook").is_dir())
        self.assertTrue((inputs / "cases" / "task-prompt.txt").is_file())
        self.assertTrue((inputs / "methods" / "conventional" / "AGENTS.fragment.md").is_file())
        # The agent can read /in: the other case and the other arm's method are not staged.
        self.assertFalse((inputs / "cases" / "readinglog").exists())
        self.assertFalse((inputs / "methods" / "markitect").exists())
        self.assertGreater(record["hostTimeoutSeconds"], MANIFEST["limits"]["totalSeconds"] + 3600)
        self.assertTrue((self.out / "results").is_dir())
        build = next(c for c in docker.calls if c[:2] == ["docker", "build"])
        self.assertEqual(build[-3:], ["--build-arg", "CODEX_VERSION=0.162.0", str(host.ROOT / "container")])
        self.assertIn("CLAUDE_VERSION=2.1.296", build)
        self.assertIn("--provenance=false", build)  # keeps the image ID stable across rebuilds
        run = next(c for c in docker.calls if c[:2] == ["docker", "run"])
        self.assertIn(host._mount(inputs.resolve(), "/in", readonly=True), run)
        commands = docker.commands()
        self.assertLess(commands.index("docker container"), commands.index("docker build"))  # name check first
        self.assertLess(commands.index("docker wait"), commands.index("docker rm"))
        self.assertNotIn("docker kill", commands)
        self.assertIsNone(docker.container)

    def test_results_are_handed_back_after_the_container_ends(self):
        docker = FakeDocker()
        with mock.patch.object(host, "hand_back", return_value="done") as hand_back:
            self.run_host(docker)
        hand_back.assert_called_once_with("mpg-fake-roombook-001", "sha256:feed", self.out.resolve() / "results")
        self.assertEqual(self.host_record()["handBack"], "done")

    def test_timeout_kills_and_records(self):
        docker = FakeDocker(wait_effect=subprocess.TimeoutExpired(["docker", "wait"], 1200))
        self.assertEqual(self.run_host(docker), 124)
        self.assertEqual(self.host_record()["status"], "host-timeout")
        self.assertIn("docker kill", docker.commands())
        self.assertLess(docker.commands().index("docker kill"), docker.commands().index("docker rm"))
        self.assertIsNone(docker.container)

    def test_interrupt_kills_and_keep_container(self):
        docker = FakeDocker(wait_effect=KeyboardInterrupt())
        self.assertEqual(self.run_host(docker, "--keep-container"), 130)
        self.assertEqual(self.host_record()["status"], "host-interrupted")
        self.assertIn("docker kill", docker.commands())
        self.assertNotIn("docker rm", docker.commands())

    def test_existing_container_stops_before_build_and_is_left_alone(self):
        docker = FakeDocker(existing=True)
        self.assertEqual(self.run_host(docker), 2)
        record = self.host_record()
        self.assertEqual((record["status"], record["containerLaunched"]), ("setup-failed", False))
        self.assertIn("already exists", record["error"])
        self.assertNotIn("handBack", record)
        self.assertNotIn("docker build", docker.commands())
        self.assertNotIn("docker rm", docker.commands())

    def test_failed_docker_run_cleans_up_what_it_may_have_created(self):
        docker = FakeDocker(run_fails=True)
        self.assertEqual(self.run_host(docker), 2)
        record = self.host_record()
        self.assertEqual(record["status"], "start-failed")
        self.assertIn("conflict", record["error"])
        self.assertIn("docker rm", docker.commands())

    def test_interrupt_during_docker_run_still_kills(self):
        docker = FakeDocker()
        original = docker.run

        def interrupted(cmd, **kwargs):
            result = original(cmd, **kwargs)
            if cmd[:2] == ["docker", "run"]:
                raise KeyboardInterrupt
            return result

        docker.run = interrupted
        self.assertEqual(self.run_host(docker), 130)
        self.assertIn("docker kill", docker.commands())
        self.assertIsNone(docker.container)

    def test_failed_docker_wait_keeps_a_running_container(self):
        docker = FakeDocker(wait_fails=True)
        self.assertEqual(self.run_host(docker), 2)
        record = self.host_record()
        self.assertEqual(record["status"], "wait-failed")
        self.assertIn("pipe closed", record["error"])
        self.assertNotIn("docker rm", docker.commands())
        self.assertEqual(docker.container, "running")

    def test_setup_failure_can_be_retried_with_the_same_out(self):
        failing = FakeDocker(wait_effect=None)
        with mock.patch.object(host, "build_image", side_effect=host.HostError("docker build failed")):
            self.assertEqual(self.run_host(failing), 2)
        self.assertEqual(self.host_record()["status"], "setup-failed")
        self.assertEqual(self.run_host(FakeDocker()), 1)
        self.assertEqual(self.host_record()["status"], "completed")
        self.assertEqual(self.run_host(FakeDocker()), 2)  # a finished run is never replaced

    def test_out_must_be_new_and_outside_git(self):
        self.out.mkdir(parents=True)
        docker = FakeDocker()
        self.assertEqual(self.run_host(docker), 2)
        self.out = self.base / "checkout" / "run"
        (self.base / "checkout" / ".git").mkdir(parents=True)
        self.assertEqual(self.run_host(docker), 2)
        self.assertEqual(docker.calls, [])

    def test_codex_needs_auth_file_and_mounts_it_readonly(self):
        manifest = self.base / "codex.json"
        manifest.write_text(json.dumps({**MANIFEST, "agent": {**MANIFEST["agent"], "kind": "codex"}}),
                            encoding="utf-8")
        docker = FakeDocker()
        missing = self.base / "no-auth.json"
        self.assertEqual(self.run_host(docker, "--codex-auth", str(missing), manifest=manifest), 2)
        self.assertEqual(docker.calls, [])
        auth = self.base / "fixture auth.json"
        auth.write_text("fixture", encoding="utf-8")
        self.run_host(docker, "--codex-auth", str(auth), manifest=manifest)
        run = next(c for c in docker.calls if c[:2] == ["docker", "run"])
        self.assertIn(host._mount(auth.resolve(), "/run/secrets/codex-auth.json", readonly=True), run)
        self.assertFalse((self.out / "inputs" / "tests").exists())  # the fake agent is staged only for fake runs

    def test_markitect_method_builds_binary_and_passes_sha(self):
        manifest = self.base / "mk.json"
        manifest.write_text(json.dumps({**MANIFEST, "method": "markitect",
                                        "markitect": {"sourceRepo": "/src", "commit": "669cecd2"}}),
                            encoding="utf-8")
        docker = FakeDocker()
        built = {"commit": "669cecd2" + "0" * 32, "sha256": "f" * 64}
        with mock.patch.object(host.shutil, "which", return_value="go"), \
                mock.patch.object(host, "build_markitect", return_value=built) as build:
            self.run_host(docker, manifest=manifest)
        self.assertEqual(build.call_args.args[1], self.out.resolve() / "inputs" / "bin" / "markitect")
        self.assertEqual(self.host_record()["markitect"], built)
        run = next(c for c in docker.calls if c[:2] == ["docker", "run"])
        self.assertIn("MPG_MARKITECT_SHA256=" + "f" * 64, run)
        self.assertIn("MPG_MARKITECT_COMMIT=" + built["commit"], run)
        self.assertFalse((self.out / "inputs" / "methods").exists())  # notes for people stay on the host

    def test_station_count_sets_the_safety_timeout(self):
        (host.ROOT / "cases" / "readinglog2").mkdir()
        (host.ROOT / "cases" / "readinglog2" / "STATIONS.json").write_text(
            station_plan("readinglog2", (1, 3, 6, 2, 2, 1)), encoding="utf-8")
        self.run_host(FakeDocker(), manifest=self.write_manifest("six.json", case="readinglog2"))
        record = self.host_record()
        self.assertEqual(record["stations"], 6)
        self.assertEqual(record["hostTimeoutSeconds"], host.host_timeout(MANIFEST, 6))
        self.assertGreater(host.host_timeout(MANIFEST, 6), host.host_timeout(MANIFEST, 4))
        self.assertTrue((self.out / "inputs" / "cases" / "readinglog2" / "STATIONS.json").is_file())

    def test_case_without_station_plan_fails_before_docker(self):
        (host.ROOT / "cases" / "roombook" / "STATIONS.json").unlink()
        docker = FakeDocker()
        self.assertEqual(self.run_host(docker), 2)
        self.assertEqual(docker.calls, [])

    def test_claude_needs_token_file_mounted_readonly_and_never_reads_it(self):
        manifest = self.write_manifest("claude.json", kind="claude", claudeVersion="2.1.296",
                                       model="claude-opus-5-5")
        docker = FakeDocker()
        self.assertEqual(self.run_host(docker, manifest=manifest), 2)  # --claude-token missing
        self.assertEqual(self.run_host(docker, "--claude-token", str(self.base / "none"), manifest=manifest), 2)
        self.assertEqual(docker.calls, [])
        token = self.base / "claude token"
        secret = "sk-ant-oat01-fixture-" + "x" * 20
        token.write_text(secret + "\n", encoding="utf-8")
        self.run_host(docker, "--claude-token", str(token), manifest=manifest)
        record = self.host_record()
        run = next(c for c in docker.calls if c[:2] == ["docker", "run"])
        self.assertIn(host._mount(token.resolve(), "/run/secrets/claude-token", readonly=True), run)
        self.assertFalse(any("codex-auth" in arg for arg in run))  # conventional: no Codex login needed
        self.assertEqual(record["secrets"], {"codexAuth": False, "claudeToken": True})
        everything = json.dumps(docker.calls) + (self.out / "host.json").read_text(encoding="utf-8")
        self.assertNotIn(secret, everything)
        self.assertNotIn("CLAUDE_CODE_OAUTH_TOKEN", everything)
        self.assertFalse((self.out / "inputs" / "tests").exists())

    def test_markitect_with_claude_needs_both_logins(self):
        manifest = self.base / "mk-claude.json"
        manifest.write_text(json.dumps({**MANIFEST, "method": "markitect",
                                        "agent": {**MANIFEST["agent"], "kind": "claude", "claudeVersion": "2.1.296"},
                                        "markitect": {"sourceRepo": "/src", "commit": "669cecd2",
                                                      "innerModel": "gpt-6-luna", "innerEffort": "high"}}),
                            encoding="utf-8")
        token = self.base / "token"
        token.write_text("fixture-token", encoding="utf-8")
        docker = FakeDocker()
        with mock.patch.object(host.shutil, "which", return_value="go"):
            code = self.run_host(docker, "--claude-token", str(token), "--codex-auth", str(self.base / "none"),
                                 manifest=manifest)
        self.assertEqual((code, docker.calls), (2, []))
        auth = self.base / "auth.json"
        auth.write_text("fixture", encoding="utf-8")
        built = {"commit": "669cecd2" + "0" * 32, "sha256": "f" * 64}
        with mock.patch.object(host.shutil, "which", return_value="go"), \
                mock.patch.object(host, "build_markitect", return_value=built):
            self.run_host(docker, "--claude-token", str(token), "--codex-auth", str(auth), manifest=manifest)
        run = next(c for c in docker.calls if c[:2] == ["docker", "run"])
        self.assertIn(host._mount(auth.resolve(), "/run/secrets/codex-auth.json", readonly=True), run)
        self.assertIn(host._mount(token.resolve(), "/run/secrets/claude-token", readonly=True), run)

    def test_fake_claude_stages_its_fake_and_takes_an_optional_token(self):
        manifest = self.write_manifest("fake-claude.json", kind="fake-claude", claudeVersion="2.1.296")
        docker = FakeDocker()
        self.run_host(docker, manifest=manifest)
        self.assertTrue((self.out / "inputs" / "tests" / "fake_claude.py").is_file())
        self.assertFalse((self.out / "inputs" / "tests" / "fake_agent.py").exists())
        run = next(c for c in docker.calls if c[:2] == ["docker", "run"])
        self.assertEqual(sum(arg == "--mount" for arg in run), 2)
        self.out = self.base / "runs" / "second"
        token = self.base / "dummy-token"
        token.write_text("fake-token", encoding="utf-8")
        self.run_host(docker, "--claude-token", str(token), manifest=manifest)
        run = [c for c in docker.calls if c[:2] == ["docker", "run"]][-1]
        self.assertIn(host._mount(token.resolve(), "/run/secrets/claude-token", readonly=True), run)

    def test_markitect_without_go_fails_clearly(self):
        with mock.patch.object(host.shutil, "which", return_value=None):
            with self.assertRaisesRegex(host.HostError, "Go is required"):
                host.build_markitect({"sourceRepo": "/src", "commit": "abc1234"}, self.base / "bin")


class HandBackTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.folder = Path(temp.name) / "Glacius Labs" / "results"
        (self.folder / "audit").mkdir(parents=True)
        (self.folder / "audit" / "run.json").write_text("{}", encoding="utf-8")
        self.owner = self.folder.stat().st_uid
        self.other = (self.owner + 1, 4242)

    def hand_back(self, operator, docker, foreign=None):
        with mock.patch.object(host, "_operator", return_value=operator), \
                mock.patch.object(host.subprocess, "run", docker.run), contextlib.ExitStack() as stack:
            if foreign is not None:  # entries not owned by the operator, before and after chown
                stack.enter_context(mock.patch.object(host, "_foreign", side_effect=foreign))
            return host.hand_back("mpg-x", "sha256:feed", self.folder)

    def test_foreign_counts_every_entry_of_another_owner(self):
        self.assertEqual(host._foreign(self.folder, self.owner), 0)
        self.assertEqual(host._foreign(self.folder, self.owner + 1), 2)

    def test_not_needed_without_operator_or_when_the_operator_owns_the_output(self):
        docker = FakeDocker()
        self.assertEqual(self.hand_back(None, docker), "not-needed")  # Windows or root
        self.assertEqual(self.hand_back((self.owner, 1000), docker), "not-needed")  # e.g. rootless engine
        self.assertEqual(docker.calls, [])

    def test_output_gets_the_parents_owner_without_following_links(self):
        docker = FakeDocker()
        self.assertEqual(self.hand_back(self.other, docker, foreign=[2, 0]), "done")
        self.assertEqual(docker.commands(), ["docker container", "docker run"])
        run = docker.calls[1]
        self.assertEqual(run[:3], ["docker", "run", "--rm"])
        for flag, value in (("--network", "none"), ("--user", "0:0"), ("--entrypoint", "chown")):
            self.assertEqual(run[run.index(flag) + 1], value)
        mounts = [run[i + 1] for i, arg in enumerate(run) if arg == "--mount"]
        self.assertEqual(mounts, [host._mount(self.folder.parent, "/reference", readonly=True),
                                  host._mount(self.folder, "/handback")])
        self.assertEqual(run[-5:], ["sha256:feed", "-R", "--no-dereference", "--reference=/reference",
                                    "/handback"])

    def test_entries_left_after_chown_are_reported(self):  # e.g. userns-remap
        self.assertEqual(self.hand_back(self.other, FakeDocker(), foreign=[2, 1]),
                         "failed: 1 entries still belong to another user")

    def test_nothing_is_handed_back_unless_docker_confirms_the_container_stopped(self):
        running = FakeDocker()
        running.container = "running"  # e.g. docker wait failed; the agent may still be inside
        unknown = FakeDocker(inspect_error="Cannot connect to the Docker daemon")
        for docker in (running, unknown):
            self.assertEqual(self.hand_back(self.other, docker, foreign=[2]),
                             "skipped: container mpg-x may still be running")
            self.assertNotIn("docker run", docker.commands())

    def test_failure_is_reported_not_raised(self):
        self.assertEqual(self.hand_back(self.other, FakeDocker(run_fails=True), foreign=[2]), "failed: conflict")


class CleanTests(unittest.TestCase):
    def test_clean_removes_stopped_labelled_containers(self):
        docker = FakeDocker()
        with mock.patch.object(host.subprocess, "run", docker.run), quiet():
            self.assertEqual(host.main(["clean"]), 0)
        ps, rm = docker.calls
        self.assertIn("label=markitect-playground=1", ps)
        self.assertNotIn("status=running", ps)
        self.assertEqual(rm, ["docker", "rm", "c1", "c2"])


@unittest.skipUnless(shutil.which("go") and shutil.which("git"), "needs go and git")
class BuildMarkitectTests(unittest.TestCase):
    def test_builds_static_linux_binary_from_commit(self):
        with tempfile.TemporaryDirectory() as temp:
            repo = Path(temp) / "product repo"
            main = repo / "src" / "cmd" / "markitect" / "main.go"
            main.parent.mkdir(parents=True)
            (repo / "go.mod").write_text("module example.com/mk\n\ngo 1.21\n", encoding="utf-8")
            main.write_text('package main\n\nfunc main() { println("ok") }\n', encoding="utf-8")
            git = ["git", "-C", str(repo), "-c", "user.name=t", "-c", "user.email=t@t.invalid"]
            subprocess.run([*git, "init", "-q"], check=True)
            subprocess.run([*git, "add", "."], check=True)
            subprocess.run([*git, "commit", "-q", "-m", "seed"], check=True)
            short = subprocess.run([*git, "rev-parse", "--short", "HEAD"], check=True,
                                   capture_output=True, text=True).stdout.strip()
            target = Path(temp) / "inputs" / "bin" / "markitect"
            result = host.build_markitect({"sourceRepo": str(repo), "commit": short}, target)
            self.assertEqual(len(result["commit"]), 40)
            self.assertTrue(result["commit"].startswith(short))
            self.assertEqual(target.read_bytes()[:4], b"\x7fELF")
            self.assertEqual(len(result["sha256"]), 64)
            self.assertTrue(result["go"].startswith("go1."))


if __name__ == "__main__":
    unittest.main()
