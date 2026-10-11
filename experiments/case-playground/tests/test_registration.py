"""Pre-registration against temporary Git repositories, and reviewer independence."""

import io
import json
import tarfile
import tempfile
import unittest
from pathlib import Path

from playground import registration
from tests import evaluation_repo
from tests.evaluation_repo import committed, git, runner


class ObserveTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.base = Path(temp.name).resolve()
        self.repo = committed(self.base / "repo", case="readinglog2")

    def observe(self, root: Path | None = None) -> dict:
        root = root or self.repo
        return registration.observe(root, runner(root))

    def test_a_clean_evaluation_folder_is_registered_with_its_tree(self):
        record = self.observe()
        self.assertEqual(record["status"], "registered")
        self.assertEqual(record["commit"], git(self.repo, "rev-parse", "HEAD"))
        self.assertEqual(record["evaluationTree"], git(self.repo, "rev-parse", "HEAD:evaluation"))
        self.assertRegex(record["recordedAt"], r"^\d{4}-\d\d-\d\dT")
        self.assertEqual(set(record), {"status", "commit", "evaluationTree", "recordedAt"})

    def test_a_changed_or_untracked_file_is_dirty_and_listed(self):
        (self.repo / "evaluation" / "config.json").write_text("{}\n", encoding="utf-8")
        (self.repo / "evaluation" / "readinglog2" / "new-holdout.py").write_text("", encoding="utf-8")
        (self.repo / "elsewhere.txt").write_text("not evaluation\n", encoding="utf-8")
        record = self.observe()
        self.assertEqual(record["status"], "dirty")
        self.assertIsNone(record["evaluationTree"])
        self.assertEqual(sorted(record["paths"]), ["evaluation/config.json", "evaluation/readinglog2/new-holdout.py"])
        self.assertIn("evaluation/config.json", registration.refusal(record))
        self.assertIn("uncommitted or untracked", registration.refusal(record))

    def test_a_staged_but_uncommitted_change_is_dirty(self):
        (self.repo / "evaluation" / "common" / "reviewer-prompt.md").write_text("changed\n", encoding="utf-8")
        git(self.repo, "add", "-A")
        self.assertEqual(self.observe()["status"], "dirty")

    def test_ignored_files_do_not_count(self):
        (self.repo / ".gitignore").write_text("__pycache__/\n", encoding="utf-8")
        git(self.repo, "add", ".gitignore")
        git(self.repo, "commit", "-q", "-m", "ignore")
        (self.repo / "evaluation" / "readinglog2" / "__pycache__").mkdir()
        (self.repo / "evaluation" / "readinglog2" / "__pycache__" / "holdout.pyc").write_bytes(b"x")
        self.assertEqual(self.observe()["status"], "registered")

    def test_a_subfolder_playground_registers_its_own_evaluation_folder(self):
        nested = self.base / "nested"
        (nested / "experiments").mkdir(parents=True)
        committed(nested)  # the repository root holds evaluation/ ...
        playground = nested / "experiments" / "case-playground"
        (playground / "evaluation").mkdir(parents=True)  # ... and a subfolder its own, untracked
        (playground / "evaluation" / "config.json").write_text("{}\n", encoding="utf-8")
        record = self.observe(playground)
        self.assertEqual(record["status"], "dirty")
        self.assertTrue(all("case-playground/evaluation" in path or path.startswith("evaluation")
                            for path in record["paths"]))
        git(nested, "add", "-A")
        git(nested, "commit", "-q", "-m", "sub")
        record = self.observe(playground)
        self.assertEqual(record["evaluationTree"], git(nested, "rev-parse",
                                                       "HEAD:experiments/case-playground/evaluation"))

    def test_no_checkout_or_no_git_is_refused(self):
        plain = self.base / "plain"
        (plain / "evaluation").mkdir(parents=True)
        record = self.observe(plain)
        self.assertEqual(record["status"], "no-checkout")
        self.assertIn("not a Git checkout", record["error"])

        def missing(_args):
            raise registration.RegistrationError("git is not installed or not on PATH")
        record = registration.observe(self.repo, missing)
        self.assertEqual((record["status"], record["error"]), ("no-checkout", "git is not installed or not on PATH"))
        self.assertIn("not pre-registered: git is not installed", registration.refusal(record))

    def test_an_uncommitted_evaluation_folder_in_a_checkout_is_not_registered(self):
        empty = self.base / "empty"
        empty.mkdir()
        git(empty, "init", "-q")
        (empty / "README.md").write_text("x\n", encoding="utf-8")
        git(empty, "add", "-A")
        git(empty, "commit", "-q", "-m", "no evaluation")
        record = self.observe(empty)
        self.assertEqual(record["status"], "no-checkout")
        self.assertIn("evaluation/ is not committed", record["error"])

    def test_the_default_runner_is_real_git(self):
        self.assertEqual(registration.observe(self.repo)["status"], "registered")
        self.assertEqual(registration.git_in(self.repo)(["rev-parse", "--is-inside-work-tree"]).stdout.strip(),
                         b"true")


class ArchiveTests(unittest.TestCase):
    def test_the_archive_holds_the_committed_bytes_after_the_working_tree_changed(self):
        with tempfile.TemporaryDirectory() as temp:
            repo = committed(Path(temp).resolve() / "repo", case="readinglog2")
            prompt = repo / "evaluation" / "common" / "reviewer-prompt.md"
            committed_prompt = prompt.read_bytes()
            tree = registration.observe(repo, runner(repo))["evaluationTree"]
            prompt.write_bytes(b"edited after the run started\r\n")
            (repo / "evaluation" / "config.json").write_text('{"reviewers": {}}', encoding="utf-8")
            data = registration.archive(repo, tree, runner(repo))
            with tarfile.open(fileobj=io.BytesIO(data)) as tar:
                names = sorted(m.name for m in tar.getmembers() if m.isfile())
                self.assertEqual(tar.extractfile("common/reviewer-prompt.md").read(), committed_prompt)
            self.assertIn("readinglog2/reference/SECRET.md", names)
            config = registration.read_json(repo, tree, "config.json", runner(repo))
            self.assertEqual(config["reviewers"]["codex"]["model"], "gpt-6.1-sol")
            target = Path(temp) / "staged"
            written = registration.extract(
                data, target, lambda parts: not registration.ignored(parts, ("reference", "validate.py")))
            self.assertEqual((target / "common" / "reviewer-prompt.md").read_bytes(), committed_prompt)
            self.assertNotIn("readinglog2/reference/SECRET.md", written)
            self.assertFalse((target / "readinglog2" / "validate.py").exists())
            self.assertTrue(registration.tree_exists(repo, tree, runner(repo)))
            self.assertFalse(registration.tree_exists(repo, "0" * 40, runner(repo)))
            self.assertFalse(registration.tree_exists(repo, git(repo, "rev-parse", "HEAD"), runner(repo)))
            with self.assertRaises(registration.RegistrationError):
                registration.read_blob(repo, tree, "missing.json", runner(repo))

    def test_extract_skips_links_and_paths_that_leave_the_target(self):
        buffer = io.BytesIO()
        with tarfile.open(fileobj=buffer, mode="w") as tar:
            for name, kind in (("ok.txt", tarfile.REGTYPE), ("../escape.txt", tarfile.REGTYPE),
                               ("link", tarfile.SYMTYPE)):
                info = tarfile.TarInfo(name)
                info.type, info.linkname = kind, "/etc/passwd" if kind == tarfile.SYMTYPE else ""
                info.size = 0 if kind == tarfile.SYMTYPE else 2
                tar.addfile(info, io.BytesIO(b"hi") if kind == tarfile.REGTYPE else None)
        with tempfile.TemporaryDirectory() as temp:
            target = Path(temp) / "t"
            self.assertEqual(registration.extract(buffer.getvalue(), target, lambda parts: True), ["ok.txt"])
            self.assertFalse((Path(temp) / "escape.txt").exists())


class CodeStateTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.repo = committed(Path(temp.name).resolve() / "repo")
        (self.repo / "playground").mkdir()
        (self.repo / "playground" / "assess.py").write_text("x = 1\n", encoding="utf-8")
        git(self.repo, "add", "-A")
        git(self.repo, "commit", "-q", "-m", "code")
        self.paths = ("playground", "cases/common/checks")

    def state(self, root: Path | None = None) -> dict:
        root = root or self.repo
        return registration.code_state(root, self.paths, runner(root))

    def test_committed_code_is_clean_with_its_commit(self):
        record = self.state()
        self.assertEqual((record["commit"], record["dirty"], record["changed"]),
                         (git(self.repo, "rev-parse", "HEAD"), False, []))
        self.assertEqual(record["paths"], list(self.paths))

    def test_changed_or_untracked_code_is_dirty_and_other_paths_do_not_count(self):
        (self.repo / "notes.md").write_text("elsewhere\n", encoding="utf-8")
        self.assertIs(self.state()["dirty"], False)
        (self.repo / "playground" / "assess.py").write_text("x = 2\n", encoding="utf-8")
        (self.repo / "cases" / "common" / "checks").mkdir(parents=True)
        (self.repo / "cases" / "common" / "checks" / "new.py").write_text("", encoding="utf-8")
        record = self.state()
        self.assertIs(record["dirty"], True)
        self.assertEqual(sorted(record["changed"]), ["cases/common/checks/new.py", "playground/assess.py"])

    def test_no_checkout_is_unknown(self):
        with tempfile.TemporaryDirectory() as temp:
            record = self.state(Path(temp))
        self.assertEqual((record["commit"], record["dirty"]), (None, None))
        self.assertTrue(record["error"])


class ReviewerTests(unittest.TestCase):
    CONFIG = {"reviewers": {"codex": {"model": "gpt-6.1-sol"}, "claude": {"model": "claude-opus-5-5"}}}

    def test_model_ids_compare_normalized(self):
        for given, key in ((" Claude-Opus-5-5[1m] ", "claude-opus-5-5"),
                           ("gpt-6.1-sol", "gpt-6.1-sol"),
                           ("anthropic/claude-opus-5-5", "claude-opus-5-5"),
                           ("openai/gpt-6.1-sol", "gpt-6.1-sol"),
                           ("openrouter/anthropic/claude-opus-5-5", "claude-opus-5-5"),
                           ("us.anthropic.claude-opus-5-5-20261001-v1:0", "claude-opus-5-5"),
                           ("anthropic.claude-opus-5-5-v2:0", "claude-opus-5-5"),
                           ("global.anthropic.claude-sonnet-4-5-20250929-v1:0", "claude-sonnet-4-5"),
                           ("claude-opus-5-5-20261001", "claude-opus-5-5"),
                           ("claude-opus-5-5@20261001", "claude-opus-5-5"),
                           ("gpt-4o-2024-08-06", "gpt-4o"),
                           ("Claude-Opus-5-5-20261001[1m]", "claude-opus-5-5"),
                           ("gpt-6-luna-v2", "gpt-6-luna-v2")):  # only Bedrock's -vN:N is a version suffix
            with self.subTest(model=given):
                self.assertEqual(registration.model_key(given), key)
        self.assertIsNone(registration.model_key(""))
        self.assertIsNone(registration.model_key(None))

    def test_aliases_are_not_full_model_ids(self):
        for model in ("opus", "Sonnet", "haiku", "fable", "opusplan", "default", "best", "opus[1m]",
                      "anthropic/opus", "claude-opus", "gpt", "codex-mini-latest", "claude-3-5-sonnet-latest",
                      "claude-opus-20261001"):
            with self.subTest(model=model):
                reason = registration.alias_reason(model)
                self.assertIsNotNone(reason)
                self.assertIn("is a model alias, not a full model id", reason)
        for model in ("claude-opus-5-5", "claude-opus-5-5[1m]", "gpt-6.1-sol", "gpt-6-luna", "o3", "o4-mini",
                      "us.anthropic.claude-sonnet-4-5-20250929-v1:0", "fake-model-1", None, ""):
            with self.subTest(model=model):
                self.assertIsNone(registration.alias_reason(model))

    def test_an_alias_or_a_dated_id_of_the_same_model_still_clashes(self):
        arms = registration.arm_models({"agent": {"model": "anthropic/Claude-Opus-5-5-20261001"}},
                                       [{"role": "verifier", "model": "us.anthropic.claude-opus-5-5-v1:0"}])
        self.assertEqual(set(arms), {"claude-opus-5-5"})
        found = registration.clashes(self.CONFIG, ["codex", "claude"], arms)
        self.assertEqual([(c["reviewer"], c["arm"]) for c in found],
                         [("claude", "agent.model anthropic/Claude-Opus-5-5-20261001")])

    def test_arm_models_include_the_inner_model_and_every_role(self):
        manifest = {"agent": {"model": "gpt-6-luna"}, "markitect": {"innerModel": "GPT-6-LUNA"}}
        roles = [{"role": "verifier", "manager": None, "model": "claude-opus-5-5[1m]"},
                 {"role": "worker", "manager": "owner", "model": None}]
        arms = registration.arm_models(manifest, roles)
        self.assertEqual(set(arms), {"gpt-6-luna", "claude-opus-5-5"})
        self.assertEqual(arms["gpt-6-luna"], "agent.model gpt-6-luna")
        self.assertEqual(arms["claude-opus-5-5"], "role verifier claude-opus-5-5[1m]")

    def test_a_reviewer_model_that_an_arm_uses_clashes(self):
        arms = registration.arm_models({"agent": {"model": "gpt-6-luna"}, "markitect": {"innerModel": "GPT-6.1-Sol"}})
        found = registration.clashes(self.CONFIG, ["codex", "claude"], arms)
        self.assertEqual(found, [{"reviewer": "codex", "model": "gpt-6.1-sol",
                                  "arm": "markitect.innerModel GPT-6.1-Sol"}])
        self.assertEqual(registration.clashes(self.CONFIG, ["claude"], arms), [])  # a subset avoids it
        self.assertIn("reviewer codex uses model gpt-6.1-sol, as does markitect.innerModel",
                      registration.describe(found))
        roles = [{"role": "reviewer", "manager": "owner", "model": "Claude-Opus-5-5"}]
        found = registration.clashes(self.CONFIG, ["claude"], registration.arm_models({}, roles))
        self.assertEqual(found[0]["arm"], "role reviewer owner Claude-Opus-5-5")

    def test_the_shipped_examples_have_independent_reviewers(self):
        config = json.loads((evaluation_repo.PLAYGROUND / "evaluation" / "config.json").read_text(encoding="utf-8"))
        for path in sorted((evaluation_repo.PLAYGROUND / "examples").glob("study-*.json")):
            with self.subTest(study=path.name):
                study = json.loads(path.read_text(encoding="utf-8"))
                arms = registration.arm_models(study)
                self.assertEqual(registration.clashes(config, study["reviewers"], arms), [])


if __name__ == "__main__":
    unittest.main()
