from __future__ import annotations

import contextlib
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import tempfile
import unittest
from unittest.mock import patch

from playground import lifecycle

PLAYGROUND = Path(__file__).resolve().parents[1]


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def make_seed(root: Path) -> Path:
    seed = root / "cases"
    common = seed / "common"
    (common / "checks").mkdir(parents=True)
    (common / "AGENTS.md").write_bytes(b"Use normal project workflow.\n")
    (common / "QUALITY.md").write_bytes(b"Assess after freeze.\n")
    (common / "checks" / "acceptance.py").write_bytes(b"# public fixed checks\n")
    (common / "__pycache__").mkdir()
    (common / "__pycache__" / "junk.pyc").write_bytes(b"\0")
    (seed / "task-prompt.txt").write_bytes(b"Do the backlog.\n")
    for case, prefix in (("roombook", "R"), ("readinglog", "B")):
        folder = seed / case
        folder.mkdir(parents=True)
        (folder / "README.md").write_bytes(f"# {case}\n".encode())
        (folder / "BACKLOG.md").write_bytes(b"Implement the public backlog.\n")
        ids = [f"{prefix}{i:02d}" for i in range(1, 13)]
        stations = [ids[:1], ids[1:4], ids[4:11], ids[11:]]
        value = {"schema": 1, "case": case, "totalItems": 12,
                 "stations": [{"id": f"S{i}", "items": wave} for i, wave in enumerate(stations, 1)]}
        (folder / "STATIONS.json").write_text(json.dumps(value), encoding="utf-8", newline="\n")
        if case == "readinglog":
            (folder / "app.py").write_bytes(b"# baseline\n")
    return seed


def declaration(repo: Path) -> dict:
    return json.loads((repo / ".study" / "station.json").read_text(encoding="utf-8"))


class LifecycleTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.seed = make_seed(self.root)
        self.repo = self.root / "work" / "readinglog"
        self.audit = self.root / "out" / "audit"

    def prepare(self, case="readinglog"):
        return lifecycle.prepare(self.seed, self.repo, self.audit, case=case, method="conventional")

    def commit_on_main(self, name: str, data: bytes) -> None:
        (self.repo / name).write_bytes(data)
        lifecycle.git(self.repo, "add", name)
        lifecycle.git(self.repo, "commit", "-q", "-m", f"add {name}")

    def test_prepare_seeds_main_without_prompt_and_never_overwrites(self):
        run = self.prepare()
        original = (self.repo / "README.md").read_bytes()
        self.assertEqual(lifecycle.git(self.repo, "rev-parse", "--abbrev-ref", "HEAD"), "main")
        self.assertEqual(lifecycle.git(self.repo, "config", "core.autocrlf"), "false")
        for relative in ("AGENTS.md", "checks/acceptance.py", "app.py"):
            self.assertTrue((self.repo / relative).is_file(), relative)
        self.assertFalse((self.repo / "task-prompt.txt").exists())
        self.assertFalse((self.repo / "__pycache__").exists())
        self.assertEqual(declaration(self.repo), {"schema": 1, "station": "S1", "items": ["B01"]})
        self.assertEqual(run["seed"]["mainCommit"], lifecycle.git(self.repo, "rev-parse", "main"))
        self.assertIn(".study/station.json", run["seed"]["rawGitManifest"])
        self.assertEqual(lifecycle.git(self.repo, "status", "--porcelain"), "")
        self.assertNotIn("authorization", run)
        with self.assertRaises(lifecycle.LifecycleError):
            self.prepare()
        self.assertEqual((self.repo / "README.md").read_bytes(), original)

    def test_git_helper_disables_repo_code_and_keeps_existing_config(self):
        with patch.dict(os.environ, {"GIT_CONFIG_COUNT": "1", "GIT_CONFIG_KEY_0": "user.name",
                                     "GIT_CONFIG_VALUE_0": "Somebody"}):
            env = lifecycle.git_env()
        self.assertEqual(env["GIT_CONFIG_KEY_0"], "user.name")
        pairs = {env[f"GIT_CONFIG_KEY_{i}"]: env[f"GIT_CONFIG_VALUE_{i}"]
                 for i in range(int(env["GIT_CONFIG_COUNT"]))}
        self.assertNotIn("safe.directory", pairs)  # root never reads someone else's repo itself
        self.assertEqual(pairs["core.autocrlf"], "false")
        self.assertEqual(pairs["core.fsmonitor"], "false")
        self.assertEqual(pairs["core.hooksPath"], os.devnull)
        self.assertEqual(pairs["user.name"], "Somebody")

    def test_repo_configured_commands_do_not_run_during_capture(self):
        self.prepare()
        marker = self.root / "ran.txt"
        script = self.root / "hook.sh"
        script.write_text(f"#!/bin/sh\necho ran >> '{marker.as_posix()}'\n", encoding="utf-8", newline="\n")
        script.chmod(0o755)
        lifecycle.git(self.repo, "config", "core.fsmonitor", script.as_posix())
        lifecycle.git(self.repo, "config", "diff.external", script.as_posix())
        lifecycle.git(self.repo, "config", "diff.evil.textconv", script.as_posix())
        (self.repo / ".git" / "info" / "attributes").write_text("*.md diff=evil\n", encoding="utf-8")
        (self.repo / "README.md").write_bytes(b"# changed\n")
        lifecycle.git(self.repo, "add", "README.md")
        (self.repo / "README.md").write_bytes(b"# changed again\n")
        lifecycle.snapshot(self.repo, self.audit)
        self.assertFalse(marker.exists())

    def test_lf_git_bytes_and_dirty_unfinished_work_are_captured_raw(self):
        global_config = self.root / "global.gitconfig"
        global_config.write_text("[core]\n\tautocrlf = true\n", encoding="utf-8")
        with patch.dict(os.environ, {"GIT_CONFIG_GLOBAL": str(global_config)}):
            self.prepare()
            expected = b"# readinglog\n"
            self.assertEqual((self.repo / "README.md").read_bytes(), expected)
            main = lifecycle.git(self.repo, "rev-parse", "main")
            self.assertEqual(lifecycle._committed_manifest(self.repo, main)["README.md"], sha(expected))
            changed = b"# readinglog edited\n"
            (self.repo / "README.md").write_bytes(changed)
            (self.repo / "untracked.txt").write_bytes(b"unfinished\n")
            result = lifecycle.snapshot(self.repo, self.audit)
        record = result["record"]
        self.assertTrue(record["branch"]["selectedHeadUnfinished"])
        self.assertTrue(record["branch"]["dirty"])
        self.assertEqual(record["rawWorktreeManifest"]["README.md"], sha(changed))
        self.assertEqual(record["rawWorktreeManifest"]["untracked.txt"], sha(b"unfinished\n"))
        self.assertEqual(Path(result["worktree"], "README.md").read_bytes(), changed)
        self.assertEqual(record["immutableMain"]["rawGitManifest"]["README.md"], sha(expected))
        self.assertEqual(Path(result["immutableMain"], "README.md").read_bytes(), expected)
        self.assertEqual(Path(result["immutableMain"]), self.audit / "snapshot-S1" / "immutable-main")
        self.assertTrue(Path(result["bundle"]).is_file())

    def test_immutable_main_contains_merged_work_only(self):
        self.prepare()
        self.commit_on_main("merged.txt", b"on main\n")
        lifecycle.git(self.repo, "switch", "-q", "-c", "feature/open")
        self.commit_on_main("open.txt", b"not merged\n")
        result = lifecycle.snapshot(self.repo, self.audit)
        main_copy = Path(result["immutableMain"])
        self.assertTrue((main_copy / "merged.txt").is_file())
        self.assertFalse((main_copy / "open.txt").exists())
        self.assertTrue(Path(result["worktree"], "open.txt").is_file())
        self.assertEqual(result["record"]["branch"]["name"], "feature/open")
        self.assertTrue(result["record"]["branch"]["headDiffersFromMain"])

    def test_advance_requires_snapshot_and_preserves_prior_snapshot(self):
        self.prepare(case="roombook")
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.advance(self.repo, self.audit)
        first = lifecycle.snapshot(self.repo, self.audit)
        snapshot_bytes = (self.audit / "snapshot-S1" / "snapshot.json").read_bytes()
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.snapshot(self.repo, self.audit)
        second_stage = lifecycle.advance(self.repo, self.audit)
        self.assertEqual(second_stage, {"schema": 1, "station": "S2", "items": ["R02", "R03", "R04"]})
        self.assertEqual(first["station"], "S1")
        second = lifecycle.snapshot(self.repo, self.audit)
        self.assertEqual(second["station"], "S2")
        self.assertEqual((self.audit / "snapshot-S1" / "snapshot.json").read_bytes(), snapshot_bytes)

    def test_in_tree_attributes_do_not_convert_the_main_checkout(self):
        self.prepare()
        (self.repo / ".gitattributes").write_bytes(b"*.md text eol=crlf\n")
        self.commit_on_main(".gitattributes", b"*.md text eol=crlf\n")
        result = lifecycle.snapshot(self.repo, self.audit)
        self.assertEqual(Path(result["immutableMain"], "README.md").read_bytes(), b"# readinglog\n")
        self.assertEqual(result["record"]["errors"], [])

    def test_missing_main_is_recorded_not_fatal(self):
        self.prepare()
        lifecycle.git(self.repo, "branch", "-m", "main", "trunk")
        result = lifecycle.snapshot(self.repo, self.audit)
        self.assertIsNone(result["immutableMain"])
        self.assertIsNone(result["record"]["immutableMain"]["commit"])
        self.assertTrue(result["record"]["errors"])
        self.assertEqual(result["record"]["branch"]["name"], "trunk")
        self.assertTrue(Path(result["worktree"], "README.md").is_file())

    def test_oversized_files_are_listed_not_copied(self):
        self.prepare()
        (self.repo / "big.bin").write_bytes(b"x" * 2048)
        with patch.object(lifecycle, "FILE_CAP", 1024):
            result = lifecycle.snapshot(self.repo, self.audit)
        self.assertEqual(result["record"]["rawWorktreeManifest"]["big.bin"], "skipped:2048")
        self.assertFalse(Path(result["worktree"], "big.bin").exists())
        self.assertTrue(Path(result["worktree"], "README.md").is_file())

    def test_four_stage_declarations_and_final_freeze_are_fixed(self):
        self.prepare()
        expected = [["B01"], ["B02", "B03", "B04"],
                    ["B05", "B06", "B07", "B08", "B09", "B10", "B11"], ["B12"]]
        for number, wave in enumerate(expected, 1):
            self.assertEqual(declaration(self.repo)["station"], f"S{number}")
            self.assertEqual(declaration(self.repo)["items"], wave)
            lifecycle.snapshot(self.repo, self.audit)
            if number < 4:
                lifecycle.advance(self.repo, self.audit)
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.advance(self.repo, self.audit)
        final = lifecycle.freeze(self.repo, self.audit, reason="finite end after S4")
        self.assertEqual(final["kind"], "final_freeze")
        self.assertEqual(final["record"]["freezeReason"], "finite end after S4")
        self.assertEqual(Path(final["immutableMain"]), self.audit / "final-freeze" / "immutable-main")
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.freeze(self.repo, self.audit, reason="duplicate")

    def write_plan(self, case: str, waves, **extra) -> None:
        folder = self.seed / case
        if not folder.exists():
            shutil.copytree(self.seed / "readinglog", folder)
        value = {"schema": 1, "case": case, **extra,
                 "stations": [{"id": f"S{i}", "items": wave} for i, wave in enumerate(waves, 1)]}
        (folder / "STATIONS.json").write_text(json.dumps(value), encoding="utf-8", newline="\n")

    def test_six_stations_advance_to_the_last_one(self):
        waves = [["B01"], ["B02", "B03", "B04"], [f"B{i:02d}" for i in range(5, 11)], ["B11", "B12"],
                 ["B13", "B14"], ["B15"]]
        self.write_plan("readinglog2", waves)
        run = lifecycle.prepare(self.seed, self.repo, self.audit, case="readinglog2", method="conventional")
        self.assertEqual(run["stationPlan"], waves)
        for number, wave in enumerate(waves, 1):
            self.assertEqual(declaration(self.repo), {"schema": 1, "station": f"S{number}", "items": wave})
            self.assertEqual(lifecycle.snapshot(self.repo, self.audit)["station"], f"S{number}")
            if number < len(waves):
                lifecycle.advance(self.repo, self.audit)
        with self.assertRaisesRegex(lifecycle.LifecycleError, "S6 is the final station"):
            lifecycle.advance(self.repo, self.audit)
        self.assertEqual(lifecycle.freeze(self.repo, self.audit, reason="done")["station"], "S6")

    def test_station_plan_must_be_ordered_unique_and_non_empty(self):
        bad = {"no stations": [], "empty wave": [["B01"], []], "reused item": [["B01"], ["B01", "B02"]],
               "blank item": [["B01"], [" "]]}
        for name, waves in bad.items():
            with self.subTest(name):
                self.write_plan("readinglog2", waves)
                with self.assertRaises(lifecycle.LifecycleError):
                    lifecycle.load_station_plan(self.seed / "readinglog2" / "STATIONS.json", "readinglog2")
        self.write_plan("readinglog2", [["B01"], ["B02"]])
        with self.assertRaisesRegex(lifecycle.LifecycleError, "does not describe"):
            lifecycle.load_station_plan(self.seed / "readinglog2" / "STATIONS.json", "roombook")
        value = json.loads((self.seed / "readinglog2" / "STATIONS.json").read_text(encoding="utf-8"))
        value["stations"][1]["id"] = "S3"
        (self.seed / "readinglog2" / "STATIONS.json").write_text(json.dumps(value), encoding="utf-8")
        with self.assertRaisesRegex(lifecycle.LifecycleError, "in order"):
            lifecycle.load_station_plan(self.seed / "readinglog2" / "STATIONS.json", "readinglog2")
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.prepare(self.seed, self.repo, self.audit, case="not-a-case", method="conventional")

    def test_actual_case_seed_is_copied_flat(self):
        seed = PLAYGROUND / "cases"
        if not (seed / "readinglog" / "STATIONS.json").is_file():
            self.skipTest("cases/ not present yet")
        lifecycle.prepare(seed, self.repo, self.audit, case="readinglog", method="conventional")
        for relative in ("AGENTS.md", "QUALITY.md", "checks/acceptance.py", "README.md",
                         "BACKLOG.md", "STATIONS.json", "app.py"):
            self.assertTrue((self.repo / relative).is_file(), relative)
        self.assertFalse((self.repo / "task-prompt.txt").exists())

    def test_actor_branch_main_and_detached_head_are_capturable(self):
        self.prepare()
        lifecycle.git(self.repo, "switch", "-q", "-c", "feature/unmerged")
        self.commit_on_main("feature.txt", b"unmerged feature ref\n")
        lifecycle.git(self.repo, "switch", "-q", "main")
        main_capture = lifecycle.snapshot(self.repo, self.audit)["record"]
        self.assertEqual(main_capture["branch"]["name"], "main")
        self.assertFalse(main_capture["branch"]["selectedHeadUnfinished"])
        self.assertIn("refs/heads/feature/unmerged", main_capture["branch"]["refs"])

        second_repo, second_audit = self.root / "detached", self.root / "audit-detached"
        lifecycle.prepare(self.seed, second_repo, second_audit, case="readinglog", method="markitect")
        lifecycle.git(second_repo, "checkout", "-q", "--detach", "HEAD")
        detached = lifecycle.snapshot(second_repo, second_audit)["record"]
        self.assertIsNone(detached["branch"]["name"])
        self.assertIn("refs/heads/main", detached["branch"]["refs"])

    def test_early_blocked_freeze_needs_reason_and_closes_transitions(self):
        self.prepare()
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.freeze(self.repo, self.audit, reason="  ")
        frozen = lifecycle.freeze(self.repo, self.audit, reason="blocked at S1: setup blocked")
        self.assertEqual(frozen["station"], "S1")
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.snapshot(self.repo, self.audit)
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.advance(self.repo, self.audit)

    def test_advance_rewrites_only_station_file_and_never_commits(self):
        self.prepare()
        changed = b"actor's unfinished change\n"
        (self.repo / "app.py").write_bytes(changed)
        (self.repo / "staged.txt").write_bytes(b"staged by the actor\n")
        lifecycle.git(self.repo, "add", "staged.txt")
        index_before = (self.repo / ".git" / "index").read_bytes()
        before_head = lifecycle.git(self.repo, "rev-parse", "HEAD")
        snap = lifecycle.snapshot(self.repo, self.audit)
        self.assertIn(b"staged.txt", Path(snap["dir"], "actor-state", "staged.diff").read_bytes())
        self.assertIn(b"app.py", Path(snap["dir"], "actor-state", "unstaged.diff").read_bytes())
        lifecycle.advance(self.repo, self.audit)
        self.assertEqual(lifecycle.git(self.repo, "rev-parse", "HEAD"), before_head)
        self.assertEqual((self.repo / ".git" / "index").read_bytes(), index_before)
        self.assertEqual((self.repo / "app.py").read_bytes(), changed)
        self.assertEqual(declaration(self.repo)["station"], "S2")
        status = lifecycle.git_bytes(self.repo, "status", "--porcelain").decode("utf-8").splitlines()
        self.assertEqual(sorted(status), [" M .study/station.json", " M app.py", "A  staged.txt"])
        transition = lifecycle._load_json(self.audit / "transitions" / "S1-to-S2.json")
        self.assertEqual((transition["from"], transition["to"]), ("S1", "S2"))

    def test_agent_reverting_station_file_does_not_break_the_trajectory(self):
        self.prepare()
        lifecycle.snapshot(self.repo, self.audit)
        lifecycle.advance(self.repo, self.audit)
        lifecycle.git(self.repo, "checkout", "--", ".study/station.json")  # e.g. a stash or reset
        second = lifecycle.snapshot(self.repo, self.audit)
        self.assertEqual(second["station"], "S2")
        self.assertEqual(second["record"]["stationDeclaration"]["status"], "modified")
        shutil.rmtree(self.repo / ".study")
        (self.repo / ".study").write_bytes(b"not a folder\n")
        third_stage = lifecycle.advance(self.repo, self.audit)
        self.assertEqual(third_stage["station"], "S3")
        third = lifecycle.snapshot(self.repo, self.audit)
        self.assertEqual(third["record"]["stationDeclaration"]["status"], "intact")

    @unittest.skipIf(os.name != "posix", "symlinks need POSIX")
    def test_symlinks_in_worktree_are_listed_not_followed(self):
        self.prepare()
        os.symlink("README.md", self.repo / "link.md")
        (self.repo / ".venv" / "bin").mkdir(parents=True)
        os.symlink("/usr/bin/python3", self.repo / ".venv" / "bin" / "python")
        os.symlink(str(self.root), self.repo / "outside")
        result = lifecycle.snapshot(self.repo, self.audit)
        manifest = result["record"]["rawWorktreeManifest"]
        self.assertEqual(manifest["link.md"], "symlink:README.md")
        self.assertEqual(manifest["outside"], f"symlink:{self.root}")
        self.assertFalse(Path(result["worktree"], "outside").exists())
        lifecycle.advance(self.repo, self.audit)

    @unittest.skipIf(os.name != "posix", "non-UTF-8 file names need POSIX")
    def test_non_utf8_file_names_are_captured(self):
        self.prepare()
        name = os.fsdecode(b"caf\xe9.txt")
        (self.repo / name).write_bytes(b"latin-1 name\n")
        lifecycle.git(self.repo, "add", "--", name)
        lifecycle.git(self.repo, "commit", "-q", "-m", "odd name")
        result = lifecycle.snapshot(self.repo, self.audit)
        self.assertTrue(Path(result["immutableMain"], name).is_file())
        self.assertIn(name, result["record"]["rawWorktreeManifest"])

    def test_capture_failure_preserves_partial_evidence_and_sweeps_first(self):
        self.prepare()
        calls = []
        with patch.object(lifecycle, "_checkout_main", side_effect=RuntimeError("clone exploded")):
            with self.assertRaises(lifecycle.LifecycleError):
                lifecycle.snapshot(self.repo, self.audit, sweep=lambda: calls.append("sweep"))
        self.assertEqual(calls, ["sweep"])
        partials = list(self.audit.glob(".snapshot-S1-*"))
        self.assertEqual(len(partials), 1)
        self.assertTrue((partials[0] / "capture-error.json").is_file())
        self.assertTrue((partials[0] / "history.bundle").is_file())

    def test_missing_repo_or_audit_is_rejected(self):
        self.prepare()
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.snapshot(self.root / "other-repo", self.audit)
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.snapshot(self.repo, self.root / "other-audit")

    def test_cli_prepare_and_snapshot(self):
        quiet = contextlib.ExitStack()
        quiet.enter_context(contextlib.redirect_stdout(io.StringIO()))
        quiet.enter_context(contextlib.redirect_stderr(io.StringIO()))
        self.addCleanup(quiet.close)
        code = lifecycle._cli(["prepare", "--seed", str(self.seed), "--repo", str(self.repo),
                               "--audit", str(self.audit), "--case", "roombook", "--method", "manual"])
        self.assertEqual(code, 0)
        self.assertEqual(lifecycle._cli(["snapshot", "--repo", str(self.repo), "--audit", str(self.audit)]), 0)
        self.assertEqual(lifecycle._cli(["advance", "--repo", str(self.root), "--audit", str(self.audit)]), 2)


def _agent_entry():
    if os.name != "posix" or os.geteuid() != 0:
        return None
    import pwd
    try:
        return pwd.getpwnam("agent")
    except KeyError:
        return None


@unittest.skipUnless(_agent_entry(), "needs root and an 'agent' user (run inside the container)")
class RootCaptureTests(unittest.TestCase):
    def test_root_capture_runs_repo_code_only_as_the_owner(self):
        entry = _agent_entry()
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        root = Path(temp.name)
        os.chmod(root, 0o755)
        repo, audit = root / "work" / "readinglog", root / "out" / "audit"
        lifecycle.prepare(make_seed(root), repo, audit, case="readinglog", method="conventional")
        marker = root / "uids.txt"
        marker.write_text("", encoding="utf-8")
        script = root / "clean.sh"
        script.write_text(f"#!/bin/sh\nid -u >> {marker}\ncat\n", encoding="utf-8")
        script.chmod(0o755)
        with (repo / ".git" / "config").open("a", encoding="utf-8") as config:
            config.write(f"[core]\n\tfsmonitor = {script}\n[diff]\n\texternal = {script}\n"
                         f"[filter \"evil\"]\n\tclean = {script}\n")
        (repo / ".git" / "info" / "attributes").write_text("*.md filter=evil\n", encoding="utf-8")
        (repo / "README.md").write_bytes(b"# dirty\n")
        for current, dirs, files in os.walk(root / "work"):
            for name in dirs + files:
                os.chown(os.path.join(current, name), entry.pw_uid, entry.pw_gid, follow_symlinks=False)
        os.chown(root / "work", entry.pw_uid, entry.pw_gid)
        os.chown(marker, entry.pw_uid, entry.pw_gid)
        swept = []
        result = lifecycle.snapshot(repo, audit, sweep=lambda: swept.append(True))
        uids = set(marker.read_text(encoding="utf-8").split())
        self.assertNotIn("0", uids)
        self.assertLessEqual(uids, {str(entry.pw_uid)})
        self.assertEqual(swept, [True])
        self.assertTrue(result["record"]["branch"]["dirty"])
        self.assertEqual(Path(result["worktree"], "README.md").read_bytes(), b"# dirty\n")


if __name__ == "__main__":
    unittest.main()
