import contextlib
import csv
import io
import json
import os
import platform
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from playground import __main__ as entry
from playground import host, image, outcome
from tests import evaluation_repo
from tests.test_image import DOCKERFILE, sample_inventory

def station_plan(case: str, sizes: tuple[int, ...]) -> str:
    ids = iter(f"X{i:02d}" for i in range(1, 100))
    return json.dumps({"schema": 1, "case": case, "stations": [
        {"id": f"S{n}", "items": [next(ids) for _ in range(size)]} for n, size in enumerate(sizes, 1)]})


def make_case(root: Path, case: str, sizes: tuple[int, ...]) -> None:
    """A case folder that discovery accepts."""
    folder = root / "cases" / case
    (folder / "checks").mkdir(parents=True)
    for name in ("README.md", "BACKLOG.md"):
        (folder / name).write_text(f"{case}\n", encoding="utf-8")
    (folder / "STATIONS.json").write_text(station_plan(case, sizes), encoding="utf-8")
    (folder / "checks" / f"{case}.py").write_text("def checks(ctx):\n    pass\n", encoding="utf-8")


def resolved(product: dict) -> dict:
    """Stands in for host.resolve_markitect (no Git checkout in the tests)."""
    return {**product, "sourceRepo": str(Path(product["sourceRepo"]).resolve()), "commit": "669cecd2" + "0" * 32}


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
    """Stands in for subprocess.run/Popen inside host; records every argv. The container
    exits `exit_code` and leaves a results/report.json of class `run_class` (None: none)."""

    def __init__(self, wait_effect=None, run_fails=False, existing=False, wait_fails=False, inspect_error=None,
                 exit_code=1, run_class="none", build_fails=False, inventory=None, oom=False):
        self.calls = []
        self.build_fails, self.inventory, self.oom = build_fails, inventory, oom
        self.inspect_error = inspect_error
        self.wait_effect = wait_effect
        self.run_fails = run_fails
        self.wait_fails = wait_fails
        self.exit_code, self.run_class, self.results = exit_code, run_class, None
        self.container = "stopped" if existing else None  # None, "running" or "stopped"

    def run(self, cmd, **kwargs):
        self.calls.append(list(cmd))
        code, out, err = 0, "", ""
        if cmd[:2] == ["docker", "version"]:
            out = "29.4.1"
        elif cmd[:2] == ["docker", "build"]:
            code = 1 if self.build_fails else 0
        elif cmd[:3] == ["docker", "image", "inspect"]:
            out = "sha256:feed"
        elif cmd[:2] == ["docker", "run"] and cmd[-1] == image.INVENTORY_IN_IMAGE:
            out = self.inventory if self.inventory is not None else sample_inventory(host.image_pins())
        elif cmd[:3] == ["docker", "container", "inspect"]:
            if self.inspect_error:
                code, err = 1, self.inspect_error
            elif self.container is None:
                code, err = 1, f"Error: No such container: {cmd[-1]}"
            elif "{{.State.OOMKilled}}" in cmd:
                out = str(self.oom).lower()
            else:
                out = str(self.container == "running").lower()
        elif cmd[:2] == ["docker", "run"]:
            if self.run_fails:
                code, err = 125, "conflict"
            else:
                out, self.container = "cid", "running"
                mounts = [next(csv.reader([cmd[i + 1]])) for i, arg in enumerate(cmd) if arg == "--mount"]
                self.results = next((Path(fields[1].split("=", 1)[1]) for fields in mounts
                                     if "target=/out" in fields), self.results)
        elif cmd[:2] == ["docker", "wait"]:
            if self.wait_effect:
                raise self.wait_effect
            if self.wait_fails:
                code, err = 1, "error during connect: pipe closed"
            else:
                out, self.container = f"{self.exit_code}\n", "stopped"
                if self.run_class is not None:
                    report = {"classification": {"class": self.run_class, "reason": "fake"}}
                    (self.results / "report.json").write_text(json.dumps(report), encoding="utf-8")
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
        for name in ("playground/__pycache__", "cases/common", "methods/conventional", "methods/markitect",
                     "tests", "container"):
            (root / name).mkdir(parents=True)
        shutil.copyfile(DOCKERFILE, root / "container" / "Dockerfile")
        make_case(root, "roombook", (1, 3, 7, 1))
        make_case(root, "readinglog", (1, 3, 7, 1))
        (root / "cases" / "task-prompt.txt").write_text("prompt\n", encoding="utf-8")
        (root / "methods" / "conventional" / "AGENTS.fragment.md").write_text("fragment\n", encoding="utf-8")
        (root / "methods" / "markitect" / "README.md").write_text("notes for people\n", encoding="utf-8")
        (root / "playground" / "runner.py").write_text("", encoding="utf-8")
        (root / "playground" / "__pycache__" / "x.pyc").write_bytes(b"")
        (root / "tests" / "fake_agent.py").write_text("", encoding="utf-8")
        (root / "tests" / "fake_claude.py").write_text("", encoding="utf-8")
        # The pre-registered evaluation files; registration reads them there (the fake root is no checkout).
        self.repo = evaluation_repo.committed(self.base / "registered")
        self.manifest_path = self.base / "manifest.json"
        self.manifest_path.write_text(json.dumps(MANIFEST), encoding="utf-8")
        self.out = self.base / "runs" / "one, two"
        self.addCleanup(self.temp.cleanup)
        for patcher in (mock.patch.object(host, "ROOT", root), evaluation_repo.use(self.repo)):
            patcher.start()
            self.addCleanup(patcher.stop)

    def write_manifest(self, name: str, **changes) -> Path:
        data = json.loads(json.dumps(MANIFEST))
        for key, value in changes.items():
            target = data["agent"] if key in ("kind", "claudeVersion", "model") else data
            if value is ...:
                del target[key]
                continue
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
        argv = host.docker_run_argv(MANIFEST, name="n", image="i", image_id="d", inputs=Path("/i"), results=Path("/r"),
                                    auth=None, markitect=None, host_os={"system": "Linux", "machine": "x86_64"})
        envs = [argv[i + 1] for i, arg in enumerate(argv) if arg == "--env"]
        self.assertEqual(envs, ["MPG_IMAGE_ID=d", "MPG_HOST_SYSTEM=Linux", "MPG_HOST_MACHINE=x86_64"])

    def test_no_auth_mount_for_fake(self):
        argv = host.docker_run_argv(MANIFEST, name="n", image="i", image_id="d", inputs=Path("/i"),
                                    results=Path("/r"), auth=None, markitect=None)
        self.assertEqual(sum(arg == "--mount" for arg in argv), 2)
        self.assertFalse(any(arg.startswith("MPG_MARKITECT") for arg in argv))


class RunTests(HostTestBase):
    def test_completed_run_stages_inputs_and_removes_container(self):
        docker = FakeDocker()
        self.assertEqual(self.run_host(docker), 1)  # stopped early, class none: the method's outcome
        record = self.host_record()
        self.assertEqual(record["status"], "completed")
        self.assertEqual((record["containerExitCode"], record["exitCode"]), (1, 1))
        pinned = host.image_pins()
        inventory = (self.out / "image-inventory.txt").read_bytes()
        self.assertEqual(inventory.decode("utf-8"), sample_inventory(pinned))
        self.assertEqual(record["image"], {"tag": "markitect-playground:codex-0.162.0-claude-2.1.296",
                                           "id": "sha256:feed", "base": pinned["base"],
                                           "snapshot": pinned["snapshot"],
                                           "dockerfileSha256": pinned["dockerfileSha256"],
                                           "inventorySha256": image.sha256(inventory)})
        self.assertIs(record["oomKilled"], False)
        self.assertEqual(record["stations"], 4)
        self.assertEqual(record["secrets"], {"codexAuth": False, "claudeToken": False})
        self.assertEqual(record["dockerVersion"], "29.4.1")
        self.assertEqual(record["hostPlatform"], {"system": platform.system(), "machine": platform.machine()})
        self.assertEqual(record["hostPlatform"], host.host_platform())
        run = next(c for c in docker.calls if c[:3] == ["docker", "run", "--detach"])
        self.assertIn(f"MPG_HOST_SYSTEM={platform.system()}", run)
        self.assertIn(f"MPG_HOST_MACHINE={platform.machine()}", run)
        inputs = self.out / "inputs"
        # the container gets the normalized manifest that host.json records
        self.assertEqual(json.loads((inputs / "manifest.json").read_text(encoding="utf-8")), record["manifest"])
        self.assertEqual(record["manifest"]["stations"], 4)
        self.assertTrue((inputs / "tests" / "fake_agent.py").is_file())
        self.assertTrue((inputs / "playground" / "runner.py").is_file())
        self.assertFalse((inputs / "playground" / "__pycache__").exists())
        self.assertTrue((inputs / "cases" / "common").is_dir())
        self.assertTrue((inputs / "cases" / "roombook" / "checks" / "roombook.py").is_file())
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
        self.assertIn(f"SOURCE_DATE_EPOCH={pinned['sourceDateEpoch']}", build)
        self.assertIn("--provenance=false", build)  # keeps the image ID stable across rebuilds
        read = next(c for c in docker.calls if c[-1] == image.INVENTORY_IN_IMAGE)
        self.assertEqual(read[:8], ["docker", "run", "--rm", "--label", "markitect-playground=1", "--network", "none",
                                    "--entrypoint"])
        self.assertEqual(read[-3:-1], ["cat", "sha256:feed"])  # the image just built, by its ID
        self.assertLess(docker.calls.index(build), docker.calls.index(read))
        self.assertLess(docker.calls.index(read), docker.calls.index(run))
        run = next(c for c in docker.calls if c[:3] == ["docker", "run", "--detach"])
        self.assertIn(host._mount(inputs.resolve(), "/in", readonly=True), run)
        commands = docker.commands()
        self.assertLess(commands.index("docker container"), commands.index("docker build"))  # name check first
        self.assertLess(commands.index("docker wait"), commands.index("docker rm"))
        self.assertNotIn("docker kill", commands)
        self.assertIsNone(docker.container)

    def test_the_run_records_its_pre_registration(self):
        self.assertEqual(self.run_host(FakeDocker()), 1)
        record = self.host_record()
        pre = record["preRegistration"]
        self.assertEqual((pre["status"], pre["evaluationTree"], pre["commit"]),
                         ("registered", evaluation_repo.git(self.repo, "rev-parse", "HEAD:evaluation"),
                          evaluation_repo.git(self.repo, "rev-parse", "HEAD")))
        self.assertEqual(record["rules"], {"exploratory": False})

    def test_changed_evaluation_files_are_refused_before_the_build_unless_exploratory(self):
        (self.repo / "evaluation" / "config.json").write_text("{}\n", encoding="utf-8")
        (self.repo / "evaluation" / "notes.md").write_text("new\n", encoding="utf-8")
        docker, err = FakeDocker(), io.StringIO()
        argv = ["run", "--manifest", str(self.manifest_path), "--out", str(self.out)]
        with mock.patch.object(host.subprocess, "run", docker.run), contextlib.redirect_stderr(err):
            self.assertEqual(host.main(argv), 2)
        self.assertIn("evaluation/config.json, evaluation/notes.md", err.getvalue())
        self.assertIn("--exploratory", err.getvalue())
        self.assertEqual(docker.calls, [])
        self.assertFalse(self.out.exists())
        self.assertEqual(self.run_host(FakeDocker(), "--exploratory"), 1)
        record = self.host_record()
        self.assertEqual(record["rules"], {"exploratory": True})
        self.assertEqual((record["preRegistration"]["status"], record["preRegistration"]["evaluationTree"]),
                         ("dirty", None))
        self.assertIn("evaluation/notes.md", record["preRegistration"]["paths"])

    def test_a_playground_outside_a_checkout_is_refused(self):
        plain = self.base / "plain"
        (plain / "evaluation").mkdir(parents=True)
        docker = FakeDocker()
        with evaluation_repo.use(plain):
            self.assertEqual(self.run_host(docker), 2)
        self.assertEqual(docker.calls, [])

    def test_an_unexpected_error_is_recorded_with_the_code_the_process_exits_with(self):
        argv = ["host", "run", "--manifest", str(self.manifest_path), "--out", str(self.out)]
        docker = FakeDocker()
        with mock.patch.object(host.subprocess, "run", docker.run), \
                mock.patch.object(host.subprocess, "Popen", docker.popen), \
                mock.patch.object(host, "stage_inputs", side_effect=RuntimeError("bug")), quiet():
            self.assertEqual(entry.main(argv), outcome.HARNESS)
        record = self.host_record()
        self.assertEqual((record["status"], record["exitCode"], record["error"]), ("harness-error", 10, "RuntimeError: bug"))

    def test_a_killed_container_is_the_environments_only_when_docker_reports_oom(self):
        docker = FakeDocker(exit_code=137, run_class=None, oom=True)
        self.assertEqual(self.run_host(docker), 11)
        record = self.host_record()
        self.assertEqual((record["containerExitCode"], record["oomKilled"], record["exitCode"]), (137, True, 11))
        oom = [c for c in docker.calls if "{{.State.OOMKilled}}" in c]
        self.assertEqual([c[-1] for c in oom], ["mpg-fake-roombook-001"])
        commands = [" ".join(c[:2]) for c in docker.calls]
        self.assertLess(docker.calls.index(oom[0]), commands.index("docker rm"))  # asked before the removal
        shutil.rmtree(self.out)
        self.assertEqual(self.run_host(FakeDocker(exit_code=137, run_class="environment")), 10)
        record = self.host_record()
        self.assertEqual((record["containerExitCode"], record["oomKilled"], record["exitCode"]), (137, False, 10))

    def test_a_failed_image_build_is_the_environments(self):
        docker = FakeDocker(build_fails=True)
        self.assertEqual(self.run_host(docker), 11)
        record = self.host_record()
        self.assertEqual((record["status"], record["exitCode"], record["containerLaunched"]),
                         ("setup-failed", 11, False))
        self.assertIn("docker build failed", record["error"])
        self.assertNotIn("docker run", docker.commands())

    def test_an_image_whose_inventory_differs_from_the_committed_one_fails_the_build(self):
        committed = host.ROOT / "container" / "inventory" / "codex-0.162.0-claude-2.1.296.txt"
        committed.parent.mkdir()
        committed.write_text(sample_inventory(host.image_pins()), encoding="utf-8")
        self.assertEqual(self.run_host(FakeDocker()), 1)  # the same inventory: the run goes on
        built = sample_inventory(host.image_pins(), dpkg=("bash 5.2.15-2+b9 amd64",))
        for inventory, fragment in ((built, "+ dpkg bash 5.2.15-2+b9 amd64"),
                                    (sample_inventory(host.image_pins(), npm=("left-pad@^1.0.0",)),
                                     "not an exact version: left-pad@^1.0.0"),
                                    ("garbage", "not an image inventory")):
            with self.subTest(fragment=fragment):
                shutil.rmtree(self.out)
                docker = FakeDocker(inventory=inventory)
                self.assertEqual(self.run_host(docker), 11)
                record = self.host_record()
                self.assertEqual((record["status"], record["exitCode"]), ("setup-failed", 11))
                self.assertIn("failed its inventory check", record["error"])
                self.assertIn(fragment, record["error"])
                self.assertEqual((self.out / "image-inventory.txt").read_text(encoding="utf-8"), inventory)  # to review
                self.assertFalse(any("--detach" in call for call in docker.calls))

    def test_without_a_committed_inventory_a_run_only_warns(self):
        docker, err = FakeDocker(), io.StringIO()
        argv = ["run", "--manifest", str(self.manifest_path), "--out", str(self.out)]
        with mock.patch.object(host.subprocess, "run", docker.run), \
                mock.patch.object(host.subprocess, "Popen", docker.popen), \
                contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(err):
            self.assertEqual(host.main(argv), 1)
        self.assertIn("warning: no committed image inventory", err.getvalue())
        self.assertIn("codex-0.162.0-claude-2.1.296.txt", err.getvalue())
        self.assertEqual(self.host_record()["status"], "completed")
        # The built inventory is printed between markers, exactly as it would be committed.
        text = err.getvalue()
        begin = host.INVENTORY_BEGIN.format(name="container/inventory/codex-0.162.0-claude-2.1.296.txt")
        printed = text.split(begin + "\n", 1)[1].split(host.INVENTORY_END, 1)[0]
        self.assertEqual(printed, (self.out / "image-inventory.txt").read_text(encoding="utf-8"))

    def test_an_unpinned_dockerfile_is_refused_before_docker(self):
        dockerfile = host.ROOT / "container" / "Dockerfile"
        base = host.image_pins()["base"]
        dockerfile.write_text(dockerfile.read_text(encoding="utf-8").replace(base, "node:22-bookworm-slim"),
                              encoding="utf-8")
        docker, err = FakeDocker(), io.StringIO()
        argv = ["run", "--manifest", str(self.manifest_path), "--out", str(self.out)]
        with mock.patch.object(host.subprocess, "run", docker.run), contextlib.redirect_stderr(err):
            self.assertEqual(host.main(argv), 2)
        self.assertIn("is not pinned by digest", err.getvalue())
        self.assertEqual(docker.calls, [])
        self.assertFalse(self.out.exists())

    def test_a_non_linux_host_gets_a_warning_and_the_run_goes_on(self):
        for system, warned in (("Windows", True), ("Darwin", True), ("Linux", False)):
            with self.subTest(system=system):
                shutil.rmtree(self.out, ignore_errors=True)
                docker, err = FakeDocker(), io.StringIO()
                argv = ["run", "--manifest", str(self.manifest_path), "--out", str(self.out)]
                with mock.patch.object(host.subprocess, "run", docker.run), \
                        mock.patch.object(host.subprocess, "Popen", docker.popen), \
                        mock.patch.object(host.platform, "system", return_value=system), \
                        contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(err):
                    self.assertEqual(host.main(argv), 1)  # the run went on
                self.assertEqual("DEC-013" in err.getvalue(), warned, err.getvalue())
                if warned:
                    self.assertIn(f"warning: this host runs {system}", err.getvalue())
                    self.assertIn("compare refuses", err.getvalue())

    def test_results_are_handed_back_after_the_container_ends(self):
        docker = FakeDocker()
        with mock.patch.object(host, "hand_back", return_value="done") as hand_back:
            self.run_host(docker)
        hand_back.assert_called_once_with("mpg-fake-roombook-001", "sha256:feed", self.out.resolve() / "results")
        self.assertEqual(self.host_record()["handBack"], "done")

    def test_the_exit_code_maps_the_run_class_and_keeps_the_runner_code(self):
        for exit_code, run_class, expected in ((0, "none", 0), (1, "none", 1), (0, "product", 12),
                                               (1, "environment", 11), (2, "harness", 10), (0, None, 10),
                                               (2, "none", 10)):
            with self.subTest(exit_code=exit_code, run_class=run_class):
                shutil.rmtree(self.out, ignore_errors=True)
                self.assertEqual(self.run_host(FakeDocker(exit_code=exit_code, run_class=run_class)), expected)
                record = self.host_record()
                self.assertEqual((record["containerExitCode"], record["exitCode"]), (exit_code, expected))

    def test_timeout_kills_and_records(self):
        docker = FakeDocker(wait_effect=subprocess.TimeoutExpired(["docker", "wait"], 1200))
        self.assertEqual(self.run_host(docker), 124)
        self.assertEqual((self.host_record()["status"], self.host_record()["exitCode"]), ("host-timeout", 124))
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
        self.assertEqual(self.run_host(docker), 11)
        record = self.host_record()
        self.assertEqual((record["status"], record["containerLaunched"]), ("setup-failed", False))
        self.assertIn("already exists", record["error"])
        self.assertNotIn("handBack", record)
        self.assertNotIn("docker build", docker.commands())
        self.assertNotIn("docker rm", docker.commands())

    def test_failed_docker_run_cleans_up_what_it_may_have_created(self):
        docker = FakeDocker(run_fails=True)
        self.assertEqual(self.run_host(docker), 11)
        record = self.host_record()
        self.assertEqual((record["status"], record["exitCode"]), ("start-failed", 11))
        self.assertIn("conflict", record["error"])
        self.assertIn("docker rm", docker.commands())

    def test_interrupt_during_docker_run_still_kills(self):
        docker = FakeDocker()
        original = docker.run

        def interrupted(cmd, **kwargs):
            result = original(cmd, **kwargs)
            if cmd[:3] == ["docker", "run", "--detach"]:
                raise KeyboardInterrupt
            return result

        docker.run = interrupted
        self.assertEqual(self.run_host(docker), 130)
        self.assertIn("docker kill", docker.commands())
        self.assertIsNone(docker.container)

    def test_failed_docker_wait_keeps_a_running_container(self):
        docker = FakeDocker(wait_fails=True)
        self.assertEqual(self.run_host(docker), 11)
        record = self.host_record()
        self.assertEqual(record["status"], "wait-failed")
        self.assertIn("pipe closed", record["error"])
        self.assertNotIn("docker rm", docker.commands())
        self.assertEqual(docker.container, "running")

    def run_prebuilt(self, docker, before_remove):
        manifest = host.manifest_module.load(self.manifest_path, playground=host.ROOT)
        inventory = self.base / "built-once-inventory.txt"
        inventory.write_text("the study's inventory\n", encoding="utf-8")
        prebuilt = {"image": {"tag": host.image_tag(manifest), "id": "sha256:built-once"}, "inventory": inventory}
        with mock.patch.object(host.subprocess, "run", docker.run),                 mock.patch.object(host.subprocess, "Popen", docker.popen), quiet():
            return host.run_manifest(manifest, self.out, auth=None, token=None, prebuilt=prebuilt,
                                     before_remove=before_remove)

    def test_a_prebuilt_image_runs_by_its_id_and_the_hook_runs_before_removal(self):
        docker, seen = FakeDocker(), []
        code, record = self.run_prebuilt(docker, lambda name: seen.append((name, docker.container)))
        self.assertEqual((code, record["image"]["id"]), (1, "sha256:built-once"))
        self.assertEqual(seen, [("mpg-fake-roombook-001", "stopped")])
        run = next(c for c in docker.calls if c[:3] == ["docker", "run", "--detach"])
        self.assertIn("sha256:built-once", run)
        self.assertFalse(any(arg.startswith("markitect-playground:") for arg in run))  # never the tag
        self.assertFalse(any(c[:2] == ["docker", "build"] for c in docker.calls))
        self.assertIsNone(docker.container)
        self.assertEqual((self.out / "image-inventory.txt").read_text(encoding="utf-8"), "the study's inventory\n")

    def test_an_interrupt_in_the_pre_removal_hook_still_removes_the_container(self):
        docker = FakeDocker()

        def interrupted(name):
            raise KeyboardInterrupt
        code, record = self.run_prebuilt(docker, interrupted)
        self.assertEqual((code, record["status"]), (130, "host-interrupted"))
        self.assertIn(["docker", "rm", "-f", "mpg-fake-roombook-001"], docker.calls)
        self.assertIsNone(docker.container)
        self.assertEqual(self.host_record()["status"], "host-interrupted")  # host.json is still written
        docker = FakeDocker()
        self.out = self.base / "runs" / "failing-hook"
        code, record = self.run_prebuilt(docker, lambda name: 1 / 0)  # an ordinary error only warns
        self.assertEqual((code, record["status"]), (1, "completed"))
        self.assertIsNone(docker.container)

    def test_setup_failure_can_be_retried_with_the_same_out(self):
        failing = FakeDocker(wait_effect=None)
        with mock.patch.object(host, "build_image", side_effect=host.HostError("docker build failed")):
            self.assertEqual(self.run_host(failing), 11)
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
        self.assertEqual(self.run_host(docker, "--codex-auth", str(missing), manifest=manifest), 11)
        self.assertEqual(docker.calls, [])
        auth = self.base / "fixture auth.json"
        auth.write_text("fixture", encoding="utf-8")
        self.run_host(docker, "--codex-auth", str(auth), manifest=manifest)
        run = next(c for c in docker.calls if c[:3] == ["docker", "run", "--detach"])
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
                mock.patch.object(host, "resolve_markitect", side_effect=resolved), \
                mock.patch.object(host, "build_markitect", return_value=built) as build:
            self.run_host(docker, manifest=manifest)
        self.assertEqual(build.call_args.args[1], self.out.resolve() / "inputs" / "bin" / "markitect")
        self.assertEqual(self.host_record()["markitect"], built)
        recorded = self.host_record()["manifest"]["markitect"]
        self.assertEqual(recorded, {"sourceRepo": str(Path("/src").resolve()), "commit": built["commit"]})
        self.assertEqual(build.call_args.args[0], recorded)
        staged = json.loads((self.out / "inputs" / "manifest.json").read_text(encoding="utf-8"))
        self.assertEqual(staged["markitect"], recorded)
        run = next(c for c in docker.calls if c[:3] == ["docker", "run", "--detach"])
        self.assertIn("MPG_MARKITECT_SHA256=" + "f" * 64, run)
        self.assertIn("MPG_MARKITECT_COMMIT=" + built["commit"], run)
        self.assertFalse((self.out / "inputs" / "methods").exists())  # notes for people stay on the host

    def test_a_markitect_binary_that_does_not_build_is_a_product_failure(self):
        manifest = self.base / "mk.json"
        manifest.write_text(json.dumps({**MANIFEST, "method": "markitect",
                                        "markitect": {"sourceRepo": "/src", "commit": "669cecd2"}}),
                            encoding="utf-8")
        failure = host.HostError("the Markitect commit does not build: go build failed", outcome.PRODUCT)
        with mock.patch.object(host.shutil, "which", return_value="go"), \
                mock.patch.object(host, "resolve_markitect", side_effect=resolved), \
                mock.patch.object(host, "build_markitect", side_effect=failure):
            self.assertEqual(self.run_host(FakeDocker(), manifest=manifest), 12)
        record = self.host_record()
        self.assertEqual((record["status"], record["failureClass"], record["exitCode"]), ("setup-failed", "product", 12))

    def test_station_count_sets_the_safety_timeout(self):
        make_case(host.ROOT, "readinglog2", (1, 3, 6, 2, 2, 1))
        self.run_host(FakeDocker(), manifest=self.write_manifest("six.json", case="readinglog2"))
        record = self.host_record()
        self.assertEqual(record["stations"], 6)
        self.assertEqual(record["hostTimeoutSeconds"], host.host_timeout(MANIFEST, 6))
        self.assertGreater(host.host_timeout(MANIFEST, 6), host.host_timeout(MANIFEST, 4))
        self.assertTrue((self.out / "inputs" / "cases" / "readinglog2" / "STATIONS.json").is_file())

    def test_stations_truncate_the_run_and_its_safety_timeout(self):
        self.run_host(FakeDocker(), manifest=self.write_manifest("two.json", stations=2))
        record = self.host_record()
        self.assertEqual((record["stations"], record["manifest"]["stations"]), (2, 2))
        self.assertEqual(record["hostTimeoutSeconds"], host.host_timeout(MANIFEST, 2))
        self.assertEqual(host.station_count({"case": "roombook", "stations": 3}), 3)
        self.assertEqual(host.station_count({"case": "roombook"}), 4)
        with self.assertRaisesRegex(host.HostError, "unknown case 'nothing'"):
            host.station_count({"case": "nothing"})
        docker = FakeDocker()
        self.out = self.base / "runs" / "too-many"
        self.assertEqual(self.run_host(docker, manifest=self.write_manifest("five.json", stations=5)), 2)
        self.assertEqual(docker.calls, [])

    def test_source_repo_defaults_to_the_checkout_holding_the_playground(self):
        data = {**MANIFEST, "method": "markitect", "markitect": {"commit": "669cecd2"}}
        manifest = self.base / "mk-default.json"
        manifest.write_text(json.dumps(data), encoding="utf-8")
        docker = FakeDocker()
        with mock.patch.object(host.shutil, "which", return_value="go"):
            self.assertEqual(self.run_host(docker, manifest=manifest), 2)  # no checkout holds the fake root
        self.assertEqual(docker.calls, [])
        (host.ROOT / ".git").mkdir()  # the playground's checkout (the run folder stays outside it)
        with mock.patch.object(host.shutil, "which", return_value="go"):
            self.assertEqual(self.run_host(docker, manifest=manifest), 2)  # not a Markitect checkout
        self.assertEqual(docker.calls, [])
        (host.ROOT / "go.mod").write_text("module github.com/Glacius-Labs/Markitect\n", encoding="utf-8")
        built ={"commit": "669cecd2" + "0" * 32, "sha256": "f" * 64}
        with mock.patch.object(host.shutil, "which", return_value="go"), \
                mock.patch.object(host, "resolve_markitect", side_effect=resolved) as resolve, \
                mock.patch.object(host, "build_markitect", return_value=built):
            self.run_host(docker, manifest=manifest)
        self.assertEqual(resolve.call_args.args[0]["sourceRepo"], str(host.ROOT.resolve()))
        self.assertEqual(self.host_record()["manifest"]["markitect"]["sourceRepo"], str(host.ROOT.resolve()))

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
        self.assertEqual(self.run_host(docker, "--claude-token", str(self.base / "none"), manifest=manifest), 11)
        self.assertEqual(docker.calls, [])
        token = self.base / "claude token"
        secret = "sk-ant-oat01-fixture-" + "x" * 20
        token.write_text(secret + "\n", encoding="utf-8")
        self.run_host(docker, "--claude-token", str(token), manifest=manifest)
        record = self.host_record()
        run = next(c for c in docker.calls if c[:3] == ["docker", "run", "--detach"])
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
        self.assertEqual((code, docker.calls), (11, []))  # the missing Codex login
        auth = self.base / "auth.json"
        auth.write_text("fixture", encoding="utf-8")
        built = {"commit": "669cecd2" + "0" * 32, "sha256": "f" * 64}
        with mock.patch.object(host.shutil, "which", return_value="go"), \
                mock.patch.object(host, "resolve_markitect", side_effect=resolved), \
                mock.patch.object(host, "build_markitect", return_value=built):
            self.run_host(docker, "--claude-token", str(token), "--codex-auth", str(auth), manifest=manifest)
        run = next(c for c in docker.calls if c[:3] == ["docker", "run", "--detach"])
        self.assertIn(host._mount(auth.resolve(), "/run/secrets/codex-auth.json", readonly=True), run)
        self.assertIn(host._mount(token.resolve(), "/run/secrets/claude-token", readonly=True), run)

    def test_fake_claude_stages_its_fake_and_takes_an_optional_token(self):
        manifest = self.write_manifest("fake-claude.json", kind="fake-claude", claudeVersion="2.1.296")
        docker = FakeDocker()
        self.run_host(docker, manifest=manifest)
        self.assertTrue((self.out / "inputs" / "tests" / "fake_claude.py").is_file())
        self.assertFalse((self.out / "inputs" / "tests" / "fake_agent.py").exists())
        run = next(c for c in docker.calls if c[:3] == ["docker", "run", "--detach"])
        self.assertEqual(sum(arg == "--mount" for arg in run), 2)
        self.out = self.base / "runs" / "second"
        token = self.base / "dummy-token"
        token.write_text("fake-token", encoding="utf-8")
        self.run_host(docker, "--claude-token", str(token), manifest=manifest)
        run = [c for c in docker.calls if c[:3] == ["docker", "run", "--detach"]][-1]
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


class BuildFailureTests(unittest.TestCase):
    def build(self, error: str) -> host.HostError:
        def capture(cmd, **kwargs):
            if cmd[0] == "go":
                raise host.HostError(f"go build ... failed (1): {error}")
            if "archive" in cmd:
                import tarfile
                with tarfile.open(cmd[cmd.index("-o") + 1], "w"):
                    pass
            return "c" * 40
        with tempfile.TemporaryDirectory() as temp, mock.patch.object(host.shutil, "which", return_value="go"), \
                mock.patch.object(host, "_capture", side_effect=capture), \
                self.assertRaises(host.HostError) as caught:
            host.build_markitect({"sourceRepo": temp, "commit": "c" * 40}, Path(temp) / "bin" / "markitect")
        return caught.exception

    def test_a_failed_go_build_is_the_products_only_for_a_compile_error_in_its_sources(self):
        for text in ("internal/x.go:3: undefined: y",
                     "# github.com/Glacius-Labs/Markitect/internal/x\nsrc/internal/x.go:3:5: undefined: y",
                     "go: downloading go1.27.1 (linux/amd64)\n# example/x\n./src/x.go:9:2: declared and not used: z"):
            with self.subTest(text=text):
                failure = self.build(text)
                self.assertEqual(failure.code, outcome.PRODUCT)
                self.assertIn("does not build", str(failure))
        for text in ("go: downloading go1.27.1: dial tcp: lookup proxy.golang.org: no such host",
                     "verifying module: Get https://sum.golang.org/lookup: i/o timeout",
                     "src/x.go:3:5: undefined: y\ngo build: write /tmp/go-build1/b001/exe/a.out: "
                     "no space left on device",
                     "open /root/.cache/go-build/00/x: permission denied",
                     "go: download go1.27.1 for linux/amd64: toolchain not available",
                     "go: go.mod requires go >= 1.27.1 (running go 1.25.0; GOTOOLCHAIN=local)",
                     "/usr/local/go/src/runtime/x.go:3:5: internal compiler error",
                     "github.com/other/dep@v1.2.3/x.go:3:5: undefined: y",
                     "signal: killed",
                     "link: running gcc failed",
                     ""):
            with self.subTest(text=text):
                self.assertEqual(self.build(text).code, outcome.ENVIRONMENT)  # when unsure: the environment's

    def test_go_failure_class_reads_only_the_output(self):
        self.assertEqual(host.go_failure_class("x.go:1:1: syntax error"), outcome.PRODUCT)
        self.assertEqual(host.go_failure_class("C:\\go\\src\\x.go:1:1: syntax error"), outcome.ENVIRONMENT)
        self.assertEqual(host.go_failure_class("something else failed"), outcome.ENVIRONMENT)


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


@unittest.skipUnless(shutil.which("git"), "needs git")
class ResolveMarkitectTests(unittest.TestCase):
    def test_absolute_source_and_full_commit(self):
        with tempfile.TemporaryDirectory() as temp:
            repo = Path(temp) / "product repo"
            repo.mkdir()
            (repo / "README.md").write_text("x\n", encoding="utf-8")
            git = ["git", "-C", str(repo), "-c", "user.name=t", "-c", "user.email=t@t.invalid"]
            subprocess.run([*git, "init", "-q"], check=True)
            subprocess.run([*git, "add", "."], check=True)
            subprocess.run([*git, "commit", "-q", "-m", "seed"], check=True)
            full = subprocess.run([*git, "rev-parse", "HEAD"], check=True, capture_output=True,
                                  text=True).stdout.strip()
            cwd = os.getcwd()
            os.chdir(temp)
            try:
                result = host.resolve_markitect({"sourceRepo": "product repo", "commit": full[:8],
                                                 "innerModel": "m"})
            finally:
                os.chdir(cwd)
            self.assertEqual(result, {"sourceRepo": str(repo.resolve()), "commit": full, "innerModel": "m"})
            with self.assertRaisesRegex(host.HostError, "is not a commit in"):
                host.resolve_markitect({"sourceRepo": str(repo), "commit": "deadbeef"})
            with self.assertRaisesRegex(host.HostError, "is not a folder"):
                host.resolve_markitect({"sourceRepo": str(repo / "missing"), "commit": full})


if __name__ == "__main__":
    unittest.main()
