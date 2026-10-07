"""Runner tool-config contracts; these tests inspect argv and never launch Codex."""
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parent))
import runner


class RunnerToolConfigurationTests(unittest.TestCase):
    def make_executable_pin(self, directory):
        executable = Path(directory) / "codex-pin.bin"
        executable.write_bytes(b"synthetic runner pin; never executed")
        digest = hashlib.sha256(executable.read_bytes()).hexdigest()
        return executable, patch.object(runner, "EXPECTED_SHA256", digest)

    def test_ordinary_argv_enables_tools_and_keeps_each_requested_sandbox(self):
        with tempfile.TemporaryDirectory(prefix="runner-tools-argv-") as directory:
            root = Path(directory)
            executable, pin_override = self.make_executable_pin(root)
            actor = root / "actor"
            actor.mkdir()
            with pin_override:
                for sandbox in ("read-only", "workspace-write"):
                    argv = runner.prospective_argv(executable, actor, sandbox=sandbox)
                    config = self.argv_config(argv)
                    self.assertEqual(config["features.shell_tool"], True)
                    self.assertEqual(config["features.unified_exec"], True)
                    self.assertEqual(argv[argv.index("--sandbox") + 1], sandbox)
                    self.assertEqual(argv[0], str(executable.resolve()))

    def test_ordinary_argv_refuses_if_either_execution_tool_is_disabled(self):
        with tempfile.TemporaryDirectory(prefix="runner-tools-disabled-") as directory:
            root = Path(directory)
            executable, pin_override = self.make_executable_pin(root)
            actor = root / "actor"
            actor.mkdir()
            with pin_override:
                for disabled in ("features.shell_tool", "features.unified_exec"):
                    with self.subTest(disabled=disabled), patch.dict(runner.CONFIG, {disabled: False}):
                        with self.assertRaisesRegex(ValueError, r"ordinary[- ]tools.*shell_tool.*unified_exec"):
                            runner.prospective_argv(executable, actor, sandbox="read-only")

    def test_metadata_route_explicitly_disables_tools_and_cannot_select_exec(self):
        with tempfile.TemporaryDirectory(prefix="runner-metadata-route-") as directory:
            executable, pin_override = self.make_executable_pin(directory)
            with pin_override:
                argv = runner.metadata_argv(executable)
            self.assertEqual(argv[1:3], ["app-server", "--stdio"])
            self.assertNotIn("exec", argv)
            config = self.argv_config(argv, prefix=("-c",))
            self.assertIs(config["features.shell_tool"], False)
            self.assertIs(config["features.unified_exec"], False)

    def test_inspection_binds_updated_config_hash_without_claiming_serving_identity(self):
        with tempfile.TemporaryDirectory(prefix="runner-config-pin-") as directory:
            executable, pin_override = self.make_executable_pin(directory)
            with pin_override:
                pin = runner.inspect(executable)
            self.assertEqual(pin["sha256"], hashlib.sha256(executable.read_bytes()).hexdigest())
            self.assertEqual(pin["prospectiveConfig"]["features.shell_tool"], True)
            self.assertEqual(pin["prospectiveConfig"]["features.unified_exec"], True)
            expected_config_hash = hashlib.sha256(
                json.dumps(pin["prospectiveConfig"], sort_keys=True, separators=(",", ":")).encode()
            ).hexdigest()
            self.assertEqual(pin["configSha256"], expected_config_hash)
            self.assertIsNone(pin["actualModel"])

    @staticmethod
    def argv_config(argv, prefix=("--config",)):
        values = {}
        width = len(prefix)
        for index in range(len(argv) - width):
            if tuple(argv[index:index + width]) == prefix:
                key, raw = argv[index + width].split("=", 1)
                values[key] = json.loads(raw)
        return values


if __name__ == "__main__":
    unittest.main()
