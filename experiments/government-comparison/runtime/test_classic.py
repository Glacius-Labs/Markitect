"""Bounded checks for Scientist's Classic native adapter mapping."""
import sys
import unittest
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
import classic

PACKET = Path(r"C:\Users\Consiliari\.codex\worktrees\standard-operating-model\Markitect\.artifacts\classic-readiness\v0.14.1-r1")
BINARY = PACKET / "artifacts" / "markitect-v0.14.1-windows-amd64.exe"


class ClassicAdapterTests(unittest.TestCase):
    def test_replaced_packet_manifest_is_rejected_before_native_call(self):
        with tempfile.TemporaryDirectory(prefix="scientist-packet-tamper-") as temporary:
            packet = Path(temporary)
            (packet / "checksums.sha256").write_text("0" * 64 + "  fake.json\n", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "external Scientist pin"):
                classic.inspect_packet(packet)

    def test_packet_hashes_and_runtime_pins(self):
        inspection = classic.inspect_packet(PACKET)
        self.assertEqual(inspection["packetFilesVerified"], 113)
        self.assertEqual(inspection["runtimeSourceSha"], "7dbd599c81540c8203a1b7f83afbc335174f4f1f")
        self.assertEqual(inspection["heldSourceSha"], "c91363b7ac4decbe87212ff0f588b5451581a152")
        self.assertEqual(inspection["binary"]["sha256"], classic.EXPECTED_BINARY_SHA256)
        self.assertNotEqual(inspection["runtimeSourceSha"], inspection["heldSourceSha"])

    def test_version_probe_runs_actual_packet_binary(self):
        result = classic.run_native({"argv": classic.probe_command(BINARY), "cwd": str(PACKET),
                                     "expectedReport": "text"}, timeout_seconds=30)
        self.assertFalse(result["timedOut"])
        self.assertEqual(result["returnCode"], 0)
        self.assertIn(b"Markitect 0.14.1 (windows/amd64)", result["stdout"])
        self.assertEqual(result["runtimeSha256"], classic.EXPECTED_BINARY_SHA256)

    def test_apply_requires_external_exact_digest_gate(self):
        with self.assertRaisesRegex(ValueError, "reviewed_digest"):
            classic.native_command("apply", binary_path=BINARY, repo_path=PACKET,
                                   config_path="examples/canonical-projection/canonical.yaml",
                                   revision="a" * 40, runtime_path=PACKET / "classic-readiness.json",
                                   execute_report=PACKET / "classic-readiness.json")

    def test_documented_flow_uses_explicit_argv_and_no_retries(self):
        spec = classic.native_command("audit", binary_path=BINARY, repo_path=PACKET,
                                      config_path="examples/canonical-projection/canonical.yaml",
                                      revision="a" * 40, runtime_path=PACKET / "classic-readiness.json")
        self.assertEqual(spec["automaticRetries"], 0)
        self.assertEqual(spec["timeoutSeconds"], 180)
        self.assertIn("controller-audit", spec["argv"])
        self.assertEqual(spec["cwd"], str(PACKET.resolve()))


if __name__ == "__main__":
    unittest.main()
