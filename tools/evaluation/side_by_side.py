"""One side-by-side page per test, with no overall score (concept 3.0 §13).

  python -I -B side_by_side.py --store DIR --measures DIR --blind DIR --reference NAME --title TEXT [--notes FILE] [--appendix FILE] --out FILE

It lays out what the other tools found, per neutral label from `blind_read.py`: the figures of `measure.py`, one JSON file
per kept name in --measures; the reader's answers in the blind reads; the hidden cases, time and cost from the kept launch,
after the reads, since a cost can tell a reader which result is which; and the key, so that a reader can stop before it.
Nothing above the key names a model or an arm. Cost is in multiples of --reference, never a dollar amount.

--notes puts a person's or a session's words at the top of the page, and --appendix after the key. No figure here adds up
to a verdict.
"""
import argparse
import json
import pathlib
import sys

QUESTIONS = (("architecture", "Architecture"), ("naming", "Naming"), ("cleanliness", "Cleanliness"), ("tests", "Tests"))


def percent(share):
    return f"{share * 100:.1f} %"


def table(labels, rows):
    head = "| | " + " | ".join(labels) + " |\n|---|" + "---|" * len(labels) + "\n"
    return head + "".join(f"| {name} | " + " | ".join(str(v) for v in values) + " |\n" for name, values in rows)


def named(items, key, count="count"):
    """The first of a list of {key, count}, as `name (count)`."""
    return f"`{items[0][key]}` ({items[0][count]})" if items else "none"


def linter(value, render):
    return "not run" if value is None else render(value)


def launch_rows(labels, launches, reference):
    base = launches[reference]["relativeCost"]
    get = lambda f: [f(launches[label]) for label in labels]
    cases = lambda r: f"{r['hiddenCases']['passed']} of {r['hiddenCases']['planned']}" if r.get("hiddenCases") else "not judged"
    weekly = lambda r: f"{r['weekly']['before']:g} to {r['weekly']['after']:g}" if r.get("weekly") else "not read"
    return [
        ("Hidden cases", get(cases)),
        ("Wall time", get(lambda r: f"{r['wallSeconds'] / 60:.0f} min")),
        ("Cost, multiples of the reference", get(lambda r: f"{r['relativeCost'] / base:.2f} ×" if r.get("relativeCost") is not None else "not recorded")),
        ("Weekly reading, before and after", get(weekly)),
    ]


def measured_rows(labels, measures):
    get = lambda f: [f(measures[label]) for label in labels]
    return [
        ("Builds", get(lambda m: "yes" if m["linters"]["builds"] else "no")),
        ("Program lines", get(lambda m: m["lines"]["program"])),
        ("Test lines", get(lambda m: m["lines"]["tests"])),
        ("Share of lines in tests", get(lambda m: percent(m["lines"]["testShare"]))),
        ("Test functions", get(lambda m: m["testFunctions"])),
        ("Go files", get(lambda m: m["files"]["count"])),
        ("Files over 500 lines", get(lambda m: m["files"]["over"])),
        ("Largest file, lines", get(lambda m: m["files"]["max"])),
        ("Functions", get(lambda m: m["functions"]["count"])),
        ("Functions over 80 lines", get(lambda m: m["functions"]["over"])),
        ("Longest function, lines", get(lambda m: m["functions"]["max"])),
        ("Packages", get(lambda m: m["structure"]["packages"])),
        ("Largest package, lines", get(lambda m: named(m["structure"]["largestPackages"], "package", "lines"))),
        ("Imports between packages", get(lambda m: m["structure"]["imports"])),
        ("Highest fan-out", get(lambda m: named(m["structure"]["fanOut"], "package"))),
        ("Longest import chain", get(lambda m: m["structure"]["longestChain"])),
        ("Import cycles", get(lambda m: len(m["structure"]["cycles"]))),
        ("Repeated lines, exact", get(lambda m: percent(m["duplication"]["exact"]))),
        ("Repeated lines, renamed", get(lambda m: percent(m["duplication"]["renamed"]))),
        ("Repeated test lines, exact", get(lambda m: percent(m["duplication"]["testsExact"]))),
        ("Dead functions (`deadcode`)", get(lambda m: linter(m["linters"]["deadcode"], lambda d: d["dead"]))),
        ("Functions only tests reach", get(lambda m: linter(m["linters"]["deadcode"], lambda d: d["onlyTests"]))),
        ("`staticcheck` findings", get(lambda m: linter(m["linters"]["staticcheck"], lambda d: d["findings"]))),
        ("`go vet` findings", get(lambda m: m["linters"]["vet"])),
        ("Files `gofmt` would change", get(lambda m: m["linters"]["gofmt"])),
    ]


def spec_rows(labels, measures):
    get = lambda f: [f(measures[label]["againstSpec"]) for label in labels]
    return [
        ("Modules found", get(lambda a: f"{a['found']} of {a['modules']}")),
        ("Modules not at their path", get(lambda a: len(a["notAtTheirPath"]))),
        ("Packages the spec does not name", get(lambda a: len(a["extraPackages"]))),
        ("Imports the spec forbids", get(lambda a: len(a["forbiddenImports"]))),
        ("The same in tests", get(lambda a: a["forbiddenImportsInTests"])),
    ]


def details(labels, measures):
    out = []
    for label in labels:
        a = measures[label]["againstSpec"]
        parts = []
        if a["missing"]:
            parts.append("missing " + ", ".join(f"`{m}`" for m in a["missing"]))
        if a["extraPackages"]:
            parts.append("extra " + ", ".join(f"`{m}`" for m in a["extraPackages"][:12]))
        if a["forbiddenImports"]:
            parts.append("forbidden " + ", ".join(f"`{f['from']}` to `{f['to']}`" for f in a["forbiddenImports"][:12]))
        if parts:
            out.append(f"- **{label}:** " + "; ".join(parts) + ".")
    return "\n".join(out)


def read_of(blind, label):
    """The valid read of a label, or None."""
    for p in sorted(pathlib.Path(blind, "reads", label).glob("a*/read.json")):
        read = json.loads(p.read_text(encoding="utf-8"))
        if read.get("valid"):
            return read
    return None


def said(values):
    """A reader's sentences as plain text: code it quotes must not read as a link."""
    return " ".join(values or []).replace("[", "\\[")


def reads_section(labels, reads):
    rows = [(title, [reads[label]["answer"][name]["score"] if reads[label] else "no read" for label in labels]) for name, title in QUESTIONS]
    rows.append(("Would maintain it", [reads[label]["answer"]["takingOver"].get("wouldMaintain", "?") if reads[label] else "no read" for label in labels]))
    rows.append(("Files the reader says it opened", [reads[label]["answer"].get("filesOpened", "?") if reads[label] else "no read" for label in labels]))
    out = [table(labels, rows)]
    for label in labels:
        read = reads[label]
        out.append(f"\n### {label}\n")
        if not read:
            out.append("No valid read.\n")
            continue
        answer = read["answer"]
        for name, title in QUESTIONS:
            part = answer[name]
            out.append(f"**{title} ({part['score']} of 5).** Good: " + (said(part.get("good")) or "nothing named.") + " Weak: " + (said(part.get("weak")) or "nothing named.") + "\n")
        taking = answer["takingOver"]
        out.append(f"**Taking it over: {taking.get('wouldMaintain', '?')}.** {said([taking.get('why', '')])} First: " + said(taking.get("changeFirst")) + "\n")
    return "\n".join(out)


def page(title, labels, launches, measures, reads, key, reference, notes=None, appendix=None):
    out = [f"# {title}\n"]
    out.append("Each result has a neutral label until the key. There is no overall score: the figures point at what is worth looking at, and the conclusion is the owner's.\n")
    if notes:
        out.append(notes.rstrip() + "\n")
    out.append("## Measured\n")
    out.append("The same script for every result, from a clean checkout, with one Go toolchain. A repeated line sits inside a run of 6 identical lines, or of 12 that differ only in names and values.\n")
    out.append(table(labels, measured_rows(labels, measures)))
    out.append("## Against the spec\n")
    out.append("Do the modules match the spec's Packages, and do the imports follow its dependencies?\n")
    out.append(table(labels, spec_rows(labels, measures)))
    extra = details(labels, measures)
    if extra:
        out.append(extra + "\n")
    out.append("## Blind reads\n")
    out.append("One fresh session per result, which saw that result and the requirements and did not know of the others. A score says where to look.\n")
    out.append(reads_section(labels, reads))
    out.append("## Hidden cases, time and cost\n")
    out.append(table(labels, launch_rows(labels, launches, reference)))
    out.append("## The key\n")
    out.append("Read this last.\n")
    # Subagents and commits go here too: above the key they would tell the arms apart.
    rows = "".join(f"| {label} | `{key[label]['name']}` | {launches[label].get('note') or ''} | {', '.join(sorted(launches[label].get('modelsAnswered') or {}))} | "
                   f"{launches[label].get('subagents', 0)} | {launches[label].get('commits', '?')} |\n" for label in labels)
    out.append("| Label | Kept launch | What it is | Models that answered | Subagents | Commits |\n|---|---|---|---|---|---|\n" + rows)
    if appendix:
        out.append(appendix.rstrip() + "\n")
    return "\n".join(out)


def build(store, measures_dir, blind, reference, title, notes=None, appendix=None):
    key = json.loads(pathlib.Path(blind, "key.json").read_text(encoding="utf-8"))
    labels = sorted(key)
    launches, measures, reads = {}, {}, {}
    for label in labels:
        name = key[label]["name"]
        launches[label] = json.loads(pathlib.Path(store, name, "launch.json").read_text(encoding="utf-8"))
        measures[label] = json.loads(pathlib.Path(measures_dir, name + ".json").read_text(encoding="utf-8"))
        launches[label]["commits"] = measures[label].get("commits", "?")
        reads[label] = read_of(blind, label)
    found = [label for label in labels if key[label]["name"] == reference]
    if not found:
        raise ValueError(f"the reference {reference} is not among the results")
    return page(title, labels, launches, measures, reads, key, found[0], notes, appendix)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    for name in ("store", "measures", "blind", "reference", "title", "out"):
        parser.add_argument("--" + name, required=True)
    parser.add_argument("--notes")
    parser.add_argument("--appendix")
    a = parser.parse_args(argv)
    try:
        notes = pathlib.Path(a.notes).read_text(encoding="utf-8") if a.notes else None
        appendix = pathlib.Path(a.appendix).read_text(encoding="utf-8") if a.appendix else None
        text = build(a.store, a.measures, a.blind, a.reference, a.title, notes, appendix)
    except (OSError, ValueError, KeyError) as e:
        print(f"side_by_side: {type(e).__name__}: {e}", file=sys.stderr)
        return 2
    pathlib.Path(a.out).write_text(text, encoding="utf-8", newline="\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
