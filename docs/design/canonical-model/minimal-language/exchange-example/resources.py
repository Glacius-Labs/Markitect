"""Explicit probe budgets; exhaustion is not a model-language violation."""
from __future__ import annotations

from functools import wraps

from name_profile import ProbeLimitError


DEFAULT_OBJECT_DEPTH = 128


def resource_checked(phase):
    def decorate(action):
        @wraps(action)
        def checked(*args, **kwargs):
            try:
                return action(*args, **kwargs)
            except RecursionError as error:
                raise ProbeLimitError(f"probe runtime recursion limit in {phase}; evaluation not completed") from error
            except MemoryError as error:
                raise ProbeLimitError(f"probe runtime memory exhaustion in {phase}; evaluation not completed") from error
        return checked
    return decorate


def check_object_depth(depth, maximum):
    if depth > maximum:
        raise ProbeLimitError(f"probe object-depth budget {maximum} exceeded; evaluation not completed")


def overall_status(source, canonical_records, lock, self_test):
    statuses = [source, lock, self_test, *(record["status"] for record in canonical_records.values())]
    if "failed" in statuses:
        return "failed"
    return "passed" if canonical_records and all(status == "passed" for status in statuses) else "not_completed"
