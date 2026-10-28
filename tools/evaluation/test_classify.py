"""Offline tests of the failure classifier and the token parser on recorded runs and transcripts (testdata/)."""
import json
import pathlib
import shutil
import sys
import tempfile
import unittest

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import classify  # noqa: E402
import tokens  # noqa: E402

FAILURES = HERE / "testdata" / "failures"
TRANSCRIPT = HERE / "testdata" / "claude-transcript-subagents"


class ClassifierTest(unittest.TestCase):
    """Each fixture is a recorded run: the E1 smoke's records and real Claude Code API errors (testdata/README.md)."""

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)

    def tearDown(self):
        self._tmp.cleanup()

    def copy(self, name):
        target = self.tmp / f"{name}-{len(list(self.tmp.iterdir()))}"
        shutil.copytree(FAILURES / name, target)
        return target

    def edited(self, name, change):
        run = self.copy(name)
        record = json.loads((run / "suite.json").read_text(encoding="utf-8"))
        change(record)
        (run / "suite.json").write_text(json.dumps(record), encoding="utf-8")
        return run

    def test_recorded_provider_failures_are_void(self):
        for name, error in (("provider-overloaded", "server_error"), ("provider-usage-limit", "rate_limit")):
            with self.subTest(name):
                result = classify.classify(FAILURES / name)
                self.assertEqual((result["class"], result["kind"]), ("void", "provider"))
                self.assertEqual(result["evidence"][0]["error"], error)

    def test_a_provider_error_the_session_recovered_from_is_a_result(self):
        self.assertEqual(classify.classify(FAILURES / "provider-recovered")["class"], "result")

    def test_a_provider_error_of_a_subagent_alone_is_a_result(self):
        self.assertEqual(classify.classify(FAILURES / "provider-subagent")["class"], "result")

    def test_a_provider_error_in_a_run_that_ended_normally_is_a_result(self):
        run = self.edited("provider-overloaded", lambda r: r.update(exitCode=0))
        self.assertEqual(classify.classify(run)["class"], "result")

    def test_a_variant_command_that_never_ran_is_void(self):
        # E1 probe-01: Git Bash rewrote the variant's path; the sandbox itself was fine.
        result = classify.classify(FAILURES / "infrastructure-not-started")
        self.assertEqual((result["class"], result["kind"]), ("void", "infrastructure"))
        self.assertIn("can't open file", result["evidence"])

    def test_sandbox_failures_and_a_refused_endpoint_are_void(self):
        run = self.edited("result-k03", lambda r: (r.update(validForCounting=False), r["sandbox"].update(error="SandboxError: the egress gateway stopped during the run")))
        self.assertEqual((classify.classify(run)["class"], classify.classify(run)["kind"]), ("void", "infrastructure"))
        run = self.edited("result-k03", lambda r: (r.update(validForCounting=False), r["sandbox"].update(residue=["eval-x-sandbox"])))
        self.assertIn("residue", classify.classify(run)["evidence"])
        run = self.edited("result-k03", lambda r: r["sandbox"]["egress"]["refused"].update({"statsig.anthropic.com": 2}))
        result = classify.classify(run)
        self.assertEqual(result["class"], "void")
        self.assertTrue(result["stopsSeries"])
        (self.tmp / "nothing").mkdir()
        self.assertEqual(classify.classify(self.tmp / "nothing")["kind"], "infrastructure")

    def test_a_variant_failure_without_a_signature_is_a_result(self):
        # E1 k-01: K's capture dropped sources; no signature named it before the series.
        self.assertEqual(classify.classify(FAILURES / "variant-capture-k01", None, "K")["class"], "result")

    def test_a_variant_runtime_signature_written_before_the_series_voids_the_run(self):
        run = self.edited("variant-capture-k01", lambda r: r["reported"].update(status="failed", cause="RuntimeError: the host lost its session"))
        signatures = {"variants": {"M": [{"id": "M-host-session-lost", "path": "reported.cause", "pattern": r"host lost its session"}]}}
        result = classify.classify(run, signatures, "M")
        self.assertEqual((result["class"], result["kind"]), ("void", "variant runtime"))
        self.assertEqual(result["evidence"]["signature"], "M-host-session-lost")
        # The same signature belongs to M only: the same run of another variant is a result.
        self.assertEqual(classify.classify(run, signatures, "K")["class"], "result")

    def test_an_ordinary_run_is_a_result_and_host_or_wrong_model_runs_are_invalid(self):
        self.assertEqual(classify.classify(FAILURES / "result-k03", None, "K"), {"class": "result", "kind": None, "evidence": None, "stopsSeries": False})
        run = self.edited("result-k03", lambda r: r.update(conditions="host", validForCounting=False))
        self.assertEqual(classify.classify(run)["class"], "invalid")
        run = self.edited("result-k03", lambda r: r.update(modelMatches=False))
        self.assertEqual(classify.classify(run)["class"], "invalid")

    def test_a_timeout_is_a_result(self):
        run = self.edited("result-k03", lambda r: r.update(timedOut=True, exitCode=None))
        self.assertEqual(classify.classify(run)["class"], "result")


class TokenParserTest(unittest.TestCase):
    """A recorded Claude Code session with 22 workflow subagents, reduced to structure and usage (testdata/README.md)."""

    # Counted by hand: grep every assistant line, keep each message ID once, add its usage
    # (`grep -rh '"type":"assistant"' | sed ... | sort -u | awk`, see testdata/README.md):
    # 27 responses: 3 in the main session and 24 in its 22 subagents.
    HAND = {"responses": 27, "inputTokens": 56, "cacheCreationTokens": 413772, "cacheReadTokens": 638602, "outputTokens": 2364,
            "thinkingTokens": 1368}

    def test_matches_the_hand_count_over_every_transcript(self):
        count = tokens.count(TRANSCRIPT)
        self.assertEqual({k: count[k] for k in self.HAND}, self.HAND)
        self.assertEqual(count["files"], 24)
        self.assertEqual(count["subagentResponses"], 24)
        self.assertEqual(count["unreadableLines"], 0)

    def test_matches_the_session_s_own_totals(self):
        count = tokens.count(TRANSCRIPT)
        self.assertEqual(count["sessionTotals"], {k: self.HAND[k] for k in ("inputTokens", "cacheCreationTokens", "cacheReadTokens",
                                                                            "outputTokens", "thinkingTokens")})

    def test_the_main_session_alone_is_far_less(self):
        main = tempfile.TemporaryDirectory()
        self.addCleanup(main.cleanup)
        shutil.copy(TRANSCRIPT / "2f3759fb-d940-4e12-85db-3bc5f6156bd6.jsonl", main.name)
        count = tokens.count(main.name)
        self.assertEqual(count["responses"], 3)
        self.assertLess(count["cacheCreationTokens"], 413772 // 5)

    def test_a_response_seen_twice_counts_once(self):
        with tempfile.TemporaryDirectory() as tmp:
            line = {"type": "assistant", "message": {"id": "msg_1", "model": "m", "usage": {"input_tokens": 3, "output_tokens": 10}}}
            later = {"type": "assistant", "message": {"id": "msg_1", "model": "m", "usage": {"input_tokens": 3, "output_tokens": 12}}}
            (pathlib.Path(tmp) / "a.jsonl").write_text(json.dumps(line) + "\n" + json.dumps(later) + "\nnot json\n")
            # The same response in a stream-json file: the stream and the transcript name it alike.
            (pathlib.Path(tmp) / "stream.jsonl").write_text(json.dumps({**line, "session_id": "s"}) + "\n")
            count = tokens.count(tmp)
        self.assertEqual((count["responses"], count["inputTokens"], count["outputTokens"], count["unreadableLines"]), (1, 3, 12, 1))


if __name__ == "__main__":
    unittest.main()
