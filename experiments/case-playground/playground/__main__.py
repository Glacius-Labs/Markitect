"""Entry point.

  python -m playground run ...       one trajectory (inside the run container)
  python -m playground host ...      build, stage and run one manifest (host)
  python -m playground assess ...    assess a finished run in its own container (host)
  python -m playground compare ...   compare two assessed runs (host)
"""

from __future__ import annotations

import sys

COMMANDS = ("run", "host", "assess", "compare")
USAGE = "usage: python -m playground {run|host|assess|compare} ..."


def main(argv: list[str] | None = None) -> int:
    args = list(sys.argv[1:] if argv is None else argv)
    if not args or args[0] not in COMMANDS:
        print(USAGE, file=sys.stderr)
        return 2
    if args[0] == "run":
        from . import runner
        return runner.main(args[1:])
    if args[0] == "assess":
        from . import evaluate
        return evaluate.main(args[1:])
    if args[0] == "compare":
        from . import compare
        return compare.main(args[1:])
    from . import host
    return host.main(args[1:])


if __name__ == "__main__":
    raise SystemExit(main())
