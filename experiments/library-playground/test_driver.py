"""Controlled protocol fixtures only: no library implementation."""

import base64
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

from driver import Limits, Process, TransportFailure


class TransportTests(unittest.TestCase):
    def run_fixture(self, script, expected_error=None, request=None, limits=None):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "fixture.py"
            path.write_text(script, encoding="utf-8")
            process = Process([sys.executable, "-I", str(path)], directory,
                              limits or Limits(request_seconds=0.5, process_seconds=3, output_bytes=4096))
            try:
                def execute():
                    response = process.request(request or {"id": "q", "op": "inspectMember", "args": {"memberId": "m"}})
                    process.finish()
                    return response
                if expected_error:
                    with self.assertRaisesRegex(TransportFailure, expected_error):
                        execute()
                else:
                    self.assertEqual(execute(), {"id": "q", "status": "refused", "reason": "Unknown member"})
            finally:
                process.close()
            trace = process.trace()
            self.assertIsNotNone(trace["exit_code"])
            self.assertLessEqual(sum(len(base64.b64decode(v)) for v in trace["streams_base64"].values()),
                                 process.limits.output_bytes)
            return trace

    def test_valid_and_exact_capture(self):
        trace = self.run_fixture('import sys\nfor line in sys.stdin:\n print(\'{"id":"q","status":"refused","reason":"Unknown member"}\',flush=True)\n')
        self.assertEqual(len(trace["pairs"]), 1)
        self.assertIn(b'"id":"q"', base64.b64decode(trace["pairs"][0]["response_base64"]))

    def test_missing(self):
        self.run_fixture('import sys\nsys.stdin.readline()\n', 'missing|premature')

    def test_timeout(self):
        self.run_fixture('import sys,time\nsys.stdin.readline()\ntime.sleep(5)\n', 'timeout')

    def test_process_deadline(self):
        self.run_fixture('import sys,time\nsys.stdin.readline()\ntime.sleep(5)\n', 'timeout|deadline',
                         limits=Limits(request_seconds=1, process_seconds=0.15))

    def test_nonzero_exit(self):
        self.run_fixture('import sys\nsys.stdin.readline()\nprint(\'{"id":"q","status":"refused","reason":"Unknown member"}\',flush=True)\nsys.exit(7)\n', 'nonzero exit')

    def test_extra_response(self):
        self.run_fixture('import sys\nsys.stdin.readline()\nprint(\'{"id":"q","status":"refused","reason":"Unknown member"}\',flush=True)\nprint("{}",flush=True)\n', 'extra stdout')

    def test_invalid_json_utf8_shape_duplicates_nonfinite(self):
        for payload in [b'{oops}\n', b'[]\n', b'\xff\n', b'{"id":"q","id":"r"}\n', b'{"n":NaN}\n']:
            with self.subTest(payload=payload):
                self.run_fixture('import sys\nsys.stdin.readline()\nsys.stdout.buffer.write(' + repr(payload) + ')\nsys.stdout.flush()\n', 'malformed|JSON object')

    def test_unterminated_line(self):
        self.run_fixture('import sys\nsys.stdin.readline()\nprint("{}",end="",flush=True)\n', 'unterminated|missing')

    def test_parser_depth_failure_retains_transport_trace(self):
        payload = b'{"nested":[]}\n'
        # Native parser depth depends on Python/platform stack limits. Inject
        # that parser outcome after real transport, rather than assume a depth.
        with patch('driver.strict_json', side_effect=RecursionError('controlled parser depth')):
            trace = self.run_fixture('import sys\nsys.stdin.readline()\nsys.stdout.buffer.write(' + repr(payload) + ')\nsys.stdout.flush()\n',
                                     'malformed response.*controlled parser depth')
        self.assertEqual(base64.b64decode(trace["pairs"][0]["response_base64"]), payload)

    def test_stdout_and_stderr_flood(self):
        for stream in ('stdout', 'stderr'):
            with self.subTest(stream=stream):
                self.run_fixture('import sys\nsys.stdin.readline()\nsys.' + stream + '.write("x"*100000)\nsys.' + stream + '.flush()\n', 'output byte limit')

    def test_no_exit_at_eof(self):
        self.run_fixture('import sys,time\nsys.stdin.readline()\nprint(\'{"id":"q","status":"refused","reason":"Unknown member"}\',flush=True)\nsys.stdin.read()\ntime.sleep(5)\n', 'EOF exit timeout')

    def test_request_bounds(self):
        self.run_fixture('import sys\nsys.stdin.read()\n', 'bounded JSONL', request={"data": "x" * 20000})

    def test_limit_validation(self):
        for kwargs in ({"request_seconds": float("inf")}, {"process_seconds": 0},
                       {"output_bytes": True}, {"request_bytes": 2**30}):
            with self.assertRaises(ValueError):
                Limits(**kwargs)

    def test_provider_environment_is_not_inherited(self):
        script = 'import os,sys\nsys.stdin.readline()\nreason="Unknown member" if "PLAY_TEST_CREDENTIAL" not in os.environ else "inherited"\nprint(\'{"id":"q","status":"refused","reason":"\'+reason+\'"}\',flush=True)\n'
        with patch.dict("os.environ", {"PLAY_TEST_CREDENTIAL": "test-only-sentinel"}):
            self.run_fixture(script)


if __name__ == "__main__":
    unittest.main()
