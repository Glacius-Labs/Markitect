"""The plan.md packet of the ordinary-Agent comparison arm; see README.md, "Plan arm".

OPERATOR MATERIAL.

    python -B plan_packet.py render REPO                         # print plan.md for a Markitect packet at HEAD
    python -B plan_packet.py build SCENARIO TARGET               # the plan arm's workspace for a scenario
    python -B plan_packet.py verify REPO --markitect EXE         # plan.md against the product's checked model

plan.md is assembled deterministically from the committed model of the
scenario's Markitect packet, not written by an Agent and not a solution plan.
Every local Definition appears in full: its identity, kind, title, part of
the project, source file and digest, its complete body, its resolved links
and its structured values with resolved references. The kinds the
Definitions use are quoted from their source. Markitect's execution settings
and artifact registry configure Markitect's own process and are left out;
plan.md states this accounting explicitly. Nothing is summarized, and no
Manager organization is prescribed.

The plan arm's workspace has the same starting bytes as the Markitect arm
except that plan.md replaces .markitect/: realization/ when the case starts
from one, and the same commit sequence, so that a model change on an accepted
stand is also a requirements change on that stand.

`verify` builds the expected content of every local Definition from the
product's own checked model (`markitect context` of every Package), a code
path independent of this renderer, and requires each in full in plan.md.
"""
import argparse
import hashlib
import io
import json
import re
import shutil
import subprocess
import sys
import tempfile
import zipfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

from materialize import GIT, IDENTITY, LEAK, commit, git  # noqa: E402

PROMPT = """Read `plan.md` and implement it accordingly.

Deliver the required implementation, automated tests and usage documentation
in `realization/`. Use the ordinary implementation, build and test tools made
available in this workspace. Work as one coding Agent. plan.md is
authoritative: follow its requirements and use the implementation choices they
leave open. If it leaves a material decision open that you would otherwise
have to invent, or if two of its requirements contradict each other, do not
decide it and change nothing: end your reply with one line per such problem
that starts with `BLOCKER:` and names the requirements involved. Nobody
answers questions during this task. Report the delivered files, checks
actually performed and any unresolved requirement.
"""
PLAN_ROOTS = ("plan.md", "realization")
LINK = re.compile(r"\[\[([^\]|]+)(?:\|[^\]]*)?\]\]")
META_KINDS = {"markitect/core/Kind", "markitect/core/Property"}


class PlanError(Exception):
    pass


# -- the restricted YAML of Definition files --------------------------------------------------

def top_level(text):
    """{key: raw lines} of a Definition file's top-level mapping."""
    blocks, key = {}, None
    for line in text.splitlines():
        match = re.match(r"^([A-Za-z]+):(.*)$", line)
        if match:
            key = match.group(1)
            blocks[key] = [match.group(2)]
        elif key is not None:
            blocks[key].append(line)
    return blocks


def scalar(text):
    text = text.strip()
    if text.startswith('"'):
        return json.loads(text)
    if text.startswith("'"):
        return text[1:-1].replace("''", "'")
    return text


def block_scalar(lines):
    """The value of `key: |-`, `key: >-` (and their keep/clip forms) or an inline scalar."""
    head = lines[0].strip()
    if not head.startswith(("|", ">")):
        return scalar(" ".join([head] + [line.strip() for line in lines[1:] if line.strip()]))
    body = lines[1:]
    while body and not body[-1].strip():
        body.pop()
    indent = min((len(line) - len(line.lstrip(" ")) for line in body if line.strip()), default=0)
    content = [line[indent:] if line.strip() else "" for line in body]
    if head.startswith("|"):
        return "\n".join(content)
    paragraphs, current = [], []
    for line in content:
        if line:
            current.append(line)
        else:
            paragraphs.append(" ".join(current))
            current = []
    paragraphs.append(" ".join(current))
    return "\n".join(paragraphs)


def imports(lines):
    """{alias: package} of the file-local imports."""
    found, package = {}, None
    for line in lines[1:]:
        match = re.match(r"^\s*-?\s*(package|as):\s*(.+?)\s*$", line)
        if not match:
            continue
        if match.group(1) == "package":
            package = scalar(match.group(2))
        else:
            found[scalar(match.group(2))] = package
    return found


def resolve(reference, package, aliases):
    """The package/id of a written reference: own id, Alias.id or package/id."""
    reference = reference.strip()
    if "." in reference and "/" not in reference.split(".", 1)[0]:
        alias, local = reference.split(".", 1)
        if alias not in aliases:
            raise PlanError("unknown import alias in %r" % reference)
        return "%s/%s" % (aliases[alias], local)
    return reference if "/" in reference else "%s/%s" % (package, reference)


def definition(path, root):
    """One Definition file as {identity, package, kind, title, body, links, spec, source, digest}."""
    raw = path.read_bytes()
    text = raw.decode("utf-8")
    blocks = top_level(text)
    package = path.parent.relative_to(root).as_posix()
    aliases = imports(blocks.get("imports", [""]))
    local = scalar(blocks["id"][0])
    kind = resolve(scalar(blocks["kind"][0]), package, aliases)
    spec_lines = blocks.get("spec", [" {}"])
    if spec_lines[0].strip() in ("{}", ""):
        spec = [line for line in spec_lines[1:] if line.strip()]
    else:
        spec = ["%s" % spec_lines[0].strip()]
    indent = min((len(line) - len(line.lstrip(" ")) for line in spec if line.strip()), default=0)
    spec = [line[indent:] for line in spec]
    spec_text = "\n".join(spec)
    for alias, target in aliases.items():  # written Alias.id references become package/id
        spec_text = re.sub(r"(?<![\w/.-])%s\.([A-Za-z][\w-]*)" % re.escape(alias), lambda m: "%s/%s" % (target, m.group(1)),
                           spec_text)
    if kind in META_KINDS:  # contract references (properties, target) are written as local ids
        resolved, key = [], None
        for line in spec_text.split("\n"):
            top = re.match(r"^([A-Za-z]+):\s*(.*)$", line)
            key = top.group(1) if top else key
            item = re.match(r"^(\s*-\s+)([A-Za-z][\w-]*)\s*$", line)
            if top and key == "target" and re.fullmatch(r"[A-Za-z][\w-]*", top.group(2)):
                line = "target: " + resolve(top.group(2), package, aliases)
            elif item and key == "properties":
                line = item.group(1) + resolve(item.group(2), package, aliases)
            resolved.append(line)
        spec_text = "\n".join(resolved)
    body = block_scalar(blocks["body"]) if "body" in blocks else ""
    links = []
    for match in LINK.finditer(body):
        target = resolve(match.group(1), package, aliases)
        if (match.group(0), target) not in links:
            links.append((match.group(0), target))
    return {"identity": "%s/%s" % (package, local), "package": package, "kind": kind,
            "title": scalar(blocks["title"][0]) if "title" in blocks else "", "body": body, "links": links,
            "spec": spec_text, "digest": hashlib.sha256(raw).hexdigest(), "aliases": aliases}


def model_definitions(markitect_dir):
    """Local Definitions and the dependency Definitions of a packet's .markitect directory, by identity."""
    root = Path(markitect_dir) / "model"
    local = {}
    for path in sorted(root.rglob("*.yaml")):
        item = definition(path, root)
        item["source"] = ".markitect/model/" + path.relative_to(root).as_posix()
        local[item["identity"]] = item
    dependencies = {}
    for manifest in sorted((Path(markitect_dir) / "dependencies").glob("*/model.yaml")):
        dep_root = manifest.parent / "model"
        for path in sorted(dep_root.rglob("*.yaml")):
            item = definition(path, dep_root)
            item["source"] = ".markitect/dependencies/%s/model/%s" % (manifest.parent.name,
                                                                      path.relative_to(dep_root).as_posix())
            dependencies[item["identity"]] = item
    return local, dependencies


# -- plan.md ---------------------------------------------------------------------------

def manifest(markitect_dir):
    text = (Path(markitect_dir) / "model.yaml").read_text(encoding="utf-8")
    blocks = top_level(text)
    return scalar(blocks["id"][0]), scalar(blocks["version"][0])


def render_text(markitect_dir):
    local, dependencies = model_definitions(markitect_dir)
    model_id, version = manifest(markitect_dir)
    kinds = sorted({item["kind"] for item in local.values()} - set(local))
    out = ["# Requirements of %s %s" % (model_id, version), "",
           "This document states the complete requirements of the project. It is generated from the project's "
           "committed requirement definitions without summarizing or omitting any of them. Each section reproduces "
           "one requirement in full: its identifier, kind, title, the part of the project it belongs to, its source "
           "file and digest, its complete text, the requirements it links to and its structured values. A link is "
           "written `[[target]]` or `[[target|label]]` in the text and resolved to an identifier below it. "
           "References in structured values are written as `package/id`.", "",
           "## Accounting", "",
           "- Requirements reproduced in full: %d." % len(local),
           "- Requirements given only by reference: none.",
           "- Further lookup material: none; this document is complete on its own.",
           "- Left out: the Markitect execution settings and artifact registry. They configure Markitect's own "
           "process and state no requirement of the project.", "",
           "## Kinds of requirement", "",
           "Each requirement has a kind. These are their meanings, quoted from their source:", ""]
    for kind in kinds:
        item = dependencies.get(kind)
        if item is None:
            raise PlanError("the kind %s is neither local nor a dependency" % kind)
        out += ["- **%s**: %s" % (kind, " ".join(item["body"].split()))]
    out += ["", "## Requirements", ""]
    for identity in sorted(local):
        item = local[identity]
        title = " — %s" % item["title"] if item["title"] else ""
        out += ["### %s%s" % (identity, title), "",
                "- Kind: %s" % item["kind"],
                "- Part of the project: %s" % item["package"],
                "- Source: `%s` (sha256 %s)" % (item["source"], item["digest"]), ""]
        if item["body"]:
            out += [item["body"], ""]
        if item["links"]:
            out += ["Links: " + "; ".join("`%s` = %s" % (written, target) for written, target in item["links"]), ""]
        if item["spec"].strip():
            out += ["Structured values:", "", "```yaml", item["spec"], "```", ""]
    text = "\n".join(line.rstrip() for line in out).rstrip() + "\n"
    return text


def render(repo):
    """plan.md of a Markitect packet at its working tree, with its digest."""
    text = render_text(Path(repo) / ".markitect")
    return {"text": text, "sha256": hashlib.sha256(text.encode("utf-8")).hexdigest()}


def export_commit(repo, revision, folder):
    archive = subprocess.run([*GIT, "-C", str(repo), "archive", "--format=zip", revision], capture_output=True,
                             check=True).stdout
    with zipfile.ZipFile(io.BytesIO(archive)) as packed:
        packed.extractall(folder)
    return Path(folder)


def build_from_packet(packet, target):
    """The plan arm's workspace mirroring every commit of the Markitect PACKET."""
    target = Path(target).resolve()
    if target.exists():
        raise PlanError("%s already exists" % target)
    revisions = git(packet, "rev-list", "--reverse", "HEAD").split()
    messages = [git(packet, "log", "-1", "--format=%s", revision).strip() for revision in revisions]
    target.mkdir(parents=True)
    git(target, "init", "-q")
    plans = []
    for revision, message in zip(revisions, messages):
        with tempfile.TemporaryDirectory(prefix="plan-commit-") as folder:
            snapshot = export_commit(packet, revision, folder)
            if (target / "realization").exists():
                shutil.rmtree(target / "realization")
            if (snapshot / "realization").is_dir():
                shutil.copytree(snapshot / "realization", target / "realization")
            text = render_text(snapshot / ".markitect")
        (target / "plan.md").write_bytes(text.encode("utf-8"))
        plans.append(hashlib.sha256(text.encode("utf-8")).hexdigest())
        commit(target, message.replace("model", "requirements"))
    audit(target)
    files = {}
    for path in sorted(target.rglob("*")):
        relative = path.relative_to(target).as_posix()
        if path.is_file() and not relative.startswith(".git/"):
            files[relative] = hashlib.sha256(path.read_bytes()).hexdigest()
    return {"target": str(target), "head": git(target, "rev-parse", "HEAD").strip(),
            "tree": git(target, "rev-parse", "HEAD^{tree}").strip(), "commits": len(revisions),
            "planSha256": plans[-1], "files": files,
            "promptSha256": hashlib.sha256(PROMPT.encode("utf-8")).hexdigest()}


def audit(target):
    problems = []
    for path in sorted(Path(target).rglob("*")):
        relative = path.relative_to(target).as_posix()
        if relative == ".git" or relative.startswith(".git/") or not path.is_file():
            continue
        if relative.split("/", 1)[0] not in PLAN_ROOTS:
            problems.append("unexpected path " + relative)
        for match in LEAK.finditer(relative + "\n" + path.read_bytes().decode("utf-8", errors="replace")):
            problems.append("%s names %r" % (relative, match.group(0)))
    if problems:
        raise PlanError("plan packet audit failed:\n  " + "\n  ".join(problems))


def build(scenario_id, target, markitect_packet=None):
    """Materialize the scenario's Markitect packet (or use the given one) and mirror it as the plan workspace."""
    import freeze  # noqa: E402  (the scenario packet with its frozen budget)
    with tempfile.TemporaryDirectory(prefix="plan-source-") as folder:
        packet = markitect_packet or freeze.scenario_packet(scenario_id, Path(folder) / "packet")["repo"]
        return build_from_packet(packet, target)


# -- verification against the product's checked model -----------------------------------------

def canonical_value(value):
    """A canonical JSON value of `markitect context` as (type, value) for the facts."""
    if isinstance(value, bool):
        return "boolean", value
    if isinstance(value, dict) and set(value) == {"integer"}:
        return "integer", int(value["integer"])
    if isinstance(value, dict) and set(value) == {"Coefficient", "Exponent"}:
        return "number", "%se%s" % (value["Coefficient"], value["Exponent"])
    if isinstance(value, dict) and set(value) == {"ID", "Package"}:
        return "reference", "%s/%s" % (value["Package"], value["ID"])
    if isinstance(value, str):
        return "string", value
    raise PlanError("unexpected canonical value %r" % (value,))


def flatten(spec, kind, identity):
    """Value facts of a canonical spec: scalars, ordered lists, object lists and unordered declarations."""
    facts = []
    for key, value in spec.items():
        if kind == "markitect/core/Property" and key == "name" and value == identity.rsplit("/", 1)[1]:
            continue  # the effective default field name, not written in the source
        if isinstance(value, list):
            unordered = kind in META_KINDS and key == "properties"
            for index, item in enumerate(value):
                if isinstance(item, dict) and not ({"integer"} == set(item) or {"ID", "Package"} == set(item)
                                                   or {"Coefficient", "Exponent"} == set(item)):
                    for field, inner in item.items():
                        kind_name, scalar_value = canonical_value(inner)
                        facts.append({"path": [key, index, field], "type": kind_name, "value": scalar_value})
                else:
                    kind_name, scalar_value = canonical_value(item)
                    facts.append({"path": [key, "*" if unordered else index], "type": kind_name, "value": scalar_value})
        elif isinstance(value, dict) and not ({"integer"} == set(value) or {"ID", "Package"} == set(value)
                                              or {"Coefficient", "Exponent"} == set(value)):
            for field, inner in value.items():
                kind_name, scalar_value = canonical_value(inner)
                facts.append({"path": [key, field], "type": kind_name, "value": scalar_value})
        else:
            kind_name, scalar_value = canonical_value(value)
            facts.append({"path": [key], "type": kind_name, "value": scalar_value})
    return facts


def canonical_facts(markitect, repo):
    """{identity: facts} of every local Definition, from the product's static selection of every Package."""
    packages = sorted({path.parent.relative_to(Path(repo) / ".markitect" / "model").as_posix()
                       for path in (Path(repo) / ".markitect" / "model").rglob("*.yaml")})
    facts = {}
    for package in packages:
        done = subprocess.run([markitect, "context", package, "--repo", str(repo), "--revision", "HEAD",
                               "--model", ".markitect/model.yaml",
                               "--dependency", ".markitect/dependencies/core/model.yaml",
                               "--dependency", ".markitect/dependencies/software/model.yaml"],
                              capture_output=True, text=True, encoding="utf-8")
        report = json.loads(done.stdout)
        for context in report.get("contexts") or []:
            for entry in context["definitions"]:
                if not any(reason.get("Kind") == "own" for reason in entry["reasons"]):
                    continue
                item = entry["definition"]
                identity = "%s/%s" % (item["identity"]["Package"], item["identity"]["ID"])
                kind = "%s/%s" % (item["kind"]["Package"], item["kind"]["ID"])
                body = item["body"] or ""
                fragments = [" ".join(part.split()) for part in re.split(r"\[\[[^\]]*\]\]", body)
                             if len(" ".join(part.split())) > 2]
                links = [{"target": "%s/%s" % (link["Target"]["Package"], link["Target"]["ID"]), "written": "\0"}
                         for link in item.get("bodyLinks") or []]
                facts[identity] = {"owner": item["identity"]["Package"], "text": fragments, "links": links,
                                   "values": flatten(item.get("spec") or {}, kind, identity)}
    if not facts:
        raise PlanError("markitect context reported no Definitions for %s" % repo)
    return facts


def verify(repo, markitect):
    import context_controls  # noqa: E402  (the independent section and fact matching)
    facts = canonical_facts(markitect, repo)
    text = render(repo)["text"]
    delivered = context_controls.full_delivery(text, facts, sorted(facts))
    incomplete = {identity: gaps for identity, gaps in delivered.items() if gaps}
    local, _ = model_definitions(Path(repo) / ".markitect")
    return {"passed": not incomplete and set(local) == set(facts), "definitions": len(facts),
            "incomplete": incomplete, "renderedOnly": sorted(set(local) - set(facts)),
            "checkedOnly": sorted(set(facts) - set(local))}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = parser.add_subparsers(dest="command", required=True)
    one = sub.add_parser("render")
    one.add_argument("repo")
    make = sub.add_parser("build")
    make.add_argument("scenario")
    make.add_argument("target")
    check = sub.add_parser("verify")
    check.add_argument("repo")
    check.add_argument("--markitect", required=True)
    args = parser.parse_args(argv)
    try:
        if args.command == "render":
            sys.stdout.reconfigure(encoding="utf-8", newline="\n")
            sys.stdout.write(render(args.repo)["text"])
            return 0
        result = build(args.scenario, args.target) if args.command == "build" else verify(args.repo, args.markitect)
    except PlanError as error:
        print("plan_packet: %s" % error, file=sys.stderr)
        return 1
    print(json.dumps(result, indent=2, ensure_ascii=False))
    return 0 if result.get("passed", True) else 1


if __name__ == "__main__":
    sys.exit(main())
