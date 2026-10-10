"""The study command without Docker: study file, schedule, preflight, failure policy, logins."""

import contextlib
import copy
import io
import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from playground import __main__ as entry
from playground import host, manifest, study

PLAYGROUND = Path(__file__).resolve().parents[1]
STUDY = {
    "schema": 1, "id": "pilot", "case": "roombook", "stations": 2,
    "arms": ["conventional", "markitect"], "firstArm": "conventional",
    "agent": {"kind": "fake", "codexVersion": "0.162.0", "model": "gpt-6-luna", "effort": "high", "maxSubagents": 3},
    "limits": {"stationSeconds": 60, "totalSeconds": 300},
    "container": {"cpus": 2, "memory": "2g", "pidsLimit": 512},
    "markitect": {"sourceRepo": "/src/Markitect", "commit": "669cecd2"},
    "reviewers": ["codex", "claude"],
}
IMAGE = "sha256:feed"
FULL = "669cecd2" + "0" * 32


def variant(**changes) -> dict:
    data = copy.deepcopy(STUDY)
    for key, value in changes.items():
        if value is ...:
            data.pop(key, None)
        else:
            data[key] = value
    return data


def capture():
    out, err = io.StringIO(), io.StringIO()
    stack = contextlib.ExitStack()
    stack.enter_context(contextlib.redirect_stdout(out))
    stack.enter_context(contextlib.redirect_stderr(err))
    return stack, out, err


def assessment(run_id: str, method: str, *, refreshed: bool = False, image: str = IMAGE) -> dict:
    return {"schema": 1, "kind": "assessment",
            "run": {"id": run_id, "case": "roombook", "method": method, "outerProvider": "fake",
                    "fairness": {"case": "roombook", "stationsPlanned": 2, "imageId": image,
                                 "hostPlatform": {"system": "Linux", "machine": "x86_64"}}},
            "evaluation": {"files": {"config": {"sha256": "aa"}}},
            "reviewers": {"codex": {"model": "m", "effort": "high", "version": "fake-reviewer 1.0"}},
            "stations": [], "classification": {"class": "none", "reason": "run completed"},
            "totals": {"publicChecks": {"passed": 0, "total": 0}, "holdouts": None, "failedMcpCalls": 0,
                       "reviewers": {"codex": {"byCategory": {}, "loginRefreshed": refreshed}}}}


def bumped(path: Path) -> str:
    login = json.loads(Path(path).read_text(encoding="utf-8"))
    return json.dumps({**login, "generation": login["generation"] + 1})


class FakeDocker:
    """Stands in for subprocess.run/Popen: Docker, plus git and go answering like a checkout.

    A run container writes runner.json into its /out (codexLoginChanged), an assessment
    container writes report.json; both "refresh" a mounted Codex login, which `docker cp`
    then copies out."""

    def __init__(self, *, exits=None, wait_effects=None, start_fails=(), login_changed=True, reviewer_refresh=False,
                 server="29.4.1|linux|amd64", running="", names="", ran_image=None, cp="file", git=None,
                 mismatch=False):
        self.calls, self.containers, self.seen_logins, self.order = [], {}, {}, []
        self.exits, self.wait_effects, self.start_fails = exits or {}, wait_effects or {}, set(start_fails)
        self.login_changed, self.reviewer_refresh, self.server = login_changed, reviewer_refresh, server
        self.running, self.names, self.ran_image, self.cp, self.git = running, names, ran_image, cp, git or {}
        self.mismatch = mismatch
        self.logins = {}  # container -> refreshed login text

    def run(self, cmd, **kwargs):
        self.calls.append(list(cmd))
        code, out, err = 0, "", ""
        if cmd[0] == "git":
            for needle, answer in self.git.items():
                if needle in " ".join(cmd):
                    return subprocess.CompletedProcess(cmd, answer[0], stdout=answer[1], stderr=answer[2])
            return subprocess.CompletedProcess(cmd, 0, stdout="", stderr="")
        if cmd[0] != "docker":
            return subprocess.CompletedProcess(cmd, 0, stdout="go version go1.27.1 linux/amd64", stderr="")
        sub = cmd[1]
        if sub == "version":
            if self.server is None:
                code, err = 1, "Cannot connect to the Docker daemon"
            else:
                out = self.server if "|" in cmd[-1] else self.server.split("|")[0]
        elif cmd[1:3] == ["image", "inspect"]:
            out = IMAGE
        elif cmd[1:3] == ["container", "inspect"]:
            name = cmd[-1]
            if name not in self.containers:
                code, err = 1, f"Error: No such container: {name}"
            elif "{{.Image}}" in cmd:
                out = self.ran_image or IMAGE
            else:
                out = str(self.containers[name] == "running").lower()
        elif sub == "ps":
            out = self.names if "--all" in cmd else self.running
        elif sub == "run":
            name = cmd[cmd.index("--name") + 1]
            if name in self.start_fails:
                return subprocess.CompletedProcess(cmd, 125, stdout="", stderr="conflict")
            self.order.append(name)
            self.containers[name] = "running"
            mounts = {}
            for index, arg in enumerate(cmd):
                if arg == "--mount":
                    fields = dict(field.split("=", 1) for field in cmd[index + 1].split(",") if "=" in field)
                    mounts[fields["target"]] = Path(fields["source"])
            self.behave(name, mounts)
            out = "cid"
        elif sub == "wait":
            name = cmd[-1]
            effect = self.wait_effects.get(name)
            if effect is not None:
                raise effect
            self.containers[name] = "stopped"
            out = f"{self.exits.get(name, 0)}\n"
        elif sub == "kill":
            self.containers[cmd[-1]] = "stopped"
        elif sub == "rm":
            self.containers.pop(cmd[-1], None)
        elif sub == "cp":
            name, _path = cmd[2].split(":", 1)
            target = Path(cmd[3])
            if name not in self.logins:
                code, err = 1, "Could not find the file in container"
            elif self.cp == "symlink":
                target.symlink_to(target.parent / "codex-auth.json")
            elif self.cp == "huge":
                target.write_bytes(b"x" * (study.LOGIN_MAX_BYTES + 1))
            else:
                target.write_text(self.logins[name], encoding="utf-8")
        return subprocess.CompletedProcess(cmd, code, stdout=out, stderr=err)

    def behave(self, name: str, mounts: dict) -> None:
        if name.startswith("mpg-assess-"):
            run_id = name[len("mpg-assess-"):]
            login = mounts.get("/assess/secrets/codex-auth.json")
            method = "markitect" if run_id.endswith("-mkt") else "conventional"
            image = "sha256:other" if self.mismatch and method == "markitect" else IMAGE
            report = assessment(run_id, method, refreshed=self.reviewer_refresh and login is not None, image=image)
            (mounts["/assess/out"] / "report.json").write_text(json.dumps(report), encoding="utf-8")
        else:
            login = mounts.get("/run/secrets/codex-auth.json")
            (mounts["/out"] / "runner.json").write_text(
                json.dumps({"codexLoginChanged": self.login_changed and login is not None}), encoding="utf-8")
        if login is not None:
            self.seen_logins[name] = login.read_text(encoding="utf-8")
            refresh = self.reviewer_refresh if name.startswith("mpg-assess-") else self.login_changed
            self.logins[name] = bumped(login) if refresh else self.seen_logins[name]

    def popen(self, cmd, **kwargs):
        self.calls.append(list(cmd))
        return mock.Mock(wait=lambda timeout=None: 0, poll=lambda: 0, kill=lambda: None)

    def started(self) -> list[str]:
        return list(self.order)


class StudyFileTests(unittest.TestCase):
    def assertRejected(self, data, fragment):
        with self.assertRaises(study.StudyError) as caught:
            study.validate(data)
        self.assertIn(fragment, str(caught.exception))

    def test_valid_study_is_normalized(self):
        result = study.validate(variant(arms=["markitect", "conventional"], container={"cpus": 2}))
        self.assertEqual(result["arms"], ["conventional", "markitect"])
        self.assertEqual(result["pairs"], 1)
        self.assertEqual(result["container"], {"cpus": 2, "memory": "8g", "pidsLimit": 2048})
        self.assertEqual(result["agent"]["claudeVersion"], manifest.DEFAULT_CLAUDE_VERSION)
        self.assertEqual(result["markitect"], STUDY["markitect"])
        self.assertEqual(result["stations"], 2)
        self.assertEqual(study.validate(variant(stations=...))["stations"], 4)  # the case's count
        self.assertEqual(study.validate(result), result)

    def test_study_rules(self):
        self.assertRejected(variant(extra=1), "unknown field(s) extra")
        self.assertRejected(variant(container=...), "missing container")
        self.assertRejected(variant(schema=2), "schema")
        self.assertRejected(variant(arms=[]), "arms")
        self.assertRejected(variant(arms=["conventional", "conventional"]), "arms")
        self.assertRejected(variant(arms=["vibes"]), "arms")
        self.assertRejected(variant(firstArm="markitect", arms=["conventional"], markitect=...), "firstArm")
        self.assertRejected(variant(pairs=0), "pairs")
        self.assertRejected(variant(id="x" * 56), "too long for its run ids")
        self.assertEqual(study.validate(variant(id="x" * 55))["id"], "x" * 55)  # x...-p1-conv has 63
        self.assertRejected(variant(reviewers=["codex", "gemini"]), "reviewers")
        self.assertRejected(variant(reviewers="codex"), "reviewers")
        self.assertRejected(variant(markitect=...), "markitect: required")
        self.assertRejected(variant(arms=["conventional"]), "markitect: only allowed")

    def test_shared_parts_use_the_manifest_validators(self):
        self.assertRejected(variant(case="nothing"), "case")
        self.assertRejected(variant(stations=5), "case roombook has 4 stations")
        self.assertRejected(variant(agent={**STUDY["agent"], "model": "bad model"}), "agent.model")
        self.assertRejected(variant(limits={"stationSeconds": 0, "totalSeconds": 1}), "limits.stationSeconds")
        self.assertRejected(variant(markitect={"commit": "HEAD", "sourceRepo": "/s"}), "markitect.commit")

    def test_load_resolves_a_relative_source_repo_from_the_study_folder(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "studies" / "s.json"
            path.parent.mkdir()
            path.write_text(json.dumps(variant(markitect={"sourceRepo": "../Markitect", "commit": "669cecd2"})),
                            encoding="utf-8")
            self.assertEqual(study.load(path)["markitect"]["sourceRepo"], str(path.parent / ".." / "Markitect"))
            path.write_text("{", encoding="utf-8")
            with self.assertRaisesRegex(study.StudyError, "not valid JSON"):
                study.load(path)

    def test_shipped_study_examples_are_valid(self):
        with mock.patch.object(manifest, "default_source_repo", return_value="/checkout"):
            for path in sorted((PLAYGROUND / "examples").glob("study-*.json")):
                with self.subTest(example=path.name):
                    loaded = study.load(path)
                    self.assertEqual(loaded["arms"], ["conventional", "markitect"])
                    self.assertNotIn("sourceRepo", json.loads(path.read_text(encoding="utf-8"))["markitect"])
            fake = study.load(PLAYGROUND / "examples" / "study-fake-roombook.json")
        self.assertEqual((fake["agent"]["kind"], fake["stations"], fake["case"]), ("fake", 2, "roombook"))


class ScheduleTests(unittest.TestCase):
    def test_pairs_alternate_which_arm_goes_first(self):
        runs = study.expand(study.validate(variant(pairs=3)))
        self.assertEqual([(r["id"], r["pair"], r["arm"], r["position"]) for r in runs], [
            ("pilot-p1-conv", 1, "conventional", 1), ("pilot-p1-mkt", 1, "markitect", 2),
            ("pilot-p2-mkt", 2, "markitect", 1), ("pilot-p2-conv", 2, "conventional", 2),
            ("pilot-p3-conv", 3, "conventional", 1), ("pilot-p3-mkt", 3, "markitect", 2)])
        runs = study.expand(study.validate(variant(firstArm="markitect", pairs=2)))
        self.assertEqual([r["id"] for r in runs], ["pilot-p1-mkt", "pilot-p1-conv", "pilot-p2-conv", "pilot-p2-mkt"])

    def test_every_run_manifest_is_a_valid_normalized_manifest(self):
        for entry_ in study.expand(study.validate(variant(pairs=2))):
            normalized = entry_["manifest"]
            self.assertEqual(manifest.validate(normalized), normalized)
            self.assertEqual(normalized["method"], entry_["arm"])
            self.assertEqual("markitect" in normalized, entry_["arm"] == "markitect")
            self.assertEqual(normalized["stations"], 2)

    def test_one_arm_repeats(self):
        runs = study.expand(study.validate(variant(arms=["conventional"], markitect=..., pairs=2)))
        self.assertEqual([r["id"] for r in runs], ["pilot-p1-conv", "pilot-p2-conv"])


class LoginTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name).resolve()
        self.source = self.root / "home" / ".codex" / "auth.json"
        self.source.parent.mkdir(parents=True)
        self.source.write_text('{"generation": 0}', encoding="utf-8")
        self.token = self.root / "claude-token"
        self.token.write_text("token-value\n", encoding="utf-8")

    def logins(self) -> study.Logins:
        logins = study.Logins("pilot", codex=self.source, claude=self.token, home=self.root / "state")
        self.addCleanup(logins.close)
        return logins

    @unittest.skipUnless(os.name == "posix", "file modes")
    def test_private_folders_and_copies(self):
        logins = self.logins()
        self.assertEqual(logins.root.parent, self.root / "state" / "logins")
        self.assertTrue(logins.root.name.startswith("pilot-"))
        folder, auth, token = logins.step("pilot-p1-conv", codex=True, claude=True)
        for path in (logins.root.parent, logins.root, logins.current, folder):
            self.assertEqual(path.stat().st_mode & 0o777, 0o700, path)
        for path in (logins.codex, logins.claude, auth, token):
            self.assertEqual(path.stat().st_mode & 0o777, 0o600, path)
        self.assertEqual(auth.read_bytes(), self.source.read_bytes())
        _folder, auth, token = logins.step("assess-pilot-p1-conv", codex=False, claude=True)
        self.assertIsNone(auth)
        self.assertIsNotNone(token)

    def test_promotion_only_for_a_reported_change_and_a_small_regular_file(self):
        logins = self.logins()
        folder, _auth, _token = logins.step("one", codex=True, claude=False)
        returned = folder / study.RETURNED
        returned.write_text('{"generation": 1}', encoding="utf-8")
        self.assertEqual(logins.promote(folder, False), "unchanged")
        self.assertIn("not reported", logins.promote(folder, None))
        self.assertEqual(logins.codex.read_text(encoding="utf-8"), '{"generation": 0}')
        for content in (b"", b"x" * (study.LOGIN_MAX_BYTES + 1)):
            returned.write_bytes(content)
            self.assertIn("not a regular file of 1 B to 64 KiB", logins.promote(folder, True))
        if os.name == "posix":
            returned.unlink()
            returned.symlink_to(self.token)
            self.assertIn("not a regular file", logins.promote(folder, True))
            returned.unlink()
        else:
            returned.unlink()
        self.assertIn("not copied out", logins.promote(folder, True))
        self.assertEqual(logins.codex.read_text(encoding="utf-8"), '{"generation": 0}')
        returned.write_text('{"generation": 1}', encoding="utf-8")
        self.assertEqual(logins.promote(folder, True), "promoted")
        self.assertEqual(logins.codex.read_text(encoding="utf-8"), '{"generation": 1}')
        self.assertEqual(logins.promotions, 1)
        logins.finish_step(folder)
        self.assertFalse(folder.exists())
        _f, auth, _t = logins.step("two", codex=True, claude=False)  # the next step starts from the newest copy
        self.assertEqual(auth.read_text(encoding="utf-8"), '{"generation": 1}')

    def test_the_source_is_written_only_on_request_and_only_if_unchanged(self):
        logins = self.logins()
        self.assertIn("not needed", logins.update_source())
        folder, _a, _t = logins.step("one", codex=True, claude=False)
        (folder / study.RETURNED).write_text('{"generation": 1}', encoding="utf-8")
        logins.promote(folder, True)
        self.assertEqual(self.source.read_text(encoding="utf-8"), '{"generation": 0}')  # nothing written yet
        self.assertEqual(logins.update_source(), "updated")
        self.assertEqual(self.source.read_text(encoding="utf-8"), '{"generation": 1}')
        if os.name == "posix":
            self.assertEqual(self.source.stat().st_mode & 0o777, 0o600)
        self.assertEqual(sorted(p.name for p in self.source.parent.iterdir()), ["auth.json"])  # no temp file left

    def test_a_source_changed_since_the_copy_is_never_replaced(self):
        logins = self.logins()
        folder, _a, _t = logins.step("one", codex=True, claude=False)
        (folder / study.RETURNED).write_text('{"generation": 1}', encoding="utf-8")
        logins.promote(folder, True)
        self.source.write_text('{"generation": 7, "fresh": "codex login"}', encoding="utf-8")
        message = logins.update_source()
        self.assertIn("changed since the study copied it", message)
        self.assertIn("codex login", message)
        self.assertEqual(self.source.read_text(encoding="utf-8"), '{"generation": 7, "fresh": "codex login"}')

    def test_close_removes_every_copy(self):
        logins = self.logins()
        logins.step("one", codex=True, claude=True)
        logins.close()
        self.assertFalse(logins.root.exists())
        self.assertTrue(logins.record["removed"])
        self.assertEqual(study.leftover_login_folders(self.root / "state"), [])

    def test_login_needs_name_what_needs_each_login(self):
        def needs(data, fake_reviewers=False, codex=False, claude=False):
            parsed = study.validate(data)
            return study.login_needs(parsed, study.expand(parsed), fake_reviewers=fake_reviewers, codex_given=codex,
                                     claude_given=claude)
        self.assertEqual(needs(STUDY, fake_reviewers=True), {"codex": [], "claude": []})  # fakes need nothing
        self.assertEqual(len(needs(STUDY, fake_reviewers=True, codex=True)["codex"]), 2)  # given: handed over
        codex = needs(variant(agent={**STUDY["agent"], "kind": "codex"}))
        self.assertEqual(codex["codex"], ["the conventional arm (agent kind codex)",
                                          "the markitect arm (agent kind codex, Markitect's inner roles)",
                                          "the codex reviewer"])
        self.assertEqual(codex["claude"], ["the claude reviewer"])
        claude = needs(variant(agent={**STUDY["agent"], "kind": "claude", "claudeVersion": "2.1.296"},
                               markitect={**STUDY["markitect"], "innerModel": "gpt-6-luna", "innerEffort": "high"}),
                       fake_reviewers=True)
        self.assertEqual(claude["codex"], ["the markitect arm (agent kind claude, Markitect's inner roles)"])
        self.assertEqual(len(claude["claude"]), 2)


class StudyRunBase(unittest.TestCase):
    """A whole study against FakeDocker, with the real playground and a temporary home."""

    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name).resolve()
        self.state = self.root / "state"
        self.source = self.root / "home" / ".codex" / "auth.json"
        self.source.parent.mkdir(parents=True)
        self.marker = "SECRET-LOGIN-MARKER-4b1d"
        self.source.write_text(json.dumps({"generation": 0, "secret": self.marker}), encoding="utf-8")
        self.token = self.root / "home" / "claude-token"
        self.token.write_text("CLAUDE-TOKEN-MARKER-77\n", encoding="utf-8")
        (self.root / "src").mkdir()
        self.out = self.root / "runs" / "study"
        for name, value in (("STATE_HOME", self.state), ("CODEX_SOURCE", self.source), ("CLAUDE_SOURCE", self.token)):
            patcher = mock.patch.object(study, name, value)
            patcher.start()
            self.addCleanup(patcher.stop)

    def study_data(self, **changes) -> dict:
        changes.setdefault("markitect", {"sourceRepo": str(self.root / "src"), "commit": "669cecd2"})
        return variant(**changes)

    def write_study(self, **changes) -> Path:
        data = self.study_data(**changes)
        path = self.root / "study.json"
        path.write_text(json.dumps(data), encoding="utf-8")
        return path

    def run_study(self, docker: FakeDocker, *extra: str, path: Path | None = None, out: Path | None = None):
        argv = ["study", str(path or self.write_study()), "--out", str(out or self.out), *extra]
        built = {"commit": FULL, "sha256": "f" * 64, "go": "go1.27.1"}

        def build_binary(product, target):
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(b"binary")
            return built

        stack, stdout, stderr = capture()
        with stack, mock.patch.object(host.subprocess, "run", docker.run), \
                mock.patch.object(host.subprocess, "Popen", docker.popen), \
                mock.patch.object(study.shutil, "which", return_value="/usr/bin/tool"), \
                mock.patch.object(study.shutil, "disk_usage", return_value=mock.Mock(free=100 << 30)), \
                mock.patch.object(host, "resolve_markitect",
                                  side_effect=lambda p: {**p, "sourceRepo": str(Path(p["sourceRepo"]).resolve()),
                                                         "commit": FULL}), \
                mock.patch.object(host, "build_markitect", side_effect=build_binary):
            code = entry.main(argv)
        return code, stdout.getvalue(), stderr.getvalue()

    def record(self) -> dict:
        return json.loads((self.out / "study.json").read_text(encoding="utf-8"))

    def steps(self, kind: str) -> list[dict]:
        return [step for step in self.record()["steps"] if step["step"] == kind]

    def everything(self, root: Path) -> str:
        return "".join(path.read_text(encoding="utf-8", errors="replace") for path in root.rglob("*")
                       if path.is_file() and not path.is_symlink())


class StudyRunTests(StudyRunBase):
    def test_a_whole_study_in_schedule_order(self):
        docker = FakeDocker()
        code, stdout, stderr = self.run_study(docker, "--codex-auth", str(self.source), "--fake-reviewers",
                                              path=self.write_study(pairs=2))
        self.assertEqual(code, 0, stdout + stderr)
        self.assertEqual(docker.started(), [
            "mpg-pilot-p1-conv", "mpg-pilot-p1-mkt", "mpg-pilot-p2-mkt", "mpg-pilot-p2-conv",
            "mpg-assess-pilot-p1-conv", "mpg-assess-pilot-p1-mkt", "mpg-assess-pilot-p2-mkt",
            "mpg-assess-pilot-p2-conv"])
        self.assertEqual(sum(call[:2] == ["docker", "build"] for call in docker.calls), 1)  # built once
        record = self.record()
        self.assertEqual((record["status"], record["exitCode"]), ("completed", 0))
        self.assertEqual([s["step"] for s in record["steps"]], ["run"] * 4 + ["assess"] * 4 + ["compare"] * 2)
        self.assertTrue(all(s["status"] in ("completed", "written") and s["exitCode"] == 0 for s in record["steps"]))
        self.assertEqual([(s["a"], s["b"]) for s in self.steps("compare")],
                         [("pilot-p1-conv", "pilot-p1-mkt"), ("pilot-p2-conv", "pilot-p2-mkt")])
        for pair in (1, 2):
            self.assertIn("Fairness fields match", (self.out / "comparisons" / f"p{pair}.md").read_text(encoding="utf-8"))
        for name in ("study.md", "study-file.json", "preflight/preflight.txt", "preflight/bin/markitect"):
            self.assertTrue((self.out / name).is_file(), name)
        self.assertEqual(json.loads((self.out / "study-file.json").read_text(encoding="utf-8"))["pairs"], 2)
        for step in self.steps("run"):
            saved = json.loads((self.out / "manifests" / f"{step['id']}.json").read_text(encoding="utf-8"))
            self.assertEqual(manifest.validate(saved), saved)
            host_record = json.loads((self.out / step["folder"] / "host.json").read_text(encoding="utf-8"))
            self.assertEqual(host_record["manifest"], saved)
            self.assertEqual(host_record["image"], {"tag": host.image_tag(saved), "id": IMAGE})
            self.assertEqual(step["imageId"], IMAGE)
            if step["arm"] == "markitect":
                self.assertEqual(saved["markitect"]["commit"], FULL)
                self.assertEqual(host_record["markitect"]["sha256"], "f" * 64)
                self.assertEqual((self.out / step["folder"] / "inputs" / "bin" / "markitect").read_bytes(), b"binary")
        versions = record["versions"]
        self.assertEqual(versions["image"], {"tag": "markitect-playground:codex-0.162.0-claude-2.1.296", "id": IMAGE})
        self.assertEqual(versions["docker"], {"version": "29.4.1", "os": "linux", "arch": "amd64"})
        self.assertEqual(versions["markitect"]["sha256"], "f" * 64)
        self.assertEqual((versions["go"], versions["hostPlatform"]), ("go1.27.1", host.host_platform()))
        self.assertIn("dirty", versions["playground"])
        self.assertEqual(record["preflight"]["status"], "passed")
        self.assertIn("image build", [c["check"] for c in record["preflight"]["checks"]])
        self.assertTrue(all(name not in docker.containers for name in docker.started()))  # every container removed
        self.assertFalse((self.state / "study.lock").exists())

    def test_the_login_is_handed_over_from_run_to_run_and_never_written_back(self):
        docker = FakeDocker()
        before = self.source.stat()
        code, stdout, stderr = self.run_study(docker, "--codex-auth", str(self.source), "--fake-reviewers")
        self.assertEqual(code, 0, stdout + stderr)
        generations = [json.loads(docker.seen_logins[name])["generation"] for name in docker.started()
                       if not name.startswith("mpg-assess-")]
        self.assertEqual(generations, [0, 1])  # the second run got the first run's refreshed login
        self.assertEqual([s["login"]["promotion"] for s in self.steps("run")], ["promoted", "promoted"])
        self.assertEqual([s["login"]["copyOut"] for s in self.steps("run")], ["copied", "copied"])
        self.assertEqual(self.source.read_text(encoding="utf-8"), json.dumps({"generation": 0, "secret": self.marker}))
        self.assertEqual((self.source.stat().st_mtime_ns, self.source.stat().st_size),
                         (before.st_mtime_ns, before.st_size))
        record = self.record()
        copies = record["logins"]["copies"]
        self.assertIn("not written", copies["sourceUpdate"])
        self.assertIn("codex login", stderr)
        self.assertTrue(copies["removed"])
        self.assertFalse(Path(copies["folder"]).exists())
        self.assertEqual(record["logins"]["codex"], str(self.source))
        self.assertNotIn(self.marker, self.everything(self.out) + stdout + stderr)  # never read, printed or hashed
        cp = [call for call in docker.calls if call[:2] == ["docker", "cp"]]
        self.assertEqual([call[2] for call in cp], [f"mpg-pilot-p1-conv:{study.AGENT_LOGIN}",
                                                    f"mpg-pilot-p1-mkt:{study.AGENT_LOGIN}"])
        for call in cp:  # copied out before the container was removed
            self.assertLess(docker.calls.index(call), docker.calls.index(["docker", "rm", "-f", call[2].split(":")[0]]))

    def test_update_login_writes_the_newest_copy_back_when_the_source_is_unchanged(self):
        code, stdout, stderr = self.run_study(FakeDocker(), "--codex-auth", str(self.source), "--fake-reviewers",
                                              "--update-login")
        self.assertEqual(code, 0, stdout + stderr)
        self.assertEqual(json.loads(self.source.read_text(encoding="utf-8")), {"generation": 2, "secret": self.marker})
        self.assertEqual(self.record()["logins"]["copies"]["sourceUpdate"], "updated")

    def test_update_login_leaves_a_source_that_changed_during_the_study(self):
        docker = FakeDocker()
        original = docker.behave

        def relogin(name, mounts):  # the user ran `codex login` while the study ran
            original(name, mounts)
            if name == "mpg-pilot-p1-mkt":
                self.source.write_text('{"generation": 50, "fresh": true}', encoding="utf-8")
        docker.behave = relogin
        code, stdout, stderr = self.run_study(docker, "--codex-auth", str(self.source), "--fake-reviewers",
                                              "--update-login")
        self.assertEqual(code, 0, stdout + stderr)
        self.assertEqual(self.source.read_text(encoding="utf-8"), '{"generation": 50, "fresh": true}')
        self.assertIn("run `codex login`", self.record()["logins"]["copies"]["sourceUpdate"])

    def test_reviewer_logins_are_handed_over_too_and_tokens_only_copied(self):
        docker = FakeDocker(reviewer_refresh=True)
        path = self.write_study(agent={**STUDY["agent"], "kind": "codex"})
        code, stdout, stderr = self.run_study(docker, path=path)  # default logins: ~/.codex, the token file
        self.assertEqual(code, 0, stdout + stderr)
        seen = [json.loads(docker.seen_logins[name])["generation"] for name in docker.started()]
        self.assertEqual(seen, [0, 1, 2, 3])  # runs, then both assessments, each from the newest copy
        assess = self.steps("assess")
        self.assertEqual([s["login"]["copyOut"] for s in assess], ["copied", "copied"])
        cp = [call[2] for call in docker.calls if call[:2] == ["docker", "cp"]]
        self.assertIn(f"mpg-assess-pilot-p1-conv:{study.evaluate.REVIEWER_LOGIN}", cp)
        run = next(c for c in docker.calls if c[:2] == ["docker", "run"] and "mpg-assess-pilot-p1-conv" in c)
        mounts = [run[i + 1] for i, arg in enumerate(run) if arg == "--mount"]
        self.assertTrue(any(m.endswith("target=/assess/secrets/claude-token,readonly") for m in mounts))
        self.assertTrue(any(m.endswith("target=/assess/secrets/codex-auth.json,readonly") for m in mounts))
        self.assertFalse(any(str(self.source) in m or str(self.token) in m for m in mounts))  # copies, never sources
        everything = self.everything(self.out) + stdout + stderr
        self.assertNotIn(self.marker, everything)
        self.assertNotIn("CLAUDE-TOKEN-MARKER-77", everything)
        self.assertEqual(self.token.read_text(encoding="utf-8"), "CLAUDE-TOKEN-MARKER-77\n")

    def test_a_failed_run_does_not_stop_the_study(self):
        docker = FakeDocker(exits={"mpg-pilot-p1-conv": 1})
        code, stdout, stderr = self.run_study(docker, "--fake-reviewers")
        self.assertEqual(code, 1, stdout + stderr)
        self.assertEqual(len(docker.started()), 4)  # both runs and both assessments
        self.assertEqual([s["exitCode"] for s in self.steps("run")], [1, 0])
        self.assertEqual(self.steps("compare")[0]["status"], "written")
        self.assertEqual(self.record()["status"], "failed")
        self.assertIsNone(self.record()["stopReason"])

    def test_the_study_stops_when_the_host_cannot_run_a_container(self):
        docker = FakeDocker(start_fails={"mpg-pilot-p1-conv"})
        code, stdout, stderr = self.run_study(docker, "--codex-auth", str(self.source), "--fake-reviewers")
        self.assertEqual(code, 1, stdout + stderr)
        self.assertEqual(docker.started(), [])
        record = self.record()
        self.assertEqual(record["status"], "stopped")
        self.assertIn("start-failed", record["stopReason"])
        self.assertEqual([s["status"] for s in record["steps"]],
                         ["start-failed", "not-run", "not-run", "not-run", "not-run"])
        self.assertFalse(Path(record["logins"]["copies"]["folder"]).exists())

    def test_a_host_timeout_stops_the_study_with_124(self):
        docker = FakeDocker(wait_effects={"mpg-pilot-p1-conv": subprocess.TimeoutExpired(["docker", "wait"], 1)})
        code, _stdout, _stderr = self.run_study(docker, "--fake-reviewers")
        self.assertEqual(code, 124)
        self.assertEqual(docker.started(), ["mpg-pilot-p1-conv"])
        self.assertEqual((self.record()["status"], self.steps("run")[0]["status"]), ("timeout", "host-timeout"))
        self.assertIn(["docker", "kill", "mpg-pilot-p1-conv"], docker.calls)

    def test_an_interrupt_stops_the_study_and_removes_every_login_copy(self):
        docker = FakeDocker(wait_effects={"mpg-pilot-p1-mkt": KeyboardInterrupt()})
        code, _stdout, _stderr = self.run_study(docker, "--codex-auth", str(self.source), "--fake-reviewers")
        self.assertEqual(code, 130)
        record = self.record()
        self.assertEqual(record["status"], "interrupted")
        self.assertFalse(Path(record["logins"]["copies"]["folder"]).exists())
        self.assertEqual(list((self.state / "logins").iterdir()), [])
        self.assertFalse((self.state / "study.lock").exists())
        # also between steps, outside any container
        self.out = self.root / "runs" / "second"
        with mock.patch.object(study.Study, "compare_pairs", side_effect=KeyboardInterrupt):
            code, _stdout, _stderr = self.run_study(FakeDocker(), "--codex-auth", str(self.source),
                                                    "--fake-reviewers")
        self.assertEqual(code, 130)
        self.assertEqual(self.record()["status"], "interrupted")
        self.assertEqual(list((self.state / "logins").iterdir()), [])
        self.assertFalse((self.state / "study.lock").exists())

    def test_the_study_stops_when_the_image_changed(self):
        docker = FakeDocker(ran_image="sha256:rebuilt")
        code, _stdout, _stderr = self.run_study(docker, "--fake-reviewers")
        self.assertEqual(code, 1)
        self.assertEqual(docker.started(), ["mpg-pilot-p1-conv"])
        self.assertIn("not the preflight build", self.record()["stopReason"])

    def test_a_fairness_mismatch_fails_the_comparison(self):
        code, _stdout, _stderr = self.run_study(FakeDocker(mismatch=True), "--fake-reviewers")
        self.assertEqual(code, 1)
        step = self.steps("compare")[0]
        self.assertEqual(step["status"], "fairness-mismatch")
        self.assertIn("fairness.imageId", [key for key, _a, _b in step["mismatches"]])
        self.assertFalse((self.out / "comparisons").exists())

    def test_a_copied_out_login_that_is_not_a_small_file_is_never_promoted(self):
        for mode in ("huge", "symlink") if os.name == "posix" else ("huge",):
            with self.subTest(mode=mode):
                self.out = self.root / "runs" / mode
                docker = FakeDocker(cp=mode)
                code, stdout, stderr = self.run_study(docker, "--codex-auth", str(self.source), "--fake-reviewers")
                self.assertEqual(code, 0, stdout + stderr)
                self.assertTrue(all("not a regular file" in s["login"]["promotion"] for s in self.steps("run")))
                generations = [json.loads(docker.seen_logins[name])["generation"] for name in docker.started()
                               if not name.startswith("mpg-assess-")]
                self.assertEqual(generations, [0, 0])

    def test_one_arm_runs_without_comparisons(self):
        code, stdout, stderr = self.run_study(FakeDocker(), "--fake-reviewers",
                                              path=self.write_study(arms=["conventional"], markitect=..., pairs=2))
        self.assertEqual(code, 0, stdout + stderr)
        self.assertEqual(self.steps("compare"), [])
        self.assertEqual([s["id"] for s in self.steps("run")], ["pilot-p1-conv", "pilot-p2-conv"])

    def test_preflight_only_builds_and_writes_nothing(self):
        docker = FakeDocker()
        code, stdout, _stderr = self.run_study(docker, "--preflight", "--fake-reviewers")
        self.assertEqual(code, 0)
        self.assertIn("preflight passed", stdout)
        self.assertFalse(self.out.exists())
        self.assertFalse(any(call[:2] in (["docker", "build"], ["docker", "run"]) for call in docker.calls))

    def test_invalid_study_file_exits_2(self):
        path = self.root / "bad.json"
        path.write_text(json.dumps(variant(arms=["vibes"])), encoding="utf-8")
        code, _stdout, stderr = self.run_study(FakeDocker(), path=path)
        self.assertEqual(code, 2)
        self.assertIn("invalid study file", stderr)


class PreflightTests(StudyRunBase):
    def test_every_problem_is_printed_at_once_and_exits_3(self):
        self.source.unlink()  # missing login
        self.token.write_text("", encoding="utf-8")  # empty token
        (self.root / "checkout" / ".git").mkdir(parents=True)
        out = self.root / "checkout" / "study"  # inside a Git checkout
        self.state.mkdir()
        (self.state / "study.lock").write_text(json.dumps({"study": "other", "pid": 1}), encoding="utf-8")
        (self.state / "logins" / "old-study-x").mkdir(parents=True)
        docker = FakeDocker(server="29.4.1|linux|arm64", running="mpg-other-run\n", names="mpg-pilot-p1-mkt\n",
                            git={"rev-list --count": (0, "3", ""), "status --porcelain": (0, " M evaluation/x", "")})
        path = self.write_study(agent={**STUDY["agent"], "kind": "codex"})
        with mock.patch.object(study, "MIN_PYTHON", (99, 0)), \
                mock.patch.object(host, "resolve_markitect", side_effect=host.HostError("bad object")):
            code, stdout, stderr = self.run_study_raw(docker, path, out)
        self.assertEqual(code, 3, stdout + stderr)
        for message in ("Python 99.0 or newer is needed",
                        "the Markitect arm needs a linux/amd64 engine",
                        "playground containers are running (mpg-other-run); run one study at a time",
                        "containers with this study's names exist (mpg-pilot-p1-mkt); remove them",
                        "another study holds", "run one study at a time",
                        "markitect.commit 669cecd2 is not a commit in", "fetch it",
                        "Codex login missing:", "run `codex login` or pass --codex-auth PATH",
                        "the codex reviewer", "the markitect arm (agent kind codex, Markitect's inner roles)",
                        "Claude Code token is empty:", "claude setup-token",
                        "is inside a Git checkout",
                        "[warn] evaluation files", "[warn] login folders", "old-study-x"):
            self.assertIn(message, stdout)
        self.assertIn("preflight failed: 9 problem(s)", stderr)
        self.assertFalse(out.exists())
        self.assertFalse(any(call[:2] in (["docker", "build"], ["docker", "run"]) for call in docker.calls))
        self.assertTrue((self.state / "study.lock").exists())  # another study's lock is never touched

    def run_study_raw(self, docker, path, out):
        stack, stdout, stderr = capture()
        with stack, mock.patch.object(host.subprocess, "run", docker.run), \
                mock.patch.object(host.subprocess, "Popen", docker.popen), \
                mock.patch.object(study.shutil, "which", side_effect=self.which), \
                mock.patch.object(study.shutil, "disk_usage", return_value=mock.Mock(free=self.free)):
            code = entry.main(["study", str(path), "--out", str(out)])
        return code, stdout.getvalue(), stderr.getvalue()

    free = 100 << 30
    missing_tools: tuple = ()

    def which(self, name):
        return None if name in self.missing_tools else f"/usr/bin/{name}"

    def checks(self, docker, **study_changes) -> dict:
        parsed = study.validate(self.study_data(**study_changes))
        runs = study.expand(parsed)
        needs = study.login_needs(parsed, runs, fake_reviewers=False, codex_given=False, claude_given=False)
        with mock.patch.object(host.subprocess, "run", docker.run), \
                mock.patch.object(study.shutil, "which", side_effect=self.which), \
                mock.patch.object(study.shutil, "disk_usage", return_value=mock.Mock(free=self.free)), \
                mock.patch.object(host, "resolve_markitect", side_effect=lambda p: {**p, "commit": FULL}):
            checks, _facts = study.preflight(parsed, runs, self.out, codex_source=self.source,
                                             claude_source=self.token, needs=needs, codex_flag="", claude_flag="")
        return {check["check"]: (check["status"], check["message"]) for check in checks}

    def test_docker_missing_or_unreachable_or_not_linux(self):
        self.missing_tools = ("docker",)
        checks = self.checks(FakeDocker())
        self.assertEqual(checks["docker"][0], "fail")
        self.assertIn("docker is not on PATH; install Docker", checks["docker"][1])
        self.assertEqual(checks["docker daemon"], ("skip", "needs docker"))
        self.missing_tools = ()
        checks = self.checks(FakeDocker(server=None))
        self.assertIn("the Docker daemon is not reachable (Cannot connect", checks["docker daemon"][1])
        self.assertIn("start Docker", checks["docker daemon"][1])
        checks = self.checks(FakeDocker(server="29.4.1|windows|amd64"))
        self.assertIn("switch Docker to Linux containers", checks["linux engine"][1])
        checks = self.checks(FakeDocker(server="29.4.1|linux|arm64"), arms=["conventional"], markitect=...)
        self.assertEqual(checks["linux engine"][0], "ok")  # arm64 is fine without the Markitect arm

    def test_markitect_tools_checkout_and_commit(self):
        self.missing_tools = ("git", "go")
        checks = self.checks(FakeDocker())
        self.assertIn("git is not on PATH; install Git", checks["git"][1])
        self.assertIn("go is not on PATH; install Go", checks["go"][1])
        self.assertEqual(checks["markitect checkout"][0], "skip")
        self.missing_tools = ()
        checks = self.checks(FakeDocker(git={"rev-parse --git-dir": (128, "", "fatal: not a git repository")}))
        self.assertIn("is not a Git checkout (fatal: not a git repository); set it to a Markitect checkout",
                      checks["markitect checkout"][1])
        checks = self.checks(FakeDocker(git={"rev-list --count": (0, "12", "")}))
        self.assertEqual(checks["markitect commit"][0], "ok")
        self.assertEqual(checks["markitect commit age"][0], "warn")
        self.assertIn("is 12 commit(s) behind HEAD", checks["markitect commit age"][1])
        self.assertEqual(self.checks(FakeDocker(), arms=["conventional"], markitect=...).get("git"), None)

    def test_logins_output_folder_disk_and_warnings(self):
        self.source.write_text("", encoding="utf-8")
        self.token.unlink()
        checks = self.checks(FakeDocker(), agent={**STUDY["agent"], "kind": "codex"})
        self.assertIn("Codex login is empty:", checks["codex login"][1])
        self.assertIn("Claude Code token missing:", checks["claude login"][1])
        self.assertIn("(needed by the claude reviewer)", checks["claude login"][1])
        self.out.mkdir(parents=True)
        self.free = 1 << 30
        with mock.patch.object(host.platform, "system", return_value="Windows"):
            checks = self.checks(FakeDocker())
        self.assertIn("output folder exists", checks["output folder"][1])
        self.assertIn("only 1.0 GiB free", checks["free disk"][1])
        self.assertEqual(checks["host platform"][0], "warn")
        self.assertIn("DEC-013", checks["host platform"][1])


if __name__ == "__main__":
    unittest.main()
