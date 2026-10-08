"""Offline fake-lifecycle tests for the copied stderr consumer wiring."""

from __future__ import annotations

import json
from pathlib import Path
import tempfile
import threading
import time
import unittest
from unittest.mock import patch

import client


class FakePump:
    def __init__(self, alive: bool = False):
        self.alive = alive

    def is_alive(self) -> bool:
        return self.alive


class FakeProcess:
    def __init__(self, return_code: int | None = 0):
        self.return_code = return_code

    def poll(self) -> int | None:
        return self.return_code


class FakeStream:
    def __init__(self, chunks: list[bytes]):
        self.chunks = iter(chunks)

    def read1(self, _limit: int) -> bytes:
        return next(self.chunks, b"")


class TrackingLock:
    def __init__(self, event: threading.Event, lock: threading.Lock):
        self.event = event
        self.lock = lock
        self.event_seen_at_enter = False

    def __enter__(self):
        self.event_seen_at_enter = self.event.is_set()
        self.lock.acquire()
        return self

    def __exit__(self, exc_type, exc, traceback):
        self.lock.release()


class StderrConsumerWiringTests(unittest.TestCase):
    def new_consumer(self):
        seen = threading.Event()
        return seen, client.StderrConsumer(seen)

    def observe(self, consumer, raw: bytes, cleanup: threading.Event):
        consumer.observe_read(raw, threading.Lock(), cleanup)

    def complete_lifecycle(self, consumer, root: Path):
        cleanup = threading.Event()
        cleanup.set()
        return consumer.finish_after_cleanup(
            root, cleanup, [FakePump(False)], FakeProcess(0), time.monotonic() + 30
        )

    def test_observed_stderr_sets_event_before_admission_lock_and_rpc_is_blocked(self):
        seen, consumer = self.new_consumer()
        cleanup = threading.Event()
        underlying = threading.Lock()
        underlying.acquire()
        lock = TrackingLock(seen, underlying)
        failures = []

        def read_pump():
            try:
                consumer.observe_read(b"ordinary diagnostic\n", lock, cleanup)
            except Exception as error:  # Keep any error generic in test output.
                failures.append(type(error).__name__)

        thread = threading.Thread(target=read_pump, daemon=True)
        thread.start()
        self.assertTrue(seen.wait(2))
        self.assertTrue(thread.is_alive())
        underlying.release()
        thread.join(2)

        self.assertFalse(thread.is_alive())
        self.assertEqual(failures, [])
        self.assertTrue(lock.event_seen_at_enter)
        with self.assertRaises(ValueError):
            client.admit_rpc("initialize", [], seen.is_set(), False)

    def test_continuation_fragment_is_accepted_only_after_cleanup_starts(self):
        seen, consumer = self.new_consumer()
        cleanup = threading.Event()
        consumer.observe_read(b"split diagnostic ", threading.Lock(), cleanup)

        with self.assertRaises(ValueError):
            consumer.observe_read(b"continuation\n", threading.Lock(), cleanup)

        cleanup.set()
        consumer.observe_read(b"continuation\n", threading.Lock(), cleanup)
        with tempfile.TemporaryDirectory(prefix="stderr-wiring-fragment-") as temp:
            receipt = self.complete_lifecycle(consumer, Path(temp).resolve())
            private = Path(temp).resolve() / "private-redacted-stderr" / "redacted-stderr.json"
            self.assertEqual(receipt["excerptStatus"], "written-private")
            self.assertIn(b"split diagnostic continuation", private.read_bytes())
        self.assertTrue(seen.is_set())

    def test_quiescent_cleanup_writes_readable_private_excerpt_and_text_free_receipt(self):
        seen, consumer = self.new_consumer()
        cleanup = threading.Event()
        stream = FakeStream([b"ordinary diagnostic ", b"detail\n"])
        consumer.observe_read(stream.read1(1024), threading.Lock(), cleanup)
        cleanup.set()
        consumer.observe_read(stream.read1(1024), threading.Lock(), cleanup)

        with tempfile.TemporaryDirectory(prefix="stderr-wiring-private-") as temp:
            root = Path(temp).resolve()
            private = root / "private-redacted-stderr" / "redacted-stderr.json"
            writer_calls = []
            original_writer = consumer._collector.write_private_excerpt

            def count_writer(directory):
                writer_calls.append(1)
                return original_writer(directory)

            with patch.object(consumer._collector, "write_private_excerpt", side_effect=count_writer):
                receipt = consumer.finish_after_cleanup(
                    root, cleanup, [FakePump(False)], FakeProcess(0), time.monotonic() + 30
                )
                first_bytes = private.read_bytes()
                second_receipt = consumer.finish_after_cleanup(
                    root, cleanup, [FakePump(False)], FakeProcess(0), time.monotonic() + 30
                )
                self.assertEqual(second_receipt, receipt)
                self.assertEqual(len(writer_calls), 1)
                self.assertEqual(private.read_bytes(), first_bytes)
            self.assertEqual(receipt["excerptStatus"], "written-private")
            self.assertIn(b"ordinary diagnostic detail", first_bytes)
            self.assertNotIn(b"ordinary diagnostic detail", json.dumps(receipt).encode())
            self.assertTrue(seen.is_set())
            self.assertEqual(list((root / "private-redacted-stderr").iterdir()), [private])

        failed_seen, failed = self.new_consumer()
        failed_cleanup = threading.Event()
        failed.observe_read(b"harmless partial write probe\n", threading.Lock(), failed_cleanup)
        failed_cleanup.set()
        with tempfile.TemporaryDirectory(prefix="stderr-wiring-partial-") as temp:
            root = Path(temp).resolve()
            private = root / "private-redacted-stderr" / "redacted-stderr.json"
            writes = []

            def partial_then_fail(directory):
                writes.append(1)
                partial_path = Path(directory) / "redacted-stderr.json"
                partial_path.write_bytes(b"synthetic partial artifact")
                raise OSError("synthetic writer failure")

            with patch.object(failed._collector, "write_private_excerpt", side_effect=partial_then_fail):
                failure_receipt = failed.finish_after_cleanup(
                    root, failed_cleanup, [FakePump(False)], FakeProcess(0), time.monotonic() + 30
                )
                again = failed.finish_after_cleanup(
                    root, failed_cleanup, [FakePump(False)], FakeProcess(0), time.monotonic() + 30
                )
            self.assertEqual(failure_receipt["excerptStatus"], "private-output-failure-artifact-may-remain")
            self.assertEqual(again, failure_receipt)
            self.assertEqual(writes, [1])
            self.assertEqual(private.read_bytes(), b"synthetic partial artifact")
            self.assertNotIn(b"harmless partial write probe", json.dumps(failure_receipt).encode())
            self.assertTrue(failed_seen.is_set())

    def test_sensitive_line_is_fully_suppressed_in_private_file(self):
        seen, consumer = self.new_consumer()
        cleanup = threading.Event()
        consumer.observe_read(b"Authorization: Bearer synthetic-sensitive-canary\n", threading.Lock(), cleanup)
        cleanup.set()

        with tempfile.TemporaryDirectory(prefix="stderr-wiring-sensitive-") as temp:
            root = Path(temp).resolve()
            receipt = self.complete_lifecycle(consumer, root)
            private = root / "private-redacted-stderr" / "redacted-stderr.json"
            excerpt = json.loads(private.read_text(encoding="utf-8"))
            self.assertEqual(receipt["excerptStatus"], "written-private")
            self.assertEqual(excerpt["lines"], [])
            self.assertNotIn(b"synthetic-sensitive-canary", private.read_bytes())
            self.assertTrue(excerpt["dropped"])

    def test_no_trigger_or_incomplete_cleanup_quiescence_never_writes(self):
        cases = ("not-triggered", "cleanup-not-started", "alive-pump", "running-native", "expired-deadline")
        for case in cases:
            with self.subTest(lifecycle=case):
                seen, consumer = self.new_consumer()
                cleanup = threading.Event()
                pumps = [FakePump(False)]
                native = FakeProcess(0)
                deadline = time.monotonic() + 30
                if case == "cleanup-not-started":
                    consumer.feed(b"ordinary diagnostic\n")
                elif case != "not-triggered":
                    consumer.feed(b"ordinary diagnostic\n")
                    cleanup.set()
                    if case == "alive-pump":
                        pumps = [FakePump(True)]
                    elif case == "running-native":
                        native = FakeProcess(None)
                    elif case == "expired-deadline":
                        deadline = time.monotonic() - 1
                else:
                    cleanup.set()

                with tempfile.TemporaryDirectory(prefix="stderr-wiring-no-write-") as temp:
                    root = Path(temp).resolve()
                    receipt = consumer.finish_after_cleanup(root, cleanup, pumps, native, deadline)
                    self.assertFalse((root / "private-redacted-stderr").exists())
                    expected = {
                        "not-triggered": "not-triggered",
                        "cleanup-not-started": "not-written-cleanup-not-started",
                        "alive-pump": "not-written-not-quiescent",
                        "running-native": "not-written-not-quiescent",
                        "expired-deadline": "not-written-cleanup-deadline",
                    }[case]
                    self.assertEqual(receipt["excerptStatus"], expected)
                    if case == "alive-pump":
                        self.assertFalse(consumer._collector._finished)

    def test_output_cap_and_turn_or_tool_rpc_admission_remain_blocked(self):
        seen, consumer = self.new_consumer()
        for method in ("turn/start", "tools/call"):
            with self.subTest(pre_stderr_method=method):
                with self.assertRaises(ValueError):
                    client.admit_rpc(method, [], False, False)
        cleanup = threading.Event()
        sample = b'"' * 250 + b"\n"
        consumer.observe_read(sample * 4, threading.Lock(), cleanup)
        cleanup.set()

        with tempfile.TemporaryDirectory(prefix="stderr-wiring-cap-") as temp:
            root = Path(temp).resolve()
            receipt = consumer.finish_after_cleanup(
                root, cleanup, [FakePump(False)], FakeProcess(0), time.monotonic() + 30
            )
            private = root / "private-redacted-stderr" / "redacted-stderr.json"
            payload = private.read_bytes()
            excerpt = json.loads(payload.decode("utf-8"))
            self.assertEqual(receipt["excerptStatus"], "written-private")
            self.assertLessEqual(len(payload), 2048)
            self.assertTrue(excerpt["outputTruncated"])
            self.assertTrue(any(item["reason"] == "output-cap" for item in excerpt["dropped"]))

        for method in ("turn/start", "tools/call"):
            with self.subTest(method=method):
                with self.assertRaises(ValueError):
                    client.admit_rpc(method, [], seen.is_set(), False)

        # A pre-existing private child is never opened or overwritten.
        collision_seen, collision = self.new_consumer()
        collision_cleanup = threading.Event()
        collision.observe_read(b"collision test diagnostic\n", threading.Lock(), collision_cleanup)
        collision_cleanup.set()
        with tempfile.TemporaryDirectory(prefix="stderr-wiring-collision-") as temp:
            root = Path(temp).resolve()
            private_dir = root / "private-redacted-stderr"
            private_dir.mkdir()
            original_file = private_dir / "redacted-stderr.json"
            original_file.write_bytes(b"pre-existing artifact")
            collision_receipt = collision.finish_after_cleanup(
                root, collision_cleanup, [FakePump(False)], FakeProcess(0), time.monotonic() + 30
            )
            self.assertEqual(collision_receipt["excerptStatus"], "not-written-private-output-failure")
            self.assertEqual(original_file.read_bytes(), b"pre-existing artifact")


if __name__ == "__main__":
    unittest.main()
