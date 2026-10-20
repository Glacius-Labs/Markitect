"""Generated trial projects of the fleet family: W resource ledgers at nesting depth D.

`fleet-w<W>-d<D>` is a hub Package (CLI, inputs, one data file, a cross-ledger
`totals` report) and W ledger Packages, each with its own cap, maximum amount
and overdraft, nested under D-1 intermediate group Packages. Per-ledger rules
stay the same size, so width and depth vary the number of Managers,
Definitions, files and integration boundaries alone. The project directory has
the layout of the hand-written projects (snapshots, execution settings,
reference realization and registry) and is generated deterministically into a
cache outside the repository; it is operator material like the rest of this
directory.
"""
import hashlib
import re
import shutil
import tempfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
RESOURCES = ("wood", "iron", "salt", "wool", "clay", "rope", "wax", "tea", "ink", "oil", "gold", "rice")
GROUPS = ("ledgers", "books", "shelves", "rows")
NAME = re.compile(r"fleet-w(\d+)-d(\d+)$")
FAULT_KINDS = ("cap", "floor", "order")


def parse(project):
    match = NAME.match(project)
    if not match:
        raise ValueError("not a fleet project: " + project)
    width, depth = int(match.group(1)), int(match.group(2))
    if not 1 <= width <= len(RESOURCES) or not 1 <= depth <= len(GROUPS) + 1:
        raise ValueError("unsupported fleet size: " + project)
    return width, depth


def ledgers(project):
    """(resource, package path, cap, maximum amount, floor) per ledger."""
    width, depth = parse(project)
    prefix = "/".join(GROUPS[:depth - 1])
    found = []
    for index, resource in enumerate(RESOURCES[:width]):
        overdraft = 0 if index % 2 == 0 else 10 * (index + 1)
        found.append((resource, (prefix + "/" if prefix else "") + resource, 1000 * (index + 1), 100 + 50 * index, -overdraft))
    return found


def groups(project):
    _, depth = parse(project)
    return ["/".join(GROUPS[:level]) for level in range(1, depth)]


def packages(project):
    return ("hub",) + tuple(groups(project)) + tuple(path for _, path, _, _, _ in ledgers(project))


def managers(project):
    return 1 + len(packages(project))


def alias(resource):
    return resource.capitalize()


def module(path):
    return path.replace("/", ".")


# -- model -------------------------------------------------------------------------

def definition(kind, ident, body, imports=()):
    lines = ["imports:", "  - package: software", "    as: Software"]
    for package, name in imports:
        lines += ["  - package: " + package, "    as: " + name]
    lines += ["id: " + ident, "kind: Software." + kind, "body: |-"]
    lines += ["  " + line if line else "" for line in body.strip("\n").split("\n")]
    return "\n".join(lines + ["spec: {}", ""])


def wrap(text, width=78):
    words, lines, line = text.split(), [], ""
    for word in words:
        if line and len(line) + 1 + len(word) > width:
            lines.append(line)
            line = word
        else:
            line = word if not line else line + " " + word
    return "\n".join(lines + [line])


def floor_text(floor):
    return "0" if floor == 0 else "%d, an overdraft of at most %d" % (floor, -floor)


def model_files(project):
    entries = ledgers(project)
    hub = [("hub", "Hub")]
    files = {}
    names = ", ".join("[[%s.account|%s]]" % (alias(r), r) for r, *_ in entries)
    ledger_imports = [(path, alias(r)) for r, path, *_ in entries]
    files["hub/system.yaml"] = definition("Concept", "system", wrap(
        "A small local CLI that keeps %d resource ledgers for one operator: %s. Each ledger has its own accounts, "
        "limits and commands, and [[totals]] summarizes them. [[inputs]], [[storage]] and [[interface]] govern every "
        "command, and [[realization]] requires code, meaningful tests and usage documentation. Use the Python 3.13 "
        "standard library, with no service, network or package installation. There are no transfers between "
        "ledgers, users, deletion or renaming of accounts." % (len(entries), names)), ledger_imports)
    files["hub/inputs.yaml"] = definition("Rule", "inputs", wrap(
        "For every command of [[system]]: an account name is trimmed of surrounding whitespace and must then be 1 to "
        "20 lowercase letters a-z. An amount is written as one or more ASCII decimal digits, with no sign, "
        "whitespace, fraction or exponent; leading zeros do not change its value, which must be at least 1 and at "
        "most the maximum amount of the ledger that receives it. Invalid input fails through [[interface]] and "
        "leaves stored data unchanged under [[storage]]."))
    files["hub/storage.yaml"] = definition("Rule", "storage", wrap(
        "[[system]] keeps the data of all its ledgers as durable UTF-8 JSON in the one file named by --db. A missing "
        "file is empty data. The directory that holds the file is not created: when it does not exist, a changing "
        "command fails through [[interface]] and a query returns the empty result. Malformed existing data fails "
        "clearly through [[interface]] and is not overwritten. A successful command commits its change by atomic "
        "replacement of the file, and a later process using the same file observes it. A failed command leaves the "
        "file unchanged, and queries never write it. The organization of the JSON inside the file is an "
        "implementation choice. Concurrent independent writers are outside this contract; [[realization]] "
        "documents that limitation."))
    files["hub/realization.yaml"] = definition("Rule", "realization", wrap(
        "Deliver working code, meaningful automated tests and a README in realization/ for [[system]] and every "
        "modeled command. Keep code, tests and documentation consistent with the model. The README documents how "
        "to run the commands and the tests, the inputs and limits, the JSON output and error behavior, the "
        "persistent data file and the lack of guarantees for concurrent independent writers. Internal "
        "organization is free within the Python standard-library command and test boundary of [[interface]]."))
    files["hub/totals.yaml"] = definition("UseCase", "totals", wrap(
        "At [[interface]], `totals` returns an object whose resources field holds one entry per ledger of "
        "[[system]], sorted by resource name in ascending code-point order. An entry is a JSON object with exactly "
        "resource (the ledger's name, such as %s), accounts (the number of its accounts) and sum (the sum of their "
        "balances). Every ledger is listed, also without accounts. A missing data file gives zero counts and "
        "sums under [[storage]]. The query never changes stored data." % entries[0][0]))
    commands = []
    for r, *_ in entries:
        commands += ["- `%s credit --account NAME --amount N` follows [[%s.credit]]." % (r, alias(r)),
                     "- `%s debit --account NAME --amount N` follows [[%s.debit]]." % (r, alias(r)),
                     "- `%s balances` follows [[%s.balances]]." % (r, alias(r))]
    commands.append("- `totals` follows [[totals]].")
    files["hub/interface.yaml"] = definition("Rule", "interface", "\n".join([
        "From realization/, invoke [[system]] as:",
        "`python -B app.py --db PATH COMMAND ...`",
        wrap("Use the Python 3.13 standard library; no network, service or external package. Every successful "
             "command writes exactly one JSON object to stdout and exits 0. Invalid input, an unknown or missing "
             "command or option, and every refusal write exactly one JSON object to stdout whose only field is "
             "error, holding an understandable nonempty string; they exit 2 and leave stored data unchanged. "
             "Expected failures never end in an unhandled crash. Exact error wording is free."),
        "", "Commands, each returning the record or object its use case names:"] + commands + [
        "", wrap("[[inputs]] governs values and [[storage]] governs the data file. Automated tests are test_*.py "
                 "modules in importable packages below realization/; from realization/, "
                 "`python -B -m unittest discover -s . -p \"test_*.py\"` runs all of them, as [[realization]] "
                 "requires. These fixed entry points do not prescribe any other module organization.")]),
        ledger_imports)
    for group in groups(project):
        files[group + "/group.yaml"] = definition("Concept", "group", wrap(
            "This group of [[Hub.system]] holds the ledgers below it in the package hierarchy. It adds no rule or "
            "command of its own."), hub)
    for r, path, cap, top, floor in entries:
        files[path + "/account.yaml"] = definition("Concept", "account", wrap(
            "An account of the %s ledger of [[Hub.system]] has a name under [[Hub.inputs]] and a whole-number "
            "balance that never exceeds %d and never falls below %s. The first [[credit]] for a name creates the "
            "account; accounts are never deleted. The account record is a JSON object with exactly account (string) "
            "and balance (integer)." % (r, cap, floor_text(floor))), hub)
        files[path + "/credit.yaml"] = definition("UseCase", "credit", wrap(
            "At [[Hub.interface]], `%s credit --account NAME --amount N` adds N, an amount under [[Hub.inputs]] of "
            "at most %d, to the balance of the [[account]] with that name, creating the account with balance 0 "
            "first when it does not exist. A result above %d is refused; reaching exactly %d is allowed. Success "
            "commits through [[Hub.storage]] and returns the account record; a refusal leaves stored data "
            "unchanged." % (r, top, cap, cap)), hub)
        files[path + "/debit.yaml"] = definition("UseCase", "debit", wrap(
            "At [[Hub.interface]], `%s debit --account NAME --amount N` subtracts N, an amount under [[Hub.inputs]] "
            "of at most %d, from the balance of the existing [[account]] with that name. A result below %d is "
            "refused; reaching exactly %d is allowed. An unknown account is refused. Success commits through "
            "[[Hub.storage]] and returns the account record; a refusal leaves stored data unchanged."
            % (r, top, floor, floor)), hub)
        files[path + "/balances.yaml"] = definition("UseCase", "balances", wrap(
            "At [[Hub.interface]], `%s balances` returns an object whose accounts field holds every [[account]] "
            "record of the %s ledger, sorted by account name in ascending code-point order. A ledger without "
            "accounts returns an empty array. The query never changes stored data." % (r, r)), hub)
    return files


def execution(project):
    count = managers(project)
    return "\n".join([
        "apiVersion: markitect.execution/v1", "strategy: shared-workspace-v1", "budget:",
        "  executionTime: %dm" % (10 + 2 * count), "  turnTimeout: 8m", "  agentTurns: %d" % (16 * count),
        "  rounds: 4", "profiles:", "  execute: project-agent", "  verify: project-agent", "packages: {}",
        "layout:", "  roots: [realization]", "  exceptions:", "    - path: realization", "      owner: hub",
        "      reason: The fixed delivery boundary puts app.py, README.md and the end-to-end tests directly in "
        "realization/, which Root would otherwise own; the hub Package realizes them, and each ledger and group "
        "Package keeps its mirrored directory.",
        "checks:", "  - id: project-tests", "    profile: python-tests", "    directory: realization",
        "    args: [-B, -m, unittest, discover, -s, ., -p, \"test_*.py\"]", "    requiredBy: hub/interface",
        "    timeout: 3m", ""])


# -- reference realization ---------------------------------------------------------

LEDGER = '''"""The @R@ ledger: accounts, their limits and commands. The section is {name: balance}."""
from hub import inputs
from hub.errors import Refusal

RESOURCE = "@R@"
CAP = @CAP@
MAX_AMOUNT = @TOP@
FLOOR = @FLOOR@


def record(section, name):
    return {"account": name, "balance": section[name]}


def credit(state, name_text, amount_text):
    section = state[RESOURCE]
    name, amount = inputs.account_name(name_text), inputs.amount(amount_text, MAX_AMOUNT)
    balance = section.get(name, 0)
    if balance + amount > CAP:
        raise Refusal("crediting %d would raise %s above %d %s" % (amount, name, CAP, RESOURCE))
    section[name] = balance + amount
    return record(section, name)


def debit(state, name_text, amount_text):
    section = state[RESOURCE]
    name, amount = inputs.account_name(name_text), inputs.amount(amount_text, MAX_AMOUNT)
    if name not in section:
        raise Refusal("no %s account %s" % (RESOURCE, name))
    if section[name] - amount < FLOOR:
        raise Refusal("debiting %d would take %s below %d %s" % (amount, name, FLOOR, RESOURCE))
    section[name] -= amount
    return record(section, name)


def balances(state):
    section = state[RESOURCE]
    return [record(section, name) for name in sorted(section)]


def validate(section):
    if not isinstance(section, dict):
        raise ValueError("the %s ledger must be an object" % RESOURCE)
    for name, balance in section.items():
        if inputs.account_name(name) != name or type(balance) is not int or not FLOOR <= balance <= CAP:
            raise ValueError("invalid %s account %r" % (RESOURCE, name))
'''

LEDGER_TEST = '''import unittest

from hub.errors import Refusal
from hub.store import empty
from @MODULE@ import ledger


class LedgerTests(unittest.TestCase):
    def setUp(self):
        self.state = empty()

    def test_credit_creates_and_adds(self):
        self.assertEqual(ledger.credit(self.state, "ann", "5"), {"account": "ann", "balance": 5})
        self.assertEqual(ledger.credit(self.state, " ann ", "7")["balance"], 12)

    def test_amount_limit(self):
        ledger.credit(self.state, "ann", str(ledger.MAX_AMOUNT))
        with self.assertRaises(Refusal):
            ledger.credit(self.state, "ann", str(ledger.MAX_AMOUNT + 1))

    def test_debit_needs_an_account_and_cover(self):
        with self.assertRaises(Refusal):
            ledger.debit(self.state, "bob", "1")
        ledger.credit(self.state, "bob", "10")
        self.assertEqual(ledger.debit(self.state, "bob", "4")["balance"], 6)
        with self.assertRaises(Refusal):
            ledger.debit(self.state, "bob", str(6 - ledger.FLOOR + 5))
        self.assertEqual(self.state[ledger.RESOURCE]["bob"], 6)

    def test_balances_are_sorted(self):
        ledger.credit(self.state, "abe", "1")
        ledger.credit(self.state, "cid", "2")
        self.assertEqual([entry["account"] for entry in ledger.balances(self.state)], ["abe", "cid"])


if __name__ == "__main__":
    unittest.main()
'''

ERRORS = '''"""The refusal that every part raises for invalid input or a violated rule."""


class Refusal(Exception):
    """A command is refused; its message becomes the error object's text."""
'''

INPUTS = '''"""Shared input rules: account names and amounts."""
import re

from hub.errors import Refusal

NAME = re.compile(r"[a-z]{1,20}")
DIGITS = re.compile(r"[0-9]+")


def account_name(text):
    name = text.strip()
    if not NAME.fullmatch(name):
        raise Refusal("an account name is 1 to 20 lowercase letters a-z, got %r" % text)
    return name


def amount(text, high):
    if not DIGITS.fullmatch(text):
        raise Refusal("an amount is a whole number written in digits, got %r" % text)
    digits = text.lstrip("0") or "0"  # leading zeros do not change the value
    value = int(digits) if len(digits) <= 9 else high + 1
    if not 1 <= value <= high:
        raise Refusal("the amount must be from 1 to %d" % high)
    return value
'''

STORE = '''"""The one data file of all ledgers: UTF-8 JSON {resource: {name: balance}}, replaced atomically."""
import json
import os
import tempfile
from pathlib import Path

from hub.errors import Refusal
from hub.ledgers import LEDGERS


class StoreError(Exception):
    """The data file cannot be read or written."""


def empty():
    return {ledger.RESOURCE: {} for ledger in LEDGERS}


def load(path):
    """Read and validate the data; a missing file is empty data."""
    path = Path(path)
    try:
        if not path.exists():
            return empty()
        state = json.loads(path.read_bytes().decode("utf-8"))
        if not isinstance(state, dict) or set(state) != {ledger.RESOURCE for ledger in LEDGERS}:
            raise ValueError("expected an object with one section per ledger")
        for ledger in LEDGERS:
            ledger.validate(state[ledger.RESOURCE])
        return state
    except (OSError, ValueError, TypeError, KeyError, RecursionError, Refusal) as error:
        raise StoreError("the data file %s cannot be read: %s" % (path, error)) from None


def save(path, state):
    """Write the data next to the target and atomically replace the file."""
    path = Path(path)
    text = json.dumps(state, ensure_ascii=False, sort_keys=True) + "\\n"
    try:
        handle, temporary = tempfile.mkstemp(prefix=path.name + ".", suffix=".tmp", dir=path.parent)
        try:
            with os.fdopen(handle, "w", encoding="utf-8") as stream:
                stream.write(text)
                stream.flush()
                os.fsync(stream.fileno())
            os.replace(temporary, path)
        except BaseException:
            os.unlink(temporary)
            raise
    except OSError as error:
        raise StoreError("the data file %s cannot be written: %s" % (path, error)) from None
'''

TOTALS = '''"""The cross-ledger totals report."""
from hub.ledgers import LEDGERS


def totals(state):
    entries = []
    for ledger in sorted(LEDGERS, key=lambda ledger: ledger.RESOURCE):
        accounts = ledger.balances(state)
        entries.append({"resource": ledger.RESOURCE, "accounts": len(accounts),
                        "sum": sum(entry["balance"] for entry in accounts)})
    return entries
'''

APP = '''"""Command line of the ledgers: python -B app.py --db PATH COMMAND ..."""
import argparse
import json
import sys
from pathlib import Path

from hub import store, totals
from hub.errors import Refusal
from hub.ledgers import LEDGERS

BY_RESOURCE = {ledger.RESOURCE: ledger for ledger in LEDGERS}


class UsageError(Exception):
    """The command line itself is invalid."""


class Parser(argparse.ArgumentParser):
    """Reports usage errors as exceptions instead of printing and exiting."""

    def error(self, message):
        raise UsageError(message)


def build_parser():
    parser = Parser(prog="app.py", add_help=False, allow_abbrev=False)
    parser.add_argument("--db", required=True)
    commands = parser.add_subparsers(dest="command", required=True, parser_class=Parser)
    for resource in BY_RESOURCE:
        actions = commands.add_parser(resource, add_help=False, allow_abbrev=False).add_subparsers(
            dest="action", required=True, parser_class=Parser)
        for action in ("credit", "debit"):
            change = actions.add_parser(action, add_help=False, allow_abbrev=False)
            change.add_argument("--account", required=True)
            change.add_argument("--amount", required=True)
        actions.add_parser("balances", add_help=False, allow_abbrev=False)
    commands.add_parser("totals", add_help=False, allow_abbrev=False)
    return parser


def run(args):
    path = Path(args.db)
    state = store.load(path)
    if args.command == "totals":
        return {"resources": totals.totals(state)}
    ledger = BY_RESOURCE[args.command]
    if args.action == "balances":
        return {"accounts": ledger.balances(state)}
    result = getattr(ledger, args.action)(state, args.account, args.amount)
    store.save(path, state)
    return result


def main(argv=None):
    try:
        result = run(build_parser().parse_args(argv))
    except (UsageError, Refusal, store.StoreError) as error:
        print(json.dumps({"error": str(error)}))
        return 2
    # ASCII escapes keep every output writable whatever the stdout encoding; the JSON value is the same.
    print(json.dumps(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
'''

CLI_TEST = '''import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

REALIZATION = Path(__file__).resolve().parents[1]
FIRST = "@R@"


class CommandLineTests(unittest.TestCase):
    def setUp(self):
        self.folder = tempfile.TemporaryDirectory()
        self.db = Path(self.folder.name) / "ledgers.json"

    def tearDown(self):
        self.folder.cleanup()

    def run_app(self, *args):
        done = subprocess.run([sys.executable, "-B", "app.py", "--db", str(self.db), *args],
                              cwd=REALIZATION, capture_output=True, text=True, encoding="utf-8")
        return done.returncode, json.loads(done.stdout)

    def ok(self, *args):
        code, result = self.run_app(*args)
        self.assertEqual(code, 0, result)
        return result

    def stored(self):
        try:
            return self.db.read_bytes()
        except OSError:  # missing, or a path the system cannot use
            return None

    def refused(self, *args):
        before = self.stored()
        code, result = self.run_app(*args)
        self.assertEqual(code, 2)
        self.assertEqual(list(result), ["error"])
        self.assertEqual(self.stored(), before)

    def test_round_trip_across_processes(self):
        self.assertEqual(self.ok(FIRST, "credit", "--account", "ann", "--amount", "9"), {"account": "ann", "balance": 9})
        self.assertEqual(self.ok(FIRST, "debit", "--account", "ann", "--amount", "4")["balance"], 5)
        self.assertEqual(self.ok(FIRST, "balances"), {"accounts": [{"account": "ann", "balance": 5}]})
        entry = next(item for item in self.ok("totals")["resources"] if item["resource"] == FIRST)
        self.assertEqual(entry, {"resource": FIRST, "accounts": 1, "sum": 5})

    def test_refusals_and_usage_errors_leave_the_file_unchanged(self):
        self.ok(FIRST, "credit", "--account", "ann", "--amount", "3")
        self.refused(FIRST, "debit", "--account", "zed", "--amount", "1")
        self.refused(FIRST, "credit", "--account", "Ann", "--amount", "1")
        self.refused(FIRST, "credit", "--account", "ann", "--amount", "0")
        self.refused(FIRST, "credit", "--account", "ann")
        self.refused("move", "--account", "ann")

    def test_malformed_file_is_reported_and_kept(self):
        self.db.write_text("[]", encoding="utf-8")
        self.refused("totals")
        self.refused(FIRST, "credit", "--account", "ann", "--amount", "1")
        self.assertEqual(self.db.read_text(encoding="utf-8"), "[]")

    def test_extreme_inputs_fail_without_a_crash(self):
        self.refused(FIRST, "credit", "--account", "ann", "--amount", "9" * 5000)
        self.refused(FIRST, "credit", "--account", "ann\\udcff", "--amount", "1")
        self.db.write_text("[" * 100000, encoding="utf-8")
        self.refused("totals")


if __name__ == "__main__":
    unittest.main()
'''


def reference_files(project):
    entries = ledgers(project)
    files = {}
    imports = "\n".join("from %s import ledger as %s" % (module(path), r) for r, path, *_ in entries)
    files["realization/hub/__init__.py"] = '"""The hub: command line, shared inputs, the data file and totals."""\n'
    files["realization/hub/errors.py"] = ERRORS
    files["realization/hub/inputs.py"] = INPUTS
    files["realization/hub/ledgers.py"] = ('"""Every ledger of the system, in one place for the command line, the data '
                                           'file and the totals."""\n%s\n\nLEDGERS = (%s,)\n'
                                           % (imports, ", ".join(r for r, *_ in entries)))
    files["realization/hub/store.py"] = STORE
    files["realization/hub/totals.py"] = TOTALS
    files["realization/app.py"] = APP
    files["realization/tests/__init__.py"] = '"""End-to-end tests of the command line."""\n'
    files["realization/tests/test_cli.py"] = CLI_TEST.replace("@R@", entries[0][0])
    for group in groups(project):
        files["realization/%s/__init__.py" % group] = '"""A group of ledgers."""\n'
    for r, path, cap, top, floor in entries:
        files["realization/%s/__init__.py" % path] = '"""The %s ledger."""\n' % r
        files["realization/%s/ledger.py" % path] = (LEDGER.replace("@R@", r).replace("@CAP@", str(cap))
                                                     .replace("@TOP@", str(top)).replace("@FLOOR@", str(floor)))
        files["realization/%s/test_ledger.py" % path] = LEDGER_TEST.replace("@MODULE@", module(path))
    rows = []
    for r, path, cap, top, floor in entries:
        rows += ["| `%s credit --account NAME --amount N` | Adds N (1 to %d) to the account, creating it; the balance stays at most %d. |" % (r, top, cap),
                 "| `%s debit --account NAME --amount N` | Subtracts N (1 to %d) from an existing account; the balance stays at least %d. |" % (r, top, floor),
                 "| `%s balances` | Returns `{\"accounts\": [...]}` sorted by account name. |" % r]
    files["realization/README.md"] = "\n".join([
        "# Ledgers", "",
        "A small local command line that keeps %d resource ledgers. It uses only the Python 3.13 standard library." % len(entries),
        "", "## Run", "", "From this directory:", "", "```text", "python -B app.py --db PATH COMMAND ...", "```", "",
        "| Command | Effect |", "| --- | --- |"] + rows + [
        "| `totals` | Returns `{\"resources\": [...]}` with the number of accounts and the sum of balances per ledger, sorted by resource. |",
        "", "An account is returned as `{\"account\": \"ann\", \"balance\": 5}`.", "",
        "## Inputs", "",
        "- An account name is trimmed and must then be 1 to 20 lowercase letters a-z.",
        "- An amount is written in digits only and is at least 1 and at most the ledger's maximum amount.", "",
        "## Output and errors", "",
        "Every successful command prints one JSON object and exits with 0. Invalid input, a refused change and a wrong "
        "command line print `{\"error\": \"...\"}` and exit with 2; the data file is then unchanged.", "",
        "## Data file", "",
        "All ledgers are kept as UTF-8 JSON in the one file given by `--db`. A missing file means empty ledgers. The "
        "file's directory is not created: without it, changes fail and the queries show empty ledgers. Each change "
        "replaces the file atomically, so a later run sees it; queries never write the file. A malformed file is "
        "reported and left as it is. The tool assumes one writer at a time: two processes changing the same file "
        "concurrently can lose an update.", "",
        "## Code and tests", "",
        "`app.py` reads the command line; `hub/` holds the inputs, the data file and the totals; each ledger has its "
        "own package with its tests beside it. Run every test with:", "",
        "```text", "python -B -m unittest discover -s . -p \"test_*.py\"", "```", ""])
    return files


def registry_files(project):
    def shard(entries):
        lines = ["apiVersion: markitect.registry/v1", "artifacts:"]
        for path, form, realizes, why in sorted(entries):
            lines += ["  - path: " + path, "    form: " + form, "    realizes: [%s]" % ", ".join(realizes), "    why: " + why]
        return "\n".join(lines) + "\n"

    entries = ledgers(project)
    hub = [
        ("realization/README.md", "doc", ["hub/realization", "hub/interface", "hub/inputs", "hub/storage"],
         "Documents every command, the inputs and limits, the output, the data file and the single-writer limitation."),
        ("realization/app.py", "code", ["hub/system", "hub/interface"],
         "Parses the command line, dispatches each command to its ledger and writes the result or error object."),
        ("realization/hub/__init__.py", "code", ["hub/system"], "Makes the hub an importable package."),
        ("realization/hub/errors.py", "code", ["hub/interface"], "Defines the refusal that every part raises."),
        ("realization/hub/inputs.py", "code", ["hub/inputs"], "Parses the shared account names and amounts."),
        ("realization/hub/ledgers.py", "code", ["hub/system"], "Lists every ledger for the command line, the data file and the totals."),
        ("realization/hub/store.py", "code", ["hub/storage"], "Reads the one data file, has each ledger validate its section and replaces the file atomically."),
        ("realization/hub/totals.py", "code", ["hub/totals"], "Builds the cross-ledger totals from each ledger's balances."),
        ("realization/tests/__init__.py", "test", ["hub/interface"], "Makes the end-to-end tests an importable package."),
        ("realization/tests/test_cli.py", "test", ["hub/interface", "hub/storage", "hub/totals"],
         "Exercises the command line end to end, including refusals that leave the file unchanged."),
    ]
    files = {"hub/registry.yaml": shard(hub)}
    for group in groups(project):
        files[group + "/registry.yaml"] = shard([("realization/%s/__init__.py" % group, "code", [group + "/group"],
                                                  "Makes the group an importable package.")])
    for r, path, *_ in entries:
        files[path + "/registry.yaml"] = shard([
            ("realization/%s/__init__.py" % path, "code", [path + "/account"], "Makes the ledger an importable package."),
            ("realization/%s/ledger.py" % path, "code", [path + "/account", path + "/credit", path + "/debit", path + "/balances"],
             "Keeps the accounts of the ledger within its limits and lists them."),
            ("realization/%s/test_ledger.py" % path, "test", [path + "/credit", path + "/debit", path + "/balances"],
             "Tests crediting, debiting and listing the ledger's accounts."),
        ])
    return files


# -- faults ------------------------------------------------------------------------

def fault_edit(project, index):
    """The edit and kind of the deliberate fault in ledger INDEX."""
    r, path, *_ = ledgers(project)[index]
    kind = FAULT_KINDS[index % len(FAULT_KINDS)]
    old, new = {
        "cap": ("    if balance + amount > CAP:\n", "    if balance + amount >= CAP:\n"),
        "floor": ("    if section[name] - amount < FLOOR:\n", "    if section[name] - amount < FLOOR - 1:\n"),
        "order": ("    return [record(section, name) for name in sorted(section)]\n",
                  "    return [record(section, name) for name in sorted(section, key=section.get)]\n"),
    }[kind]
    summary = {"cap": "credit refuses reaching exactly the cap of %s", "floor": "debit lets %s fall one below its floor",
               "order": "balances of %s are sorted by balance instead of name"}[kind] % r
    return kind, "realization/%s/ledger.py" % path, old, new, summary


# -- generation --------------------------------------------------------------------

def write(root, files):
    for relative, text in files.items():
        path = Path(root) / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8", newline="\n")


def generate(project, target):
    """Write the project directory: snapshots/f1, execution.yaml, reference/realization and reference/registry."""
    target = Path(target)
    snapshot = target / "snapshots" / "f1"
    write(snapshot / "model", model_files(project))
    width, depth = parse(project)
    write(snapshot, {"model.yaml": "formatVersion: 1\nid: %s\nversion: 0.1.0\nroot: model\ndependencies:\n"
                                   "  markitect-core: \"1.0.0\"\n  software-basics: \"0.1.0\"\n" % "ledgers"})
    shutil.copyfile(HERE / "stockroom" / "snapshots" / "s1" / "model-lock.yaml", snapshot / "model-lock.yaml")
    write(target, {"execution.yaml": execution(project)})
    write(target / "reference", reference_files(project))
    write(target / "reference" / "registry", registry_files(project))
    return target


def project_dir(project):
    """The generated project directory, cached outside the repository by generator digest."""
    parse(project)
    digest = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()[:12]
    target = Path(tempfile.gettempdir()) / "markitect-apply-trials" / ("%s-%s" % (project, digest))
    if not (target / ".complete").is_file():
        staging = target.with_name(target.name + ".tmp-%d" % __import__("os").getpid())
        if staging.exists():
            shutil.rmtree(staging)
        generate(project, staging)
        (staging / ".complete").write_text("", encoding="utf-8")
        try:
            staging.rename(target)
        except OSError:  # a parallel generation won; its result is identical
            shutil.rmtree(staging, ignore_errors=True)
    return target
