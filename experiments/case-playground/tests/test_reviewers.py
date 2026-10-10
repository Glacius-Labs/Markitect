"""Reviewer commands, prompt, schema validation and runs with the fake reviewer CLI."""

import json
import os
import sys
import tempfile
import unittest
from pathlib import Path

from playground import codex_agent, reviewers

PLAYGROUND = Path(__file__).resolve().parents[1]
FAKE = [sys.executable, str(Path(__file__).resolve().parent / "fake_reviewer.py")]
SCHEMA = json.loads((PLAYGROUND / "evaluation" / "common" / "reviewer-schema.json").read_text(encoding="utf-8"))
TEMPLATE = (PLAYGROUND / "evaluation" / "common" / "reviewer-prompt.md").read_text(encoding="utf-8")
CONFIG = json.loads((PLAYGROUND / "evaluation" / "config.json").read_text(encoding="utf-8"))
TOKEN = "fake-oauth-token-value-for-tests"
VALID = {"findings": [{"category": "regression", "severity": "high", "item": "B01", "rule": None,
                       "evidence": [{"path": "app.py", "line": 12}], "detail": "finish is not idempotent"}],
         "obligations": {"covered": 3, "total": 4}, "notes": ""}


def wave(root: Path, **overrides) -> dict:
    data = {"station": 2, "items": ["B02", "B03"], "earlierItems": ["B01"],
            "itemTexts": {"B02": "- B02 (B01): import", "B03": "- B03 (B01): summary"},
            "rules": {"README.md": "# Readinglog\nRules."}, "groundTruth": None, "lastMessage": "Done.",
            "diff": "diff --git a/app.py b/app.py\n+new\n", "backlog": "# Backlog\n- B01: x\n- B02 (B01): import\n",
            "repo": str(root / "repo"), "bundle": str(root / "bundle")}
    data.update(overrides)
    return data


class SchemaTests(unittest.TestCase):
    def test_valid_answer_passes(self):
        self.assertEqual(reviewers.validate(VALID, SCHEMA), [])

    def test_invalid_answers_are_named(self):
        bad = json.loads(json.dumps(VALID))
        bad["findings"][0]["category"] = "style"
        bad["findings"][0]["extra"] = 1
        del bad["obligations"]["covered"]
        bad["notes"] = None
        errors = "\n".join(reviewers.validate(bad, SCHEMA))
        for expected in ("'style' is not one of", "unexpected extra", "missing covered", "$.notes: expected string"):
            self.assertIn(expected, errors)
        self.assertIn("below 0", "\n".join(reviewers.validate({**VALID, "obligations": {"covered": -1, "total": 1}},
                                                             SCHEMA)))

    def test_schema_is_strict_compatible_and_matches_categories(self):
        def objects(node):
            if isinstance(node, dict):
                if node.get("type") == "object":
                    yield node
                for value in node.values():
                    yield from objects(value)
        for node in objects(SCHEMA):  # OpenAI strict mode: all properties required, nothing extra
            self.assertIs(node.get("additionalProperties"), False)
            self.assertEqual(sorted(node["required"]), sorted(node["properties"]))
        finding = SCHEMA["properties"]["findings"]["items"]["properties"]
        self.assertEqual(tuple(finding["category"]["enum"]), reviewers.CATEGORIES)
        self.assertEqual(tuple(finding["severity"]["enum"]), reviewers.SEVERITIES)

    def test_parse_json_text_tolerates_a_fence(self):
        self.assertEqual(reviewers.parse_json_text("```json\n{\"a\": 1}\n```"), {"a": 1})
        self.assertEqual(reviewers.parse_json_text("Here: {\"a\": 2} done"), {"a": 2})


class PromptTests(unittest.TestCase):
    def test_extract_items_from_the_v1_backlog(self):
        backlog = (PLAYGROUND / "cases" / "readinglog" / "BACKLOG.md").read_text(encoding="utf-8")
        ids = ["B01", "B02", "B03", "B04", "B05", "B06", "B07", "B08", "B09", "B10", "B11", "B12"]
        found = reviewers.extract_items(backlog, ["B02", "B04", "B12", "B99"], ids)
        self.assertTrue(found["B02"].startswith("- B02 (B01): README atomic"))
        self.assertNotIn("B03", found["B02"])
        self.assertIn("integrate baseline/regression tests", found["B04"])
        self.assertIn("grep alone is not acceptance", found["B12"])
        self.assertIsNone(found["B99"])

    def test_items_under_headings_end_at_the_next_heading(self):
        backlog = "# Backlog\n\n### B13 Optional pages\n\nPages become optional.\nMore text.\n\n### B14 Tests\nx\n"
        found = reviewers.extract_items(backlog, ["B13"], ["B13", "B14"])
        self.assertEqual(found["B13"], "### B13 Optional pages\n\nPages become optional.\nMore text.")

    def test_template_is_neutral(self):
        lowered = TEMPLATE.lower()
        for word in ("conventional", "arm", "method a", "codex", "claude"):
            self.assertNotRegex(lowered, rf"\b{word}\b")
        for category in reviewers.CATEGORIES:
            self.assertIn(f"`{category}`", TEMPLATE)
        self.assertIn(".markitect/", TEMPLATE)
        self.assertIn("realizations of the requirements", TEMPLATE)

    def test_compose_prompt_holds_the_inputs_and_respects_the_byte_limit(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            truth = {"case": "x", "rules": [{"id": "R1", "text": "audit"}],
                     "wave": {"station": "S2", "items": [{"id": "B02", "obligations": ["a", "b"]}]}}
            prompt = reviewers.compose_prompt(TEMPLATE, wave(root, groundTruth=truth))
            for text in ("- Released items: B02, B03", "- Released in earlier waves: B01", "- B03 (B01): summary",
                         "### README.md", "\"obligations\"", "Done.", "+new", str(root / "repo")):
                self.assertIn(text, prompt)
            self.assertTrue(prompt.startswith(TEMPLATE.rstrip()))
            self.assertIn("No ground truth exists", reviewers.compose_prompt(TEMPLATE, wave(root)))
            big = reviewers.compose_prompt(TEMPLATE, wave(root, diff="+x\n" * 100_000), max_bytes=20_000)
            self.assertLessEqual(len(big.encode("utf-8")), 20_000)
            self.assertIn("full text in", big)
            self.assertIn("wave.diff", big)

    def test_wave_ground_truth_and_obligation_count(self):
        truth = {"case": "c", "rules": [{"id": "R1", "text": "t"}],
                 "waves": [{"station": "S1", "items": [{"id": "B01", "obligations": ["a", "b"]}]},
                           {"station": 2, "items": [{"id": "B02", "obligations": ["c"]}, {"id": "B03"}]}]}
        self.assertEqual(reviewers.wave_ground_truth(truth, 1)["wave"]["items"][0]["id"], "B01")
        self.assertEqual(reviewers.obligation_count(reviewers.wave_ground_truth(truth, 2)), 1)
        self.assertIsNone(reviewers.wave_ground_truth(truth, 3))
        self.assertIsNone(reviewers.wave_ground_truth(None, 1))


class CommandTests(unittest.TestCase):
    def test_codex_command(self):
        argv = reviewers.codex_command(CONFIG["reviewers"]["codex"], schema_path=Path("s.json"),
                                       output_path=Path("o.json"), prompt="P")
        self.assertEqual(argv[:2], ["codex", "exec"])
        for pair in (["--sandbox", "read-only"], ["-m", "gpt-6.1-sol"], ["-c", 'model_reasoning_effort="high"'],
                     ["--output-schema", "s.json"], ["-o", "o.json"], ["--", "P"]):
            self.assertIn(pair, [argv[i:i + 2] for i in range(len(argv) - 1)])
        self.assertIn("--json", argv)
        self.assertIn("--skip-git-repo-check", argv)
        self.assertNotIn("--dangerously-bypass-approvals-and-sandbox", argv)

    def test_claude_command(self):
        argv = reviewers.claude_command(CONFIG["reviewers"]["claude"], schema=SCHEMA, repo_dir=Path("repo"),
                                        prompt="P")
        pairs = [argv[i:i + 2] for i in range(len(argv) - 1)]
        self.assertEqual(argv[:2], ["claude", "-p"])
        for pair in (["--output-format", "json"], ["--model", "claude-opus-5-5"], ["--tools", "Read,Grep,Glob"],
                     ["--allowedTools", "Read,Grep,Glob"], ["--setting-sources", ""], ["--add-dir", "repo"],
                     ["--", "P"]):
            self.assertIn(pair, pairs)
        self.assertEqual(json.loads(argv[argv.index("--json-schema") + 1]), SCHEMA)
        self.assertNotIn("--bare", argv)  # --bare ignores OAuth tokens
        self.assertNotIn("--dangerously-skip-permissions", argv)


class RunReviewerTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name).resolve()
        os.chmod(self.root, 0o755)
        (self.root / "repo").mkdir()
        (self.root / "repo" / "app.py").write_text("print('x')\n", encoding="utf-8")
        self.bundle = self.root / "bundle"
        self.prompt = reviewers.compose_prompt(TEMPLATE, wave(self.root), max_bytes=20_000)
        reviewers.write_bundle(self.bundle, wave(self.root), self.prompt, SCHEMA)
        self.auth = self.root / "auth.json"
        self.auth.write_text('{"fake": "login"}', encoding="utf-8")
        if codex_agent.container_mode():
            codex_agent.give_to_agent(self.root, recursive=True)

    def review(self, provider: str, plan: dict | None = None, **cfg_overrides) -> dict:
        plan_file = self.bundle / "fake-reviewer-plan.json"  # read by the fake from its working directory
        plan_file.write_text(json.dumps(plan or {}), encoding="utf-8")
        cfg = {**CONFIG["reviewers"][provider], **cfg_overrides}
        return reviewers.run_reviewer(provider, cfg, prompt=self.prompt, schema=SCHEMA,
                                      schema_path=self.bundle / "reviewer-schema.json",
                                      repo_dir=self.root / "repo", bundle_dir=self.bundle,
                                      out_dir=self.root / "out" / provider, codex_auth=self.auth,
                                      claude_token=TOKEN, executable=FAKE)

    def all_output(self) -> str:
        return "".join(p.read_text(encoding="utf-8", errors="replace") for p in (self.root / "out").rglob("*")
                       if p.is_file())

    def test_codex_review_in_a_fresh_home_with_only_the_login(self):
        record = self.review("codex")
        self.assertEqual(record["status"], "ok", record)
        self.assertEqual(len(record["findings"]), 2)
        self.assertEqual(record["obligations"], {"covered": 1, "total": 2})
        self.assertIn("home=auth.json ", record["notes"])
        self.assertEqual(record["usage"], {"input": 500, "cachedInput": 100, "output": 50})
        self.assertFalse(record["loginRefreshed"])
        self.assertEqual(record["model"], "gpt-6.1-sol")
        argv = json.loads((self.root / "out" / "codex" / "argv.json").read_text(encoding="utf-8"))
        self.assertEqual(argv[-1], "<prompt.md>")
        for base in (Path(tempfile.gettempdir()), reviewers.REVIEWER_HOMES):  # homes are removed after the run
            self.assertFalse(list(base.glob("mpg-reviewer-home-*/.codex/auth.json")))

    def test_claude_review_gets_the_token_only_through_its_environment(self):
        record = self.review("claude")
        self.assertEqual(record["status"], "ok", record)
        self.assertIn("token=present", record["notes"])
        self.assertEqual(record["resolvedModels"], ["fake-claude-model"])
        self.assertEqual(record["usage"]["input"], 700)
        self.assertNotIn(TOKEN, self.all_output())
        self.assertNotIn(TOKEN, json.dumps(record))

    def test_both_reviewers_see_the_identical_prompt(self):
        notes = [self.review(name)["notes"] for name in ("codex", "claude")]
        self.assertEqual({note.split("prompt=")[1] for note in notes}.__len__(), 1)

    def test_invalid_garbage_and_fenced_answers(self):
        record = self.review("codex", {"codex": "invalid"})
        self.assertEqual(record["status"], "invalid")
        self.assertTrue(record["validationErrors"])
        self.assertTrue((self.root / "out" / "codex" / "output.txt").read_text(encoding="utf-8"))
        self.assertEqual(self.review("claude", {"claude": "garbage"})["status"], "invalid")
        self.assertEqual(self.review("claude", {"claude": "fenced"})["status"], "ok")

    def test_crash_error_result_and_timeout_are_recorded_not_raised(self):
        crashed = self.review("codex", {"codex": "crash"})
        self.assertEqual((crashed["status"], crashed["exitCode"]), ("error", 3))
        errored = self.review("claude", {"claude": "error-result"})
        self.assertEqual(errored["status"], "error")
        hung = self.review("codex", {"codex": "hang"}, timeoutSeconds=2)
        self.assertEqual((hung["status"], hung["timedOut"]), ("error", True))

    def test_missing_credentials(self):
        record = reviewers.run_reviewer("claude", CONFIG["reviewers"]["claude"], prompt="p", schema=SCHEMA,
                                        schema_path=self.bundle / "reviewer-schema.json", repo_dir=self.root,
                                        bundle_dir=self.bundle, out_dir=self.root / "out" / "x",
                                        claude_token=None, executable=FAKE)
        self.assertEqual((record["status"], record["error"]), ("error", "Claude token missing"))

    def test_cli_version(self):
        self.assertEqual(reviewers.cli_version("codex", FAKE), "fake-reviewer 1.0")


class SummaryTests(unittest.TestCase):
    def test_summary_and_agreement(self):
        def finding(item, category, severity="high"):
            return {"category": category, "severity": severity, "item": item, "rule": None, "evidence": [],
                    "detail": ""}
        codex = {"status": "ok", "findings": [finding("B01", "regression"), finding("b02", "missed_obligation", "low")],
                 "obligations": {"covered": 1, "total": 2}}
        claude = {"status": "ok", "findings": [finding("B02", "missed_obligation"), finding(None, "false_claim")],
                  "obligations": {"covered": 2, "total": 2}}
        summary = reviewers.summarize(codex)
        self.assertEqual((summary["findings"], summary["byCategory"]["regression"], summary["bySeverity"]["low"]),
                         (2, 1, 1))
        agree = reviewers.agreement({"codex": codex, "claude": claude})
        self.assertEqual((agree["both"], agree["codexOnly"], agree["claudeOnly"]), (1, 1, 1))
        self.assertEqual(agree["shared"], [{"item": "B02", "category": "missed_obligation"}])
        self.assertIsNone(reviewers.agreement({"codex": codex, "claude": {"status": "invalid"}}))
        self.assertIsNone(reviewers.summarize({"status": "error"})["findings"])


if __name__ == "__main__":
    unittest.main()
