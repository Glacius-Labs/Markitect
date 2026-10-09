from __future__ import annotations

import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

PLAYGROUND = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(PLAYGROUND))
import lifecycle
import workspace_policy


PUBLIC = PLAYGROUND / "public"


class WorkspaceSnapshotTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        self.repo = root / "repo"
        self.audit = root / "audit"

    def prepare(self, *, completion_target="workspace_snapshot"):
        return lifecycle.prepare(
            PUBLIC, self.repo, self.audit, case="readinglog", method="Conventional",
            profile={"runtime": "provider-free-fixture"}, completion_target=completion_target,
        )

    def test_workspace_snapshot_retains_dirty_and_untracked_files_across_advance(self):
        self.prepare()
        tracked = self.repo / "app.py"
        tracked.write_bytes(b"# actor implementation\n")
        lifecycle._git(self.repo, "add", "app.py")
        untracked = self.repo / "unfinished.py"
        untracked.write_bytes(b"unfinished work stays in the workspace\n")
        head_before = lifecycle._git(self.repo, "rev-parse", "HEAD")

        captured = lifecycle.snapshot(self.repo, self.audit)
        self.assertEqual(captured["completionTarget"], "workspace_snapshot")
        self.assertEqual(captured["assessmentCandidate"]["kind"], "workspace_snapshot")
        self.assertEqual(captured["assessmentCandidate"]["path"], "worktree")
        self.assertEqual(captured["assessmentCandidate"]["manifest"], captured["rawWorktreeManifest"])
        self.assertEqual(
            (self.audit / "snapshot-S1" / "worktree" / "unfinished.py").read_bytes(),
            untracked.read_bytes(),
        )

        next_station = lifecycle.advance(self.repo, self.audit)
        self.assertEqual(next_station["station"], "S2")
        self.assertEqual(lifecycle._git(self.repo, "rev-parse", "HEAD"), head_before)
        self.assertEqual(tracked.read_bytes(), b"# actor implementation\n")
        self.assertEqual(untracked.read_bytes(), b"unfinished work stays in the workspace\n")
        self.assertTrue(lifecycle._git(self.repo, "status", "--porcelain"))

    def test_advance_rejects_workspace_tampered_after_snapshot(self):
        self.prepare()
        lifecycle.snapshot(self.repo, self.audit)
        (self.repo / "late-change.txt").write_text("not in the frozen snapshot\n", encoding="utf-8")

        with self.assertRaisesRegex(lifecycle.LifecycleError, "changed after its station snapshot"):
            lifecycle.advance(self.repo, self.audit)
        station = json.loads((self.repo / ".study" / "station.json").read_text(encoding="utf-8"))
        self.assertEqual(station["station"], "S1")

    def test_advance_detects_workspace_race_while_preparing_transition(self):
        self.prepare()
        lifecycle.snapshot(self.repo, self.audit)
        original_manifest = lifecycle._manifest
        repo_reads = 0

        def racing_manifest(root, **kwargs):
            nonlocal repo_reads
            if Path(root).resolve() == self.repo.resolve() and kwargs.get("exclude_git"):
                repo_reads += 1
                if repo_reads == 2:
                    (self.repo / "raced.txt").write_text("concurrent actor change\n", encoding="utf-8")
            return original_manifest(root, **kwargs)

        with patch.object(lifecycle, "_manifest", side_effect=racing_manifest):
            with self.assertRaisesRegex(lifecycle.LifecycleError, "changed while preparing station advance"):
                lifecycle.advance(self.repo, self.audit)
        station = json.loads((self.repo / ".study" / "station.json").read_text(encoding="utf-8"))
        self.assertEqual(station["station"], "S1")

    def test_default_target_stays_main_merge_and_does_not_render_public_rules(self):
        source_agents = (PUBLIC / "common" / "AGENTS.md").read_bytes()
        source_quality = (PUBLIC / "common" / "QUALITY.md").read_bytes()
        run = lifecycle.prepare(
            PUBLIC, self.repo, self.audit, case="readinglog", method="Conventional",
            profile={"runtime": "provider-free-fixture"},
        )
        self.assertEqual(run["completionTarget"], "main_merge")
        self.assertEqual((self.repo / "AGENTS.md").read_bytes(), source_agents)
        self.assertEqual((self.repo / "QUALITY.md").read_bytes(), source_quality)
        captured = lifecycle.snapshot(self.repo, self.audit)
        self.assertEqual(captured["completionTarget"], "main_merge")
        self.assertNotIn("assessmentCandidate", captured)

    def test_workspace_policy_changes_only_mechanical_completion_clauses(self):
        self.prepare()
        requirements = {
            path.relative_to(PUBLIC).as_posix(): path.read_bytes()
            for path in (
                PUBLIC / "cases" / "readinglog" / "README.md",
                PUBLIC / "cases" / "readinglog" / "BACKLOG.md",
                PUBLIC / "cases" / "readinglog" / "STATIONS.json",
                PUBLIC / "common" / "checks" / "acceptance.py",
            )
        }
        agents = workspace_policy.render(
            "AGENTS.md", (PUBLIC / "common" / "AGENTS.md").read_text(encoding="utf-8")
        )
        quality = workspace_policy.render(
            "QUALITY.md", (PUBLIC / "common" / "QUALITY.md").read_text(encoding="utf-8")
        )
        fragment = workspace_policy.render(
            "AGENTS.fragment.md", (PUBLIC / "conventional" / "AGENTS.fragment.md").read_text(encoding="utf-8")
        )
        prompt = workspace_policy.render(
            "task-prompt.txt", (PUBLIC / "task-prompt.txt").read_text(encoding="utf-8")
        )

        for output in (agents, quality, fragment, prompt):
            self.assertNotIn("merge finished", output)
            self.assertNotIn("merge the fertigen Änderungen", output)
        self.assertIn("all requirements are public from the beginning", agents)
        self.assertIn("Critical required behavior failure", quality)
        self.assertIn("S4 needs behavior and reference consistency", fragment)
        self.assertIn("Hier liegt das Backlog.", prompt)
        for relative, original in requirements.items():
            self.assertEqual((PUBLIC / relative).read_bytes(), original)
            parts = Path(relative).parts
            repo_relative = Path(*parts[1:]) if parts[0] == "common" else Path(*parts[2:])
            self.assertEqual((self.repo / repo_relative).read_bytes(), original)

    def test_renderer_fails_closed_when_a_reviewed_clause_changes(self):
        changed = (PUBLIC / "task-prompt.txt").read_text(encoding="utf-8").replace(
            "merge die fertigen Änderungen", "integrate the finished changes"
        )
        with self.assertRaises(workspace_policy.PolicyRenderError):
            workspace_policy.render("task-prompt.txt", changed)


if __name__ == "__main__":
    unittest.main()
