"""Render arm B's ordinary documentation from the canonical Library model.

The document is the Library model's Definition bodies, verbatim, under plain
headings. Body links become Markdown links to the linked section with the link
text in LINKS, and SUBSTITUTIONS replace the few phrases that name the model
itself. The policy values come from the loan-policy spec. Nothing else is
added except headings, one introductory sentence and the policy table frame.

Profiles m1 and m2 are Phase A's Library M1 and M2. Phase B's m3, m4 and m5
apply the overlays under phase-b/ cumulatively on top of m2: each overlay
replaces or adds Definition files and may list deleted ones in deleted.txt.

Usage, from the repository root:
  python -I -B tools/comparison/requirements.py render --profile m1 [--model DIR] [--out FILE]
  python -I -B tools/comparison/requirements.py check --profile m1 --doc FILE [--model DIR]
  python -I -B tools/comparison/requirements.py materialize --profile m3 --out NEW_DIR [--model DIR]

`check` re-renders, compares the bytes and confirms that every Definition body
occurs in the document; it prints the per-Definition binding as JSON.
"""
import argparse
import hashlib
import json
import pathlib
import re
import sys

REPO = pathlib.Path(__file__).resolve().parents[2]
DEFAULT_MODEL = REPO / "testdata" / "library" / "m1"
PHASE_B = pathlib.Path(__file__).resolve().parent / "phase-b"
PROFILES = ("m1", "m2", "m3", "m4", "m5")

# Document order: (heading level, title, Definition keys "package/id").
SECTIONS = [
    (2, "Library", ["library/library"]),
    (2, "Shared input rule", ["library/inputs"]),
    (2, "Delivery", ["library/realization"]),
    (2, "Callable interface", ["library/callable-interface"]),
    (2, "Members", []),
    (3, "Member", ["members/member"]),
    (3, "Eligibility", ["members/eligibility"]),
    (3, "Register a member", ["members/register-member"]),
    (3, "Set member status", ["members/set-member-status"]),
    (3, "Inspect a member", ["members/inspect-member"]),
    (2, "Copies", []),
    (3, "Copy", ["catalog/copy"]),
    (3, "Availability", ["catalog/availability"]),
    (3, "Register a copy", ["catalog/register-copy"]),
    (3, "Inspect a copy", ["catalog/inspect-copy"]),
    (2, "Lending", []),
    (3, "Lending policy", ["lending/LendingPolicy", "lending/maxOpenLoans", "lending/loanPeriodDays",
                           "lending/shortLoanPeriodDays", "lending/maxRenewals", "lending/loan-policy"]),
    (3, "Loan", ["lending/loan"]),
    (3, "Borrow", ["lending/borrow"]),
    (3, "Return", ["lending/return"]),
    (3, "Renew", ["lending/renew"]),
    (3, "Inspect loans", ["lending/inspect-loans"]),
    (2, "Overdue notices", []),
    (3, "Assess overdue loans", ["lending/reminders/assess-overdue"]),
]

# Link text and target section for every linkable Definition.
LINKS = {
    "library/library": ("the library", "Library"),
    "library/inputs": ("the shared input rule", "Shared input rule"),
    "library/realization": ("the delivery rule", "Delivery"),
    "library/callable-interface": ("the callable interface", "Callable interface"),
    "members/member": ("member", "Member"),
    "members/eligibility": ("the eligibility rule", "Eligibility"),
    "members/register-member": ("member registration", "Register a member"),
    "members/set-member-status": ("member status change", "Set member status"),
    "members/inspect-member": ("member inspection", "Inspect a member"),
    "catalog/copy": ("copy", "Copy"),
    "catalog/availability": ("the availability rule", "Availability"),
    "catalog/register-copy": ("copy registration", "Register a copy"),
    "catalog/inspect-copy": ("copy inspection", "Inspect a copy"),
    "lending/LendingPolicy": ("the lending policy", "Lending policy"),
    "lending/maxOpenLoans": ("maximum open loans", "Lending policy"),
    "lending/loanPeriodDays": ("loan period", "Lending policy"),
    "lending/shortLoanPeriodDays": ("short loan period", "Lending policy"),
    "lending/maxRenewals": ("maximum renewals", "Lending policy"),
    "lending/renew": ("renewal", "Renew"),
    "lending/loan-policy": ("the lending policy", "Lending policy"),
    "lending/loan": ("loan", "Loan"),
    "lending/borrow": ("borrowing", "Borrow"),
    "lending/return": ("return", "Return"),
    "lending/inspect-loans": ("loan inspection", "Inspect loans"),
    "lending/reminders/assess-overdue": ("overdue assessment", "Assess overdue loans"),
}

# The only wording changes: phrases that name the model itself. Each must
# occur exactly the stated number of times in its Definition body.
SUBSTITUTIONS = [
    ("lending/LendingPolicy", "States the numeric", "The lending policy states the numeric", 1),
    ("lending/loan-policy", "in this definition's spec", "in the table above", 2),
    ("library/callable-interface", "offers every operation of the model", "offers every operation in this document", 1),
    ("library/realization", "must agree with the model.", "must agree with this document.", 1),
]

POLICY_ROWS = [("lending/maxOpenLoans", "Maximum open loans"), ("lending/loanPeriodDays", "Loan period in calendar days"),
               ("lending/shortLoanPeriodDays", "Short loan period in calendar days"), ("lending/maxRenewals", "Maximum renewals per loan")]
INTRO = "This document describes the small library application to build. Links point to the related sections."
FORBIDDEN = ("[[", "]]", "model", "Definition", "definition", "spec", "Package")
M2_EDIT = ("  maxOpenLoans: 5\n", "  maxOpenLoans: 3\n")


class ModelError(ValueError):
    pass


def slug(title):
    return re.sub(r"[^a-z0-9 -]", "", title.lower()).replace(" ", "-")


def parse_block(lines, start):
    """Return the indented block (two-space base indentation) after a key."""
    block, i = [], start
    while i < len(lines) and (lines[i] == "" or lines[i].startswith("  ")):
        block.append(lines[i])
        i += 1
    while block and block[-1] == "":
        block.pop()
    return block, i


def fold(block):
    """YAML folded scalar with strip chomping, for flush two-space text."""
    paragraphs, current = [], []
    for line in block:
        if line == "":
            paragraphs.append(" ".join(current))
            current = []
            continue
        if line.startswith("   "):
            raise ModelError("more-indented folded lines are not supported")
        current.append(line[2:])
    paragraphs.append(" ".join(current))
    return "\n".join(p for p in paragraphs)


def literal(block):
    return "\n".join(line[2:] if line else "" for line in block)


def parse_definition(text):
    """Parse the YAML subset the Library Definitions use; refuse anything else."""
    lines = text.split("\n")
    if lines and lines[-1] == "":
        lines.pop()
    definition = {"imports": {}, "spec": {}}
    i = 0
    while i < len(lines):
        line = lines[i]
        if line == "imports:":
            i += 1
            while i + 1 < len(lines) and lines[i].startswith("  - package: ") and lines[i + 1].startswith("    as: "):
                definition["imports"][lines[i + 1][8:].strip()] = lines[i][13:].strip()
                i += 2
        elif line.startswith("id: ") or line.startswith("kind: "):
            key, value = line.split(": ", 1)
            definition[key] = value.strip()
            i += 1
        elif line in ("body: >-", "body: |-"):
            block, i = parse_block(lines, i + 1)
            definition["body"] = fold(block) if line.endswith(">-") else literal(block)
        elif line == "spec: {}":
            i += 1
        elif line == "spec:":
            block, i = parse_block(lines, i + 1)
            j = 0
            while j < len(block):
                key, _, value = block[j].strip().partition(":")
                if value.strip():
                    definition["spec"][key] = value.strip()
                    j += 1
                else:
                    items = []
                    j += 1
                    while j < len(block) and block[j].startswith("    - "):
                        items.append(block[j][6:].strip())
                        j += 1
                    definition["spec"][key] = items
        else:
            raise ModelError("unsupported line: " + line)
    for key in ("id", "kind", "body"):
        if key not in definition:
            raise ModelError("missing " + key)
    return definition


def model_files(model_dir, profile):
    """The profile's Definition files as {"model/...": bytes}."""
    if profile not in PROFILES:
        raise ModelError("unknown profile " + profile)
    base = pathlib.Path(model_dir)
    files = {path.relative_to(base).as_posix(): path.read_bytes() for path in sorted((base / "model").rglob("*.yaml"))}
    if profile != "m1":
        text = files["model/lending/loan-policy.yaml"].decode("utf-8")
        if text.count(M2_EDIT[0]) != 1:
            raise ModelError("the M2 edit needs exactly one maxOpenLoans: 5 line")
        files["model/lending/loan-policy.yaml"] = text.replace(*M2_EDIT).encode("utf-8")
    for stage in PROFILES[2:PROFILES.index(profile) + 1]:
        overlay = PHASE_B / stage
        for path in sorted((overlay / "model").rglob("*.yaml")):
            files[path.relative_to(overlay).as_posix()] = path.read_bytes()
        deleted = overlay / "deleted.txt"
        for name in deleted.read_text(encoding="utf-8").split() if deleted.is_file() else []:
            if name not in files:
                raise ModelError("deleted file is not in the model: " + name)
            del files[name]
    return dict(sorted(files.items()))


def present_sections(definitions):
    """SECTIONS restricted to the profile's Definitions; headings without any left are dropped."""
    out = []
    for level, title, keys in SECTIONS:
        kept = [key for key in keys if key in definitions]
        if kept or not keys:
            out.append((level, title, kept))
    return out


def load_model(model_dir, profile):
    """Load every local Definition of the Library model, keyed "package/id"."""
    definitions = {}
    for name, raw in model_files(model_dir, profile).items():
        definition = parse_definition(raw.decode("utf-8"))
        package = pathlib.PurePosixPath(name).parent.relative_to("model").as_posix()
        definition.update(package=package, file=name, sha256=hashlib.sha256(raw).hexdigest())
        key = package + "/" + definition["id"]
        if key in definitions:
            raise ModelError("duplicate Definition " + key)
        definitions[key] = definition
    covered = [key for _, _, keys in present_sections(definitions) for key in keys]
    if sorted(covered) != sorted(definitions) or len(set(covered)) != len(covered):
        raise ModelError("sections do not cover exactly the model's Definitions")
    return definitions


def resolve(definition, target):
    alias, _, ident = target.rpartition(".")
    if not alias:
        return definition["package"] + "/" + target
    if alias not in definition["imports"]:
        raise ModelError("unknown link alias " + alias)
    return definition["imports"][alias] + "/" + ident


def render_body(definition, key):
    body = definition["body"]
    for owner, old, new, count in SUBSTITUTIONS:
        if owner == key:
            if body.count(old) != count:
                raise ModelError(f"substitution {old!r} expected {count} times in {key}")
            body = body.replace(old, new)

    def link(match):
        target = resolve(definition, match.group(1))
        if target not in LINKS:
            raise ModelError("unlinkable target " + target)
        text, section = LINKS[target]
        if match.group(2):
            text = match.group(2)
        before = body[:match.start()]
        if text.startswith("the ") and re.search(r"\b[Tt]he $", before):
            text = text[4:]
        # Capitalize at the start of a sentence or paragraph, not after a line break.
        if not before.strip() or re.search(r"([.!?]|\n\n)\s*$", before):
            text = text[0].upper() + text[1:]
        return f"[{text}](#{slug(section)})"

    body = re.sub(r"\[\[([^\]|]+)(?:\|([^\]]+))?\]\]", link, body)
    if key == "library/callable-interface":
        body = re.sub(r"(?m)^## ", "### ", body)
    for word in FORBIDDEN:
        if re.search(r"\b" + re.escape(word) + r"\b" if word.isalpha() else re.escape(word), body):
            raise ModelError(f"rendered body of {key} still contains {word!r}")
    return body


def property_meaning(definition):
    spec = definition["spec"]
    if spec != {"type": "integer", "minCount": "1", "maxCount": "1"}:
        raise ModelError("unsupported Property spec")
    return "Exactly one integer value is required."


def render(model_dir, profile):
    definitions = load_model(model_dir, profile)
    policy = definitions["lending/loan-policy"]["spec"]
    out = ["# Small library requirements", "", INTRO, ""]
    rows = [key for key, _ in POLICY_ROWS]
    for level, title, keys in present_sections(definitions):
        out += ["#" * level + " " + title, ""]
        for key in keys:
            if key in rows:
                continue
            out += [render_body(definitions[key], key), ""]
            if key == "lending/LendingPolicy":
                out += ["| Policy value | Value | Meaning |", "| --- | --- | --- |"]
                for row_key, label in POLICY_ROWS:
                    if row_key not in definitions:
                        continue
                    prop = definitions[row_key]
                    meaning = render_body(prop, row_key) + " " + property_meaning(prop)
                    out.append(f"| {label} (`{prop['id']}`) | {policy[prop['id']]} | {meaning} |")
                out.append("")
    return "\n".join(out).rstrip("\n") + "\n", definitions


def normalize(text):
    return " ".join(text.split())


def check(model_dir, profile, doc_path):
    rendered, definitions = render(model_dir, profile)
    actual = pathlib.Path(doc_path).read_bytes()
    doc = normalize(actual.decode("utf-8"))
    bindings = []
    for _, title, keys in present_sections(definitions):
        for key in keys:
            body = render_body(definitions[key], key)
            bindings.append({"definition": key, "file": definitions[key]["file"],
                             "sha256": definitions[key]["sha256"], "section": title,
                             "substitutions": [old for owner, old, _, _ in SUBSTITUTIONS if owner == key],
                             "found": normalize(body) in doc})
    result = {"profile": profile, "doc_sha256": hashlib.sha256(actual).hexdigest(),
              "identical_to_render": actual == rendered.encode("utf-8"),
              "all_found": all(b["found"] for b in bindings), "bindings": bindings}
    return result


def materialize(model_dir, profile, out):
    """Write the profile's complete model (manifest, lock, Definitions) to a new directory."""
    out = pathlib.Path(out)
    out.mkdir(parents=True, exist_ok=False)
    for name in ("model.yaml", "model-lock.yaml"):
        (out / name).write_bytes((pathlib.Path(model_dir) / name).read_bytes())
    for name, raw in model_files(model_dir, profile).items():
        (out / name).parent.mkdir(parents=True, exist_ok=True)
        (out / name).write_bytes(raw)
    return sorted(p.relative_to(out).as_posix() for p in out.rglob("*") if p.is_file())


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("command", choices=["render", "check", "materialize"])
    p.add_argument("--profile", choices=PROFILES, required=True)
    p.add_argument("--model", default=str(DEFAULT_MODEL))
    p.add_argument("--out")
    p.add_argument("--doc")
    a = p.parse_args(argv)
    if a.command == "materialize":
        if not a.out:
            p.error("materialize needs --out")
        print(json.dumps(materialize(a.model, a.profile, a.out), indent=2))
        return 0
    if a.command == "render":
        text, _ = render(a.model, a.profile)
        if a.out:
            pathlib.Path(a.out).write_bytes(text.encode("utf-8"))
        else:
            sys.stdout.buffer.write(text.encode("utf-8"))
        return 0
    if not a.doc:
        p.error("check needs --doc")
    result = check(a.model, a.profile, a.doc)
    print(json.dumps(result, indent=2))
    return 0 if result["identical_to_render"] and result["all_found"] else 1


if __name__ == "__main__":
    sys.exit(main())
