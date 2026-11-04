"""The pages of the large Makerspace model: parse them, check their form, write the requirements document and the facts.

  python -I -B pages.py check MODEL_DIR [REGISTRY_DIR]   the form of every page, links, operations, the dependency graph
  python -I -B pages.py render MODEL_DIR OUT_FILE        the requirements document every variant receives as task.md
  python -I -B pages.py facts MODEL_DIR OUT_FILE         the operations as data, for the generated hidden cases

The model is the one source: the requirements document is written from it, so both forms of the task say the same.
"""
import argparse
import json
import pathlib
import re
import sys

LINK = re.compile(r"\[\[([^\]|]+)(?:\|([^\]]+))?\]\]")
ROLES = ("member", "staff", "admin")
# Words that would tell a variant working from the document about the model's own language.
FORBIDDEN = re.compile(r"\bPackages?\b|\bDefinitions?\b|\bKinds?\b|\[\[|\bmodel\b|\bManagers?\b")
SHARED_FIRST = ("order-of-checks", "argument-values", "values", "refusal-codes", "authorization", "audit", "audit-entry", "time",
                "computed-times", "money", "list-order", "identifier-limits", "integer-limits", "request-limits", "list-audit-entries")
# The order of the module sections: the design's build order.
MODULES = ("protocol", "members", "equipment", "ledger", "notices", "households", "guardians", "waivers", "ppe", "lockouts", "inspections",
           "staffing", "parts", "rooms", "billing", "deposits", "passes", "packs", "promotions", "scholarships", "refunds", "dunning", "lockers", "guests",
           "volunteering", "training", "loans", "materials", "consumables", "bookings", "safety", "reporting", "access", "instructors", "classes",
           "assessments", "waitlists", "payouts", "events", "hazmat", "purchasing", "stocktake", "workorders", "escalations", "projects",
           "quotes", "jobs", "compliance", "finance")


class Page:
    def __init__(self, name, kind, title, body, front):
        self.name, self.kind, self.title, self.body, self.front = name, kind, title, body, front

    @property
    def module(self):
        return self.name.split("/")[0] if "/" in self.name else None


def read_page(path, name):
    text = path.read_text(encoding="utf-8").replace("\r\n", "\n")
    m = re.match(r"---\n(.*?)\n---\n", text, re.S)
    if not m:
        raise ValueError(f"{name}: no front matter")
    front = dict(line.split(":", 1) for line in m.group(1).splitlines() if ":" in line and not line.startswith(" "))
    front = {k.strip(): v.strip() for k, v in front.items()}
    rest = text[m.end():].lstrip("\n")
    head = re.match(r"# (.+)\n", rest)
    if not head:
        raise ValueError(f"{name}: no heading")
    return Page(name, front.get("kind", ""), head.group(1).strip(), rest[head.end():].strip("\n"), front)


def load(root):
    root = pathlib.Path(root)
    pages = {}
    for path in sorted(root.rglob("*.md")):
        name = path.relative_to(root).with_suffix("").as_posix()
        if name.startswith("registry"):
            continue
        pages[name] = read_page(path, name)
    return pages


def parse_argument(line, where):
    m = re.match(r"\s*- `(\w+)`: (.*)$", line)
    if not m:
        raise ValueError(f"{where}: argument line not understood: {line.strip()}")
    name, spec = m.group(1), m.group(2)
    arg = {"name": name}
    plain = LINK.sub(lambda l: l.group(2) or l.group(1), spec)
    ref = re.search(r"naming an existing \[\[([^\]|]+)", spec)
    if spec.startswith("identifier"):
        arg["type"] = "identifier"
        if ref:
            arg["ref"] = ref.group(1)
    elif spec.startswith("text"):
        arg["type"] = "text"
    elif spec.startswith(("timestamp", "date", "month")):
        arg["type"] = spec.split()[0].rstrip(",")
    elif spec.startswith(("flag", "boolean")):
        arg["type"] = "flag"
    elif spec.startswith("integer from"):
        r = re.match(r"integer from `?(\d+)`? to `?(\d+)`?", spec)
        if not r:
            raise ValueError(f"{where}: integer range not understood: {spec}")
        arg.update(type="integer", min=int(r.group(1)), max=int(r.group(2)))
    elif "`" in plain:
        arg.update(type="enum", values=re.findall(r"`([^`]+)`", plain))
    else:
        raise ValueError(f"{where}: argument type not understood: {spec}")
    return arg


def parse_operation(page):
    lines = page.body.split("\n")
    op = {"page": page.name, "title": page.title, "args": [], "refusals": [], "result": [], "permissions": {}, "audit": None}
    section = None
    for line in lines:
        if line.startswith("- **Wire name:**"):
            m = re.match(r"- \*\*Wire name:\*\* `(\w+)`, a (mutation|read)\.", line)
            if not m:
                raise ValueError(f"{page.name}: wire name line not understood: {line}")
            op["wire"], op["mutation"] = m.group(1), m.group(2) == "mutation"
            section = None
        elif line.startswith("- **Arguments"):
            section = "args" if "only `actor` and `at`" not in line else None
        elif line.startswith("- **Refusals"):
            section = "refusals"
        elif line.startswith("- **Result:**"):
            section = "result"
        elif line.startswith("- **Permissions:**"):
            for role, perm in re.findall(r"\b(member|staff|admin) (all|own)\b", line):
                op["permissions"][role] = perm
            section = None
        elif line.startswith("- **Audit target:**"):
            op["audit"] = re.sub(r"^- \*\*Audit target:\*\* ", "", line).rstrip(".")
            section = None
        elif section == "args" and line.startswith("  - "):
            op["args"].append(parse_argument(line, page.name))
        elif section == "refusals" and re.match(r"\s+\d+\. ", line):
            m = re.match(r"\s+\d+\. `(\w+)`", line)
            if not m:
                raise ValueError(f"{page.name}: refusal line not understood: {line}")
            op["refusals"].append(m.group(1))
        elif section == "result" and line.startswith("  - "):
            m = re.match(r"\s+- `(\w+)`", line)
            if not m:
                raise ValueError(f"{page.name}: result line not understood: {line}")
            op["result"].append(m.group(1))
    if "wire" not in op:
        raise ValueError(f"{page.name}: no wire name")
    return op


def operations(pages):
    ops = {}
    for page in pages.values():
        if page.kind == "Operation":
            op = parse_operation(page)
            if op["wire"] in ops:
                raise ValueError(f"{op['wire']}: on {ops[op['wire']]['page']} and {page.name}")
            ops[op["wire"]] = op
    return ops


def resolve(pages, target):
    if target in pages:
        return target
    # A link to a module names its folder.
    if any(n.startswith(target + "/") for n in pages):
        return target
    # A short link names a page by its last part when that is unique.
    found = [n for n in pages if n.rsplit("/", 1)[-1] == target or n == target.lower()]
    return found[0] if len(found) == 1 else None


def uses_of(pages):
    """Module -> modules its interface says it uses."""
    uses = {}
    for page in pages.values():
        if page.kind == "Interface":
            line = next((l for l in page.body.split("\n") if l.startswith("- **Uses:**")), "")
            uses[page.module] = sorted({t.split("/")[0] for t, _ in LINK.findall(line)})
    return uses


def check(root, registry=None):
    pages = load(root)
    errors = []
    try:
        ops = operations(pages)
    except ValueError as e:
        return [str(e)], {}
    for page in pages.values():
        for target, _ in LINK.findall(page.body):
            if not resolve(pages, target):
                errors.append(f"{page.name}: broken link [[{target}]]")
    shared = set(re.findall(r"^\| `(\w+)` \|", pages["refusal-codes"].body, re.M)) if "refusal-codes" in pages else set()
    codes = {}
    for page in pages.values():
        if page.kind == "Interface":
            for code in re.findall(r"^\| `(\w+)` \|", page.body, re.M):
                codes.setdefault(code, []).append(page.module)
    for op in ops.values():
        where = op["page"]
        if not op["permissions"]:
            errors.append(f"{where}: no permissions")
        if not op["result"]:
            errors.append(f"{where}: no result")
        if op["mutation"] and not op["audit"]:
            errors.append(f"{where}: a mutation without an audit target")
        if not op["mutation"] and op["audit"]:
            errors.append(f"{where}: a read with an audit target")
        for code in op["refusals"]:
            if code not in shared and code not in codes:
                errors.append(f"{where}: refusal `{code}` is in no interface and is not shared")
    for code, owners in codes.items():
        if len(owners) > 1:
            errors.append(f"refusal `{code}` is listed by several interfaces: {', '.join(owners)}")
        if code in shared:
            errors.append(f"refusal `{code}` is shared and also listed by {owners[0]}")
    for page in pages.values():
        if page.module and page.module not in MODULES:
            errors.append(f"{page.name}: module {page.module} is not in the module list")
    # One-way dependencies, by the build order.
    uses = uses_of(pages)
    order = {m: i for i, m in enumerate(MODULES)}
    for module, used in uses.items():
        for other in used:
            if other in order and module in order and module != "protocol" and order[other] >= order[module]:
                errors.append(f"{module} uses {other}, which comes later")
    if registry:
        registry = pathlib.Path(registry)
        for module in MODULES:
            if f"{module}/{module}-interface" in pages or module == "protocol":
                if not (registry / f"{module}.md").is_file():
                    errors.append(f"no registry page for {module}")
    return errors, ops


def reduce_links(text, pages):
    def plain(m):
        if m.group(2):
            return m.group(2)
        target = resolve(pages, m.group(1))
        title = pages[target].title if target in pages else m.group(1)
        return title if title[:2].isupper() else title[0].lower() + title[1:]
    return LINK.sub(plain, text).replace("this model", "this document").replace("of the model", "of this document").replace("the model", "this document")


PREFACE = """# Makerspace requirements: large (fresh build)

This document describes the complete product for the first delivery.

How to read an operation in this document. Its text states the operation's effect. A list below it states: its wire name, the exact `op` value, and whether the operation is a mutation, which changes state and is audited, or a read; its arguments besides `actor` and `at`, which every operation takes, all of them required; an argument that names an existing record must name one that exists (check 5 of the order of checks); its refusals, its own, checked at check 7 of the order of checks in the order listed, the first that applies deciding the code; its result, the members of a successful `result`, each with its type; its permissions, for each role `all` or `own`, and a role that is not listed may not use the operation; and for a mutation its audit target, the `target` of its audit entry: the name of an argument or of a new identifier whose value is the target, `a/b` joining two values with a slash, for example `laser-101/ada` for course `laser-101` and member `ada`, and `month` being the month string, for example `2026-03`.

What a module's interface states is what other modules may use of it: its operations, its records, the refusal codes it adds and the capabilities other modules ask of it that are not operations. Other modules use nothing else of it; the module's other pages are internal to it, except its rules that bind other modules. It also names the interfaces of the modules it uses.
"""


def section(page, pages, level="###"):
    return f"{level} {page.title}\n\n{reduce_links(page.body, pages)}\n"


def render(root):
    pages = load(root)
    out = [PREFACE]
    out.append(section(pages["project"], pages, "##").replace("## Makerspace", "## Overview", 1))
    out.append(section(pages["architecture"], pages, "##"))
    out.append("## Rules every module follows\n")
    done = {"project", "architecture", "Operation", "Interface", "documentation", "tests"}
    for name in SHARED_FIRST:
        out.append(section(pages[name], pages))
        done.add(name)
    rest = sorted(n for n in pages if "/" not in n and n not in done)
    for name in rest:
        out.append(section(pages[name], pages))
        done.add(name)
    kind_rank = {"Interface": 0, "std/Rule": 1, "std/Concept": 2, "Operation": 3}
    for module in MODULES:
        chosen = [p for p in pages.values() if p.module == module]
        if not chosen:
            continue
        out.append(f"## Module `{module}`\n")
        for page in sorted(chosen, key=lambda p: (kind_rank.get(p.kind, 9), p.name)):
            out.append(section(page, pages))
    out.append("## Delivery\n")
    out.append(section(pages["documentation"], pages))
    out.append(section(pages["tests"], pages))
    return "\n".join(out)


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("cmd", choices=["check", "render", "facts"])
    p.add_argument("model")
    p.add_argument("out", nargs="?")
    a = p.parse_args(argv)
    if a.cmd == "check":
        errors, ops = check(a.model, a.out)
        for e in errors:
            print(e)
        print(f"{len(load(a.model))} pages, {len(ops)} operations, {len(errors)} problems")
        return 1 if errors else 0
    if a.cmd == "render":
        text = render(a.model)
        bad = sorted({m.group(0) for m in FORBIDDEN.finditer(text)})
        if bad:
            for m in FORBIDDEN.finditer(text):
                start = text.rfind("\n", 0, m.start()) + 1
                print(f"forbidden word {m.group(0)!r}: {text[start:text.find(chr(10), m.end())][:160]}", file=sys.stderr)
            return 2
        pathlib.Path(a.out).write_text(text, encoding="utf-8")
        print(f"{len(text.split())} words")
        return 0
    errors, ops = check(a.model)
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 2
    pathlib.Path(a.out).write_text(json.dumps(ops, indent=1, sort_keys=True), encoding="utf-8")
    print(f"{len(ops)} operations")
    return 0


if __name__ == "__main__":
    sys.exit(main())
