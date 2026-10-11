"""The one table of exit codes for `host run`, `assess`, `compare` and `study`.

The codes keep a method outcome apart from failures of the harness, the environment and
the product, which are never outcomes of the method (DEC-010). `--help` of these
commands prints TABLE; the README's "Exit codes" section copies it.
"""
from __future__ import annotations

from typing import Iterable

OK, METHOD, INVALID, PREFLIGHT = 0, 1, 2, 3
HARNESS, ENVIRONMENT, PRODUCT = 10, 11, 12
TIMEOUT, INTERRUPTED = 124, 130

TABLE = {
    OK: "completed. host run: every wave ran and class none; assess, compare: written; study: every step 0",
    METHOD: "method outcome: the run stopped early with class none (agent time used up)",
    INVALID: "invalid input or refused by a rule: manifest, study file, options, folder, pre-registration, "
             "reviewer models, fairness mismatch",
    PREFLIGHT: "study preflight failed, including the image and binary builds",
    HARNESS: "harness failure: runner error, snapshot or wave release failed, assess-error.txt, study-error.txt, "
             "an unexpected error of the command",
    ENVIRONMENT: "environment failure: logins, no session id, host status setup-failed, start-failed or "
                 "wait-failed (image build included), a container killed (exit 137) under its memory limit, "
                 "image changed",
    PRODUCT: "product failure: the Markitect binary does not build, setup blocked by the product, markitect "
             "check could not run",
    TIMEOUT: "host safety timeout",
    INTERRUPTED: "interrupted",
}
# A run report's class (report.py) and the code it gives; class "none" is the method's.
CLASS_CODES = {"harness": HARNESS, "environment": ENVIRONMENT, "product": PRODUCT}
# A study's code is the first of these among its step codes.
STUDY_ORDER = (INTERRUPTED, TIMEOUT, HARNESS, ENVIRONMENT, PRODUCT, METHOD, OK)
KILLED = 137  # a container's exit after SIGKILL, as the kernel's OOM killer sends under --memory


def help_text() -> str:
    """TABLE as an argparse epilog."""
    return "exit codes:\n" + "\n".join(f"  {code:>3}  {text}" for code, text in TABLE.items())


def _host_status(status: str | None, failure_class: str | None = None) -> int | None:
    """The code of a host status that is not `completed`: the container did not finish.
    `harness-error` is an unexpected error of the command (the code __main__ exits
    with); a setup failure is the environment's unless `failure_class` names another
    class (a Markitect binary that does not build is the product's)."""
    if status == "host-timeout":
        return TIMEOUT
    if status == "host-interrupted":
        return INTERRUPTED
    if status == "harness-error":
        return HARNESS
    if status == "setup-failed" and failure_class in CLASS_CODES:
        return CLASS_CODES[failure_class]
    return None if status == "completed" else ENVIRONMENT  # setup-failed, start-failed, wait-failed


def host_run(status: str | None, container_exit: int | None, run_class: str | None,
             failure_class: str | None = None, memory_limited: bool = False) -> int:
    """`host run`: the host status, then a container killed under its memory limit
    (exit 137: the environment's), then the class in results/report.json, then the
    runner's own 0 (every wave ran) or 1 (stopped early). No readable report, an unknown
    class or any other runner exit is a harness failure."""
    code = _host_status(status, failure_class)
    if code is not None:
        return code
    if container_exit == KILLED and memory_limited:
        return ENVIRONMENT
    if run_class in CLASS_CODES:
        return CLASS_CODES[run_class]
    if run_class == "none" and container_exit in (OK, METHOD):
        return container_exit
    return HARNESS


def assess(status: str | None, container_exit: int | None, memory_limited: bool = False) -> int:
    """`assess`: a finished container that did not exit 0 failed to write the assessment
    (assess-error.txt), unless it was killed under its memory limit (the environment's);
    failed holdouts or reviews inside a written one do not count."""
    code = _host_status(status)
    if code is not None:
        return code
    if container_exit == KILLED and memory_limited:
        return ENVIRONMENT
    return OK if container_exit == 0 else HARNESS


def study(codes: Iterable[int]) -> int:
    """The worst step code in STUDY_ORDER. A step refused inside a study (2) counts as a
    harness failure, as does any code outside the order: the preflight should have caught it."""
    ranks = [STUDY_ORDER.index(code) if code in STUDY_ORDER else STUDY_ORDER.index(HARNESS) for code in codes]
    return STUDY_ORDER[min(ranks, default=STUDY_ORDER.index(OK))]
