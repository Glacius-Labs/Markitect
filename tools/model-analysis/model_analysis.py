"""Analyze a Markitect canonical model like code: dependencies, references, impact and lint signals.

The input is the JSON report of `markitect check`. Its per-Manager contexts,
joined, give every local Definition with its kind, body, typed `spec`
references and resolved `bodyLinks`, plus the imported type contracts they
use. The report also carries the registry records and the authored execution
ownership. File-local imports are not in the report; with --repo the tool
reads them from each Definition's source file.

  python -I -B model_analysis.py graph REPORT [--json]
  python -I -B model_analysis.py refs REPORT PACKAGE/ID [--json]
  python -I -B model_analysis.py impact OLD_REPORT NEW_REPORT [--transitive] [--json]
  python -I -B model_analysis.py lint REPORT [--repo DIR] [--rules RULES.json] [--json]
  python -I -B model_analysis.py lock REPORT [--repo DIR]

An edge is a typed `spec` reference or a body link from a local Definition.
`kind` is a type use, not an edge. Lint findings are warnings (register
ENH-002), never failures: `lint` exits 0 whatever it finds. Exit code 2 means
unusable input.
"""
import argparse
import collections
import fnmatch
import json
import pathlib
import re
import sys

CORE = "markitect/core"
META_KINDS = {(CORE, "Kind"), (CORE, "Property")}
WORDS_WARNING, WORDS_STRONG = 150, 300
TERM_PACKAGES = 3
TERM_PATTERN = r"[A-Za-z][A-Za-z0-9]*"


class InputError(Exception):
    pass


def text(identity):
    return identity[0] + "/" + identity[1]


def parse_identity(value):
    package, _, local = value.rpartition("/")
    if not package or not local:
        raise InputError(f"not a PACKAGE/ID Definition address: {value!r}")
    return package, local


def _identity(value):
    return value["Package"], value["ID"]


def _is_reference(value):
    return isinstance(value, dict) and set(value) == {"Package", "ID"}


def _spec_references(value, at):
    """(target, JSON pointer) for every reference value; objects and lists are walked."""
    if _is_reference(value):
        yield _identity(value), at
    elif isinstance(value, dict) and set(value) != {"integer"}:
        for name in sorted(value):
            yield from _spec_references(value[name], at + "/" + name.replace("~", "~0").replace("/", "~1"))
    elif isinstance(value, list):
        for i, item in enumerate(value):
            yield from _spec_references(item, f"{at}/{i}")


def read_json(path):
    """A check report as redirected by any shell; PowerShell 5.1 writes UTF-16 with a BOM."""
    data = pathlib.Path(path).read_bytes()
    encoding = "utf-16" if data[:2] in (b"\xff\xfe", b"\xfe\xff") else "utf-8-sig"
    try:
        return json.loads(data.decode(encoding))
    except (UnicodeDecodeError, ValueError) as error:
        raise InputError(f"{path}: not a JSON check report: {error}") from None


class Model:
    """The joined view of one check report."""

    def __init__(self, report):
        if report.get("status") != "passed" or not report.get("contexts"):
            raise InputError("needs a passed `markitect check` report with contexts")
        self.definitions = {}
        self.local = set()
        for context in report["contexts"]:
            for entry in context.get("definitions") or []:
                definition = entry["definition"]
                identity = _identity(definition["identity"])
                self.definitions[identity] = definition
                if entry.get("canonicalModelDigest") == context.get("canonicalModelDigest"):
                    self.local.add(identity)
        structural = (report.get("stages") or {}).get("structural") or {}
        self.models = ((structural.get("model") or {}).get("models")) or []
        self.records = ((structural.get("registry") or {}).get("records")) or []
        ownership = (((structural.get("execution") or {}).get("declarations") or {}).get("Ownership") or {}).get("Value") or []
        self.area_owners = {(area.get("Owner") or {}).get("Value") for area in ownership}
        self.area_owners |= {record.get("Owner") for record in self.records}
        self.edges = []  # (source, target, via, at); via is "spec" or "link"
        for identity in sorted(self.local):
            definition = self.definitions[identity]
            for name in sorted(definition.get("spec") or {}):
                for target, at in _spec_references(definition["spec"][name], "/spec/" + name):
                    self.edges.append((identity, target, "spec", at))
            for i, link in enumerate(definition.get("bodyLinks") or []):
                self.edges.append((identity, _identity(link["Target"]), "link", f"/bodyLinks/{i}"))

    def kind(self, identity):
        return _identity(self.definitions[identity]["kind"])

    def is_content(self, identity):
        return self.kind(identity) not in META_KINDS

    def content(self):
        return sorted(i for i in self.local if self.is_content(i))

    def packages(self, identities):
        return sorted({i[0] for i in identities})

    def local_model_definitions(self):
        """The Definition count the report gives for the local model, or None."""
        for m in self.models:
            if m.get("local"):
                return m.get("definitions")
        return None


def strongly_connected(nodes, successors):
    """Tarjan's components, iterative; each component sorted, list sorted."""
    index, low, on_stack, stack, result, counter = {}, {}, set(), [], [], [0]
    for root in sorted(nodes):
        if root in index:
            continue
        work = [(root, iter(sorted(successors.get(root, ()))))]
        index[root] = low[root] = counter[0]
        counter[0] += 1
        stack.append(root)
        on_stack.add(root)
        while work:
            node, children = work[-1]
            for child in children:
                if child not in index:
                    index[child] = low[child] = counter[0]
                    counter[0] += 1
                    stack.append(child)
                    on_stack.add(child)
                    work.append((child, iter(sorted(successors.get(child, ())))))
                    break
                if child in on_stack:
                    low[node] = min(low[node], index[child])
            else:
                work.pop()
                if work:
                    low[work[-1][0]] = min(low[work[-1][0]], low[node])
                if low[node] == index[node]:
                    component = []
                    while True:
                        member = stack.pop()
                        on_stack.discard(member)
                        component.append(member)
                        if member == node:
                            break
                    result.append(sorted(component))
    return sorted(result)


def package_graph(model):
    """Package dependency graph over all edges, with cycles and coupling."""
    definition_edges = {(s, t) for s, t, _, _ in model.edges if s != t}
    cross = {(s, t) for s, t in definition_edges if s[0] != t[0]}
    successors = collections.defaultdict(set)
    for s, t in cross:
        successors[s[0]].add(t[0])
    local = set(model.packages(model.local))
    nodes = local | {t for targets in successors.values() for t in targets}
    afferent = collections.defaultdict(set)
    for source, targets in successors.items():
        for target in targets:
            afferent[target].add(source)
    coupling = []
    for package in sorted(nodes):
        ca, ce = len(afferent[package]), len(successors[package])
        coupling.append({"package": package, "local": package in local,
                         "afferent": ca, "efferent": ce, "instability": round(ce / (ca + ce), 3) if ca + ce else None})
    cycles = [c for c in strongly_connected(nodes, successors) if len(c) > 1]
    vias = collections.Counter(via for _, _, via, _ in model.edges)
    return {
        "localDefinitions": len(model.local),
        "contentDefinitions": len(model.content()),
        "packages": len(local),
        "contentPackages": len(model.packages(model.content())),
        "specReferences": vias["spec"],
        "links": vias["link"],
        "definitionEdges": len(definition_edges),
        "crossPackageDefinitionEdges": len(cross),
        "crossPackageShare": round(len(cross) / len(definition_edges), 3) if definition_edges else None,
        "packageEdges": sum(len(t) for t in successors.values()),
        "cycles": cycles,
        "coupling": coupling,
        "reportedLocalDefinitions": model.local_model_definitions(),
    }


def references(model, identity):
    """Incoming edges, instances of a Kind and registry artifacts that realize the Definition."""
    if identity not in model.definitions:
        raise InputError(f"unknown Definition {text(identity)}")
    incoming = [{"from": text(s), "via": via, "at": at} for s, t, via, at in model.edges if t == identity]
    instances = sorted(text(i) for i in model.local if model.kind(i) == identity)
    artifacts = sorted(({"path": r.get("Path"), "form": r.get("Form"), "owner": r.get("Owner")}
                        for r in model.records if text(identity) in (r.get("Realizes") or [])), key=lambda a: a["path"])
    return {"definition": text(identity), "local": identity in model.local, "incoming": incoming,
            "instances": instances, "artifacts": artifacts}


def _content_view(definition):
    """What a Definition means: provenance, coordinates and display text excluded."""
    return json.dumps({"kind": definition.get("kind"), "title": definition.get("title"), "body": definition.get("body"),
                       "spec": definition.get("spec") or {},
                       "links": [link["Target"] for link in definition.get("bodyLinks") or []]}, sort_keys=True)


def impact(old, new, transitive=False):
    """Changed, added and removed local Definitions and who references them."""
    changed = sorted(i for i in old.local & new.local if _content_view(old.definitions[i]) != _content_view(new.definitions[i]))
    added = sorted(new.local - old.local)
    removed = sorted(old.local - new.local)
    seeds = set(changed) | set(added) | set(removed)
    referrers = collections.defaultdict(set)
    for model in (old, new):
        for s, t, _, _ in model.edges:
            if s in new.local or s in old.local:
                referrers[t].add(s)
        for i in model.local:
            referrers[model.kind(i)].add(i)
    affected, frontier = set(), set(seeds)
    while frontier:
        reached = {s for t in frontier for s in referrers[t]} - seeds - affected
        affected |= reached
        frontier = reached if transitive else set()
    targets = {text(i) for i in seeds | affected}
    artifacts = sorted({(r.get("Path"), d) for model in (old, new) for r in model.records
                        for d in r.get("Realizes") or [] if d in targets})
    return {"changed": [text(i) for i in changed], "added": [text(i) for i in added], "removed": [text(i) for i in removed],
            "affected": [text(i) for i in sorted(affected)],
            "artifacts": [{"path": p, "realizes": d} for p, d in artifacts]}


# Imports are not in the check report. This reader understands the import
# forms the language contract allows: a block list of `package`/`as` mappings
# or a flow list `[]` / `[{package: x, as: Y}]`, with plain or quoted scalars.
_SCALAR = r"""("(?:[^"\\]|\\.)*"|'(?:[^']|'')*'|[^,\s{}\[\]#][^,{}\[\]#]*?)"""


def _unquote(value):
    value = value.strip()
    if value[:1] == '"':
        return json.loads(value)
    if value[:1] == "'":
        return value[1:-1].replace("''", "'")
    return value


def read_imports(source):
    """[(package, alias)] from a Definition file's text, or None when its `imports` form is not understood."""
    lines = source.splitlines()
    for n, line in enumerate(lines):
        match = re.match(r"imports:\s*(.*?)\s*(#.*)?$", line)
        if not match:
            continue
        rest = match.group(1)
        if rest:
            if not (rest.startswith("[") and rest.endswith("]")):
                return None
            result = []
            for entry in re.findall(r"\{([^{}]*)\}", rest):
                fields = [re.fullmatch(r"\s*(package|as)\s*:\s*" + _SCALAR + r"\s*", f) for f in entry.split(",")]
                if not all(fields):
                    return None
                result.append(_import_entry(f.groups() for f in fields))
            return result
        result, current = [], None
        for item in lines[n + 1:]:
            if not item.strip() or item.lstrip().startswith("#"):
                continue
            if not item[:1].isspace() and not item.startswith("-"):
                break
            field = re.match(r"\s*(-\s+)?(package|as)\s*:\s*" + _SCALAR + r"\s*(#.*)?$", item)
            if not field:
                return None
            if field.group(1):
                current = []
                result.append(current)
            if current is None:
                return None
            current.append((field.group(2), field.group(3)))
        return [_import_entry(e) for e in result]
    return []


def _import_entry(fields):
    entry = {name: _unquote(value) for name, value in fields}
    return entry.get("package"), entry.get("as")


def definition_imports(model, repo):
    """{local Definition: [(package, alias)]} read from source files under repo; unread files are listed."""
    imports, unread = {}, []
    for identity in sorted(model.local):
        path = pathlib.Path(repo) / (model.definitions[identity].get("source") or {}).get("Path", "")
        try:
            parsed = read_imports(path.read_text(encoding="utf-8"))
        except OSError:
            parsed = None
        if parsed is None:
            unread.append(text(identity))
        else:
            imports[identity] = parsed
    return imports, unread


def read_rules(path):
    """Dependency-direction rules: {"layers": [[pattern, ...], ...], "forbid": [{"from", "to", "reason"}]}.

    Layers run from top to bottom: a Package may depend on its own or a lower
    layer only. Patterns are fnmatch globs over Package paths.
    """
    rules = read_json(path)
    if not isinstance(rules, dict) or set(rules) - {"layers", "forbid"}:
        raise InputError(f"{path}: rules need only `layers` and/or `forbid`")
    return rules


def _layer(rules, package):
    for depth, patterns in enumerate(rules.get("layers") or []):
        if any(fnmatch.fnmatchcase(package, p) for p in patterns):
            return depth
    return None


def lint(model, repo=None, rules=None, term_pattern=TERM_PATTERN):
    """Warnings per register ENH-002. Each has a signal, a severity and a subject."""
    findings = []

    def warn(signal, subject, message, severity="warning"):
        findings.append({"signal": signal, "severity": severity, "subject": subject, "message": message})

    content = model.content()
    for identity in content:
        words = len((model.definitions[identity].get("body") or "").split())
        if words > WORDS_WARNING:
            warn("size", text(identity), f"body has {words} words", "strong" if words > WORDS_STRONG else "warning")
    connected = {s for s, t, _, _ in model.edges if s != t} | {t for s, t, _, _ in model.edges if s != t}
    for identity in content:
        if identity not in connected:
            warn("isolated", text(identity), "no typed reference or body link in or out")
    for package in model.packages(content):
        if package not in model.area_owners:
            warn("no-area", package, "holds content Definitions but owns no registered artifact or ownership area")

    used = collections.defaultdict(set)  # Kind -> effective field names set by local Definitions
    for identity in model.local:
        used[model.kind(identity)].update((model.definitions[identity].get("spec") or {}).keys())
    for kind in sorted(used):
        if kind in META_KINDS or kind not in model.definitions or model.kind(kind) != (CORE, "Kind"):
            continue
        for prop, _ in _spec_references((model.definitions[kind].get("spec") or {}).get("properties") or [], ""):
            if prop not in model.definitions:
                continue
            field = (model.definitions[prop].get("spec") or {}).get("name") or prop[1]
            if field not in used[kind]:
                warn("unset-field", text(kind), f"field `{field}` is set by no local Definition")

    ids = {i[1].lower() for i in model.definitions}
    term_packages = collections.defaultdict(set)
    for identity in model.local:
        for term in re.findall(r"`([^`\n]+)`", model.definitions[identity].get("body") or ""):
            if re.fullmatch(term_pattern, term) and term.lower() not in ids:
                term_packages[term].add(identity[0])
    for term in sorted(term_packages):
        if len(term_packages[term]) >= TERM_PACKAGES:
            warn("undefined-term", term, f"used in {len(term_packages[term])} Packages without a Definition of its own")

    package_edges = sorted({(s[0], t[0]) for s, t, _, _ in model.edges if s[0] != t[0]})
    if rules:
        for source, target in package_edges:
            above, below = _layer(rules, source), _layer(rules, target)
            if above is not None and below is not None and below < above:
                warn("direction", f"{source} -> {target}", f"layer {above} depends on higher layer {below}")
            for rule in rules.get("forbid") or []:
                if fnmatch.fnmatchcase(source, rule["from"]) and fnmatch.fnmatchcase(target, rule["to"]):
                    warn("direction", f"{source} -> {target}", "forbidden: " + rule.get("reason", f"{rule['from']} -> {rule['to']}"))

    notes = []
    if repo is None:
        notes.append("imports not read (no --repo): unused-import and import-cycle signals skipped")
    else:
        imports, unread = definition_imports(model, repo)
        if unread:
            notes.append(f"imports not understood in {len(unread)} files: " + ", ".join(unread[:5]))
        targets = collections.defaultdict(set)
        for s, t, _, _ in model.edges:
            targets[s].add(t[0])
        import_graph = collections.defaultdict(set)
        for identity, entries in sorted(imports.items()):
            for package, alias in entries:
                if package != identity[0]:
                    import_graph[identity[0]].add(package)
                if package not in targets[identity] and package != model.kind(identity)[0]:
                    warn("unused-import", text(identity), f"imports `{package}`" + (f" as {alias}" if alias else "") + " but uses nothing from it")
        nodes = set(import_graph) | {p for ps in import_graph.values() for p in ps}
        for cycle in strongly_connected(nodes, import_graph):
            if len(cycle) > 1:
                warn("import-cycle", " <-> ".join(cycle), f"{len(cycle)} Packages import each other in a cycle")
    order = {"strong": 0, "warning": 1}
    findings.sort(key=lambda f: (f["signal"], order[f["severity"]], f["subject"], f["message"]))
    return {"findings": findings, "counts": dict(collections.Counter(f["signal"] for f in findings)), "notes": notes}


def _manifest(model_entry, repo):
    """The manifest path a model entry names, resolved against repo or its Origin; None when absent."""
    source = model_entry.get("source") or {}
    candidates = [pathlib.Path(repo or ".") / source.get("Path", "")]
    if source.get("Origin"):
        candidates.append(pathlib.Path(source["Origin"]) / source.get("Path", ""))
    return next((c for c in candidates if c.is_file()), None)


def lock_lines(report, repo=None):
    """model-lock.yaml text for every non-local model of a report, and notes on what could not be bound."""
    structural = (report.get("stages") or {}).get("structural") or {}
    models = sorted((m for m in ((structural.get("model") or {}).get("models") or []) if not m.get("local")), key=lambda m: m["id"])
    lines, notes = ["formatVersion: 1", "models:" + ("" if models else " {}")], []
    for m in models:
        version = None
        manifest = _manifest(m, repo)
        if manifest:
            found = re.search(r"^version:\s*(.+?)\s*(#.*)?$", manifest.read_text(encoding="utf-8"), re.M)
            version = _unquote(found.group(1)) if found else None
        if version is None:
            notes.append(f"{m['id']}: version not found; give --repo so the manifest can be read")
        digest = m.get("canonicalModelDigest") or ""
        if digest and not digest.startswith("sha256:"):
            digest = "sha256:" + digest
        lines += [f"  {m['id']}:", f"    version: {version or 'UNKNOWN'}", f"    digest: {digest or 'UNKNOWN'}"]
    return "\n".join(lines) + "\n", notes


def format_graph(g):
    out = [f"Local Definitions: {g['localDefinitions']} ({g['contentDefinitions']} content) in {g['packages']} Packages ({g['contentPackages']} with content)",
           f"Edges: {g['specReferences']} spec references, {g['links']} body links",
           f"Definition edges: {g['definitionEdges']}, cross-Package {g['crossPackageDefinitionEdges']}" +
           (f" ({g['crossPackageShare']:.0%})" if g["crossPackageShare"] is not None else ""),
           f"Package edges: {g['packageEdges']}",
           f"Cycles: {len(g['cycles'])}"]
    out += [f"  {len(c)} Packages: " + ", ".join(c) for c in g["cycles"]]
    if g["reportedLocalDefinitions"] not in (None, g["localDefinitions"]):
        out.append(f"Note: the report's local model has {g['reportedLocalDefinitions']} Definitions; contexts gave {g['localDefinitions']}")
    out.append("Coupling (Ca afferent, Ce efferent, I instability):")
    for c in g["coupling"]:
        instability = "-" if c["instability"] is None else f"{c['instability']:.2f}"
        out.append(f"  {c['package']}{'' if c['local'] else ' (imported)'}: Ca {c['afferent']}, Ce {c['efferent']}, I {instability}")
    return "\n".join(out)


def format_refs(r):
    out = [f"{r['definition']}{'' if r['local'] else ' (imported)'}", f"Incoming: {len(r['incoming'])}"]
    out += [f"  {i['from']} ({i['via']} {i['at']})" for i in r["incoming"]]
    if r["instances"]:
        out += [f"Instances of this Kind: {len(r['instances'])}"] + [f"  {i}" for i in r["instances"]]
    out += [f"Realizing artifacts: {len(r['artifacts'])}"] + [f"  {a['path']} ({a['form']}, owner {a['owner']})" for a in r["artifacts"]]
    return "\n".join(out)


def format_impact(r):
    out = []
    for key in ("changed", "added", "removed", "affected"):
        out += [f"{key.capitalize()}: {len(r[key])}"] + [f"  {i}" for i in r[key]]
    out += [f"Artifacts: {len(r['artifacts'])}"] + [f"  {a['path']} (realizes {a['realizes']})" for a in r["artifacts"]]
    return "\n".join(out)


def format_lint(r):
    out = [f"{len(r['findings'])} warnings" + (" (" + ", ".join(f"{k} {v}" for k, v in sorted(r["counts"].items())) + ")" if r["counts"] else "")]
    out += [f"  [{f['signal']}{'!' if f['severity'] == 'strong' else ''}] {f['subject']}: {f['message']}" for f in r["findings"]]
    out += ["Note: " + n for n in r["notes"]]
    return "\n".join(out)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    commands = parser.add_subparsers(dest="command", required=True)
    for name in ("graph", "refs", "lint", "lock"):
        sub = commands.add_parser(name)
        sub.add_argument("report")
        if name == "refs":
            sub.add_argument("definition", help="PACKAGE/ID")
        if name in ("lint", "lock"):
            sub.add_argument("--repo", help="directory the report's source paths are relative to")
        if name == "lint":
            sub.add_argument("--rules", help="JSON dependency-direction rules")
            sub.add_argument("--term-pattern", default=TERM_PATTERN, help="regular expression a backticked term must match")
        if name != "lock":
            sub.add_argument("--json", action="store_true")
    sub = commands.add_parser("impact")
    sub.add_argument("old")
    sub.add_argument("new")
    sub.add_argument("--transitive", action="store_true", help="follow referrers of referrers")
    sub.add_argument("--json", action="store_true")
    args = parser.parse_args(argv)
    try:
        if args.command == "lock":
            output, notes = lock_lines(read_json(args.report), args.repo)
            sys.stdout.write(output)
            for note in notes:
                print("note:", note, file=sys.stderr)
            return 0
        if args.command == "impact":
            result, formatter = impact(Model(read_json(args.old)), Model(read_json(args.new)), args.transitive), format_impact
        else:
            model = Model(read_json(args.report))
            if args.command == "graph":
                result, formatter = package_graph(model), format_graph
            elif args.command == "refs":
                result, formatter = references(model, parse_identity(args.definition)), format_refs
            else:
                rules = read_rules(args.rules) if args.rules else None
                result, formatter = lint(model, args.repo, rules, args.term_pattern), format_lint
    except (InputError, OSError, KeyError, TypeError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 2
    print(json.dumps(result, indent=2, ensure_ascii=False) if args.json else formatter(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
