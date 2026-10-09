from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import lifecycle


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def make_seed(root: Path) -> Path:
    seed = root / "public"
    common = seed / "common"
    (common / "checks").mkdir(parents=True)
    (common / "AGENTS.md").write_bytes(b"Use normal project workflow.\n")
    (common / "QUALITY.md").write_bytes(b"Assess after freeze.\n")
    (common / "checks" / "acceptance.py").write_bytes(b"# public fixed checks\n")
    for case, prefix in (("roombook", "R"), ("readinglog", "B")):
        folder = seed / "cases" / case
        folder.mkdir(parents=True)
        (folder / "README.md").write_bytes(f"# {case}\n".encode())
        (folder / "BACKLOG.md").write_bytes(b"Implement the public backlog.\n")
        ids = [f"{prefix}{i:02d}" for i in range(1, 13)]
        stations = [ids[:1], ids[1:4], ids[4:11], ids[11:]]
        value = {"schema": 1, "case": case, "totalItems": 12,
                 "stations": [{"id": f"S{i}", "items": wave, "requiresTeam": i == 3}
                              for i, wave in enumerate(stations, 1)]}
        (folder / "STATIONS.json").write_text(json.dumps(value), encoding="utf-8", newline="\n")
        if case == "readinglog":
            (folder / "app.py").write_bytes(b"# baseline\n")
    return seed


class LifecycleTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.seed = make_seed(self.root)
        self.repo = self.root / "actor"
        self.audit = self.root / "audit"

    def prepare(self, case="readinglog"):
        return lifecycle.prepare(self.seed, self.repo, self.audit, case=case,
                                 method="Conventional", profile={"runtime": "offline-fixture"},
                                 pins={"markitect": None})

    def test_prepare_does_not_overwrite_and_records_execution_disabled(self):
        run = self.prepare()
        original = (self.repo / "README.md").read_bytes()
        self.assertEqual(run["authorization"], {
            "execution_authorized": False, "status": "disabled", "source": "not supplied"})
        self.assertFalse(run["providerLauncher"])
        self.assertTrue((self.repo / "AGENTS.md").is_file())
        self.assertTrue((self.repo / "checks" / "acceptance.py").is_file())
        self.assertTrue((self.repo / "app.py").is_file())
        self.assertEqual((self.repo / ".study" / "station.json").read_bytes().count(b"B01"), 1)
        with self.assertRaises(lifecycle.LifecycleError):
            self.prepare()
        self.assertEqual((self.repo / "README.md").read_bytes(), original)

    def test_lf_git_bytes_and_dirty_unfinished_work_are_captured_raw(self):
        global_config = self.root / "global.gitconfig"
        global_config.write_text("[core]\n\tautocrlf = true\n", encoding="utf-8")
        with patch.dict(os.environ, {"GIT_CONFIG_GLOBAL": str(global_config)}):
            self.prepare()
            expected = b"# readinglog\n"
            self.assertEqual((self.repo / "README.md").read_bytes(), expected)
            snap_commit = lifecycle._git(self.repo, "rev-parse", "main")
            self.assertEqual(lifecycle._committed_manifest(self.repo, snap_commit)["README.md"], sha(expected))
            changed = b"# readinglog edited\n"
            (self.repo / "README.md").write_bytes(changed)
            (self.repo / "untracked.txt").write_bytes(b"unfinished\n")
            result = lifecycle.snapshot(self.repo, self.audit)
        self.assertEqual(result["branch"]["selectedHeadUnfinished"], True)
        self.assertEqual(result["branch"]["overallUnfinished"], "unknown")
        self.assertEqual(result["branch"]["dirty"], True)
        self.assertEqual(result["rawWorktreeManifest"]["README.md"], sha(changed))
        self.assertEqual(result["rawWorktreeManifest"]["untracked.txt"], sha(b"unfinished\n"))
        self.assertEqual((self.audit / "snapshot-S1" / "worktree" / "README.md").read_bytes(), changed)
        self.assertEqual(result["immutableMain"]["rawGitManifest"]["README.md"], sha(b"# readinglog\n"))
        self.assertEqual((self.audit / "snapshot-S1" / "immutable-main" / "README.md").read_bytes(), b"# readinglog\n")

    def test_advance_requires_snapshot_and_preserves_prior_snapshot(self):
        self.prepare(case="roombook")
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.advance(self.repo, self.audit)
        first = lifecycle.snapshot(self.repo, self.audit)
        snapshot_bytes = (self.audit / "snapshot-S1" / "snapshot.json").read_bytes()
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.snapshot(self.repo, self.audit)
        second_stage = lifecycle.advance(self.repo, self.audit)
        self.assertEqual(second_stage["station"], "S2")
        self.assertEqual(second_stage["items"], ["R02", "R03", "R04"])
        self.assertEqual(second_stage["previousSnapshotSha256"], first["snapshotSha256"])
        self.assertEqual((self.audit / "snapshot-S1" / "snapshot.json").read_bytes(), snapshot_bytes)
        second = lifecycle.snapshot(self.repo, self.audit)
        self.assertEqual(second["station"], "S2")
        self.assertEqual((self.audit / "snapshot-S1" / "snapshot.json").read_bytes(), snapshot_bytes)

    def test_advance_rejects_mutated_snapshot_copy(self):
        self.prepare()
        lifecycle.snapshot(self.repo, self.audit)
        frozen_copy = self.audit / "snapshot-S1" / "worktree" / "README.md"
        frozen_copy.write_bytes(b"tampered archive copy\n")
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.advance(self.repo, self.audit)
        self.assertEqual(json.loads((self.repo / ".study" / "station.json").read_text())["station"], "S1")

    def test_four_stage_declarations_and_final_freeze_are_fixed(self):
        self.prepare()
        expected = [
            ["B01"], ["B02", "B03", "B04"],
            ["B05", "B06", "B07", "B08", "B09", "B10", "B11"], ["B12"],
        ]
        for number, wave in enumerate(expected, 1):
            declaration = json.loads((self.repo / ".study" / "station.json").read_text(encoding="utf-8"))
            self.assertEqual(declaration["station"], f"S{number}")
            self.assertEqual(declaration["items"], wave)
            if number < 4:
                lifecycle.snapshot(self.repo, self.audit)
                lifecycle.advance(self.repo, self.audit)
        final = lifecycle.freeze(self.repo, self.audit, reason="finite end after S4")
        self.assertEqual(final["kind"], "final_freeze")
        self.assertEqual(final["freezeReason"], "finite end after S4")
        self.assertEqual(final["assessment"], {"status": "pending"})
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.freeze(self.repo, self.audit, reason="duplicate")

    def test_actual_public_seed_is_copied_flat_without_running_case_code(self):
        actual_seed = Path(__file__).resolve().parents[1] / "public"
        lifecycle.prepare(actual_seed, self.repo, self.audit, case="readinglog",
                          method="Conventional", profile={"runtime": "offline-fixture"})
        for relative in ("AGENTS.md", "QUALITY.md", "checks/acceptance.py", "README.md",
                         "BACKLOG.md", "STATIONS.json", "app.py", "tests/test_baseline.py"):
            self.assertTrue((self.repo / relative).is_file(), relative)
        self.assertEqual(lifecycle._load_json(self.repo / "STATIONS.json")["case"], "readinglog")

    def test_actor_branch_main_and_detached_head_are_capturable(self):
        self.prepare()
        lifecycle._git(self.repo, "switch", "-c", "feature/unmerged")
        (self.repo / "feature.txt").write_bytes(b"unmerged feature ref\n")
        lifecycle._git(self.repo, "add", "feature.txt")
        lifecycle._git_check(self.repo, "diff", "--cached", "--check")
        lifecycle._git(self.repo, "commit", "-m", "fixture feature commit")
        lifecycle._git(self.repo, "switch", "main")
        main_capture = lifecycle.snapshot(self.repo, self.audit)
        self.assertEqual(main_capture["branch"]["name"], "main")
        self.assertFalse(main_capture["branch"]["selectedHeadUnfinished"])
        self.assertEqual(main_capture["branch"]["overallUnfinished"], "unknown")
        self.assertIn("refs/heads/feature/unmerged", main_capture["branch"]["refs"])

        second_repo = self.root / "actor-detached"
        second_audit = self.root / "audit-detached"
        lifecycle.prepare(self.seed, second_repo, second_audit, case="readinglog",
                          method="Markitect", profile={"runtime": "offline-fixture"})
        lifecycle._git(second_repo, "checkout", "--detach", "HEAD")
        detached = lifecycle.snapshot(second_repo, second_audit)
        self.assertIsNone(detached["branch"]["name"])
        self.assertIn("refs/heads/main", detached["branch"]["refs"])

    def test_early_blocked_freeze_needs_reason_and_closes_transitions(self):
        self.prepare()
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.freeze(self.repo, self.audit, reason="  ")
        frozen = lifecycle.freeze(self.repo, self.audit, reason="blocked at S1: product binding unresolved")
        self.assertEqual(frozen["station"], "S1")
        self.assertEqual(frozen["freezeReason"], "blocked at S1: product binding unresolved")
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.snapshot(self.repo, self.audit)
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.advance(self.repo, self.audit)

    def test_authorization_and_execution_metadata_are_preserved_as_external_input(self):
        authorization = {"execution_authorized": True, "authority": "external-order-7", "stratum": "sonnet"}
        execution = {"status": "completed_externally", "modelId": "pinned-effective-id", "calls": 2}
        lifecycle.prepare(self.seed, self.repo, self.audit, case="readinglog", method="Conventional",
                          profile={"runtime": "fixture"}, authorization=authorization)
        lifecycle.record_execution(self.audit, metadata=execution)
        captured = lifecycle.freeze(self.repo, self.audit, reason="finite S1 fixture")
        self.assertEqual(captured["authorization"], authorization)
        self.assertEqual(captured["execution"]["metadata"], execution)
        self.assertFalse(captured["providerLauncher"])

    def test_advance_preserves_unstaged_actor_work_and_never_commits(self):
        self.prepare()
        changed = b"actor's unfinished change\n"
        (self.repo / "app.py").write_bytes(changed)
        before_head = lifecycle._git(self.repo, "rev-parse", "HEAD")
        lifecycle.snapshot(self.repo, self.audit)
        lifecycle.advance(self.repo, self.audit)
        self.assertEqual(lifecycle._git(self.repo, "rev-parse", "HEAD"), before_head)
        self.assertEqual((self.repo / "app.py").read_bytes(), changed)
        self.assertEqual(json.loads((self.repo / ".study" / "station.json").read_text())["station"], "S2")
        transition = lifecycle._load_json(self.audit / "transitions" / "S1-to-S2.json")
        self.assertFalse(transition["committedByPlayground"])
        self.assertEqual(transition["actorDelta"], "left_uncommitted")

    def test_advance_refuses_staged_index_without_touching_it(self):
        self.prepare()
        (self.repo / "staged.txt").write_bytes(b"retain index state\n")
        lifecycle._git(self.repo, "add", "staged.txt")
        index_before = (self.repo / ".git" / "index").read_bytes()
        snapshot = lifecycle.snapshot(self.repo, self.audit)
        self.assertEqual((self.repo / ".git" / "index").read_bytes(), index_before)
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.advance(self.repo, self.audit)
        self.assertEqual((self.repo / ".git" / "index").read_bytes(), index_before)
        self.assertEqual(json.loads((self.repo / ".study" / "station.json").read_text())["station"], "S1")
        self.assertEqual(snapshot["branch"]["indexSha256"], sha(index_before))

    def test_capture_detects_mid_copy_change_and_preserves_partial_evidence(self):
        self.prepare()
        original_manifest = lifecycle._manifest
        repo_reads = 0

        def manifest_with_race(root, **kwargs):
            nonlocal repo_reads
            if Path(root).resolve() == self.repo.resolve() and kwargs.get("exclude_git"):
                repo_reads += 1
                if repo_reads == 2:
                    (self.repo / "README.md").write_bytes(b"changed during capture\n")
            return original_manifest(root, **kwargs)

        with patch.object(lifecycle, "_manifest", side_effect=manifest_with_race):
            with self.assertRaises(lifecycle.LifecycleError):
                lifecycle.snapshot(self.repo, self.audit)
        partials = list(self.audit.glob(".snapshot-S1-*"))
        self.assertEqual(len(partials), 1)
        self.assertTrue((partials[0] / "capture-error.json").is_file())

    def test_wrong_repo_or_audit_binding_is_rejected(self):
        self.prepare()
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.snapshot(self.root / "other-repo", self.audit)
        with self.assertRaises(lifecycle.LifecycleError):
            lifecycle.snapshot(self.repo, self.root / "other-audit")


if __name__ == "__main__":
    unittest.main()
