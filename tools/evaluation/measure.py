"""What can be measured about a kept result, the same way for every result (concept 3.0 §13).

  python -I -B measure.py --kept KEPT_DIR [--architecture PAGE] [--tools DIR] --out FILE

It reads a clean checkout of the kept repository and runs nothing the result wrote: no test and no program, only Go's
own build and analysis tools, without network. The figures point at what is worth looking at, and none of them is a
verdict.

- **Structure:** Go packages, lines per package, and the imports between them: how many, the highest fan-in and fan-out,
  the longest chain, and any cycle.
- **Sizes:** files and functions, with the largest. A function ends at the first closing brace in the first column,
  which is where gofmt puts it.
- **Tests:** the share of lines in test files, and how many test functions.
- **Duplication:** the share of a program's lines inside a repeated run of lines. Blank lines, comments and lone
  brackets do not count. `exact` looks for 6 lines in a row that are the same after trimming; `renamed` for 12 that are
  the same once every name, string and number is the same, which also finds a copy with other names.
- **Dead code:** `deadcode`, from the program's main and its tests. `onlyTests` are the functions only tests reach.
- **Linters:** `go vet`, `gofmt -l` and `staticcheck`.

`--architecture` names the spec's page with the module table (`| module | model | may depend on |`), and adds the part
measured against the spec: do the modules match the Packages, and do the imports follow its dependencies? A row of
`nothing` allows no module, and `cmd/...` is the program's entry, which no table lists.

Every result is measured with one Go toolchain, GO_TOOLCHAIN, which must be in the module cache, since nothing here
reaches the network. `deadcode` and `staticcheck` come from `--tools`, built with that toolchain at the versions in PINNED.
Without them those rows say not run.
"""
import argparse
import json
import os
import pathlib
import re
import statistics
import subprocess
import sys
import tempfile

# Isolated mode (-I) leaves the script's directory off the path.
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import kept  # noqa: E402

PINNED = {"staticcheck": "honnef.co/go/tools/cmd/staticcheck@v0.8.2", "deadcode": "golang.org/x/tools/cmd/deadcode@v0.52.0"}
GO_TOOLCHAIN = "go1.27.1"
EXACT_LINES = 6
RENAMED_LINES = 12
LONG_FILE = 500
LONG_FUNCTION = 80
TOP = 5
SKIP_DIRS = {".git", "vendor", "testdata", "node_modules"}
GO_WORDS = set("break case chan const continue default defer else fallthrough for func go goto if import interface map package range return select struct switch type var".split())
FUNCTION = re.compile(r"^func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)")
TEST_FUNCTION = re.compile(r"^func\s+(?:Test|Fuzz|Example)\w*\(")
STRING = re.compile(r'"(?:[^"\\]|\\.)*"|`[^`]*`|\'(?:[^\'\\]|\\.)*\'')
NUMBER = re.compile(r"\b\d[\w.]*\b")
NAME = re.compile(r"\b[A-Za-z_]\w*\b")
MODULE_TABLE = re.compile(r"^\|\s*Module\s*\|\s*Model\s*\|\s*May depend on\s*\|\s*$")
MODULE_ROW = re.compile(r"^\|\s*`([a-z0-9_-]+)`\s*\|[^|]*\|\s*(.*?)\s*\|\s*$")
UNREACHABLE = re.compile(r"^(.*?):\d+:\d+: unreachable func: (.+)$")
QUOTED = re.compile(r"`([a-z0-9_-]+)`")


class MeasureError(Exception):
    pass


def environment(cache):
    env = {k: v for k, v in os.environ.items() if not k.upper().startswith(("ANTHROPIC_", "CLAUDE_", "MARKITECT_", "GO"))}
    env.update(GOFLAGS="-mod=readonly", GOPROXY="off", GOTOOLCHAIN=GO_TOOLCHAIN, GOWORK="off", CGO_ENABLED="0", GOCACHE=str(cache))
    return env


def tool(argv, cwd, env, timeout=900):
    p = subprocess.run(argv, cwd=cwd, env=env, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=timeout)
    return p.returncode, p.stdout, p.stderr


def go_files(root):
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
        for name in filenames:
            if name.endswith(".go"):
                yield pathlib.Path(dirpath) / name


def module_root(checkout):
    roots = sorted((p.parent for p in pathlib.Path(checkout).rglob("go.mod") if ".git" not in p.parts), key=lambda p: (len(p.parts), str(p)))
    if not roots:
        raise MeasureError("the result has no go.mod")
    return roots[0]


def spread(values):
    values = sorted(values)
    if not values:
        return {"count": 0, "median": 0, "p90": 0, "max": 0}
    return {"count": len(values), "median": statistics.median(values), "p90": values[min(len(values) - 1, int(len(values) * 0.9))], "max": values[-1]}


def significant(line):
    """A line for the duplication count, or None."""
    text = line.strip()
    if not text or text.startswith("//") or not re.search(r"\w", text):
        return None
    return text


def renamed(text):
    text = NUMBER.sub("0", STRING.sub('""', text))
    return NAME.sub(lambda m: m.group(0) if m.group(0) in GO_WORDS else "x", text)


def repeated_share(files, size, shape=lambda s: s):
    """The share of lines inside a run of `size` lines that occurs more than once."""
    seen, total = {}, 0
    for name, lines in files.items():
        kept_lines = [shape(s) for s in lines]
        total += len(kept_lines)
        for i in range(len(kept_lines) - size + 1):
            seen.setdefault(tuple(kept_lines[i:i + size]), []).append((name, i))
    covered = set()
    for places in seen.values():
        if len(places) > 1:
            for name, i in places:
                covered.update((name, j) for j in range(i, i + size))
    return round(len(covered) / total, 4) if total else 0.0


def sizes(root):
    """Lines, files, functions, tests and duplication, from the text alone."""
    root = pathlib.Path(root)
    program, tests, program_lines, test_lines, functions = {}, {}, 0, 0, []
    test_functions, file_sizes = 0, {}
    for path in go_files(root):
        rel = path.relative_to(root).as_posix()
        lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
        is_test = rel.endswith("_test.go")
        file_sizes[rel] = len(lines)
        (tests if is_test else program)[rel] = [t for t in map(significant, lines) if t is not None]
        if is_test:
            test_lines += len(lines)
            test_functions += sum(1 for s in lines if TEST_FUNCTION.match(s))
        else:
            program_lines += len(lines)
            start = None
            for i, s in enumerate(lines):
                m = FUNCTION.match(s)
                if m:
                    start = (m.group(1), i)
                    if s.rstrip().endswith("}"):
                        functions.append((1, rel, m.group(1)))
                        start = None
                elif start and s.startswith("}"):
                    functions.append((i - start[1] + 1, rel, start[0]))
                    start = None
    all_lines = program_lines + test_lines
    largest_files = sorted(file_sizes.items(), key=lambda kv: (-kv[1], kv[0]))[:TOP]
    longest = sorted(functions, key=lambda f: (-f[0], f[1], f[2]))[:TOP]
    return {
        "lines": {"program": program_lines, "tests": test_lines, "testShare": round(test_lines / all_lines, 4) if all_lines else 0.0},
        "files": {**spread(list(file_sizes.values())), "over": sum(v > LONG_FILE for v in file_sizes.values()), "overLines": LONG_FILE,
                  "program": len(program), "tests": len(tests), "largest": [{"file": f, "lines": n} for f, n in largest_files]},
        "functions": {**spread([f[0] for f in functions]), "over": sum(f[0] > LONG_FUNCTION for f in functions), "overLines": LONG_FUNCTION,
                      "longest": [{"function": n, "file": f, "lines": size} for size, f, n in longest]},
        "testFunctions": test_functions,
        "duplication": {"exact": repeated_share(program, EXACT_LINES), "renamed": repeated_share(program, RENAMED_LINES, renamed),
                        "testsExact": repeated_share(tests, EXACT_LINES)},
    }


def go_list(root, env):
    code, out, err = tool(["go", "list", "-e", "-json", "./..."], root, env)
    if code != 0:
        raise MeasureError(f"go list: {err.strip()[:500]}")
    decoder, at, found = json.JSONDecoder(), 0, []
    while at < len(out):
        while at < len(out) and out[at].isspace():
            at += 1
        if at >= len(out):
            break
        value, at = decoder.raw_decode(out, at)
        found.append(value)
    return found


def cycles(edges):
    """Every group of packages that can reach each other (Tarjan)."""
    index, low, on, stack, groups = {}, {}, set(), [], []

    def visit(v):
        index[v] = low[v] = len(index)
        stack.append(v)
        on.add(v)
        for w in edges.get(v, ()):
            if w not in index:
                visit(w)
                low[v] = min(low[v], low[w])
            elif w in on:
                low[v] = min(low[v], index[w])
        if low[v] == index[v]:
            group = []
            while True:
                w = stack.pop()
                on.discard(w)
                group.append(w)
                if w == v:
                    break
            if len(group) > 1:
                groups.append(sorted(group))

    previous = sys.getrecursionlimit()
    sys.setrecursionlimit(max(previous, 10000))
    try:
        for v in sorted(edges):
            if v not in index:
                visit(v)
    finally:
        sys.setrecursionlimit(previous)
    return groups


def longest_chain(edges):
    memo = {}

    def depth(v, path):
        if v in path:
            return 0
        if v not in memo:
            memo[v] = 1 + max((depth(w, path | {v}) for w in edges.get(v, ())), default=0)
        return memo[v]

    return max((depth(v, frozenset()) for v in edges), default=0)


def structure(root, env):
    """The packages and the imports between them, with paths relative to the module's root."""
    packages = go_list(root, env)
    module = next((p["Module"]["Path"] for p in packages if p.get("Module")), "")
    relative = {}
    for p in packages:
        path = p["ImportPath"]
        relative[path] = "." if path == module else path[len(module) + 1:] if path.startswith(module + "/") else path
    edges, test_edges, lines, errors = {}, {}, {}, []
    for p in packages:
        rel = relative[p["ImportPath"]]
        if p.get("Error"):
            errors.append(f"{rel}: {p['Error'].get('Err', '')[:200]}")
        edges[rel] = sorted({relative[i] for i in p.get("Imports") or [] if i in relative})
        test_edges[rel] = sorted({relative[i] for i in (p.get("TestImports") or []) + (p.get("XTestImports") or []) if i in relative and relative[i] != rel})
        count = 0
        for name in p.get("GoFiles") or []:
            count += len(pathlib.Path(p["Dir"], name).read_text(encoding="utf-8", errors="replace").splitlines())
        lines[rel] = count
    fan_in = {v: 0 for v in edges}
    for targets in edges.values():
        for t in targets:
            fan_in[t] = fan_in.get(t, 0) + 1
    fan_out = {v: len(t) for v, t in edges.items()}
    top = lambda d: [{"package": k, "count": n} for k, n in sorted(d.items(), key=lambda kv: (-kv[1], kv[0]))[:TOP]]
    return {
        "module": module, "packages": len(edges), "linesPerPackage": spread([n for n in lines.values() if n]),
        "largestPackages": [{"package": k, "lines": n} for k, n in sorted(lines.items(), key=lambda kv: (-kv[1], kv[0]))[:TOP]],
        "imports": sum(fan_out.values()), "fanOut": top(fan_out), "fanIn": top(fan_in),
        "longestChain": longest_chain(edges), "cycles": cycles(edges), "errors": errors,
    }, edges, test_edges, lines


def linters(root, env, tools):
    code, _, err = tool(["go", "build", "./..."], root, env)
    result = {"builds": code == 0}
    code, _, err = tool(["go", "vet", "./..."], root, env)
    result["vet"] = sum(1 for s in err.splitlines() if re.match(r"^\S+\.go:\d+", s.removeprefix("vet: ")))
    _, goroot, _ = tool(["go", "env", "GOROOT"], root, env)
    code, out, err = tool([str(pathlib.Path(goroot.strip(), "bin", "gofmt")), "-l", "."], root, env)
    result["gofmt"] = len([s for s in out.splitlines() if not any(part in SKIP_DIRS for part in pathlib.PurePath(s).parts)])
    result["staticcheck"] = result["deadcode"] = None
    # A program that does not build gives these tools nothing to say about it.
    if not tools or not result["builds"]:
        return result
    suffix = ".exe" if os.name == "nt" else ""
    check, dead = pathlib.Path(tools, "staticcheck" + suffix), pathlib.Path(tools, "deadcode" + suffix)
    if check.is_file():
        _, out, err = tool([str(check), "-f", "json", "./..."], root, env)
        counts = {}
        for s in out.splitlines():
            try:
                code_name = json.loads(s).get("code", "?")
            except ValueError:
                continue
            counts[code_name] = counts.get(code_name, 0) + 1
        result["staticcheck"] = {"findings": sum(counts.values()), "byCheck": dict(sorted(counts.items(), key=lambda kv: (-kv[1], kv[0]))[:10])}
    if dead.is_file():
        def unreachable(*flags):
            _, out, _ = tool([str(dead), *flags, "./..."], root, env)
            found = (UNREACHABLE.match(s) for s in out.splitlines())
            # With the file: two helpers of one name in two packages are two functions.
            return {f"{pathlib.PurePath(m.group(1)).as_posix()}: {m.group(2)}" for m in found if m}
        with_tests, from_main = unreachable("-test"), unreachable()
        result["deadcode"] = {"dead": len(with_tests), "onlyTests": len(from_main - with_tests), "examples": sorted(with_tests)[:TOP]}
    return result


def read_table(page):
    """The module table of the spec's architecture page: module -> the modules it may depend on, or None for any."""
    table = {}
    within = False
    for line in pathlib.Path(page).read_text(encoding="utf-8").splitlines():
        if MODULE_TABLE.match(line):
            within = True
        elif within and not line.startswith("|"):
            break
        elif within:
            m = MODULE_ROW.match(line)
            if m:
                if m.group(1) in table:
                    raise MeasureError(f"{page} lists the module {m.group(1)} twice")
                cell = m.group(2)
                table[m.group(1)] = [] if cell == "nothing" else QUOTED.findall(cell)
    if not table:
        raise MeasureError(f"{page} has no module table")
    return table


def against_spec(table, edges, test_edges, lines):
    """Do the modules match the Packages, and do the imports follow the spec's dependencies?"""
    at = {}
    for rel in edges:
        last = rel.rsplit("/", 1)[-1]
        if last in table and (last not in at or len(rel) < len(at[last])):
            at[last] = rel
    missing = sorted(set(table) - set(at))
    mapped = {rel: name for name, rel in at.items()}
    extra = sorted(rel for rel in edges if rel not in mapped and rel != "." and not rel.startswith("cmd/") and "/cmd/" not in rel)
    forbidden, forbidden_tests = [], []
    for rel, name in sorted(mapped.items()):
        allowed = set(table[name])
        for target in edges.get(rel, ()):
            other = mapped.get(target)
            if other and other not in allowed:
                forbidden.append({"from": name, "to": other})
        for target in test_edges.get(rel, ()):
            other = mapped.get(target)
            if other and other not in allowed:
                forbidden_tests.append({"from": name, "to": other})
    outside = sorted(rel for rel, name in mapped.items() if rel != name)
    module_lines = [lines[rel] for rel in mapped]
    return {
        "modules": len(table), "found": len(at), "missing": missing, "notAtTheirPath": outside, "extraPackages": extra,
        "forbiddenImports": forbidden, "forbiddenImportsInTests": len(forbidden_tests),
        "linesPerModule": spread(module_lines),
    }


def measure(folder, architecture=None, tools=None):
    folder = pathlib.Path(folder)
    with tempfile.TemporaryDirectory(ignore_cleanup_errors=True) as tmp:
        tmp = pathlib.Path(tmp)
        record = kept.clone(folder, tmp / "repo")
        root = module_root(tmp / "repo")
        env = environment(tmp / "cache")
        result = {"format": 1, "name": record["name"], "commit": record["commit"], "root": root.relative_to(tmp / "repo").as_posix(),
                  "toolchain": GO_TOOLCHAIN, "tools": PINNED}
        result["commits"] = int(kept.git("rev-list", "--count", "HEAD", cwd=tmp / "repo").strip())
        result.update(sizes(root))
        shape, edges, test_edges, lines = structure(root, env)
        result["structure"] = shape
        result["linters"] = linters(root, env, tools)
        if architecture:
            result["againstSpec"] = against_spec(read_table(architecture), edges, test_edges, lines)
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--kept", required=True)
    parser.add_argument("--architecture")
    parser.add_argument("--tools")
    parser.add_argument("--out", required=True)
    a = parser.parse_args(argv)
    try:
        result = measure(a.kept, a.architecture, a.tools)
    except (MeasureError, kept.KeptError, OSError, ValueError, KeyError, subprocess.TimeoutExpired) as e:
        print(f"measure: {type(e).__name__}: {e}", file=sys.stderr)
        return 2
    pathlib.Path(a.out).write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({k: result[k] for k in ("name", "lines", "duplication")}))
    return 0


if __name__ == "__main__":
    sys.exit(main())
