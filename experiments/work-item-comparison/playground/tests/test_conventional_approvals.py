"""Provider-free positive and fail-closed fixtures for scoped Git approval."""

from __future__ import annotations

import pathlib
import hashlib
import json
import os
import shutil
import sys
import subprocess
import tempfile
import threading
import unittest
from unittest.mock import patch

PLAYGROUND = pathlib.Path(__file__).resolve().parents[1]
sys.path.insert(0, str(PLAYGROUND))

from conventional.approvals import ScopedGitApprovalBroker, _argv, _git_action, _outer_argv  # noqa: E402
from conventional.backends import (BackendSpecError, _app_server_environment, _runtime_config_args,
                                   run, validate_runtime_options)  # noqa: E402


class ScopedGitApprovalTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.repo = pathlib.Path(self.temp.name).resolve()
        (self.repo / ".git").mkdir()
        (self.repo / ".git" / "config").write_text(
            "[core]\n\trepositoryformatversion = 0\n\tfilemode = false\n\tbare = false\n\tlogallrefupdates = true\n",
            encoding="utf-8")
        self.shell = self.shell_config()
        self.git_command = self.git("add -- src/file.py")
        self.broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", self.shell)
        self.item = {
            "threadId": "thread-owned", "turnId": "turn-owned",
            "item": {"type": "commandExecution", "id": "item-1",
                     "command": self.git_command, "cwd": str(self.repo)},
        }

    def tearDown(self) -> None:
        self.temp.cleanup()

    def request(self, command: str | None = None, **overrides: object) -> dict:
        command = command or self.git_command
        params = {
            "kind": "command", "threadId": "thread-owned", "turnId": "turn-owned",
            "itemId": "item-1", "startedAtMs": 10, "command": command,
            "cwd": str(self.repo), "availableDecisions": ["accept", "decline", "cancel"],
        }
        params.update(overrides)
        return {"id": "request-1", "method": "item/commandExecution/requestApproval", "params": params}

    def shell_config(self) -> dict[str, str]:
        executable = self.repo / "pwsh.exe"
        git_executable = self.repo / "git.exe"
        executable.write_bytes(b"offline test-only pwsh fixture")
        git_executable.write_bytes(b"offline test-only git fixture")
        return {"executable": str(executable),
                "pinSha256": hashlib.sha256(executable.read_bytes()).hexdigest(),
                "gitExecutable": str(git_executable),
                "gitPinSha256": hashlib.sha256(git_executable.read_bytes()).hexdigest()}

    def git(self, command: str) -> str:
        return (f'"{self.shell["gitExecutable"]}" -c core.hooksPath=/dev/null '
                f"-c core.fsmonitor=false {command}")

    def test_accepts_one_exact_scoped_git_stage_request(self) -> None:
        decision = self.broker.decide(self.request(), self.item)
        self.assertEqual("accept", decision.decision)
        self.assertEqual((self.shell["gitExecutable"], "-c", "core.hooksPath=/dev/null", "-c",
                          "core.fsmonitor=false", "add", "--", "src/file.py"), decision.action)
        self.assertTrue(decision.reason)

    def test_accepts_narrow_branch_switch_commit_and_merge_forms(self) -> None:
        for command in (self.git("branch study/fix"), self.git("switch -c study/fix"),
                        self.git("switch main"), self.git("add --all"),
                        self.git('commit -m "fix scoped test"'), self.git("merge --ff-only study/fix"),
                        self.git("merge --no-ff study/fix")):
            with self.subTest(command=command):
                broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", self.shell)
                item = {**self.item, "item": {**self.item["item"], "command": command}}
                self.assertEqual("accept", broker.decide(self.request(command), item).decision)

    def test_unknown_shell_network_and_destructive_git_commands_decline(self) -> None:
        git_path = f'"{self.shell["gitExecutable"]}"'
        unguarded = (f"{git_path} switch main", f"{git_path} branch study/fix",
                     f"{git_path} add -- src/file.py", f'{git_path} commit -m "unguarded"',
                     f"{git_path} merge --ff-only study/fix")
        for command in (*unguarded, "git add .", "git add ../outside.py", "git add -- src/a.py && git push",
                        "git push origin main", "git reset --hard", "git commit --amend",
                        self.git("checkout main"), self.git("checkout -B main"), self.git("merge --abort"),
                        self.git("branch -D study/fix"), self.git("-c core.hooksPath=anything commit -m change"),
                        self.git("add ../outside.py"), self.git("reset --hard"),
                        f'"{self.shell["gitExecutable"]}" add -- src/file.py',
                        f'"{self.shell["gitExecutable"]}" -c core.hooksPath=/dev/null add -- src/file.py'):
            with self.subTest(command=command):
                broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", self.shell)
                item = {**self.item, "item": {**self.item["item"], "command": command}}
                decision = broker.decide(self.request(command), item)
                self.assertEqual("decline", decision.decision)
                self.assertIsNone(decision.action)

        other_git = self.repo / "other-git.exe"
        other_git.write_bytes(b"unbound executable")
        shadow = f'"{other_git}" add -- src/file.py'
        broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", self.shell)
        item = {**self.item, "item": {**self.item["item"], "command": shadow}}
        self.assertEqual("decline", broker.decide(self.request(shadow), item).decision)

    def test_mismatched_or_ambiguous_request_context_declines(self) -> None:
        cases = (
            {"threadId": "other"}, {"turnId": "other"}, {"itemId": "other"},
            {"cwd": str(self.repo.parent)}, {"networkApprovalContext": {"host": "example.test"}},
            {"additionalPermissions": {"fileSystem": {"write": ["/"]}}},
            {"proposedExecpolicyAmendment": {"execpolicy_amendment": ["git"]}},
            {"availableDecisions": ["acceptForSession", "decline"]},
            {"availableDecisions": {"accept": True}},
            {"availableDecisions": ["accept", {"acceptWithExecpolicyAmendment": {}}]},
            {"futurePermission": True}, {"startedAtMs": "10"},
        )
        for override in cases:
            with self.subTest(override=override):
                broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", self.shell)
                self.assertEqual("decline", broker.decide(self.request(**override), self.item).decision)

    def test_one_time_accept_is_allowed_when_other_choices_are_also_offered(self) -> None:
        for decisions in (None, ["accept", "acceptForSession", "decline", "cancel"]):
            broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", self.shell)
            request = self.request() if decisions is None else self.request(availableDecisions=decisions)
            if decisions is None:
                request["params"].pop("availableDecisions")
            self.assertEqual("accept", broker.decide(request, self.item).decision)

    def test_protected_branch_creation_and_windows_shell_wrappers_decline(self) -> None:
        for command in (self.git("branch main"), self.git("switch -c main"), self.git("checkout -b main"),
                        "git merge study/fix",
                        "C:/Program Files/Codex/pwsh.exe -Command 'git add -- src/file.py'"):
            with self.subTest(command=command):
                broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", self.shell)
                item = {**self.item, "item": {**self.item["item"], "command": command}}
                self.assertEqual("decline", broker.decide(self.request(command), item).decision)

        # Exact command pretty-print captured from the frozen Windows S1 pilot.
        observed = "\"C:\\Users\\Consiliari\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe\" -Command 'Get-Location; git status --short --branch'"
        observed_argv = _outer_argv(observed)
        self.assertEqual("C:\\Users\\Consiliari\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe",
                         observed_argv[0])
        self.assertIsNone(_git_action(_argv(observed_argv[2]), str(self.repo), self.shell["gitExecutable"]))

    def test_pinned_windows_pwsh_wrapper_accepts_only_one_inner_git_action(self) -> None:
        shell = self.shell_config()
        executable = shell["executable"]
        git = self.git("add -- src/file.py")
        safe = f'"{executable}" -NoProfile -NonInteractive -Command \'& {git}\''
        broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", shell)
        item = {**self.item, "item": {**self.item["item"], "command": safe}}
        self.assertEqual("accept", broker.decide(self.request(safe), item).decision)

        invalid_commands = (
            f'"{executable}" -Command \'{git}\'',
            f'"{executable}" -ExecutionPolicy Bypass -NoProfile -Command \'{git}\'',
            f'"{executable}" -NonInteractive -Command \'{git}\'',
            f'"{executable}" -Command \'Get-Location; git status --short --branch\'',
            f'"{executable}" -NoProfile -Command \'{self.git("push origin main")}\'',
            f'"{executable}" -NoProfile -Command \'{git}\' -NoExit',
            f'"{executable}" -NoProfile -Command \'git add -- src/file.py\'',
            f'"{executable}" -NoProfile -Command \'&& {git}\'',
            f'"{executable}" -NoProfile -Command \'&  {git}\'',
        )
        for command in invalid_commands:
            with self.subTest(command=command):
                broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", shell)
                item = {**self.item, "item": {**self.item["item"], "command": command}}
                self.assertEqual("decline", broker.decide(self.request(command), item).decision)

        shell_path = pathlib.Path(executable)
        shell_path.write_bytes(b"changed after source pin")
        broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", shell)
        item = {**self.item, "item": {**self.item["item"], "command": safe}}
        self.assertEqual("decline", broker.decide(self.request(safe), item).decision)

        shell = self.shell_config()
        git_path = pathlib.Path(shell["gitExecutable"])
        git_path.write_bytes(b"changed git after source pin")
        safe = f'"{shell["executable"]}" -NoProfile -Command \'& {self.git("add -- src/file.py")}\''
        broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", shell)
        item = {**self.item, "item": {**self.item["item"], "command": safe}}
        self.assertEqual("decline", broker.decide(self.request(safe), item).decision)

    def test_git_approval_shell_pin_is_checked_before_backend_start(self) -> None:
        shell = self.shell_config()
        options = {"sandbox": "workspace-write", "approvalPolicy": "on-request",
                   "memoryEnabled": False, "scopedGitApproval": True, "allowLoginShell": False,
                   "gitApprovalShell": shell}
        self.assertEqual(shell, validate_runtime_options(options, "model", "high", "codex-app-server")["gitApprovalShell"])
        for invalid, backend in (
                ({**options, "gitApprovalShell": {**shell, "pinSha256": "0" * 64}}, "codex-app-server"),
                ({**options, "gitApprovalShell": {**shell, "gitPinSha256": "0" * 64}}, "codex-app-server"),
                (options, "codex-cli"),
                ({key: value for key, value in options.items() if key != "gitApprovalShell"}, "codex-app-server"),
                ({key: value for key, value in options.items() if key != "scopedGitApproval"}, "codex-app-server"),
                ({**options, "allowLoginShell": None}, "codex-app-server"),
                ({**options, "allowLoginShell": True}, "codex-app-server"),
                ({**options, "gitApprovalShell": {**shell, "futureOption": True}}, "codex-app-server")):
            with self.subTest(invalid=invalid, backend=backend), self.assertRaises(BackendSpecError):
                validate_runtime_options(invalid, "model", "high", backend)

    @unittest.skipUnless(os.name == "nt", "Windows Git executable pinning is the target")
    def test_real_guarded_git_commit_works_with_an_inert_owned_config(self) -> None:
        git_path = shutil.which("git")
        if not git_path or pathlib.Path(git_path).name.casefold() != "git.exe":
            self.skipTest("an absolute Git for Windows executable is unavailable")
        repo = self.repo / "real-git-repo"
        repo.mkdir()
        env = os.environ.copy()
        env.update({"GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.devnull,
                    "USERPROFILE": str(repo), "HOME": str(repo), "GIT_TERMINAL_PROMPT": "0",
                    })
        env.pop("GIT_CONFIG_PARAMETERS", None)
        env.pop("GIT_CONFIG_COUNT", None)

        def git_bootstrap(*args: str) -> subprocess.CompletedProcess[str]:
            return subprocess.run([git_path, *args], cwd=repo, env=env, text=True,
                                  capture_output=True, timeout=20, check=True)

        git_bootstrap("init", "-q")
        git_bootstrap("config", "user.name", "Approval Fixture")
        git_bootstrap("config", "user.email", "approval-fixture@example.invalid")
        (repo / "tracked.txt").write_text("fixture\n", encoding="utf-8")

        git_file = pathlib.Path(git_path)
        shell = self.shell_config()
        shell.update({"gitExecutable": str(git_file.resolve()),
                      "gitPinSha256": hashlib.sha256(git_file.read_bytes()).hexdigest()})
        broker = ScopedGitApprovalBroker(str(repo), "thread-owned", "turn-owned", shell)

        def approve(command: str, item_id: str) -> tuple[str, ...]:
            item = {"threadId": "thread-owned", "turnId": "turn-owned",
                    "item": {"type": "commandExecution", "id": item_id,
                             "command": command, "cwd": str(repo)}}
            request = {"id": f"request-{item_id}", "method": "item/commandExecution/requestApproval",
                       "params": {"kind": "command", "threadId": "thread-owned", "turnId": "turn-owned",
                                  "itemId": item_id, "startedAtMs": 10, "command": command,
                                  "cwd": str(repo)}}
            result = broker.decide(request, item)
            self.assertEqual("accept", result.decision, result.reason)
            return result.action

        prefix = f'"{git_path}" -c core.hooksPath=/dev/null -c core.fsmonitor=false '
        for item_id, suffix in (("stage", "add -- tracked.txt"),
                                ("commit", 'commit -m "guarded hook fixture"')):
            argv = approve(prefix + suffix, item_id)
            completed = subprocess.run(argv, cwd=repo, env=env, text=True,
                                       capture_output=True, timeout=20)
            self.assertEqual(0, completed.returncode, (argv, completed.stdout, completed.stderr))

    @unittest.skipUnless(os.name == "nt", "Windows Git hook-path semantics are the target")
    def test_command_prefix_suppresses_hook_while_broker_rejects_hook_config(self) -> None:
        git_path = shutil.which("git")
        if not git_path or pathlib.Path(git_path).name.casefold() != "git.exe":
            self.skipTest("an absolute Git for Windows executable is unavailable")
        repo = self.repo / "hook-config-repo"
        repo.mkdir()
        hooks = repo / "configured-hooks"
        hooks.mkdir()
        marker = repo / "hook-ran.txt"
        env = os.environ.copy()
        env.update({"GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.devnull,
                    "USERPROFILE": str(repo), "HOME": str(repo), "HOOK_SENTINEL": marker.as_posix()})
        env.pop("GIT_CONFIG_PARAMETERS", None)
        env.pop("GIT_CONFIG_COUNT", None)

        def git_bootstrap(*args: str) -> None:
            subprocess.run([git_path, *args], cwd=repo, env=env, check=True,
                           capture_output=True, timeout=20)

        git_bootstrap("init", "-q")
        git_bootstrap("config", "user.name", "Approval Fixture")
        git_bootstrap("config", "user.email", "approval-fixture@example.invalid")
        git_bootstrap("config", "core.hooksPath", str(hooks))
        hook = hooks / "pre-commit"
        hook.write_text('#!/bin/sh\nprintf hook-ran > "$HOOK_SENTINEL"\nexit 1\n', encoding="utf-8")
        os.chmod(hook, 0o755)
        (repo / "tracked.txt").write_text("fixture\n", encoding="utf-8")
        git_file = pathlib.Path(git_path).resolve()
        shell = self.shell_config()
        shell.update({"gitExecutable": str(git_file),
                      "gitPinSha256": hashlib.sha256(git_file.read_bytes()).hexdigest()})
        prefix = f'"{git_path}" -c core.hooksPath=/dev/null -c core.fsmonitor=false '
        for suffix in ("add -- tracked.txt", 'commit -m "guard configured hook"'):
            completed = subprocess.run([*(_outer_argv(prefix + suffix))], cwd=repo, env=env,
                                       capture_output=True, timeout=20)
            self.assertEqual(0, completed.returncode, completed.stderr)
        self.assertFalse(marker.exists(), "the command-local /dev/null prefix did not suppress the hook")

        command = prefix + 'commit -m "broker must decline"'
        item = {"threadId": "thread-owned", "turnId": "turn-owned",
                "item": {"type": "commandExecution", "id": "hook-config",
                         "command": command, "cwd": str(repo)}}
        request = {"id": "request-hook-config", "method": "item/commandExecution/requestApproval",
                   "params": {"kind": "command", "threadId": "thread-owned", "turnId": "turn-owned",
                              "itemId": "hook-config", "startedAtMs": 10,
                              "command": command, "cwd": str(repo)}}
        broker = ScopedGitApprovalBroker(str(repo), "thread-owned", "turn-owned", shell)
        self.assertEqual("decline", broker.decide(request, item).decision)

    @unittest.skipUnless(os.name == "nt", "the pinned Windows PowerShell executable is required")
    def test_pinned_powershell_call_operator_runs_approved_git_commit(self) -> None:
        pwsh = pathlib.Path(
            r"C:\Users\Consiliari\.cache\codex-runtimes\codex-primary-runtime\dependencies\native\powershell\pwsh.exe")
        git_path = shutil.which("git")
        if not pwsh.is_file():
            self.skipTest("the frozen pinned PowerShell executable is unavailable")
        if not git_path or pathlib.Path(git_path).name.casefold() != "git.exe":
            self.skipTest("an absolute Git for Windows executable is unavailable")
        repo = self.repo / "powershell-git-repo"
        repo.mkdir()
        env = {key: value for key, value in os.environ.items()
               if not key.upper().startswith("GIT_")}
        env.update({"GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.devnull,
                    "USERPROFILE": str(repo), "HOME": str(repo), "GIT_TERMINAL_PROMPT": "0"})
        subprocess.run([git_path, "init", "-q"], cwd=repo, env=env, check=True,
                       capture_output=True, timeout=20)
        subprocess.run([git_path, "config", "user.name", "PowerShell Fixture"], cwd=repo,
                       env=env, check=True, capture_output=True, timeout=20)
        subprocess.run([git_path, "config", "user.email", "powershell-fixture@example.invalid"], cwd=repo,
                       env=env, check=True, capture_output=True, timeout=20)
        (repo / "tracked.txt").write_text("fixture\n", encoding="utf-8")

        git_file = pathlib.Path(git_path).resolve()
        shell = {"executable": str(pwsh.resolve()),
                 "pinSha256": hashlib.sha256(pwsh.read_bytes()).hexdigest(),
                 "gitExecutable": str(git_file),
                 "gitPinSha256": hashlib.sha256(git_file.read_bytes()).hexdigest()}
        broker = ScopedGitApprovalBroker(str(repo), "thread-owned", "turn-owned", shell)
        prefix = f'& "{git_file}" -c core.hooksPath=/dev/null -c core.fsmonitor=false '

        def run_approved(suffix: str, item_id: str) -> None:
            inner = prefix + suffix
            rendered = f'"{pwsh}" -NoProfile -NonInteractive -Command \'{inner}\''
            item = {"threadId": "thread-owned", "turnId": "turn-owned",
                    "item": {"type": "commandExecution", "id": item_id,
                             "command": rendered, "cwd": str(repo)}}
            request = {"id": f"request-{item_id}", "method": "item/commandExecution/requestApproval",
                       "params": {"kind": "command", "threadId": "thread-owned", "turnId": "turn-owned",
                                  "itemId": item_id, "startedAtMs": 10,
                                  "command": rendered, "cwd": str(repo)}}
            decision = broker.decide(request, item)
            self.assertEqual("accept", decision.decision, decision.reason)
            completed = subprocess.run([str(pwsh), "-NoProfile", "-NonInteractive", "-Command", inner],
                                       cwd=repo, env=env, capture_output=True, timeout=20)
            self.assertEqual(0, completed.returncode, (completed.stdout, completed.stderr))

        run_approved("add -- tracked.txt", "stage")
        run_approved('commit -m "PowerShell pinned commit"', "commit")
        status = subprocess.run([git_path, "rev-parse", "--verify", "HEAD"], cwd=repo, env=env,
                               capture_output=True, timeout=20)
        self.assertEqual(0, status.returncode, status.stderr)

    @unittest.skipUnless(os.name == "nt", "Windows Git executable pinning is the target")
    def test_forbidden_local_git_execution_configurations_decline(self) -> None:
        git_path = shutil.which("git")
        if not git_path or pathlib.Path(git_path).name.casefold() != "git.exe":
            self.skipTest("an absolute Git for Windows executable is unavailable")
        repo = self.repo / "forbidden-config-repo"
        repo.mkdir()
        env = os.environ.copy()
        env.update({"GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.devnull,
                    "USERPROFILE": str(repo), "HOME": str(repo)})
        subprocess.run([git_path, "init", "-q"], cwd=repo, env=env, check=True,
                       capture_output=True, timeout=20)
        shell = self.shell_config()
        git_file = pathlib.Path(git_path)
        shell.update({"gitExecutable": str(git_file.resolve()),
                      "gitPinSha256": hashlib.sha256(git_file.read_bytes()).hexdigest()})
        command = (f'"{git_path}" -c core.hooksPath=/dev/null -c core.fsmonitor=false '
                   'commit -m "must decline"')
        item = {"threadId": "thread-owned", "turnId": "turn-owned",
                "item": {"type": "commandExecution", "id": "config-check",
                         "command": command, "cwd": str(repo)}}
        base = (repo / ".git" / "config").read_text(encoding="utf-8")
        dangerous = (
            '[filter "evil"]\n\tclean = cmd /c echo owned\n',
            '[commit]\n\tgpgsign = true\n',
            '[merge "evil"]\n\tdriver = cmd /c echo owned\n',
            '[include]\n\tpath = ../outside.gitconfig\n',
        )
        for extra in dangerous:
            with self.subTest(config=extra):
                (repo / ".git" / "config").write_text(base + extra, encoding="utf-8")
                broker = ScopedGitApprovalBroker(str(repo), "thread-owned", "turn-owned", shell)
                request = {"id": "request-config", "method": "item/commandExecution/requestApproval",
                           "params": {"kind": "command", "threadId": "thread-owned",
                                      "turnId": "turn-owned", "itemId": "config-check",
                                      "startedAtMs": 10, "command": command, "cwd": str(repo)}}
                self.assertEqual("decline", broker.decide(request, item).decision)

    def test_git_directory_and_config_must_not_be_symlinks(self) -> None:
        config = self.repo / ".git" / "config"
        external = self.repo / "external-config"
        external.write_text(config.read_text(encoding="utf-8"), encoding="utf-8")
        saved_config = self.repo / "saved-config"
        config.replace(saved_config)
        try:
            config.symlink_to(external)
        except OSError as exc:
            saved_config.replace(config)
            self.skipTest(f"Windows symlink creation is unavailable: {exc}")
        self.assertEqual("decline", self.broker.decide(self.request(), self.item).decision)
        config.unlink()
        saved_config.replace(config)

        git_dir = self.repo / ".git"
        saved_git_dir = self.repo / "saved-git-dir"
        git_dir.replace(saved_git_dir)
        try:
            git_dir.symlink_to(saved_git_dir, target_is_directory=True)
        except OSError as exc:
            saved_git_dir.replace(git_dir)
            self.skipTest(f"Windows directory symlink creation is unavailable: {exc}")
        self.assertEqual("decline", self.broker.decide(self.request(), self.item).decision)

    def test_request_must_match_an_observed_command_item_and_request_id_is_one_shot(self) -> None:
        broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", self.shell)
        self.assertEqual("decline", broker.decide(self.request(), None).decision)
        broker = ScopedGitApprovalBroker(str(self.repo), "thread-owned", "turn-owned", self.shell)
        self.assertEqual("accept", broker.decide(self.request(), self.item).decision)
        self.assertEqual("decline", broker.decide(self.request(), self.item).decision)

    def test_runtime_option_requires_explicit_app_server_on_request_policy(self) -> None:
        options = {"sandbox": "workspace-write", "approvalPolicy": "on-request",
                   "memoryEnabled": False, "scopedGitApproval": True, "allowLoginShell": False,
                   "gitApprovalShell": self.shell}
        self.assertEqual(options, validate_runtime_options(options, "model", "high", "codex-app-server"))
        args = _runtime_config_args(options, "model", "high")
        self.assertIn("allow_login_shell=false", args)
        for invalid, backend in (({**options, "approvalPolicy": "never"}, "codex-app-server"),
                                 (options, "codex-cli"),
                                 ({**options, "scopedGitApproval": 1}, "codex-app-server")):
            with self.subTest(invalid=invalid, backend=backend), self.assertRaises(BackendSpecError):
                validate_runtime_options(invalid, "model", "high", backend)

    def test_scoped_app_server_environment_removes_inherited_git_controls(self) -> None:
        options = {"scopedGitApproval": True}
        with patch.dict(os.environ, {
                "GIT_CONFIG_COUNT": "1", "GIT_CONFIG_KEY_0": "core.hooksPath",
                "GIT_CONFIG_VALUE_0": "C:/unsafe/hooks", "GIT_DIR": "C:/unsafe/repo/.git"}, clear=False):
            env = _app_server_environment(options)
        self.assertEqual("1", env["GIT_CONFIG_NOSYSTEM"])
        self.assertEqual(os.devnull, env["GIT_CONFIG_GLOBAL"])
        self.assertFalse(any(key.upper().startswith("GIT_") and key not in {
            "GIT_CONFIG_NOSYSTEM", "GIT_CONFIG_GLOBAL"} for key in env))
        self.assertIsNone(_app_server_environment({"scopedGitApproval": False}))

    def test_app_server_receives_exact_one_time_decision_and_audited_request(self) -> None:
        fixture = self.repo / "fake_server.py"
        shell = self.shell_config()
        pretty_command = f'"{shell["executable"]}" -NoProfile -NonInteractive -Command \'& {self.git("add -- src/file.py")}\''
        fixture.write_text(r'''import json, os, sys
assert "allow_login_shell=false" in sys.argv, sys.argv
assert os.environ.get("GIT_CONFIG_NOSYSTEM") == "1", os.environ.get("GIT_CONFIG_NOSYSTEM")
assert os.environ.get("GIT_CONFIG_GLOBAL") == os.devnull, os.environ.get("GIT_CONFIG_GLOBAL")
assert "GIT_CONFIG_COUNT" not in os.environ
assert "GIT_DIR" not in os.environ
def send(value): print(json.dumps(value), flush=True)
for line in sys.stdin:
    message=json.loads(line); method=message.get("method")
    if method == "initialize": send({"id": message["id"], "result": {}})
    elif method == "thread/start": send({"id": message["id"], "result": {"threadId": "thread-owned"}})
    elif method == "turn/start":
        send({"id": message["id"], "result": {"turnId": "turn-owned"}})
        item={"type":"commandExecution","id":"item-1","command":sys.argv[2],"cwd":sys.argv[1]}
        send({"method":"item/started","params":{"threadId":"thread-owned","turnId":"turn-owned","item":item}})
        send({"id":"request-1","method":"item/commandExecution/requestApproval","params":{
            "kind":"command","threadId":"thread-owned","turnId":"turn-owned","itemId":"item-1",
            "startedAtMs":10,"command":item["command"],"cwd":item["cwd"],
            "availableDecisions":["accept","decline","cancel"]}})
    elif message.get("id") == "request-1":
        assert message == {"id":"request-1","result":{"decision":"accept"}}, message
        send({"method":"turn/completed","params":{"threadId":"thread-owned","turn":{"id":"turn-owned","status":"completed"}}})
''', encoding="utf-8")
        command = [sys.executable, str(fixture), str(self.repo), pretty_command]
        options = {"sandbox": "workspace-write", "approvalPolicy": "on-request",
                   "memoryEnabled": False, "scopedGitApproval": True, "allowLoginShell": False,
                   "gitApprovalShell": shell}
        events: list[dict] = []
        with patch.dict(os.environ, {
                "GIT_CONFIG_COUNT": "1", "GIT_CONFIG_KEY_0": "core.hooksPath",
                "GIT_CONFIG_VALUE_0": "C:/unsafe/hooks", "GIT_DIR": "C:/unsafe/repo/.git"}, clear=False):
            result = run({"backend": "codex-app-server", "command": command,
                          "cwd": str(self.repo), "model": "fixture-model", "effort": "high",
                          "timeoutSeconds": 3, "runtimeOptions": options},
                         "fixture prompt", None, events.append, threading.Event())
        self.assertEqual("completed", result["state"], (result, events))
        policy = next(event["parsed"] for event in events if event.get("stream") == "runtime-policy")
        self.assertEqual("scoped-git-approval-env-v1", policy["policy"])
        self.assertNotIn("C:/unsafe/hooks", json.dumps(events))
        audit = next(event["parsed"] for event in events if event.get("stream") == "approval-broker")
        self.assertEqual("accept", audit["decision"])
        self.assertEqual(pretty_command, audit["request"]["params"]["command"])
        response = next(event["parsed"] for event in events if event.get("stream") == "client"
                        and event.get("parsed", {}).get("result", {}).get("decision"))
        self.assertEqual({"id": "request-1", "result": {"decision": "accept"}}, response)


if __name__ == "__main__":
    unittest.main()
