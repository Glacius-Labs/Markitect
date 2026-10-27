"""Runner mechanics against Git plumbing and a fake engine, never a provider."""

import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import unittest

import runner

REPO = Path(__file__).resolve().parents[2]
IMAGE = "sha256:" + "a" * 64
LOGIN = {"auth_mode": "chatgpt", "OPENAI_API_KEY": None, "last_refresh": "2026-10-20T00:00:00Z",
         "tokens": {"id_token": "ID-SECRET-VALUE", "access_token": "ACCESS-SECRET-VALUE",
                    "refresh_token": "REFRESH-SECRET-VALUE", "account_id": "account-1"}}

FAKE_ENGINE = r'''
import json, os, sys, time
args = sys.argv[1:]
log = os.environ["FAKE_DIR"]
with open(os.path.join(log, "calls.jsonl"), "a") as f:
    f.write(json.dumps(args) + "\n")
if args[:2] == ["image", "inspect"]:
    print(os.environ["FAKE_IMAGE"])
elif args[0] in ("ps", "volume", "kill", "rm"):
    pass
elif "/usr/local/bin/codex" in args:
    if os.environ.get("FAKE_SLEEP"):
        time.sleep(float(os.environ["FAKE_SLEEP"]))
    for event in ({"type": "thread.started", "thread_id": "t-1"}, {"type": "turn.started"},
                  {"type": "item.completed", "item": {"type": "command_execution", "command": "go test ./..."}},
                  {"type": "item.completed", "item": {"type": "agent_message", "text": "Done."}},
                  {"type": "turn.completed", "usage": {"input_tokens": 10, "output_tokens": 2}}):
        print(json.dumps(event))
else:
    script = args[-1]
    data = sys.stdin.buffer.read()
    if "fixture.bundle" in script:
        print("c" * 40, "t" * 40)
    elif "auth.json" in script and "cat >" in script:
        open(os.path.join(log, "installed-login.json"), "wb").write(data)
    elif "result.bundle" in script:
        sys.stderr.write("d" * 40 + "\n" + "e" * 40 + "\n")
        sys.stdout.write("bundle")
    elif script.startswith("tar") or "tar -c" in script:
        sys.stdout.write("tar")
'''


def git(*args, cwd):
    return subprocess.run(["git", "-C", str(cwd), *args], check=True, capture_output=True, text=True).stdout.strip()


class FixtureTests(unittest.TestCase):
    def test_new_continue_and_root_fixtures(self):
        with tempfile.TemporaryDirectory() as tmp:
            tmp = Path(tmp)
            m1, m2 = tmp / "m1.md", tmp / "m2.md"
            m1.write_bytes(b"limit 5\r\nkept bytes\n")
            m2.write_bytes(b"limit 3\n")
            first = runner.build_fixture("new", tmp / "s1.bundle", "Library requirements", m1)
            again = runner.build_fixture("new", tmp / "s1-again.bundle", "Library requirements", m1)
            self.assertEqual(first["commit"], again["commit"])  # fixed identity and date
            clone = tmp / "clone"
            subprocess.run(["git", "clone", "-q", "-b", "main", str(tmp / "s1.bundle"), str(clone)], check=True, capture_output=True)
            self.assertEqual(git("ls-tree", "-r", "--name-only", "HEAD", cwd=clone), "requirements.md")
            self.assertEqual(subprocess.run(["git", "-C", str(clone), "cat-file", "blob", "HEAD:requirements.md"],
                                            capture_output=True, check=True).stdout, m1.read_bytes())
            (clone / "realization").mkdir()
            (clone / "realization" / "go.mod").write_text("module library\n", encoding="utf-8")
            (clone / ".markitect").mkdir()
            (clone / ".markitect" / "model.yaml").write_text("x\n", encoding="utf-8")
            git("add", "-A", cwd=clone)
            subprocess.run(["git", "-C", str(clone), "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "work"], check=True)
            subprocess.run(["git", "-C", str(clone), "bundle", "create", str(tmp / "r.bundle"), "HEAD"], check=True, capture_output=True)
            work = git("rev-parse", "HEAD", cwd=clone)
            nxt = runner.build_fixture("continue", tmp / "s2.bundle", "Library M2 requirements", m2, tmp / "r.bundle")
            self.assertEqual(nxt["base"], work)
            self.assertNotEqual(nxt["commit"], work)
            same = runner.build_fixture("continue", tmp / "s3.bundle", "unused", None, tmp / "r.bundle")
            self.assertEqual(same["commit"], work)
            root = runner.build_fixture("root", tmp / "s4.bundle", "Library M2 realization", m2, tmp / "r.bundle", work, [".markitect"])
            self.assertEqual(root["realization_tree"], nxt["realization_tree"])
            check = tmp / "check"
            subprocess.run(["git", "clone", "-q", "-b", "main", str(tmp / "s4.bundle"), str(check)], check=True, capture_output=True)
            self.assertEqual(git("rev-list", "--count", "HEAD", cwd=check), "1")
            self.assertEqual(sorted(git("ls-tree", "-r", "--name-only", "HEAD", cwd=check).split()), ["realization/go.mod", "requirements.md"])
            with self.assertRaises(runner.RunError):
                runner.build_fixture("new", tmp / "s1.bundle", "exists", m1)


class ParityTests(unittest.TestCase):
    def test_codex_switches_equal_the_manager_provider_switches(self):
        source = (REPO / "internal/adapters/codexappserver/profile.go").read_text(encoding="utf-8")
        body = source[source.index("func providerArgv()"):]
        body = body[:body.index("\n}\n")]
        literals = [json.loads('"' + s + '"') if not s.startswith("`") else s[1:-1]
                    for s in re.findall(r'"((?:[^"\\]|\\.)*)"|(`[^`]*`)', body) for s in s if s]
        literals = literals[literals.index("app-server") + 1:]
        mcp = next(i for i, s in enumerate(literals) if s.startswith("mcp_servers="))
        self.assertEqual(literals[mcp - 1], "-c")
        del literals[mcp - 1:mcp + 1]
        self.assertEqual(literals, runner.CODEX_SWITCHES)

    def test_session_limits_and_hardening_equal_a_manager_session(self):
        profile = (REPO / "examples/runtime-profile.example.yaml").read_text(encoding="utf-8")
        values = dict(re.findall(r"^\s+(cpus|memoryBytes|pids|tmpBytes|network|effort): (.*)$", profile, re.M))
        self.assertEqual(values["cpus"], runner.SESSION_LIMITS["cpus"])
        self.assertEqual(values["memoryBytes"], runner.SESSION_LIMITS["memory"])
        self.assertEqual(values["pids"], runner.SESSION_LIMITS["pids"])
        self.assertEqual(values["tmpBytes"], runner.SESSION_LIMITS["tmp"])
        self.assertEqual(values["network"], "''")
        self.assertEqual(values["effort"], runner.EFFORT)
        argv = runner.session_argv("mk-cmp-x", IMAGE, "w", "h", "prompt")
        spec = (REPO / "internal/adapters/container/spec.go").read_text(encoding="utf-8")
        for flag in ("--read-only", "--init", "no-new-privileges", "/tmp:rw,exec,nosuid,nodev,mode=1777,size="):
            self.assertIn(flag, spec)
            self.assertTrue(any(flag in a for a in argv), flag)
        self.assertNotIn("--network", argv)  # the engine default network, like the Manager session
        # The aggregate ceiling of the six Manager sessions an Apply can run at once.
        self.assertEqual(argv[argv.index("--cpus") + 1], "6")
        self.assertEqual(argv[argv.index("--memory") + 1], str(6 * 2147483648))
        self.assertEqual(argv[argv.index("--pids-limit") + 1], "1152")
        self.assertEqual(argv[-1], "prompt")
        self.assertIn('approval_policy="never"', argv)
        self.assertIn('sandbox_mode="danger-full-access"', argv)
        thread = "0199a213-81c0-7800-8aa1-bbab2a035a53"
        resumed = runner.session_argv("mk-cmp-x", IMAGE, "w", "h", "prompt", thread)
        codex = resumed[resumed.index(IMAGE) + 1:]
        self.assertEqual(codex[:2], ["exec", "resume"])
        self.assertEqual(codex[-2:], [thread, "prompt"])
        self.assertEqual(codex[2:-2], argv[argv.index(IMAGE) + 2:-1])  # identical switches

    def test_stream_errors_are_recorded_but_only_turn_failures_fail(self):
        with tempfile.TemporaryDirectory() as tmp:
            transcript = Path(tmp) / "t.jsonl"
            transcript.write_text("\n".join(json.dumps(e) for e in (
                {"type": "thread.started", "thread_id": "x"}, {"type": "error", "message": "Reconnecting... 1/5"},
                {"type": "turn.completed", "usage": {"input_tokens": 1}})) + "\nnot json\n", encoding="utf-8")
            summary = runner.summarize(transcript)
            self.assertTrue(summary["turn_completed"])
            self.assertEqual((len(summary["errors"]), summary["failures"], summary["malformed_lines"]), (1, [], 1))

    @unittest.skipUnless(shutil.which("sh") and shutil.which("xargs"), "needs a POSIX shell")
    def test_capture_keeps_sources_an_over_broad_gitignore_would_drop(self):
        with tempfile.TemporaryDirectory() as tmp:
            work = Path(tmp) / "work"
            subprocess.run(["git", "init", "-q", str(work)], check=True)
            for path, data in ((".gitignore", b"library\n*.out\n"), ("realization/go.mod", b"module library\n"),
                               ("realization/cmd/library/main.go", b"package main\n"), ("realization/library", b"\x7fELF"),
                               ("realization/cover.out", b"mode: set\n"), ("realization/README.md", b"usage\n")):
                (work / path).parent.mkdir(parents=True, exist_ok=True)
                (work / path).write_bytes(data)
            # Replace /tmp/ first: on Linux the temporary directory itself lies
            # below /tmp/, so replacing it after /work would rewrite that path too.
            script = runner.CAPTURE.replace("/tmp/", Path(tmp).as_posix() + "/").replace("/work", work.as_posix())
            subprocess.run([shutil.which("sh"), "-c", script.replace("; cat ", "; true ")], check=True, capture_output=True)
            tracked = git("ls-tree", "-r", "--name-only", "HEAD", cwd=work).split()
            self.assertIn("realization/cmd/library/main.go", tracked)  # ignored by "library", kept
            self.assertIn("realization/README.md", tracked)
            self.assertNotIn("realization/library", tracked)  # the build output stays ignored
            self.assertNotIn("realization/cover.out", tracked)

    def test_secret_scan_reads_compressed_git_objects(self):
        with tempfile.TemporaryDirectory() as tmp:
            tmp = Path(tmp)
            repo = tmp / "repo"
            subprocess.run(["git", "init", "-q", str(repo)], check=True)
            (repo / "notes.txt").write_bytes(b"token ACCESS-SECRET-VALUE here\n" * 50)
            git("add", "-A", cwd=repo)
            subprocess.run(["git", "-C", str(repo), "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "x"], check=True)
            out = tmp / "out"
            out.mkdir()
            subprocess.run(["git", "-C", str(repo), "bundle", "create", str(out / "result.bundle"), "HEAD"], check=True, capture_output=True)
            self.assertNotIn(b"ACCESS-SECRET-VALUE", (out / "result.bundle").read_bytes())  # compressed
            self.assertEqual(runner.files_containing(out, [b"ACCESS-SECRET-VALUE"]), (["result.bundle"], []))

    def test_session_login_drops_the_refresh_token(self):
        copy, secrets = runner.session_login(json.dumps(LOGIN).encode())
        self.assertEqual(json.loads(copy)["tokens"]["refresh_token"], "")
        self.assertNotIn(b"REFRESH-SECRET-VALUE", copy)
        self.assertEqual(len(secrets), 3)
        for wrong in ({**LOGIN, "auth_mode": "apikey"}, {**LOGIN, "OPENAI_API_KEY": "sk"}, {**LOGIN, "tokens": {}}):
            with self.assertRaises(runner.RunError):
                runner.session_login(json.dumps(wrong).encode())


class LockTests(unittest.TestCase):
    def test_lock_is_exclusive_and_removed_only_by_its_holder(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "NATIVE-RUN.lock"
            with runner.native_lock(path, "comparison arm B", "run-1"):
                self.assertEqual(json.loads(path.read_text())["run"], "run-1")
                with self.assertRaises(runner.RunError):
                    with runner.native_lock(path, "other", "run-2"):
                        pass
                self.assertTrue(path.exists())
            self.assertFalse(path.exists())


class SessionTests(unittest.TestCase):
    def run_fake(self, tmp, **env):
        engine = tmp / "engine.py"
        engine.write_text(FAKE_ENGINE, encoding="utf-8")
        login = tmp / "auth.json"
        login.write_text(json.dumps(LOGIN), encoding="utf-8")
        fixture = tmp / "fixture.bundle"
        fixture.write_bytes(b"bundle bytes")
        old = dict(os.environ)
        os.environ.update(FAKE_DIR=str(tmp), FAKE_IMAGE=IMAGE, **env)
        try:
            return runner.run_session([sys.executable, "-I", str(engine)], fixture, IMAGE, login, tmp / "out", "mk-cmp-test", 2700 if not env else 1)
        finally:
            os.environ.clear()
            os.environ.update(old)

    def test_completed_session_is_captured_and_cleaned_up(self):
        with tempfile.TemporaryDirectory() as tmp:
            tmp = Path(tmp)
            result = self.run_fake(tmp)
            self.assertEqual(result["status"], "completed", result.get("error"))
            self.assertEqual((result["input_commit"], result["result_commit"]), ("c" * 40, "d" * 40))
            self.assertEqual(result["transcript"]["final_message"], "Done.")
            self.assertEqual(result["transcript"]["usage"], {"input_tokens": 10, "output_tokens": 2})
            self.assertEqual(json.loads((tmp / "installed-login.json").read_text())["tokens"]["refresh_token"], "")
            calls = [json.loads(line) for line in (tmp / "calls.jsonl").read_text().splitlines()]
            self.assertEqual([c[:2] for c in calls if c[0] == "volume" and c[1] == "rm"], [["volume", "rm"]] * 2)
            steps = [s["step"] for s in result["steps"]]
            self.assertLess(steps.index("logout"), steps.index("raw-tar"))
            self.assertEqual(result["secret_in_outputs"], [])
            self.assertEqual(result["secret_scan_unreadable"], ["result.bundle"])  # the fake engine's bundle
            listed = (tmp / "out" / "files.sha256").read_text().splitlines()
            self.assertTrue(any(line.endswith("  result.json") for line in listed))
            self.assertFalse(any(line.endswith("files.sha256") for line in listed))
            for path in (tmp / "out").rglob("*"):
                if path.is_file():
                    self.assertNotIn(b"SECRET-VALUE", path.read_bytes(), path)

    def test_deadline_kills_the_session_and_records_incomplete(self):
        with tempfile.TemporaryDirectory() as tmp:
            tmp = Path(tmp)
            result = self.run_fake(tmp, FAKE_SLEEP="5")
            self.assertTrue(result["session_timed_out"])
            self.assertEqual(result["status"], "incomplete")
            self.assertIn("kill", [s["step"] for s in result["steps"]])


if __name__ == "__main__":
    unittest.main()
