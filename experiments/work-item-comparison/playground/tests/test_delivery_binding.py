"""Offline finite-order admission rejects changed budget and executable."""
from copy import deepcopy
import hashlib
import json
from pathlib import Path
import sys
import unittest
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
