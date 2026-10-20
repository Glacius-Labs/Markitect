"""Black-box check registry of the Apply trial projects.

The trials reuse the process and data helpers of the test-project oracles
(`experiments/test-projects/oracle_kit.py`): a check drives the realized
command line `python -B app.py --db PATH COMMAND ...` from `realization/` and
sees only stdout, the exit code, the data file and, where a model statement
binds internal structure, the named files. Unlike those cumulative stages, a
trial check names the expectation profiles it belongs to (for example the base
model, its change and its overspecified variant) and receives the profile's
parameters. A check whose prerequisite is missing is reported as skipped,
never as passed.
"""
import sys
import tempfile
import traceback
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "test-projects"))

from oracle_kit import CheckFailure, Env, crash_guard, expect_equal, fail  # noqa: E402,F401  (re-exported to the checks)


class Skip(Exception):
    """A prerequisite of the check, such as a previous realization, was not supplied."""


class Checks:
    """The checks of one trial project, ordered, each with its profiles and cited Definitions."""

    def __init__(self, project, profiles):
        self.project = project
        self.profiles = profiles  # {profile: {"snapshot": name, ...parameters}}
        self.checks = []

    def check(self, check_id, profiles, definitions, title):
        def register(function):
            if any(known["id"] == check_id for known in self.checks):
                raise ValueError("duplicate check " + check_id)
            unknown = set(profiles) - set(self.profiles)
            if unknown or not profiles:
                raise ValueError("unknown or missing profile for %s: %s" % (check_id, sorted(unknown)))
            if not definitions or any(definition.count("/") < 1 for definition in definitions):
                raise ValueError("a check cites at least one package/id Definition: " + check_id)
            self.checks.append({"id": check_id, "profiles": list(profiles), "definitions": list(definitions),
                                "title": title, "run": function})
            return function
        return register

    def selected(self, profile):
        return [known for known in self.checks if profile in known["profiles"]]

    def assess(self, realization, profile, python, extra=None):
        parameters = dict(self.profiles[profile], **(extra or {}))
        results = []
        for known in self.selected(profile):
            with tempfile.TemporaryDirectory(prefix="trial-oracle-") as root:
                env = Env(realization, python, root)
                try:
                    known["run"](env, parameters)
                    status, detail = "pass", ""
                except Skip as reason:
                    status, detail = "skip", str(reason)
                except CheckFailure as error:
                    status, detail = "fail", str(error)
                except Exception:  # an oracle defect must never read as a pass
                    status, detail = "fail", "oracle error: " + traceback.format_exc(limit=3)
            results.append({"id": known["id"], "title": known["title"], "definitions": known["definitions"],
                            "status": status, "detail": detail})
        count = {status: sum(1 for item in results if item["status"] == status) for status in ("pass", "fail", "skip")}
        return {"project": self.project, "profile": profile, "realization": str(realization),
                "passed": count["fail"] == 0 and count["skip"] == 0,
                "summary": dict(total=len(results), passed=count["pass"], failed=count["fail"], skipped=count["skip"]),
                "checks": results}


def unusable_query(env, *query):
    """Point env at a path the system cannot use; a query may read it as missing data or fail cleanly."""
    env.db = env.root / ("x" * 5000 + ".json")
    result = env.run(*query)
    crash_guard(result)
    if result.obj is None or (result.code, set(result.obj) == {"error"}) not in ((0, False), (2, True)):
        fail("`%s` on an unusable data path must succeed or fail with the error object, got %s"
             % (" ".join(query), result.describe()))


def unusable_change(env, *change):
    """On the unusable path of `unusable_query`, a change fails with the error object."""
    result = env.run(*change)
    crash_guard(result)
    if result.code != 2 or result.obj is None or set(result.obj) != {"error"}:
        fail("`%s` on an unusable data path must fail with the error object, got %s"
             % (" ".join(change), result.describe()))


def python_command(text):
    return text.split() if text else [sys.executable]
