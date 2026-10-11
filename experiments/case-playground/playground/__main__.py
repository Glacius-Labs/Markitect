"""Entry point.

  python -m playground run ...       one trajectory (inside the run container)
  python -m playground host ...      build, stage and run one manifest (host)
  python -m playground assess ...    assess a finished run in its own container (host)
  python -m playground compare ...   compare two assessed runs (host)
  python -m playground study ...     preflight, both arms, assessments and comparisons (host)

Exit codes: outcome.py. An error no command handled is a harness failure (10), never
the method outcome 1 that Python's own traceback exit would suggest.
"""

from __future__ import annotations

import sys
import traceback

from . import outcome

COMMANDS = ("run", "host", "assess", "compare", "study")
USAGE = "usage: python -m playground {run|host|assess|compare|study} ..."


def main(argv: list[str] | None = None) -> int:
    args = list(sys.argv[1:] if argv is None else argv)
    if not args or args[0] not in COMMANDS:
        print(USAGE, file=sys.stderr)
        return outcome.INVALID
    try:
        return _command(args)
    except KeyboardInterrupt:
        print("interrupted", file=sys.stderr)
        return outcome.INTERRUPTED
    except Exception:
        traceback.print_exc()
        return outcome.HARNESS


def _command(args: list[str]) -> int:
    if args[0] == "run":
        from . import runner
        return runner.main(args[1:])
    if args[0] == "assess":
        from . import evaluate
        return evaluate.main(args[1:])
    if args[0] == "compare":
        from . import compare
        return compare.main(args[1:])
    if args[0] == "study":
        from . import study
        return study.main(args[1:])
    from . import host
    return host.main(args[1:])


if __name__ == "__main__":
    raise SystemExit(main())
