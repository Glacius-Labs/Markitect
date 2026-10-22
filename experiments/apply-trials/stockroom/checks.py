"""Stockroom checks: black-box expectations per model profile.

Profiles: `s1` is snapshot s1 (version 0.1.0); `s2` is snapshot s2 (0.2.0:
movement quantity at most 500, reorder points, `list --low`, 0.1.0 data stays
usable); `o` is s1 with the three overspecified statements of case S-O;
`n40` is s1 with the maximum name length 40 (case S-N40); `free` is s1 with
the maximum name length left to the realization from 40 to 120 and stated in
its README (case S-FREE). Error wording is free except where profile `o`
fixes it. The data file is inspected only as the profile's storage statement
binds it, and the README only where `free` binds it.
"""
import ast
import json
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from trial_kit import Checks, Env, Skip, expect_equal, fail, no_crash, unusable_change, unusable_query  # noqa: E402

PROFILES = {
    "s1": {"snapshot": "s1", "maxQty": 1000, "reorder": False, "maxName": 60},
    "s1-sol3": {"snapshot": "s1", "maxQty": 1000, "reorder": False, "maxName": 60},
    "s2": {"snapshot": "s2", "maxQty": 500, "reorder": True, "maxName": 60},
    "o": {"snapshot": "s1", "maxQty": 1000, "reorder": False, "maxName": 60},
    "n40": {"snapshot": "s1", "maxQty": 1000, "reorder": False, "maxName": 40},
    "free": {"snapshot": "s1", "maxQty": 1000, "reorder": False, "maxName": None, "nameRange": (40, 120)},
}
ALL = ["s1", "s1-sol3", "s2", "o", "n40", "free"]
FIXED_NAME = ["s1", "s1-sol3", "s2", "o", "n40"]
CHECKS = Checks("stockroom", PROFILES)
check = CHECKS.check


def record(p, sku, name, on_hand, reorder=0):
    item = {"sku": sku, "name": name, "onHand": on_hand}
    if p["reorder"]:
        item["reorderPoint"] = reorder
    return item


def add(env, sku, name="Bolt", reorder=None, refuse=False):
    args = ["add-item", "--sku", sku, "--name", name] + ([] if reorder is None else ["--reorder", reorder])
    return env.refuse(*args) if refuse else env.ok(*args)


def receive(env, sku, qty, refuse=False):
    args = ["receive", "--sku", sku, "--qty", str(qty)]
    return env.refuse(*args) if refuse else env.ok(*args)


def issue(env, sku, qty, refuse=False):
    args = ["issue", "--sku", sku, "--qty", str(qty)]
    return env.refuse(*args) if refuse else env.ok(*args)


def listing(env, *extra):
    result = env.ok("list", *extra)
    if set(result) != {"items"} or not isinstance(result["items"], list):
        fail("list must return exactly {\"items\": [...]}, got %s" % result)
    return result["items"]


def fill(env, sku, total, step):
    """Receive in steps of at most `step` until the item holds `total`."""
    held = 0
    while held < total:
        part = min(step, total - held)
        held = receive(env, sku, part)["onHand"]
    expect_equal(held, total, "on hand after receiving")


# -- every profile --------------------------------------------------------------------

@check('db-path', ['s1-sol3'], ['stockroom/interface', 'stockroom/storage'],
       'Separated leading-hyphen DB paths support public reads, changes and refusals')
def _(env, p):
    with tempfile.TemporaryDirectory(prefix='stockroom-path-') as folder:
        realization = Path(folder) / 'realization'
        shutil.copytree(env.realization, realization, ignore=shutil.ignore_patterns('__pycache__'))
        path = realization / '-stock.json'
        def call(options, args, code=0):
            done = subprocess.run([*env.python, '-B', 'app.py', *options, *args], cwd=realization,
                                  capture_output=True, timeout=15)
            expect_equal(done.returncode, code, 'public DB-path command exit')
            value = json.loads(done.stdout)
            if code:
                expect_equal(list(value), ['error'], 'refusal object')
                if not isinstance(value['error'], str) or not value['error'].strip():
                    fail('refusal must explain the failure')
            return value
        options = ['--db', '-stock.json']
        expect_equal(call(options, ['list']), {'items': []}, 'missing-file query')
        if path.exists():
            fail('query created the data file')
        expected = {'sku': 'ABC-123', 'name': '-Bolt', 'onHand': 0}
        expect_equal(call(options, ['add-item', '--sku', 'ABC-123', '--name', '-Bolt']), expected, 'added item')
        expect_equal(call(options, ['receive', '--sku', 'ABC-123', '--qty', '10'])['onHand'], 10, 'received stock')
        expect_equal(call(options, ['issue', '--sku', 'ABC-123', '--qty', '3'])['onHand'], 7, 'issued stock')
        before = path.read_bytes()
        expected['onHand'] = 7
        for spelling in (options, ['--db=-stock.json'], ['--db', './-stock.json']):
            expect_equal(call(spelling, ['list']), {'items': [expected]}, 'same durable data through each spelling')
        call(options, ['receive', '--sku', 'ABC-123', '--qty', '0'], 2)
        call(['--db'], [], 2)
        expect_equal(path.read_bytes(), before, 'queries/refusals preserve the file')

@check("list-empty", ALL, ["stockroom/list", "stockroom/storage"], "A missing data file lists no items")
def _(env, p):
    expect_equal(listing(env), [], "list without a data file")


@check("add-record", ALL, ["stockroom/add-item", "stockroom/item", "stockroom/inputs", "stockroom/interface"],
       "add-item returns the new record with normalized SKU and name and 0 on hand")
def _(env, p):
    expect_equal(add(env, " bol-006 ", "  Bolt M6 "), record(p, "BOL-006", "Bolt M6", 0), "added record")


@check("persist", ALL, ["stockroom/storage", "stockroom/receive", "stockroom/list"],
       "A later process sees committed changes in a UTF-8 JSON file")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    receive(env, "BOL-006", 5)
    expect_equal(listing(env), [record(p, "BOL-006", "Bolt", 5)], "list after add and receive")
    env.stored_json()


@check("sku-identity", ALL, ["stockroom/inputs", "stockroom/item", "stockroom/receive", "stockroom/issue"],
       "A SKU is the same item in any case and padding, for every command")
def _(env, p):
    add(env, "bol-006", "Bolt")
    add(env, " BOL-006 ", "Again", refuse=True)
    expect_equal(receive(env, " bol-006", 5)["onHand"], 5, "receive with a lowercase SKU")
    expect_equal(issue(env, "bOl-006 ", 2)["onHand"], 3, "issue with a mixed-case padded SKU")
    expect_equal(listing(env), [record(p, "BOL-006", "Bolt", 3)], "list")


@check("sku-format", ALL, ["stockroom/inputs", "stockroom/add-item"], "Malformed SKUs are refused")
def _(env, p):
    for text in ("AB-123", "ABCD-123", "ABC-12", "ABC-1234", "ABC123", "A1C-123", "ABC_123", "ÄBC-123", "", "   "):
        add(env, text, refuse=True)
    expect_equal(listing(env), [], "items after refusals")


@check("name-rules", FIXED_NAME, ["stockroom/inputs", "stockroom/add-item"],
       "Names are trimmed, nonempty and at most the profile's maximum of code points")
def _(env, p):
    top = p["maxName"]
    expect_equal(add(env, "AAA-001", "é" * top)["name"], "é" * top, "%d non-ASCII code points" % top)
    expect_equal(add(env, "AAA-002", "  " + "x" * top + "  ")["name"], "x" * top, "%d characters after trimming" % top)
    expect_equal(add(env, "AAA-004", "-Bolt")["name"], "-Bolt", "a name that starts with a hyphen")
    for name in ("", "   ", "x" * (top + 1), "é" * (top + 1), "  " + "x" * (top + 1) + "  "):
        add(env, "AAA-003", name, refuse=True)


@check("name-hyphens", ALL, ["stockroom/inputs", "stockroom/add-item", "stockroom/interface"],
       "Names that start with or consist of hyphens are names, not options")
def _(env, p):
    for index, name in enumerate(("-Bolt", "--Bolt", "-9", "-", "-x" + "y" * 30)):
        expect_equal(add(env, "HYP-%03d" % index, name)["name"], name, "the name %r" % name)


def accepts(env, length, serial):
    """Whether add-item accepts a name of LENGTH non-ASCII code points; a refusal must be the clean one."""
    result = env.run("add-item", "--sku", "LEN-%03d" % serial, "--name", "é" * length)
    if result.code == 0 and result.obj is not None and result.obj.get("name") == "é" * length:
        return True
    if result.code == 2 and result.obj is not None and set(result.obj) == {"error"}:
        return False
    fail("add-item with a name of %d code points neither succeeds nor fails cleanly: %s" % (length, result.describe()))


@check("name-limit-free", ["free"], ["stockroom/inputs", "stockroom/add-item", "stockroom/realization"],
       "The chosen maximum name length is one value from 40 to 120, enforced in code points and stated in the README")
def _(env, p):
    low, high = p["nameRange"]
    if not accepts(env, 1, 0):
        fail("a one-character name is refused")
    top, step, serial = 1, 1, 1
    while top < 400 and accepts(env, top + step, serial):  # bracket the maximum by doubling, then halve
        top, step, serial = top + step, step * 2, serial + 1
    while step > 1:
        step //= 2
        serial += 1
        if accepts(env, top + step, serial):
            top += step
    if not low <= top <= high:
        fail("the maximum name length is %d, the model allows %d to %d" % (top, low, high))
    for length, expected in ((top, True), (top + 1, False), (low, True)):
        serial += 1
        if accepts(env, length, serial) is not expected:
            fail("the maximum %d is not applied consistently at %d code points" % (top, length))
    expect_equal(add(env, "PAD-001", "  " + "x" * top + "  ")["name"], "x" * top, "%d characters after trimming" % top)
    readme = env.realization / "README.md"
    lines = [line for line in (readme.read_text(encoding="utf-8", errors="replace") if readme.is_file() else "").splitlines()
             if "name" in line.lower()]
    if not any(re.search(r"(?<![0-9])%d(?![0-9])" % top, line) for line in lines):
        fail("no README line about names states the maximum %d: %s" % (top, lines[:5]))


@check("duplicate-names", ALL, ["stockroom/item"], "Different items may share a name")
def _(env, p):
    add(env, "AAA-001", "Washer")
    add(env, "AAA-002", "Washer")
    expect_equal([item["sku"] for item in listing(env)], ["AAA-001", "AAA-002"], "both items")


@check("qty-format", ALL, ["stockroom/inputs", "stockroom/receive", "stockroom/issue"],
       "Quantities are ASCII digits without sign, space or fraction; leading zeros keep the value")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    for text in ("0", "000", "-1", "+1", "1.0", "1e2", " 5", "5 ", "", "five", "５"):
        receive(env, "BOL-006", text, refuse=True)
    expect_equal(receive(env, "BOL-006", "0007")["onHand"], 7, "receive 0007")
    for text in ("0", "-1", "+1", "1.5", "５"):
        issue(env, "BOL-006", text, refuse=True)


@check("qty-max", ALL, ["stockroom/inputs", "stockroom/receive", "stockroom/issue"],
       "A movement quantity is at most the profile's maximum")
def _(env, p):
    top = p["maxQty"]
    add(env, "BOL-006", "Bolt")
    expect_equal(receive(env, "BOL-006", top)["onHand"], top, "receive the maximum")
    receive(env, "BOL-006", top + 1, refuse=True)
    expect_equal(receive(env, "BOL-006", top)["onHand"], 2 * top, "receive the maximum again")
    issue(env, "BOL-006", top + 1, refuse=True)
    expect_equal(issue(env, "BOL-006", top)["onHand"], top, "issue the maximum")


@check("receive-cap", ALL, ["stockroom/stock-limits", "stockroom/receive"],
       "Receiving may reach exactly 9999 on hand, never more")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    fill(env, "BOL-006", 9999, p["maxQty"])
    receive(env, "BOL-006", 1, refuse=True)
    expect_equal(listing(env)[0]["onHand"], 9999, "on hand after the refused receipt")


@check("issue-exact", ALL, ["stockroom/stock-limits", "stockroom/issue"], "Issuing exactly the stock leaves 0")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    receive(env, "BOL-006", 10)
    expect_equal(issue(env, "BOL-006", 10), record(p, "BOL-006", "Bolt", 0), "issue everything")


@check("issue-over", ALL, ["stockroom/stock-limits", "stockroom/issue"],
       "Issuing more than is on hand is refused without a change")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    issue(env, "BOL-006", 1, refuse=True)
    receive(env, "BOL-006", 10)
    issue(env, "BOL-006", 11, refuse=True)
    expect_equal(listing(env)[0]["onHand"], 10, "on hand after refusals")


@check("unknown-sku", ALL, ["stockroom/receive", "stockroom/issue"], "Unknown SKUs are refused without a change")
def _(env, p):
    receive(env, "XYZ-999", 1, refuse=True)
    issue(env, "XYZ-999", 1, refuse=True)
    add(env, "BOL-006", "Bolt")
    receive(env, "XYZ-999", 1, refuse=True)
    issue(env, "XYZ-999", 1, refuse=True)


@check("list-order", ALL, ["stockroom/list"], "list is sorted by SKU, not by name or insertion")
def _(env, p):
    add(env, "ZED-001", "Alpha")
    add(env, "ABC-002", "Zulu")
    add(env, "MID-500", "Mike")
    expect_equal([item["sku"] for item in listing(env)], ["ABC-002", "MID-500", "ZED-001"], "SKU order")


@check("other-items", ALL, ["stockroom/receive", "stockroom/issue", "stockroom/add-item"],
       "A command changes only its own item")
def _(env, p):
    add(env, "AAA-001", "Anchor")
    add(env, "BBB-002", "Bracket")
    receive(env, "BBB-002", 8)
    before = listing(env)[0]
    receive(env, "AAA-001", 4)
    issue(env, "AAA-001", 1)
    add(env, "CCC-003", "Clamp")
    expect_equal(listing(env)[1], record(p, "BBB-002", "Bracket", 8), "the untouched item")
    expect_equal(listing(env)[0], dict(before, onHand=3), "the changed item")


@check("reads-untouched", ALL, ["stockroom/storage", "stockroom/list"], "Queries never write the data file")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    receive(env, "BOL-006", 3)
    env.reads_leave_data_untouched([["list"]] + ([["list", "--low"]] if p["reorder"] else []))


@check("malformed-kept", ALL, ["stockroom/storage", "stockroom/interface"],
       "Malformed data fails clearly and is not overwritten")
def _(env, p):
    for content in (b"{not json", b"\xff\xfe\x00"):
        env.write_db(content)
        env.refuse("list")
        add(env, "BOL-006", "Bolt", refuse=True)
        if env.raw() != content:
            fail("the malformed data file was changed")


@check("missing-directory", ALL, ["stockroom/storage", "stockroom/interface"],
       "A missing data directory is not created: changes fail and queries are empty")
def _(env, p):
    env.db = env.root / "absent" / "stock.json"
    expect_equal(listing(env), [], "list without the data directory")
    add(env, "BOL-006", "Bolt", refuse=True)
    if (env.root / "absent").exists():
        fail("the missing data directory was created")


@check("edge-robustness", ALL, ["stockroom/interface", "stockroom/inputs", "stockroom/storage"],
       "Extreme inputs fail through the error object, never with a crash")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    receive(env, "BOL-006", "9" * 5000, refuse=True)
    expect_equal(receive(env, "BOL-006", "0" * 5000 + "7")["onHand"], 7, "receive 5000 leading zeros and 7")
    no_crash(env, "add-item", "--sku", "NUT-010", "--name", "Nut \udcff")  # undecodable byte: the model is silent
    listing(env)
    env.write_db("[" * 100000)
    env.refuse("list")
    unusable_query(env, "list")
    unusable_change(env, "add-item", "--sku", "BOL-006", "--name", "Bolt")


@check("usage-errors", ALL, ["stockroom/interface"], "Command-line errors use the error object and exit 2")
def _(env, p):
    for args in (["receive", "--sku", "BOL-006"], ["move", "--sku", "BOL-006"], ["list", "--bogus"], [],
                 ["add-item", "--sku", "BOL-006"], ["issue", "--qty", "1"]):
        env.refuse(*args)


@check("no-leftovers", ALL, ["stockroom/storage"], "Atomic replacement leaves no temporary files")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    receive(env, "BOL-006", 9)
    issue(env, "BOL-006", 4)
    receive(env, "BOL-006", 9999, refuse=True)
    found = env.leftover_temp_files()
    if found:
        fail("files left beside the data file: %s" % found)


# -- s2: reorder points and data of version 0.1.0 --------------------------------------

@check("reorder-option", ["s2"], ["stockroom/add-item", "stockroom/inputs", "stockroom/item"],
       "--reorder sets a reorder point from 0 to 9999; it defaults to 0")
def _(env, p):
    expect_equal(add(env, "AAA-001", "A", "15")["reorderPoint"], 15, "--reorder 15")
    expect_equal(add(env, "AAA-002", "B", "0009")["reorderPoint"], 9, "--reorder 0009")
    expect_equal(add(env, "AAA-003", "C", "9999")["reorderPoint"], 9999, "--reorder 9999")
    expect_equal(add(env, "AAA-004", "D", "0")["reorderPoint"], 0, "--reorder 0")
    expect_equal(add(env, "AAA-005", "E")["reorderPoint"], 0, "without --reorder")
    for text in ("-1", "10000", "x", "1.5", "", " 3"):
        add(env, "AAA-006", "F", text, refuse=True)


@check("list-low", ["s2"], ["stockroom/list", "stockroom/item"],
       "list --low holds only items below their reorder point, in SKU order")
def _(env, p):
    add(env, "DDD-004", "D", "5")
    add(env, "AAA-001", "A", "10")
    add(env, "BBB-002", "B", "10")
    add(env, "CCC-003", "C", "0")
    receive(env, "AAA-001", 9)
    receive(env, "BBB-002", 10)
    expect_equal([item["sku"] for item in listing(env, "--low")], ["AAA-001", "DDD-004"], "low items")
    expect_equal(len(listing(env)), 4, "all items")
    issue(env, "BBB-002", 1)
    expect_equal([item["sku"] for item in listing(env, "--low")], ["AAA-001", "BBB-002", "DDD-004"],
                 "low items after an issue")


@check("upgrade", ["s2"], ["stockroom/storage", "stockroom/item"],
       "A data file written by the 0.1.0 realization stays usable with reorder point 0")
def _(env, p):
    previous = p.get("previous")
    if not previous:
        raise Skip("no 0.1.0 realization supplied (--previous)")
    with tempfile.TemporaryDirectory(prefix="trial-previous-") as root:
        old = Env(previous, env.python, root)
        old.ok("add-item", "--sku", "NUT-010", "--name", "Nut")
        old.ok("add-item", "--sku", "BOL-006", "--name", "Bolt é")
        old.ok("receive", "--sku", "NUT-010", "--qty", "12")
        env.write_db(old.raw())
    expect_equal(listing(env), [record(p, "BOL-006", "Bolt é", 0), record(p, "NUT-010", "Nut", 12)],
                 "items of the 0.1.0 data file")
    expect_equal(listing(env, "--low"), [], "low items of the 0.1.0 data file")
    expect_equal(receive(env, "BOL-006", 3), record(p, "BOL-006", "Bolt é", 3), "receive on 0.1.0 data")
    add(env, "CAP-001", "Cap", "4")
    expect_equal([item["sku"] for item in listing(env, "--low")], ["CAP-001"], "low items after an add")


# -- o: the overspecified statements -------------------------------------------------

@check("unknown-sku-message", ["o"], ["stockroom/interface"],
       "receive and issue of an unknown SKU write exactly 'unknown SKU: <normalized SKU>'")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    for command in ("receive", "issue"):
        result = env.refuse(command, "--sku", " xyz-999 ", "--qty", "1")
        expect_equal(result.obj, {"error": "unknown SKU: XYZ-999"}, command + " of an unknown SKU")


def canonical(document):
    return json.dumps(document, ensure_ascii=False, indent=2) + "\n"


@check("storage-format", ["o"], ["stockroom/storage"],
       "The data file is a two-space-indented object keyed by sorted SKU with name and onHand")
def _(env, p):
    add(env, "NUT-010", "Nut é")
    add(env, "BOL-006", "Bolt")
    receive(env, "NUT-010", 7)
    text = env.raw().decode("utf-8")
    expected = {"BOL-006": {"name": "Bolt", "onHand": 0}, "NUT-010": {"name": "Nut é", "onHand": 7}}
    if text != canonical(expected):
        fail("data file text %r differs from the required %r" % (text, canonical(expected)))


@check("storage-format-strict", ["o"], ["stockroom/storage"],
       "A file in another JSON organization is malformed data and is kept")
def _(env, p):
    content = b'{"items": [{"sku": "BOL-006", "name": "Bolt", "onHand": 1}]}\n'
    env.write_db(content)
    env.refuse("list")
    if env.raw() != content:
        fail("the data file was changed")


CONSTANTS = {"MAX_QUANTITY": 1000, "MAX_ON_HAND": 9999, "MAX_NAME_LENGTH": 60}


def module_constants(path):
    """Top-level integer constants, assigned plainly or with an annotation."""
    values = {}
    for node in ast.parse(path.read_text(encoding="utf-8")).body:
        if isinstance(node, ast.Assign) and len(node.targets) == 1:
            target = node.targets[0]
        elif isinstance(node, ast.AnnAssign):
            target = node.target
        else:
            continue
        if isinstance(target, ast.Name) and isinstance(node.value, ast.Constant) and type(node.value.value) is int:
            values[target.id] = node.value.value
    return values


@check("limits-module", ["o"], ["stockroom/realization", "stockroom/inputs", "stockroom/stock-limits"],
       "The limits are the constants of realization/limits.py, and the behavior follows them")
def _(env, p):
    module = env.realization / "limits.py"
    if not module.is_file():
        fail("realization/limits.py does not exist")
    values = module_constants(module)
    for name, value in CONSTANTS.items():
        expect_equal(values.get(name), value, "limits.py " + name)
    with tempfile.TemporaryDirectory(prefix="trial-limits-") as root:
        copy = Path(root) / "realization"
        shutil.copytree(env.realization, copy, ignore=shutil.ignore_patterns("__pycache__"))
        text = (copy / "limits.py").read_text(encoding="utf-8")
        for name, value in {"MAX_QUANTITY": 7, "MAX_ON_HAND": 20, "MAX_NAME_LENGTH": 5}.items():
            text, count = re.subn(r"(?m)^(%s(?:\s*:\s*int)?\s*=\s*)[0-9][0-9_]*" % name, r"\g<1>%d" % value, text)
            if count != 1:
                fail("limits.py must assign %s exactly once" % name)
        (copy / "limits.py").write_text(text, encoding="utf-8")
        probe_root = Path(root) / "probe"
        probe_root.mkdir()
        probe = Env(copy, env.python, probe_root)
        add(probe, "AAA-001", "abcdef", refuse=True)
        add(probe, "AAA-001", "abcde")
        receive(probe, "AAA-001", 8, refuse=True)
        receive(probe, "AAA-001", 7)
        receive(probe, "AAA-001", 7)
        receive(probe, "AAA-001", 6)
        receive(probe, "AAA-001", 1, refuse=True)
