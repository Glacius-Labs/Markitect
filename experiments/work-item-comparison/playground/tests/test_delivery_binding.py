"""Offline finite-order admission rejects changed budget and executable."""
from copy import deepcopy
import hashlib
import json
from pathlib import Path
import sys
import unittest
import tempfile
from datetime import datetime, timezone
from unittest.mock import patch
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
import delivery_binding as binding
class DeliveryBindingTests(unittest.TestCase):
    def setUp(self):
        self.plan=json.loads((Path(__file__).resolve().parents[1]/"pilots/conventional-variant-adapter-delivery-20261009/plan.json").read_text())
    def test_changed_check_accounting_cannot_admit(self):
        self.plan["maxRuntimeStartupChecks"]=3
        with self.assertRaisesRegex(ValueError,"six startup-check"):
            binding.verify(self.plan,Path(sys.executable))
    def test_other_executable_cannot_admit(self):
        with self.assertRaisesRegex(ValueError,"executable binding changed"):
            binding.verify(self.plan,Path(sys.executable))

    def test_closed_delivery_cannot_restart_even_with_exact_existing_executable(self):
        executable=Path("C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe")
        with self.assertRaisesRegex(ValueError,"delivery order is closed"):
            binding.verify(self.plan,executable)

    def snapshot_plan(self):
        return json.loads((Path(__file__).resolve().parents[1]/"pilots/conventional-workspace-snapshot-execution-20261009/plan.json").read_bytes())

    def test_snapshot_v2_exact_frozen_plan_admits_without_native_dispatch(self):
        class Clock(datetime):
            @classmethod
            def now(cls,tz=None):return cls(2026,10,9,21,0,tzinfo=timezone.utc)
        executable=Path("C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe")
        with patch.object(binding,"datetime",Clock), patch.object(Path,"exists",return_value=False):
            raw=binding.verify(self.snapshot_plan(),executable)
        self.assertEqual(hashlib.sha256(raw).hexdigest(),binding.SNAPSHOT_ORDER_SHA256)

    def test_snapshot_changed_target_or_extra_case_declines(self):
        for mutation in ("target","case"):
            plan=self.snapshot_plan()
            if mutation=="target":plan["completionTarget"]="main_merge"
            else:plan["trajectories"].append(deepcopy(plan["trajectories"][0]))
            with self.assertRaisesRegex(ValueError,"one frozen workspace"):
                binding.verify(plan,Path(sys.executable))

    def test_snapshot_reservation_is_single_use_in_external_owned_fixture(self):
        with tempfile.TemporaryDirectory() as folder:
            destination=Path(folder)/"case"
            ledger=Path(folder)/"ledger.json"
            plan={"id":"conventional-workspace-snapshot-execution-20261009","executionLedgerPath":str(ledger)}
            with patch.object(binding,"SNAPSHOT_DESTINATION",str(destination)),patch.object(binding,"SNAPSHOT_LEDGER",str(ledger)):
                binding.reserve(plan,destination)
                first=ledger.read_bytes()
                with self.assertRaises(FileExistsError):binding.reserve(plan,destination)
                self.assertEqual(ledger.read_bytes(),first)
