"""Fleet checks: black-box expectations of a generated fleet project, built from its parameters.

One check per ledger covers that ledger's own cap, maximum amount, floor,
unknown accounts and order, so a fault in one ledger fails exactly that
ledger's check. The shared checks cover the hub's inputs, totals, data file
and command line. Error wording and the data file's organization are free.
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import fleet  # noqa: E402
from trial_kit import Checks, expect_equal, fail, unusable_change, unusable_query  # noqa: E402

PROFILE = ["f1"]


def credit(env, resource, name, amount, refuse=False):
    args = [resource, "credit", "--account", name, "--amount", str(amount)]
    return env.refuse(*args) if refuse else env.ok(*args)


def debit(env, resource, name, amount, refuse=False):
    args = [resource, "debit", "--account", name, "--amount", str(amount)]
    return env.refuse(*args) if refuse else env.ok(*args)


def accounts(env, resource):
    result = env.ok(resource, "balances")
    if set(result) != {"accounts"} or not isinstance(result["accounts"], list):
        fail("%s balances must return exactly {\"accounts\": [...]}, got %s" % (resource, result))
    return result["accounts"]


def totals(env):
    result = env.ok("totals")
    if set(result) != {"resources"} or not isinstance(result["resources"], list):
        fail("totals must return exactly {\"resources\": [...]}, got %s" % result)
    return result["resources"]


def build(project):
    entries = fleet.ledgers(project)
    resources = [r for r, *_ in entries]
    checks = Checks(project, {"f1": {"snapshot": "f1"}})
    check = checks.check
    first, first_path, _, first_top, _ = entries[0]

    @check("empty-state", PROFILE, ["hub/storage", "hub/totals"] + [p + "/balances" for _, p, *_ in entries],
           "A missing data file is empty ledgers with zero totals")
    def _(env, p):
        for resource in resources:
            expect_equal(accounts(env, resource), [], resource + " balances")
        expect_equal(totals(env), [{"resource": r, "accounts": 0, "sum": 0} for r in sorted(resources)], "totals")

    def ledger_check(resource, path, cap, top, floor):
        @check("ledger-" + resource, PROFILE, [path + "/account", path + "/credit", path + "/debit", path + "/balances"],
               "The %s ledger keeps its cap %d, maximum amount %d and floor %d and lists by name" % (resource, cap, top, floor))
        def _(env, p):
            balance = 0
            while balance < cap:
                balance = credit(env, resource, "abe", min(top, cap - balance))["balance"]
            expect_equal(balance, cap, "abe reaches exactly the cap")
            credit(env, resource, "abe", 1, refuse=True)
            credit(env, resource, "cid", top + 1, refuse=True)
            expect_equal(credit(env, resource, "bob", 10), {"account": "bob", "balance": 10}, "a new account")
            expect_equal(debit(env, resource, "bob", 10 - floor), {"account": "bob", "balance": floor}, "bob at the floor")
            debit(env, resource, "bob", 1, refuse=True)
            debit(env, resource, "zoe", 1, refuse=True)
            credit(env, resource, "dan", 5)
            credit(env, resource, "ann", 9)
            expect_equal(accounts(env, resource), [{"account": "abe", "balance": cap}, {"account": "ann", "balance": 9},
                                                   {"account": "bob", "balance": floor}, {"account": "dan", "balance": 5}],
                         resource + " balances")

    for resource, path, cap, top, floor in entries:
        ledger_check(resource, path, cap, top, floor)

    @check("account-names", PROFILE, ["hub/inputs", first_path + "/credit"],
           "Account names are trimmed and must be 1 to 20 lowercase letters a-z")
    def _(env, p):
        for name in ("Ann", "an1", "", "   ", "a" * 21, "ann-b", "änn", "ann bob"):
            credit(env, first, name, 1, refuse=True)
        expect_equal(credit(env, first, "  ann  ", 1), {"account": "ann", "balance": 1}, "trimmed name")
        expect_equal(credit(env, first, "a" * 20, 1)["account"], "a" * 20, "20 letters")

    @check("amount-format", PROFILE, ["hub/inputs", first_path + "/credit", first_path + "/debit"],
           "Amounts are ASCII digits without sign, space or fraction; leading zeros keep the value")
    def _(env, p):
        for text in ("0", "000", "-1", "+1", "1.0", "1e2", " 5", "5 ", "", "five", "５"):
            credit(env, first, "ann", text, refuse=True)
        expect_equal(credit(env, first, "ann", "0007")["balance"], 7, "credit 0007")
        debit(env, first, "ann", "-1", refuse=True)
        expect_equal(credit(env, first, "ann", first_top)["balance"], 7 + first_top, "the maximum amount")

    if len(entries) > 1:
        second = entries[1][0]

        @check("ledgers-independent", PROFILE, ["hub/system", first_path + "/account", entries[1][1] + "/account"],
               "The same account name in two ledgers is two separate accounts")
        def _(env, p):
            credit(env, first, "ann", 4)
            credit(env, second, "ann", 6)
            debit(env, second, "ann", 1)
            expect_equal(accounts(env, first), [{"account": "ann", "balance": 4}], first + " balances")
            expect_equal(accounts(env, second), [{"account": "ann", "balance": 5}], second + " balances")

    @check("totals", PROFILE, ["hub/totals"] + [p + "/balances" for _, p, *_ in entries],
           "totals counts the accounts and sums the balances of every ledger, sorted by resource")
    def _(env, p):
        expected = []
        for index, (resource, *_rest) in enumerate(entries):
            credit(env, resource, "ann", index + 1)
            if index % 2 == 0:
                credit(env, resource, "bob", 2)
            expected.append({"resource": resource, "accounts": 2 if index % 2 == 0 else 1,
                             "sum": index + 1 + (2 if index % 2 == 0 else 0)})
        expect_equal(totals(env), sorted(expected, key=lambda entry: entry["resource"]), "totals")

    @check("persist", PROFILE, ["hub/storage", first_path + "/credit"],
           "A later process sees committed changes in a UTF-8 JSON file")
    def _(env, p):
        credit(env, first, "ann", 3)
        expect_equal(accounts(env, first), [{"account": "ann", "balance": 3}], "balances in a later process")
        env.stored_json()

    @check("reads-untouched", PROFILE, ["hub/storage", "hub/totals", first_path + "/balances"],
           "Queries never write the data file")
    def _(env, p):
        credit(env, first, "ann", 3)
        env.reads_leave_data_untouched([[resource, "balances"] for resource in resources] + [["totals"]])

    @check("malformed-kept", PROFILE, ["hub/storage", "hub/interface"], "Malformed data fails clearly and is kept")
    def _(env, p):
        for content in (b"{not json", b"\xff\xfe\x00"):
            env.write_db(content)
            env.refuse("totals")
            credit(env, first, "ann", 1, refuse=True)
            if env.raw() != content:
                fail("the malformed data file was changed")

    @check("missing-directory", PROFILE, ["hub/storage", "hub/interface"],
           "A missing data directory is not created: changes fail and queries are empty")
    def _(env, p):
        env.db = env.root / "absent" / "ledgers.json"
        expect_equal(accounts(env, first), [], "balances without the data directory")
        credit(env, first, "ann", 1, refuse=True)
        if (env.root / "absent").exists():
            fail("the missing data directory was created")

    @check("usage-errors", PROFILE, ["hub/interface"], "Command-line errors use the error object and exit 2")
    def _(env, p):
        for args in ([first, "credit", "--account", "ann"], [first], ["coal", "balances"], [first, "balances", "--x"],
                     [], ["totals", "--all"], [first, "transfer", "--account", "ann", "--amount", "1"]):
            env.refuse(*args)

    @check("edge-robustness", PROFILE, ["hub/interface", "hub/inputs", "hub/storage"],
           "Extreme inputs fail through the error object, never with a crash")
    def _(env, p):
        credit(env, first, "ann", 1)
        credit(env, first, "ann", "9" * 5000, refuse=True)
        expect_equal(credit(env, first, "ann", "0" * 5000 + "7")["balance"], 8, "credit 5000 leading zeros and 7")
        credit(env, first, "ann\udcff", 1, refuse=True)
        env.write_db("[" * 100000)
        env.refuse("totals")
        unusable_query(env, "totals")
        unusable_change(env, first, "credit", "--account", "ann", "--amount", "1")

    @check("no-leftovers", PROFILE, ["hub/storage"], "Atomic replacement leaves no temporary files")
    def _(env, p):
        credit(env, first, "ann", 9)
        debit(env, first, "ann", 4)
        credit(env, first, "ann", first_top + 1, refuse=True)
        found = env.leftover_temp_files()
        if found:
            fail("files left beside the data file: %s" % found)

    return checks
