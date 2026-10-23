"""Discrimination self-check of the Roombook oracle; see README.md.

    python -B selfcheck.py [--reference DIR] [--mutants DIR]

Always checks that every cited Definition exists in the snapshots and that
degenerate command lines (silent, always-error, empty-object, crashing) fail.
With --reference the reference passes every stage; with --mutants (layout
<stage>/<name>/app.py) each mutant fails at least one check. Keep references
and mutants outside the repository.
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from checks import REG  # noqa: E402
from oracle_kit import selfcheck_main  # noqa: E402

if __name__ == "__main__":
    sys.exit(selfcheck_main(REG, Path(__file__).resolve().parents[1]))
