"""Focused offline tests for the bounded private stderr excerpt collector."""

from __future__ import annotations

import json
from pathlib import Path
import tempfile
import threading
import unittest
from unittest.mock import patch

from redacted_stderr import RedactedStderrCollector


class RedactedStderrTests(unittest.TestCase):
    def make_collector(self, known_paths: tuple[str, ...] = ()):
        self.stop = threading.Event()
        return RedactedStderrCollector(self.stop, known_paths=known_paths)

    def excerpt(self, collector: RedactedStderrCollector) -> dict:
        return json.loads(collector.render_private().decode("utf-8"))

    def test_useful_message_survives_and_finish_is_metadata_only(self):
        collector = self.make_collector()
        collector.feed(b"ERROR service unavailable\n")

        metadata = collector.finish()
        encoded = json.dumps(metadata, ensure_ascii=False).encode("utf-8")
        self.assertTrue(self.stop.is_set())
        self.assertNotIn(b"service unavailable", encoded)
        self.assertFalse(any("text" in row for row in metadata.get("lines", [])))

        excerpt = self.excerpt(collector)
        self.assertEqual(excerpt["lines"][0]["line"], 1)
        self.assertIn("service unavailable", excerpt["lines"][0]["text"])
        self.assertLessEqual(len(collector.render_private()), 2048)

    def test_first_nonempty_chunk_stops_even_when_every_line_is_suppressed(self):
        collector = self.make_collector()
        original = collector._redact
        stop_states = []

        def assert_stopped_before_redaction(text):
            stop_states.append(self.stop.is_set())
            return original(text)

        with patch.object(collector, "_redact", side_effect=assert_stopped_before_redaction):
            collector.feed(b"Authorization: Bearer benign-canary-value\n")

        self.assertTrue(self.stop.is_set())
        self.assertEqual(stop_states, [True])
        output = collector.render_private()
        self.assertNotIn(b"benign-canary-value", output)
        excerpt = json.loads(output.decode("utf-8"))
        self.assertEqual(excerpt["lines"], [])
        self.assertTrue(excerpt["dropped"])

    def test_credential_patterns_are_suppressed_as_whole_lines(self):
        samples = (
            b"Authorization: Bearer bearer-canary\n",
            b"Authorization: Basic basic-canary\n",
            b"token=token-canary\n",
            b'{"secret":"secret-canary"}\n',
            b"password: password-canary\n",
            b"authcode=authcode-canary\n",
            b"api_key=api-key-canary\n",
            b"private_key=private-key-canary\n",
            b"privateKey=private-key-camel-canary\n",
            b"signing_key=signing-key-canary\n",
            b"eyJhbGciOiJub25lIn0.eyJzdWIiOiJqd3QtY2FuYXJ5In0.signature\n",
            b"-----BEGIN PRIVATE KEY-----\n",
            b"AWS_SECRET_ACCESS_KEY=cloud-canary\n",
        )
        for sample in samples:
            with self.subTest(kind=sample.split(b"=", 1)[0][:18]):
                collector = self.make_collector()
                collector.feed(sample)
                self.assertNotIn(sample.strip(), collector.render_private())
                self.assertEqual(self.excerpt(collector)["lines"], [])

    def test_url_email_absolute_and_known_paths_are_redacted(self):
        known = "C:\\Users\\Scientist\\Documents\\Probe"
        collector = self.make_collector((known,))
        collector.feed(
            ("contact https://example.test/path?q=private alice@example.test "
             + known + " and /home/scientist/private\n").encode("utf-8")
        )

        line = self.excerpt(collector)["lines"][0]
        self.assertNotIn("https://", line["text"])
        self.assertNotIn("alice@example.test", line["text"])
        self.assertNotIn(known, line["text"])
        self.assertNotIn("/home/scientist/private", line["text"])
        self.assertTrue(line["redacted"])

        path_cases = (
            ("forward-slash UNC", "UNC //server/share/team folder/file.txt"),
            ("quoted Windows path with spaces", 'File: "C:\\Users\\Synthetic User\\Work Files\\diag.txt"'),
            ("quoted POSIX path with spaces", 'File: "/home/synthetic user/work files/diag.txt"'),
        )
        for label, message in path_cases:
            with self.subTest(path_kind=label):
                candidate = self.make_collector()
                candidate.feed((message + "\n").encode("utf-8"))
                rendered = self.excerpt(candidate)["lines"][0]
                self.assertIn("[path]", rendered["text"])
                self.assertNotIn(message.split(" ", 1)[1], rendered["text"])
                self.assertIn("path", rendered["redacted"])

    def test_structured_json_suppression_and_redaction_failure_fail_closed(self):
        collector = self.make_collector()
        collector.feed(
            b'{"message":"broken"\n'
            b'{"api\\u005fkey":"escaped-canary"}\n'
            b'{"message":"ordinary diagnostic"}\n'
        )

        excerpt = self.excerpt(collector)
        self.assertEqual([item["line"] for item in excerpt["lines"]], [3])
        self.assertEqual(excerpt["lines"][0]["text"], '{"message":"ordinary diagnostic"}')
        self.assertNotIn(b"escaped-canary", collector.render_private())
        self.assertEqual(
            [(item["line"], item["reason"]) for item in excerpt["dropped"]],
            [(1, "malformed-structured-line"), (2, "encoded-structured-line")],
        )

        prefixed_cases = (
            (
                "prefixed escaped password JSON",
                b'2026-10-08T12:00:00Z ERROR {"pass\\u0077ord":"prefixed-password-canary"}\n',
                "encoded-structured-line",
            ),
            (
                "prefixed malformed JSON",
                b'2026-10-08T12:00:00Z ERROR {"message":"broken"\n',
                "malformed-structured-line",
            ),
            (
                "encoded field in JSON array",
                b'ERROR [{"api\\u005fkey":"array-key-canary"}]\n',
                "encoded-structured-line",
            ),
            (
                "ordinary prefixed valid JSON",
                b'2026-10-08T12:00:00Z ERROR {"message":"ordinary prefixed diagnostic"}\n',
                None,
            ),
        )
        for label, sample, expected_reason in prefixed_cases:
            with self.subTest(prefixed_json=label):
                candidate = self.make_collector()
                candidate.feed(sample)
                candidate_excerpt = self.excerpt(candidate)
                if expected_reason is None:
                    self.assertEqual(len(candidate_excerpt["lines"]), 1)
                    self.assertEqual(candidate_excerpt["lines"][0]["text"], sample.decode().rstrip("\n"))
                else:
                    self.assertEqual(candidate_excerpt["lines"], [])
                    self.assertEqual(candidate_excerpt["dropped"][0]["reason"], expected_reason)

        failed = self.make_collector()
        with patch.object(failed, "_redact", side_effect=RuntimeError("synthetic-only")):
            failed.feed(b"ordinary diagnostic\n")
        failure_excerpt = self.excerpt(failed)
        self.assertEqual(failure_excerpt["lines"], [])
        self.assertEqual(failure_excerpt["dropped"], [{"line": 1, "reason": "redaction-failure"}])

    def test_valid_sgr_is_removed_but_other_escape_and_controls_drop_line(self):
        collector = self.make_collector()
        collector.feed(b"\x1b[31mERROR\x1b[0m harmless\n")
        collector.feed(b"ERROR hidden\x1b]0;title-canary\x07\n", cleanup=True)
        collector.feed(b"ERROR bidi\xe2\x80\xaehidden\n", cleanup=True)

        excerpt = self.excerpt(collector)
        self.assertEqual(len(excerpt["lines"]), 1)
        self.assertEqual(excerpt["lines"][0]["text"], "ERROR harmless")
        self.assertTrue(excerpt["lines"][0]["sgrRemoved"])
        self.assertEqual(len(excerpt["dropped"]), 2)
        self.assertNotIn(b"title-canary", collector.render_private())

    def test_chunk_boundaries_crlf_and_split_utf8_are_reassembled(self):
        collector = self.make_collector()
        collector.feed(b"first ")
        collector.feed(b"caf\xc3", cleanup=True)
        collector.feed(b"\xa9\r", cleanup=True)
        collector.feed(b"\nsecond line\n", cleanup=True)

        excerpt = self.excerpt(collector)
        self.assertEqual([item["line"] for item in excerpt["lines"]], [1, 2])
        self.assertEqual(excerpt["lines"][0]["text"], "first café")
        self.assertEqual(excerpt["lines"][1]["text"], "second line")

    def test_invalid_utf8_and_unterminated_fragment_are_not_rendered(self):
        collector = self.make_collector()
        collector.feed(b"valid line\ninvalid \xff data\nunfinished-canary", cleanup=True)

        excerpt = self.excerpt(collector)
        self.assertEqual([item["text"] for item in excerpt["lines"]], ["valid line"])
        self.assertNotIn(b"unfinished-canary", collector.render_private())
        self.assertTrue(excerpt.get("incompleteLineDropped"))
        self.assertTrue(any(item["reason"] == "invalid-utf8" for item in excerpt["dropped"]))

    def test_only_first_four_physical_lines_count_even_when_dropped(self):
        collector = self.make_collector()
        collector.feed(
            b"Authorization: Bearer first-canary\n"
            b"two\nthree\nfour\nfifth-must-not-appear\n"
        )

        excerpt = self.excerpt(collector)
        self.assertEqual([item["line"] for item in excerpt["lines"]], [2, 3, 4])
        self.assertNotIn(b"fifth-must-not-appear", collector.render_private())
        self.assertNotIn(b"first-canary", collector.render_private())
        self.assertTrue(excerpt.get("lineCapReached"))

    def test_input_cap_excludes_a_line_cut_by_the_1024_byte_boundary(self):
        collector = self.make_collector()
        prefix = b"safe\n" * 3
        collector.feed(prefix + b"x" * (1024 - len(prefix)))

        excerpt = self.excerpt(collector)
        self.assertEqual([item["line"] for item in excerpt["lines"]], [1, 2, 3])
        self.assertTrue(excerpt.get("inputTruncated"))
        self.assertNotIn(b"x" * 20, collector.render_private())

    def test_output_cap_drops_whole_expanded_lines_with_visible_marker(self):
        collector = self.make_collector()
        for index in range(4):
            collector.feed(("\"" * 250 + "\n").encode(), cleanup=index > 0)

        output = collector.render_private()
        self.assertLessEqual(len(output), 2048)
        excerpt = json.loads(output.decode("utf-8"))
        self.assertTrue(excerpt.get("outputTruncated"))
        self.assertTrue(any(item["reason"] == "output-cap" for item in excerpt["dropped"]))
        self.assertLess(len(excerpt["lines"]), 4)
        self.assertTrue(all(item["text"] == '"' * 250 for item in excerpt["lines"]))
        self.assertTrue(set(item["line"] for item in excerpt["lines"]).isdisjoint(
            item["line"] for item in excerpt["dropped"] if item["reason"] == "output-cap"
        ))

    def test_private_write_is_exclusive_and_receipt_does_not_contain_excerpt(self):
        collector = self.make_collector()
        collector.feed(b"useful synthetic diagnostic\n")
        with tempfile.TemporaryDirectory(prefix="stderr-private-test-") as temp:
            directory = Path(temp).resolve()
            path = collector.write_private_excerpt(directory)
            self.assertEqual(path.name, "redacted-stderr.json")
            self.assertIn(b"useful synthetic diagnostic", path.read_bytes())
            receipt = json.dumps(collector.finish(), ensure_ascii=False).encode("utf-8")
            self.assertNotIn(b"useful synthetic diagnostic", receipt)
            with self.assertRaises((FileExistsError, RuntimeError, ValueError)):
                collector.write_private_excerpt(directory)


if __name__ == "__main__":
    unittest.main()
