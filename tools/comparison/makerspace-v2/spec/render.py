"""Render the single-agent arm's Makerspace v2 documentation from the model.

Every Definition body appears verbatim, with body links reduced to their
display text and SUBSTITUTIONS replacing the phrases that name the model
itself. Typed `spec` fields are written as fixed sentences and tables, so the
document carries exactly what the model states: operations with arguments,
refusals, results, permissions and audit targets; records with their fields;
value types with their formats and ranges; refusal codes; and the
architecture (Packages, dependency rules, placements, interfaces).

`architecture` prints only the architecture section, which the neutral
requirements copy (../requirements/w0.md) carries verbatim.

Usage, from the repository root:
  python -I -B tools/comparison/makerspace-v2/spec/render.py render [--out FILE]
  python -I -B tools/comparison/makerspace-v2/spec/render.py check [--doc FILE]
  python -I -B tools/comparison/makerspace-v2/spec/render.py architecture
"""
import argparse
import json
import pathlib
import re
import sys

HERE = pathlib.Path(__file__).resolve().parent
MODEL = HERE.parent / "model"
DOC = HERE / "requirements-w0.md"
LINK = re.compile(r"\[\[([^\]|]+)(?:\|([^\]]+))?\]\]")
SUBSTITUTIONS = [("this model", "this document"), ("of the model", "of this document")]
AREAS = ["members", "training", "equipment", "bookings", "materials", "billing", "safety", "reporting"]
TITLES = {"members": "Members", "training": "Training", "equipment": "Equipment", "bookings": "Bookings",
          "materials": "Materials", "billing": "Billing", "safety": "Safety", "reporting": "Reporting"}
ROLES = [("member-role", "member"), ("staff-role", "staff"), ("admin-role", "admin")]


# ---------------------------------------------------------------- loading
def scalar_value(text):
    text = text.strip()
    if text.startswith('"'):
        return json.loads(text)  # the model's double-quoted strings are JSON strings
    if re.fullmatch(r"-?\d+", text):
        return int(text)
    if text in ("true", "false"):
        return text == "true"
    return text


def split_flow(text):
    """Split a YAML flow sequence or mapping body at top-level commas."""
    parts, depth, quoted, current = [], 0, False, ""
    for ch in text:
        if ch == '"' and (len(current) - len(current.rstrip("\\"))) % 2 == 0:
            quoted = not quoted
        if not quoted and ch in "[{":
            depth += 1
        if not quoted and ch in "]}":
            depth -= 1
        if ch == "," and depth == 0 and not quoted:
            parts.append(current)
            current = ""
        else:
            current += ch
    if current.strip():
        parts.append(current)
    return [p.strip() for p in parts]


def flow_value(text):
    text = text.strip()
    if text.startswith("[") and text.endswith("]"):
        return [flow_value(p) for p in split_flow(text[1:-1])]
    if text.startswith("{") and text.endswith("}"):
        out = {}
        for part in split_flow(text[1:-1]):
            key, value = part.split(":", 1)
            out[key.strip()] = flow_value(value)
        return out
    return scalar_value(text)


def parse(path):
    """Parse the generated Definition layout: imports, id, kind, body |-, spec."""
    lines = path.read_text(encoding="utf-8").split("\n")
    out = {"imports": {}, "spec": {}}
    seen_spec = False
    i = 0
    while i < len(lines):
        line = lines[i]
        if line.startswith("  - {package:"):
            item = flow_value(line[4:])
            out["imports"][item["as"]] = item["package"]
        elif line.startswith("id: "):
            out["id"] = line[4:]
        elif line.startswith("kind: "):
            out["kind"] = line[6:]
        elif line == "body: |-":
            body = []
            i += 1
            while i < len(lines) and (lines[i].startswith("  ") or lines[i] == ""):
                if lines[i] == "" and i + 1 < len(lines) and lines[i + 1].startswith("spec"):
                    break
                body.append(lines[i][2:])
                i += 1
            out["body"] = "\n".join(body).rstrip("\n")
            continue
        elif line == "spec: {}":
            pass
        elif line.startswith("  ") and not line.startswith("    ") and ":" in line and seen_spec:
            key, _, rest = line[2:].partition(":")
            if rest.strip():
                out["spec"][key] = flow_value(rest)
            else:
                items = []
                while i + 1 < len(lines) and lines[i + 1].startswith("    - "):
                    i += 1
                    items.append(flow_value(lines[i][6:]))
                out["spec"][key] = items
        elif line in ("imports:", "spec:"):
            seen_spec = seen_spec or line == "spec:"
        elif line:
            raise ValueError(f"{path}: unexpected line {i + 1}: {line!r}")
        i += 1
    return out


def load():
    model = {}
    for path in sorted(MODEL.rglob("*.yaml")):
        d = parse(path)
        d["package"] = path.parent.relative_to(MODEL).as_posix()
        model[(d["package"], d["id"])] = d
    return model


def resolve(model, d, ref):
    """Resolve a short or alias-qualified reference written in d."""
    if "." in ref and ref.split(".", 1)[0] in d["imports"]:
        alias, id_ = ref.split(".", 1)
        return (d["imports"][alias], id_)
    return (d["package"], ref)


def kind_of(d):
    return d["kind"].split(".")[-1]


# ---------------------------------------------------------------- rendering
def text(body):
    out = LINK.sub(lambda m: m.group(2) or m.group(1).split(".")[-1], body)
    for old, new in SUBSTITUTIONS:
        out = out.replace(old, new)
    return out


def value_range(spec):
    low, high = spec.get("lowest"), spec.get("highest")
    if low is not None and high is not None:
        return f"{low} to {high}"
    if low is not None:
        return f"{low} or more"
    if high is not None:
        return f"at most {high}"
    return ""


def type_phrase(model, d, ref):
    t = model[resolve(model, d, ref)]
    k = kind_of(t)
    if k == "Scalar":
        s = t["spec"]
        if s["base"] in ("integer", "number"):
            return " ".join(filter(None, [s["base"], value_range(s)]))
        if s["base"] == "boolean":
            return "boolean"
        return t["id"].replace("-", " ")
    if k == "Enumeration":
        values = [f"`{m['value']}`" for m in t["spec"]["members"]]
        return "enumeration " + (", ".join(values[:-1]) + " or " + values[-1] if len(values) > 1 else values[0])
    return f"{t['id'].replace('-', ' ')} record"


def field_phrase(model, d, f):
    phrase = type_phrase(model, d, f["fieldType"])
    if f.get("list"):
        phrase = f"list of {phrase}"
    if f.get("refersTo"):
        target = model[resolve(model, d, f["refersTo"])]["id"].replace("-", " ")
        phrase += f", an existing {target}"
    if f.get("nullable"):
        phrase += ", nullable"
    return phrase


def render_operation(model, d):
    s = d["spec"]
    out = [f"#### `{s['wireName']}`", "", ("Mutation. " if s["mutation"] else "Read. ") + text(d["body"]), ""]
    args = s.get("arguments", [])
    out.append("- Arguments: " + ("; ".join(f"`{a['name']}` ({field_phrase(model, d, a)})" for a in args) if args
                                  else "none") + ", besides `actor` and `at`.")
    refusals = s.get("refusals", [])
    if refusals:
        out.append("- Refusals, in this order: " + "; ".join(
            f"`{model[resolve(model, d, r['refusal'])]['spec']['code']}` when {r['when']}" for r in refusals) + ".")
    results = s.get("results", [])
    out.append("- Result: " + ("; ".join(f"`{r['name']}` ({field_phrase(model, d, r)})" for r in results) or "none") + ".")
    given = {model[resolve(model, d, p["role"])]["id"]: p["scope"] for p in s.get("permissions", [])}
    unknown = set(given) - {role for role, _ in ROLES}
    if unknown:
        raise ValueError(f"{d['id']}: permission for a role the document does not list: {sorted(unknown)}")
    out.append("- Permissions: " + ", ".join(f"{name} {given.get(role, 'none')}" for role, name in ROLES) + ".")
    if "auditTarget" in s:
        out.append(f"- Audit target: `{s['auditTarget']}`.")
    return out + [""]


def render_entity(model, d):
    out = [f"#### Record: {d['id'].replace('-', ' ')}", "", text(d["body"]), ""]
    if "key" in d["spec"]:
        out += [f"Identified by `{d['spec']['key']}`.", ""]
    out += ["| Field | Type |", "| --- | --- |"]
    out += [f"| `{f['name']}` | {field_phrase(model, d, f)} |" for f in d["spec"]["fields"]]
    return out + [""]


def render_values(model, defs):
    out = []
    scalars = [d for d in defs if kind_of(d) == "Scalar"]
    enums = [d for d in defs if kind_of(d) == "Enumeration"]
    if scalars:
        out += ["| Value type | Base | Range | Rule |", "| --- | --- | --- | --- |"]
        for d in scalars:
            s = d["spec"]
            rng = value_range(s)
            unit = ""
            if "measuredIn" in s:
                unit = " (" + model[resolve(model, d, s["measuredIn"])]["id"] + ")"
            out.append(f"| {d['id'].replace('-', ' ')} | {s['base']}{unit} | {rng} | {text(d['body'])} |")
        out.append("")
    for d in enums:
        out += [f"Enumeration {d['id'].replace('-', ' ')}: {text(d['body'])}", ""]
        out += [f"- `{m['value']}`" + (f": {m['meaning']}" if m.get("meaning") else "") for m in d["spec"]["members"]]
        out.append("")
    return out


def render_codes(model, defs):
    codes = [d for d in defs if kind_of(d) == "RefusalCode"]
    if not codes:
        return []
    return ["| Code | Meaning |", "| --- | --- |"] + [f"| `{d['spec']['code']}` | {text(d['body'])} |" for d in codes] + [""]


def scope_phrase(d):
    """Where a rule applies, as the model's appliesTo states it."""
    applies = d["spec"].get("appliesTo")
    if not applies:
        return f"Applies to the {d['package'].split('/')[-1]} package."
    if applies == ["makerspace"]:
        return "Applies to every package."
    names = [p.split("/")[-1] for p in applies]
    return "Applies to the " + (", ".join(names[:-1]) + " and " + names[-1] if len(names) > 1 else names[0]) + \
        (" packages." if len(names) > 1 else " package.")


def render_rules(defs):
    out = []
    for d in defs:
        if kind_of(d) in ("Rule", "TestObligation", "DocumentationObligation"):
            out += [text(d["body"]), "", f"*{scope_phrase(d)}*", ""]
    return out


def package(model, name):
    return [d for (p, _), d in model.items() if p == f"makerspace/{name}" or (name == "" and p == "makerspace")]


def render_architecture(model):
    project = model[("makerspace", "makerspace")]["spec"]
    out = ["## Architecture", "",
           f"The program is written in {project['language']}; the module root is `{project['moduleRoot']}`. "
           f"`{project['buildCommand']}` builds it and `{project['testCommand']}` runs its tests. "
           "No dependency beyond the language's standard library is permitted.", "",
           "Each Package below is one Go package under the module root, named after it.", ""]
    rules = sorted((d for d in package(model, "") if kind_of(d) == "DependencyRule"), key=lambda d: d["id"])
    out += ["| Package | May depend on | Rule |", "| --- | --- | --- |"]
    for d in rules:
        pkg = d["spec"]["appliesTo"][0].split("/")[-1]
        may = ", ".join(p.split("/")[-1] for p in d["spec"]["mayDependOn"]) or "nothing"
        out.append(f"| `{pkg}` | {may} | {text(d['body'])} |")
    out += ["", "A dependency is any use of another Package's records, rules or operations, in the documentation and in the code. "
            "The standard library is not counted.", ""]
    for d in sorted((d for d in package(model, "") if kind_of(d) == "Placement"), key=lambda d: d["id"]):
        concern = model[resolve(model, d, d["spec"]["concern"])]["id"]
        out += [f"- **{concern.capitalize()}** is realized in `{d['spec']['realizedIn'].split('/')[-1]}`: {text(d['body'])}"]
    out.append("")
    for area in AREAS:
        for d in package(model, area):
            if kind_of(d) == "Interface":
                ops = ", ".join(f"`{model[resolve(model, d, o)]['spec']['wireName']}`" for o in d["spec"].get("operations", []))
                types = ", ".join(model[resolve(model, d, t)]["id"].replace("-", " ") for t in d["spec"].get("types", []))
                out += [f"- **{TITLES[area]} interface:** {text(d['body'])} Operations: {ops}. Types: {types}."]
    return out + [""]


def render(model):
    project = model[("makerspace", "makerspace")]
    out = ["# Makerspace requirements: W0 (fresh build)", "",
           "This document describes the complete product for the first delivery.", "",
           "## Overview", "", text(project["body"]), ""]
    out += render_architecture(model)
    out += ["## Program interface", ""] + render_rules(package(model, "protocol"))
    pol = package(model, "policies")
    out += ["## Shared rules and policies", ""] + render_rules(pol)
    out += ["### Value types", ""] + render_values(model, pol)
    out += [f"- Unit {d['id']} (`{d['spec']['symbol']}`): {text(d['body'])}" for d in pol if kind_of(d) == "Unit"] + [""]
    out += [line for d in pol if kind_of(d) == "Entity" for line in render_entity(model, d)]
    out += ["### Roles", ""] + [f"- `{model[('makerspace/policies', r)]['id'].split('-')[0]}`: {text(model[('makerspace/policies', r)]['body'])}"
                                for r, _ in ROLES] + [""]
    out += ["### Shared refusal codes", ""] + render_codes(model, pol)
    out += ["### Operations", ""]
    out += [line for d in pol if kind_of(d) in ("Operation", "AuditedOperation") for line in render_operation(model, d)]
    out += ["### Audited operations", "", text(model[("makerspace/policies", "AuditedOperation")]["body"]),
            "", text(model[("makerspace/policies", "auditTarget")]["body"]), ""]
    for area in AREAS:
        defs = package(model, area)
        out += [f"## {TITLES[area]}", ""] + render_rules(defs)
        out += render_values(model, defs) + render_codes(model, defs)
        out += [line for d in defs if kind_of(d) == "Entity" for line in render_entity(model, d)]
        out += [line for d in defs if kind_of(d) in ("Operation", "AuditedOperation") for line in render_operation(model, d)]
    out += ["## Delivery", ""] + render_rules(package(model, ""))
    return "\n".join(out).rstrip("\n") + "\n"


def check(model, doc):
    expected = render(model)
    if doc != expected:
        return "the document differs from a fresh render"
    for (pkg, id_), d in model.items():
        if text(d["body"]) not in doc:
            return f"body of {pkg}/{id_} is missing"
    neutral = (HERE.parent / "requirements" / "w0.md").read_text(encoding="utf-8")
    section = "\n".join(render_architecture(model)).replace("## Architecture", "## 16 Architecture", 1).rstrip("\n") + "\n"
    if not neutral.endswith("\n\n" + section):
        return "requirements/w0.md does not end with the model's architecture section"
    return None


def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=["render", "check", "architecture"])
    parser.add_argument("--out", type=pathlib.Path, default=DOC)
    parser.add_argument("--doc", type=pathlib.Path, default=DOC)
    args = parser.parse_args(argv)
    model = load()
    if args.command == "render":
        args.out.write_text(render(model), encoding="utf-8", newline="\n")
        return 0
    if args.command == "architecture":
        sys.stdout.buffer.write("\n".join(render_architecture(model)).encode("utf-8"))
        return 0
    problem = check(model, args.doc.read_text(encoding="utf-8"))
    if problem:
        print(problem)
        return 1
    print(f"{args.doc.name} matches the model ({len(model)} Definitions)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
