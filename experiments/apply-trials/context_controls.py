"""Independent controls of model context completeness; see README.md, "Model context controls".

OPERATOR MATERIAL.

    python -B context_controls.py materialize SNAPSHOT TARGET      # c1 or c1-stale as a committed packet
    python -B context_controls.py selection --markitect EXE [--package PKG ...]
    python -B context_controls.py check --payload FILE --reader PKG|.|plan [--expected FILE]
    python -B context_controls.py selftest [--markitect EXE]

The control model `tariff` (context/snapshots/c1) has a Manager under test,
shop/pricing, with own Definitions, an outgoing and an incoming neighbour, a
modeled-scope policy, typed Kind and Property contracts with nested, list,
boolean and number values and a reference, an explicit freedom sentence and
unrelated Definitions that must not be selected. context/expected.json holds
the expectations, written by hand from the model files and the documented
selection contract, never from a product or plan.md renderer.

`selection` compares the product's actual static selection (`markitect
context`) with the expected own, required and forbidden Definitions and their
inclusion reasons. `check` judges a delivered payload (a captured Brief,
prompt or lookup response, or a plan.md packet) against the expected facts:
a Definition counts as delivered in full only when one section of the payload
that starts with its identity holds every text fragment, its decisive final
sentence, its freedom sentence, every link and reference target and every
structured value. An identity alone, an inventory line or a summary is a
reference, not full content. Formatting may differ; meaning and values may
not. `selftest` proves that the checker accepts an independently built
payload and rejects each declared fault, and that the plan.md packet of the
control model carries every Definition in full.
"""
import argparse
import json
import re
import shutil
import subprocess
import sys
from decimal import Decimal, InvalidOperation
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

from materialize import DEPENDENCIES, GIT, IDENTITY, commit, git  # noqa: E402

CONTROL = HERE / "context"
EXPECTED = CONTROL / "expected.json"
MODEL_ARGS = ["--model", ".markitect/model.yaml", "--dependency", ".markitect/dependencies/core/model.yaml",
              "--dependency", ".markitect/dependencies/software/model.yaml"]
NUMBER = re.compile(r"[-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?")


class ControlError(Exception):
    pass


def expected(path=EXPECTED):
    return json.loads(Path(path).read_text(encoding="utf-8"))


# -- packet ------------------------------------------------------------------------

def materialize(snapshot, target):
    """A committed packet of the control model: model, dependencies and execution settings, no realization."""
    source = CONTROL / "snapshots" / snapshot
    target = Path(target).resolve()
    if not source.is_dir():
        raise ControlError("unknown control snapshot " + snapshot)
    if target.exists():
        raise ControlError("%s already exists" % target)
    markitect = target / ".markitect"
    shutil.copytree(source / "model", markitect / "model")
    for name in ("model.yaml", "model-lock.yaml"):
        shutil.copyfile(source / name, markitect / name)
    shutil.copytree(DEPENDENCIES, markitect / "dependencies")
    shutil.copyfile(CONTROL / "execution.yaml", markitect / "execution.yaml")
    git(target, "init", "-q")
    commit(target, "Current project state")
    return {"snapshot": snapshot, "target": str(target), "head": git(target, "rev-parse", "HEAD").strip(),
            "tree": git(target, "rev-parse", "HEAD^{tree}").strip()}


# -- actual selection ----------------------------------------------------------------

REASON_WORDS = {"own": "own", "kind-contract": "kind-contract", "property-contract": "property-contract",
                "outgoing": "outgoing", "incoming": "incoming", "modeled-scope": "modeled-scope"}


def identity(value):
    """package/id from the product's identity forms: a string or an object with package and id."""
    if isinstance(value, str):
        return value
    if isinstance(value, dict):
        package = value.get("Package", value.get("package", ""))
        local = value.get("ID", value.get("Id", value.get("id", "")))
        return "%s/%s" % (package, local) if package else local
    return str(value)


def actual_selection(markitect, repo, package):
    """{identity: sorted reason kinds} of the product's static selection for PACKAGE."""
    done = subprocess.run([markitect, "context", package, "--repo", str(repo), "--revision", "HEAD", *MODEL_ARGS],
                          capture_output=True, text=True, encoding="utf-8")
    try:
        report = json.loads(done.stdout)
    except ValueError:
        raise ControlError("markitect context %s printed no JSON: %s" % (package, done.stderr[-800:]))
    found = {}

    def walk(node):
        if isinstance(node, dict):
            reasons = node.get("Reasons", node.get("reasons"))
            definition = node.get("Definition", node.get("definition"))
            if isinstance(reasons, list) and isinstance(definition, dict):
                ident = identity(definition.get("Identity", definition.get("identity")))
                kinds = {str(reason.get("Kind", reason.get("kind", ""))) for reason in reasons if isinstance(reason, dict)}
                found.setdefault(ident, set()).update(kinds)
            for value in node.values():
                walk(value)
        elif isinstance(node, list):
            for value in node:
                walk(value)

    walk(report)
    if not found:
        raise ControlError("markitect context %s reported no selected Definition (status %s)"
                           % (package, report.get("status")))
    return {key: sorted(value) for key, value in found.items()}, report.get("status")


def compare_selection(actual, expectation):
    """Missing required entries and reasons, forbidden entries, and the documented connected contracts."""
    problems = []
    for own in expectation["own"]:
        if "own" not in actual.get(own, []):
            problems.append("own %s is not selected as own (got %s)" % (own, actual.get(own)))
    for ident, reasons in expectation["required"].items():
        got = actual.get(ident)
        if got is None:
            problems.append("required %s is not selected" % ident)
        elif not set(reasons) <= set(got):
            problems.append("required %s is selected for %s, expected the reason %s" % (ident, got, reasons))
    for ident in expectation.get("connectedContracts", []):
        if ident not in actual:
            problems.append("connected contract %s is not selected" % ident)
    for ident in expectation["forbidden"]:
        if ident in actual:
            problems.append("forbidden %s is selected for %s" % (ident, actual[ident]))
    connected = [ident for ident in expectation.get("connectedContracts", []) if ident not in actual]
    listed = set(expectation["own"]) | set(expectation["required"]) | set(expectation.get("connectedContracts", []))
    return {"passed": not problems, "problems": problems, "connectedContractsMissing": connected,
            "unlisted": sorted(ident for ident in actual if ident not in listed)}


# -- payload sections and facts ----------------------------------------------------------

def normalize(text):
    return re.sub(r"\s+", " ", text).strip()


def headers(payload, identities):
    """(offset, identity) of every unindented line that starts with a known identity, after markup characters.

    Indented lines, such as list items of structured values, never start a section.
    """
    found = []
    ordered = sorted(identities, key=len, reverse=True)
    for match in re.finditer(r"(?m)^(?![ \t])[>#*`_\-| ]*", payload):
        rest = payload[match.end():match.end() + 200]
        for ident in ordered:
            if rest.startswith(ident) and not re.match(r"[\w/\-]", rest[len(ident):len(ident) + 1]):
                found.append((match.start(), ident))
                break
    return found


def sections(payload, identities):
    """{identity: [section text, ...]}: from a header line to the next header line of any identity."""
    marks = headers(payload, identities)
    result = {}
    for index, (offset, ident) in enumerate(marks):
        end = marks[index + 1][0] if index + 1 < len(marks) else len(payload)
        result.setdefault(ident, []).append(payload[offset:end])
    return result


def value_pattern(fact):
    kind, value = fact["type"], fact["value"]
    if kind == "boolean":
        return r"(?i:%s)\b" % ("true" if value else "false")
    if kind == "integer":
        return r"(?<![\w.])%d(?![\w.])" % value
    if kind == "number":
        return None
    if kind == "reference":
        options = [re.escape(value)] + ([re.escape(fact["written"])] if fact.get("written") else [])
        return r"[\"']?(?:%s)(?![\w-])" % "|".join(options)
    return r"[\"']?%s[\"']?(?![\w/-])" % re.escape(str(value))


def find_value(section, fact, start=0):
    """The end offset of the fact's key followed by its value, at or after START, or None."""
    keys = [key for key in fact["path"] if isinstance(key, str)]
    key = keys[-1]
    separator = r"[\"'*`]*\s*[:=]\s*[\[{(\"'`]*\s*"
    if isinstance(fact["path"][-1], int):  # a list item: the key's list, then the item in order
        for match in re.finditer(r"(?<![\w/-])%s%s" % (re.escape(key), separator), section[start:]):
            return match.start() + start
        return None
    pattern = value_pattern(fact)
    for match in re.finditer(r"(?<![\w/-])%s%s" % (re.escape(key), separator), section[start:]):
        tail = section[start + match.end():start + match.end() + 80]
        if pattern is None:
            token = NUMBER.match(tail)
            try:
                if token and Decimal(token.group(0)) == Decimal(fact["value"]):
                    return start + match.end() + token.end()
            except InvalidOperation:
                continue
        elif re.match(pattern, tail):
            return start + match.end()
    return None


def list_items_in_order(section, facts):
    """Facts of one list (path [key, index] or [key, index, field]) appear in index order."""
    missing = []
    groups = {}
    for fact in facts:
        groups.setdefault(fact["path"][1], []).append(fact)
    position = 0
    for index in sorted(groups):
        first = None
        for fact in groups[index]:
            if len(fact["path"]) == 2:  # a scalar list item after the list's key
                key_at = find_value(section, fact)
                if key_at is None:
                    missing.append(fact)
                    continue
                pattern = value_pattern(fact)
                match = re.compile(pattern).search(section, max(position, key_at))
                if not match:
                    missing.append(fact)
                    continue
                first = match.start() if first is None else first
                position = match.end()
            else:  # a field of an object item
                item = {"path": fact["path"][2:], "type": fact["type"], "value": fact["value"],
                        "written": fact.get("written")}
                found = find_value(section, item, position)
                if found is None:
                    missing.append(fact)
                    continue
                first = found if first is None else min(first, found)
        if first is not None:
            position = max(position, first + 1)
    return missing


def section_gaps(section, fact):
    """The facts of one Definition that SECTION does not carry."""
    text = normalize(section)
    gaps = []
    for fragment in fact.get("text", []):
        if normalize(fragment) not in text:
            gaps.append("text %r" % fragment)
    for name in ("decisive", "freedom"):
        if fact.get(name) and normalize(fact[name]) not in text:
            gaps.append("%s %r" % (name, fact[name]))
    for link in fact.get("links", []):
        if link["target"] not in section and not re.search(r"\[\[%s(?:\]\]|\|)" % re.escape(link["written"]), section):
            gaps.append("link to %s" % link["target"])
    unordered = [value for value in fact.get("values", []) if "*" in value["path"]]
    for value in unordered:  # a member of a declaration list whose order carries no meaning
        key = re.search(r"(?<![\w/-])%s[\"'*`]*\s*[:=]" % re.escape(value["path"][0]), section)
        if not key or not re.compile(value_pattern(value)).search(section, key.end()):
            gaps.append("value %s contains %s" % (value["path"][0], value["value"]))
    scalars = [value for value in fact.get("values", []) if "*" not in value["path"]
               and not any(isinstance(key, int) for key in value["path"])]
    for value in scalars:
        if find_value(section, value) is None:
            gaps.append("value %s = %s" % ("/".join(map(str, value["path"])), value["value"]))
    lists = {}
    for value in fact.get("values", []):
        if "*" not in value["path"] and any(isinstance(key, int) for key in value["path"]):
            lists.setdefault(value["path"][0], []).append(value)
    for key, items in lists.items():
        for value in list_items_in_order(section, items):
            gaps.append("value %s = %s" % ("/".join(map(str, value["path"])), value["value"]))
    return gaps


def full_delivery(payload, facts, identities):
    """{identity: gaps of its best section, or ['no section']} for the given identities."""
    found = sections(payload, facts.keys())
    result = {}
    for ident in identities:
        candidates = found.get(ident, [])
        if not candidates:
            result[ident] = ["no section starts with this identity"]
            continue
        best = min((section_gaps(section, facts[ident]) for section in candidates), key=len)
        result[ident] = best
    return result


OWNER = r"(?:owned by|owner:?|realized by|belongs to)\s*\**`?%s(?![\w/-])"


def check_payload(payload, reader, expectation=None):
    """Judge one delivered payload for READER: a Package path, "." for Root, or "plan" for the plan.md arm."""
    expectation = expectation or expected()
    facts = expectation["facts"]
    problems, report = [], {"reader": reader}
    if reader == "plan":
        incomplete = {ident: gaps for ident, gaps in full_delivery(payload, facts, sorted(facts)).items() if gaps}
        report["incomplete"] = incomplete
        problems += ["%s not in full: %s" % (ident, gaps) for ident, gaps in incomplete.items()]
    elif reader == ".":
        found = sections(payload, facts.keys())
        for ident, owner in expectation["rootInventory"].items():
            texts = found.get(ident, [])
            if not any(re.search(OWNER % re.escape(owner), text) for text in texts):
                problems.append("Root's inventory lacks %s with its owner %s" % (ident, owner))
    else:
        selection = expectation["selection"][reader]
        required = list(selection["own"]) + list(selection["required"])
        delivered = full_delivery(payload, facts, [ident for ident in required if ident in facts])
        incomplete = {ident: gaps for ident, gaps in delivered.items() if gaps}
        report["incomplete"] = incomplete
        problems += ["%s not in full: %s" % (ident, gaps) for ident, gaps in incomplete.items()]
        found = sections(payload, facts.keys())
        for ident in selection["required"]:
            if ident not in facts:
                continue
            owner = facts[ident]["owner"]
            texts = found.get(ident, [])
            if owner != reader and texts and not any(re.search(OWNER % re.escape(owner), text) for text in texts):
                problems.append("%s is not attributed to its owner %s" % (ident, owner))
            if owner != reader and any(re.search(OWNER % re.escape(reader), text) for text in texts):
                problems.append("%s is presented as owned by the reader %s" % (ident, reader))
        for ident in selection["forbidden"]:
            fact = facts.get(ident, {})
            canary = normalize(fact.get("decisive", ""))
            if ident in found or (canary and canary in normalize(payload)):
                problems.append("forbidden %s is delivered" % ident)
    report["problems"] = problems
    report["passed"] = not problems
    return report


# -- faults ------------------------------------------------------------------------

def section_span(payload, ident, facts):
    marks = headers(payload, facts.keys())
    for index, (offset, found) in enumerate(marks):
        if found == ident:
            end = marks[index + 1][0] if index + 1 < len(marks) else len(payload)
            # the section whose content is fullest, so that an inventory line is not mutated instead
            if not section_gaps(payload[offset:end], facts[ident]):
                return offset, end
    raise ControlError("no full section of %s to mutate" % ident)


def mutate(payload, fault, expectation=None):
    """Apply one declared fault to a payload that delivers its Definition in full."""
    facts = (expectation or expected())["facts"]
    op = fault["op"]
    if op == "truncate":
        return payload[:int(len(payload) * fault["keep"])]
    start, end = section_span(payload, fault["definition"], facts)
    section = payload[start:end]
    if op == "delete-section":
        changed = ""
    elif op == "header-only":
        changed = section.split("\n", 1)[0] + "\n"
    elif op in ("delete-text", "replace-text"):
        old = fault["text"] if op == "delete-text" else fault["from"]
        new = "" if op == "delete-text" else fault["to"]
        pattern = r"\s+".join(map(re.escape, old.split()))
        changed, count = re.subn(pattern, new, section, count=1)
        if not count:
            raise ControlError("%s: the text is not in the section" % fault["id"])
    elif op == "replace-value":
        separator = r"([\"'*`]*\s*[:=]\s*[\[{(\"'`]*\s*)"
        pattern = r"(?<![\w/-])(%s)%s%s(?![\w.])" % (re.escape(fault["key"]), separator, re.escape(fault["from"]))
        if fault["to"]:
            changed, count = re.subn(pattern, lambda m: m.group(1) + m.group(2) + fault["to"], section, count=1,
                                     flags=re.IGNORECASE)
        else:
            changed, count = re.subn(pattern, "", section, count=1, flags=re.IGNORECASE)
        if not count:
            raise ControlError("%s: the value is not in the section" % fault["id"])
    elif op == "delete-reference":
        changed = section
        for text in (fault["target"], fault["written"]):
            changed = re.sub(r"(?<![\w/-])%s(?![\w-])" % re.escape(text), "", changed)
    elif op == "replace-owner":
        owner = facts[fault["definition"]]["owner"]
        changed, count = re.subn(OWNER % re.escape(owner), lambda m: m.group(0).replace(owner, fault["to"]),
                                 section, count=1)
        if not count:
            raise ControlError("%s: no owner attribution in the section" % fault["id"])
    else:
        raise ControlError("unknown fault operation " + op)
    return payload[:start] + changed + payload[end:]


# -- an independent positive payload -------------------------------------------------------

def definition_files(snapshot):
    """{package/id: file} of a control snapshot, from each file's folder and its top-level `id:` line."""
    model = CONTROL / "snapshots" / snapshot / "model"
    found = {}
    for path in sorted(model.rglob("*.yaml")):
        local = re.search(r"(?m)^id: (\S+)$", path.read_text(encoding="utf-8")).group(1)
        found["%s/%s" % (path.parent.relative_to(model).as_posix(), local)] = path
    return found


def source_payload(snapshot, reader, expectation=None):
    """The raw Definition files of the expected selection, each under a header naming identity, owner and reason.

    This is a deliberately different representation from any product Brief: it
    is the authored YAML itself. It establishes that the checker accepts full
    content in another format, not what the product renders.
    """
    expectation = expectation or expected()
    files = definition_files(snapshot)
    facts = expectation["facts"]
    if reader == "plan":
        chosen = {ident: "" for ident in facts}
    else:
        selection = expectation["selection"][reader]
        chosen = {ident: "own" for ident in selection["own"]}
        chosen.update({ident: ", ".join(reasons) for ident, reasons in selection["required"].items()})
    parts = ["# Context of %s" % reader, ""]
    for ident in chosen:
        if ident not in facts:
            continue
        path = files[ident]
        owner = facts[ident]["owner"]
        attribution = "" if owner == reader or reader == "plan" else " (owned by %s; included: %s)" % (owner, chosen[ident])
        parts += ["%s%s" % (ident, attribution), "```yaml", path.read_text(encoding="utf-8").rstrip(), "```", ""]
    return "\n".join(parts)


def root_inventory_payload(expectation=None):
    expectation = expectation or expected()
    return "\n".join("- %s, owned by %s" % (ident, owner) for ident, owner in expectation["rootInventory"].items()) + "\n"


# -- self-test ------------------------------------------------------------------------

def selftest(markitect=None):
    expectation = expected()
    problems, facts = [], {}
    for reader in ("shop/pricing", "shop/marketing", "plan"):
        payload = source_payload("c1", reader, expectation)
        verdict = check_payload(payload, reader, expectation)
        if not verdict["passed"]:
            problems.append("the source payload for %s is rejected: %s" % (reader, verdict["problems"]))
        stale = source_payload("c1-stale", reader, expectation)
        if reader != "shop/marketing" and check_payload(stale, reader, expectation)["passed"]:
            problems.append("the stale model's payload for %s is accepted" % reader)
    payload = source_payload("c1", "shop/pricing", expectation)
    caught = {}
    for fault in expectation["faults"]:
        verdict = check_payload(mutate(payload, fault, expectation), "shop/pricing", expectation)
        caught[fault["id"]] = "detected" if not verdict["passed"] else "MISSED"
        if verdict["passed"]:
            problems.append("fault %s is not detected" % fault["id"])
    facts["faults"] = caught
    # A listed linked/scoped contract is part of the expected selection, even when it has no special reason.
    pricing = expectation["selection"]["shop/pricing"]
    complete_selection = {ident: ["own"] for ident in pricing["own"]}
    complete_selection.update({ident: list(reasons) for ident, reasons in pricing["required"].items()})
    complete_selection.update({ident: ["connected"] for ident in pricing["connectedContracts"]})
    missing_contract = dict(complete_selection)
    missing_contract.pop(pricing["connectedContracts"][0])
    missing_result = compare_selection(missing_contract, pricing)
    facts["connected-contract-omission"] = missing_result
    if missing_result["passed"] or pricing["connectedContracts"][0] not in missing_result["connectedContractsMissing"]:
        problems.append("a missing connected contract is accepted")
    widened = payload + "\n" + source_payload("c1", "shop/marketing", expectation)
    if check_payload(widened, "shop/pricing", expectation)["passed"]:
        problems.append("a payload that adds marketing's context is accepted for pricing")
    root = root_inventory_payload(expectation)
    if not check_payload(root, ".", expectation)["passed"]:
        problems.append("Root's inventory payload is rejected")
    shortened = "\n".join(line for line in root.splitlines() if "banner" not in line)
    if check_payload(shortened, ".", expectation)["passed"]:
        problems.append("Root's inventory without banner is accepted")
    if check_payload(root, "shop/pricing", expectation)["passed"]:
        problems.append("an inventory of identities is accepted as full delivery")
    import plan_packet  # noqa: E402  (the plan.md formatter is under test here, not a source of expectations)
    with __import__("tempfile").TemporaryDirectory(prefix="context-plan-") as folder:
        packet = materialize("c1", Path(folder) / "repo")
        plan = plan_packet.render(Path(packet["target"]))
        verdict = check_payload(plan["text"], "plan", expectation)
        if not verdict["passed"]:
            problems.append("plan.md of the control model is incomplete: %s" % verdict["problems"])
        facts["plan.md"] = "every Definition in full" if verdict["passed"] else verdict["problems"]
        for fault in expectation["faults"]:
            if fault["op"] in ("truncate", "replace-owner"):
                continue
            if check_payload(mutate(plan["text"], fault, expectation), "plan", expectation)["passed"]:
                problems.append("plan.md fault %s is not detected" % fault["id"])
        if markitect:
            for reader in ("shop/pricing", "shop/marketing"):
                actual, status = actual_selection(markitect, packet["target"], reader)
                compared = compare_selection(actual, expectation["selection"][reader])
                facts["selection " + reader] = {"status": status, **compared}
    return {"passed": not problems, "problems": problems, "facts": facts}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = parser.add_subparsers(dest="command", required=True)
    make = sub.add_parser("materialize")
    make.add_argument("snapshot", choices=("c1", "c1-stale"))
    make.add_argument("target")
    select = sub.add_parser("selection")
    select.add_argument("--markitect", required=True)
    select.add_argument("--package", action="append")
    judge = sub.add_parser("check")
    judge.add_argument("--payload", required=True)
    judge.add_argument("--reader", required=True)
    judge.add_argument("--expected", default=str(EXPECTED))
    test = sub.add_parser("selftest")
    test.add_argument("--markitect")
    args = parser.parse_args(argv)
    try:
        if args.command == "materialize":
            result = materialize(args.snapshot, args.target)
        elif args.command == "selection":
            import tempfile
            with tempfile.TemporaryDirectory(prefix="context-selection-") as folder:
                packet = materialize("c1", Path(folder) / "repo")
                result = {}
                for package in args.package or sorted(expected()["selection"]):
                    actual, status = actual_selection(args.markitect, packet["target"], package)
                    result[package] = {"status": status, "actual": actual,
                                       **compare_selection(actual, expected()["selection"][package])}
        elif args.command == "check":
            result = check_payload(Path(args.payload).read_text(encoding="utf-8"), args.reader,
                                   expected(args.expected))
        else:
            result = selftest(args.markitect)
    except ControlError as error:
        print("context_controls: %s" % error, file=sys.stderr)
        return 1
    print(json.dumps(result, indent=2, ensure_ascii=False))
    return 0 if result.get("passed", True) else 1


if __name__ == "__main__":
    sys.exit(main())
