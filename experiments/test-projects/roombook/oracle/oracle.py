"""Assess a Roombook realization at one snapshot stage; see README.md.

    python -B oracle.py --realization PATH/TO/realization --stage 1|2|3 [--python CMD]

Prints a JSON report with a pass or fail per check; exits 0 only if all pass.
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from checks import REG  # noqa: E402
from oracle_kit import oracle_main  # noqa: E402

if __name__ == "__main__":
    sys.exit(oracle_main(REG))
