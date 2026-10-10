import json
import os
import subprocess
import sys
import tempfile
import time
import unittest
from pathlib import Path
from unittest import mock

from playground import assess, methods

PLAYGROUND = Path(__file__).resolve().parents[1]
GIT_IDENTITY = ["-c", "user.name=Test", "-c", "user.email=test@example.invalid"]
MANIFEST = {"schema": 1, "id": "test", "case": "readinglog", "method": "markitect",
            "agent": {"kind": "codex", "codexVersion": "0.162.0", "model": "gpt-6-luna",
                      "effort": "high", "maxSubagents": 3},
            "limits": {"stationSeconds": 5400, "totalSeconds": 14400}}

# Shaped like the runtime.yaml the product's setup writes (go yaml.v3, indent 4).
RUNTIME_YAML = """apiVersion: markitect.example.org/project-run/v1alpha1
mode: controlled-local
requireIsolation: false
agents:
    '["project.markitect.example.org/v1alpha1","Manager","","project-owner"]':
        command: /usr/lib/codex/vendor/bin/codex
        args: []
        transport: codex-app-server
        appServer:
            reasoningEffort: high
            permissionProfile: :workspace
            helpers:
                enabled: true
                maxDepth: 1
        model: gpt-6-luna
        modelOptions: null
        providerVersion: codex-cli 0.162.0
        instructionPaths:
            - AGENTS.md
        runtimeFiles:
            - path: /usr/lib/codex/vendor/bin/codex
              mode: "0755"
        environment:
            - PATH
        pricing:
            inputMicrosPerMillion: 1000000
verifier:
    command: /usr/lib/codex/vendor/bin/codex
    transport: codex-app-server
    appServer:
        reasoningEffort: high
    model: gpt-6-luna
    providerVersion: codex-cli 0.162.0
review:
    agents:
        '["project.markitect.example.org/v1alpha1","Manager","","project-owner"]':
            command: /usr/lib/codex/vendor/bin/codex
            transport: codex-app-server
            appServer:
                reasoningEffort: high
            model: gpt-6-luna
            providerVersion: codex-cli 0.162.0
    maxRounds: 3
limits:
    maxDepth: 8
"""
ROLE = {"executor": "codex-app-server", "providerVersion": "codex-cli 0.162.0",
        "command": "/usr/lib/codex/vendor/bin/codex", "model": "gpt-6-luna", "effort": "high"}

# Stand-in for the product CLI: records argv, returns preview JSON with digests,
# writes only with --write and the matching --expect, and refuses writes on main.
FAKE_MARKITECT = r'''
import json, pathlib, subprocess, sys
LOG, FAIL, RUNTIME = __LOG__, __FAIL__, __RUNTIME__
args = sys.argv[1:]
with open(LOG, "a", encoding="utf-8") as handle:
    handle.write(json.dumps(args) + "\n")
action = args[1]
def opt(name):
    return args[args.index(name) + 1] if name in args else None
def stop(message):
    sys.stderr.write(f"markitect project {action}: {message}\n")
    sys.exit(2)
repo, write = pathlib.Path(opt("--repo")), "--write" in args
if action == FAIL:
    stop("simulated product failure")
if write:
    branch = subprocess.run(["git", "branch", "--show-current"], cwd=repo, capture_output=True, text=True).stdout.strip()
    if branch in ("", "main", "master"):
        stop("writing requires an isolated non-protected Git branch")
digests = {"init": "a" * 64, "onboard": "b" * 64, "setup": "c" * 64}
if write and action in ("onboard", "setup") and opt("--expect") != digests[action]:
    stop("expected digest does not match")
if action == "init":
    if write:
        (repo / ".markitect" / "model").mkdir(parents=True)
        (repo / ".markitect" / "project.yaml").write_text("name: case\n", encoding="utf-8")
        (repo / ".markitect" / "model" / "manager.yaml").write_text("kind: Manager\n", encoding="utf-8")
    print(json.dumps({"apiVersion": "project/v1", "name": opt("--name"), "digest": digests["init"], "files": []}))
elif action == "onboard":
    if write:
        with open(repo / "AGENTS.md", "a", encoding="utf-8") as handle:
            handle.write("\n## Markitect\n\nUse the project MCP tools.\n")
    print(json.dumps({"apiVersion": "onboarding/v1", "digest": digests["onboard"], "files": []}))
elif action == "setup":
    if write:
        (repo / ".markitect" / "runtime.yaml").write_text(RUNTIME, encoding="utf-8")
    print(json.dumps({"apiVersion": "setup/v1", "editPlan": {"digest": digests["setup"]},
                      "mutation": {"files": [{"path": ".markitect/runtime.yaml", "content": RUNTIME}]}}))
elif action == "check":
    print(json.dumps({"status": "succeeded", "findings": None, "coverage": {"conforming": False}}))
    sys.exit(1)
'''


def git(repo: Path, *args: str) -> str:
    return subprocess.run(["git", *GIT_IDENTITY, *args], cwd=repo, check=True,
                          capture_output=True, text=True, encoding="utf-8").stdout.strip()


def make_repo(root: Path) -> Path:
    repo = root / "repo"
    repo.mkdir()
    (repo / "AGENTS.md").write_text("# Project rules\n\nRead README.md first.\n", encoding="utf-8", newline="\n")
    (repo / "README.md").write_text("# Case\n", encoding="utf-8", newline="\n")
    git(repo, "init", "-q", "--initial-branch=main")
    git(repo, "config", "core.autocrlf", "false")
    git(repo, "add", "-A")
    git(repo, "commit", "-q", "-m", "seed")
    return repo


def local_runner(fake_binary: Path | None = None):
    """Test-mode `runAsAgent`: same contract, current user, runs the fake through Python."""
    def run(cmd, cwd, timeout, stdout_path, stderr_path, extra_env=None):
        if fake_binary is not None and Path(cmd[0]) == fake_binary:
            cmd = [sys.executable, *cmd]
        started = time.monotonic()
        with open(stdout_path, "wb") as out, open(stderr_path, "wb") as err:
            code = subprocess.run(cmd, cwd=cwd, stdout=out, stderr=err, timeout=timeout,
                                  env=dict(os.environ, **(extra_env or {}))).returncode
        return {"exitCode": code, "timedOut": False, "seconds": round(time.monotonic() - started, 3)}
    return run


class ConventionalSetupTest(unittest.TestCase):
    def setup_with(self, kind: str, *, existing_claude_md: bool = False):
        folder = tempfile.TemporaryDirectory()
        self.addCleanup(folder.cleanup)
        root = Path(folder.name)
        repo = make_repo(root)
        if existing_claude_md:
            (repo / "CLAUDE.md").write_text("# Own rules\n", encoding="utf-8", newline="\n")
            git(repo, "add", "CLAUDE.md")
            git(repo, "commit", "-q", "-m", "own CLAUDE.md")
        agent = {**MANIFEST["agent"], "kind": kind, "claudeVersion": "2.1.296"}
        ctx = {"manifest": dict(MANIFEST, method="conventional", agent=agent), "case": "readinglog",
               "inDir": PLAYGROUND, "outDir": root / "out", "agentHome": root / "home", "runAsAgent": local_runner()}
        return repo, methods.setup("conventional", repo, ctx)

    def test_claude_router_is_committed_before_the_method(self):
        repo, result = self.setup_with("fake-claude")
        self.assertEqual(result["status"], "ready", result["error"])
        self.assertEqual((repo / "CLAUDE.md").read_bytes(), b"@AGENTS.md\n")
        self.assertEqual(git(repo, "log", "-2", "--format=%s").splitlines(),
                         ["Install conventional workflow", "Add CLAUDE.md router to AGENTS.md"])
        self.assertEqual(result["notes"]["claudeRouter"], {"path": "CLAUDE.md", "content": "@AGENTS.md\n",
                                                           "added": True})
        self.assertEqual(git(repo, "status", "--porcelain"), "")

    def test_existing_claude_md_is_kept_and_codex_gets_no_router(self):
        repo, result = self.setup_with("claude", existing_claude_md=True)
        self.assertEqual((repo / "CLAUDE.md").read_text(encoding="utf-8"), "# Own rules\n")
        self.assertFalse(result["notes"]["claudeRouter"]["added"])
        repo, result = self.setup_with("codex")
        self.assertFalse((repo / "CLAUDE.md").exists())
        self.assertNotIn("claudeRouter", result["notes"])
        self.assertEqual((result["blockedBy"], result["roles"]), (None, None))

    def test_appends_fragment_and_commits(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            repo = make_repo(root)
            ctx = {"manifest": dict(MANIFEST, method="conventional"), "case": "readinglog", "inDir": PLAYGROUND,
                   "outDir": root / "out", "agentHome": root / "home", "runAsAgent": local_runner()}
            result = methods.setup("conventional", repo, ctx)
            self.assertEqual((result["status"], result["error"], result["mcpServers"]), ("ready", None, {}))
            self.assertEqual(result["commit"], git(repo, "rev-parse", "HEAD"))
            self.assertEqual(git(repo, "log", "-1", "--format=%s"), "Install conventional workflow")
            self.assertEqual(git(repo, "status", "--porcelain"), "")
            fragment = (PLAYGROUND / "methods" / "conventional" / "AGENTS.fragment.md").read_text(encoding="utf-8")
            agents = (repo / "AGENTS.md").read_text(encoding="utf-8")
            self.assertTrue(agents.startswith("# Project rules\n\nRead README.md first.\n\n"))
            self.assertTrue(agents.endswith(fragment.strip("\n") + "\n"))
            self.assertTrue(all(step["exitCode"] == 0 for step in result["steps"]))
            self.assertTrue((root / "out" / result["steps"][-1]["stdout"]).is_file())


class MarkitectSetupTest(unittest.TestCase):
    def setUp(self):
        self.root = Path(tempfile.mkdtemp())
        self.addCleanup(assess._rmtree, self.root)
        self.repo = make_repo(self.root)
        self.log = self.root / "argv.jsonl"
        self.installed = self.root / "usr-bin" / "markitect"
        self.codex = self.root / "codex-native"
        self.codex.write_bytes(b"\x7fELF fake")

    def run_setup(self, fail: str | None = None, kind: str = "codex", binary: bool = True) -> dict:
        in_dir = self.root / "in"
        (in_dir / "bin").mkdir(parents=True)
        script = (FAKE_MARKITECT.replace("__LOG__", repr(str(self.log))).replace("__FAIL__", repr(fail))
                  .replace("__RUNTIME__", repr(RUNTIME_YAML)))
        if binary:
            (in_dir / "bin" / "markitect").write_text(script, encoding="utf-8")
        manifest = dict(MANIFEST, agent={**MANIFEST["agent"], "kind": kind, "claudeVersion": "2.1.296"},
                        markitect={"sourceRepo": "unused", "commit": "669cecd2"})
        if kind in ("claude", "fake-claude"):  # the outer model is no Codex model
            manifest["agent"]["model"] = "claude-opus-5-5"
            manifest["markitect"].update(innerModel="gpt-6-luna", innerEffort="high")
        ctx = {"manifest": manifest, "case": "readinglog", "inDir": in_dir, "outDir": self.root / "out",
               "agentHome": self.root / "home", "runAsAgent": local_runner(self.installed),
               "markitectBinary": self.installed, "codexExecutable": str(self.codex)}
        return methods.setup("markitect", self.repo, ctx)

    def calls(self) -> list[list[str]]:
        return [json.loads(line) for line in self.log.read_text(encoding="utf-8").splitlines()]

    def test_runs_preview_then_write_with_returned_digests(self):
        result = self.run_setup()
        self.assertEqual((result["status"], result["error"]), ("ready", None))
        repo = str(self.repo.resolve())
        setup = ["setup", "--repo", repo, "--provider", "codex", "--model", "gpt-6-luna", "--effort", "high",
                 "--provider-executable", str(self.codex), "--input-micros-per-million", "1000000",
                 "--output-micros-per-million", "1000000", "--max-cost-micros", str(14400 * 50_000)]
        self.assertEqual(self.calls(), [
            ["project", "init", "--repo", repo, "--name", "readinglog"],
            ["project", "init", "--repo", repo, "--name", "readinglog", "--write"],
            ["project", "onboard", "--repo", repo, "--provider", "codex"],
            ["project", "onboard", "--repo", repo, "--provider", "codex", "--expect", "b" * 64, "--write"],
            ["project", *setup],
            ["project", *setup, "--expect", "c" * 64, "--write"],
            ["project", "check", "--repo", repo],
        ])
        self.assertEqual(result["mcpServers"], {"markitect": {
            "command": str(self.installed), "args": ["project", "mcp", "--repo", repo]}})
        self.assertEqual(git(self.repo, "branch", "--show-current"), "main")
        self.assertEqual(git(self.repo, "branch", "--list", methods.MARKITECT_BRANCH), "")
        self.assertEqual(git(self.repo, "log", "-1", "--format=%s"), "Install Markitect project workflow")
        self.assertEqual(result["commit"], git(self.repo, "rev-parse", "HEAD"))
        self.assertEqual(git(self.repo, "status", "--porcelain"), "")
        self.assertTrue((self.repo / ".markitect" / "runtime.yaml").is_file())
        self.assertEqual(result["notes"]["initialCheck"],
                         {"exitCode": 1, "status": "succeeded", "coverageConforming": False})
        self.assertEqual(result["notes"]["markitectSha256"], methods.sha_file(self.installed))
        self.assertTrue(all(isinstance(step["seconds"], float) for step in result["steps"]))
        self.assertEqual(result["roles"], [{"role": "worker", "manager": "project-owner", **ROLE},
                                           {"role": "reviewer", "manager": "project-owner", **ROLE},
                                           {"role": "verifier", "manager": None, **ROLE}])
        self.assertEqual((result["notes"]["runtimeSource"], result["notes"]["onboardProvider"]),
                         ("setup-write output", "codex"))
        self.assertFalse((self.repo / "CLAUDE.md").exists())

    def test_claude_outer_agent_gets_router_and_both_onboardings(self):
        result = self.run_setup(kind="claude")
        self.assertEqual((result["status"], result["error"]), ("ready", None))
        repo = str(self.repo.resolve())
        onboard = [call for call in self.calls() if call[1] == "onboard"]
        self.assertEqual(onboard[0], ["project", "onboard", "--repo", repo, "--provider", "both"])
        self.assertEqual(result["notes"]["onboardProvider"], "both")
        setup = next(call for call in self.calls() if call[1] == "setup")
        self.assertEqual(setup[setup.index("--provider") + 1], "codex")  # inner roles stay on Codex
        self.assertEqual((setup[setup.index("--model") + 1], setup[setup.index("--effort") + 1]),
                         ("gpt-6-luna", "high"))
        self.assertTrue((self.repo / "CLAUDE.md").read_text(encoding="utf-8").startswith("@AGENTS.md\n"))
        self.assertEqual(git(self.repo, "log", "-2", "--format=%s").splitlines(),
                         ["Install Markitect project workflow", "Add CLAUDE.md router to AGENTS.md"])
        self.assertEqual(len(result["roles"]), 3)

    def test_failure_sources(self):
        self.assertEqual(self.run_setup(fail="setup")["blockedBy"], "product")
        assess._rmtree(self.root / "in")
        missing = self.run_setup(binary=False)
        self.assertEqual((missing["status"], missing["blockedBy"]), ("blocked", "harness"))
        self.assertIn("binary missing", missing["error"])

    def test_product_failure_blocks_setup(self):
        result = self.run_setup(fail="onboard")
        self.assertEqual((result["status"], result["blockedBy"]), ("blocked", "product"))
        self.assertIn("simulated product failure", result["error"])
        self.assertEqual((result["commit"], result["mcpServers"]), (None, {}))
        failed = result["steps"][-1]
        self.assertEqual((failed["name"], failed["exitCode"]), ("onboard-preview", 2))
        self.assertIn("simulated", (self.root / "out" / failed["stderr"]).read_text(encoding="utf-8"))


class RuntimeRolesTest(unittest.TestCase):
    def test_reads_roles_from_runtime_yaml(self):
        roles = methods.runtime_roles(RUNTIME_YAML)
        self.assertEqual([(r["role"], r["manager"]) for r in roles],
                         [("worker", "project-owner"), ("reviewer", "project-owner"), ("verifier", None)])

    def test_tolerates_other_yaml_shapes(self):
        text = ("# comment\n---\nagents:\n"
                "  \"a: b\":   # quoted key with a colon\n"
                "    model: 'it''s'\n    transport: exec # trailing comment\n"
                "    notes: |\n      model: not-this\n      effort: nor-this\n"
                "    effort: low\n"
                "  plain:\n    model: m2\n    list:\n    - x: 1\n      model: inner\n    effort: ~\n"
                "verifier: null\n")
        self.assertEqual(methods.runtime_roles(text), [
            {"role": "worker", "manager": "a: b", "executor": "exec", "providerVersion": None, "command": None,
             "model": "it's", "effort": "low"},
            {"role": "worker", "manager": "plain", "executor": None, "providerVersion": None, "command": None,
             "model": "m2", "effort": None}])
        self.assertEqual(methods.runtime_roles(""), [])
        self.assertEqual(methods.runtime_roles("not: [yaml\n  - at all"), [])


class NativeCodexTest(unittest.TestCase):
    def test_finds_vendored_binary_behind_node_launcher(self):
        with tempfile.TemporaryDirectory() as folder:
            package = Path(folder) / "codex"
            (package / "bin").mkdir(parents=True)
            (package / "package.json").write_text("{}", encoding="utf-8")
            launcher = package / "bin" / "codex.js"
            launcher.write_text("#!/usr/bin/env node\n", encoding="utf-8")
            native = package / "node_modules" / "codex-linux-x64" / "vendor" / "x86_64" / "bin" / "codex"
            native.parent.mkdir(parents=True)
            native.write_bytes(b"\x7fELF binary")
            with mock.patch.object(methods.shutil, "which", return_value=str(launcher)):
                self.assertEqual(Path(methods.find_native_codex()), native.resolve())
            with mock.patch.object(methods.shutil, "which", return_value=None):
                self.assertIsNone(methods.find_native_codex())


if __name__ == "__main__":
    unittest.main()
