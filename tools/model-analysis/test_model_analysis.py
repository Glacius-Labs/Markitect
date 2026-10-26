"""Model analysis over small synthetic check reports and Definition files."""

import copy
import io
import json
import pathlib
import tempfile
import unittest
from contextlib import redirect_stdout

import model_analysis as ma

LOCAL, CORE_DIGEST = "sha256:" + "1" * 64, "sha256:" + "2" * 64


def ref(address):
    package, local = ma.parse_identity(address)
    return {"Package": package, "ID": local}


def definition(address, kind, body="Text.", spec=None, links=()):
    package, local = ma.parse_identity(address)
    return {"identity": ref(address), "kind": ref(kind), "title": None, "body": body, "spec": spec or {},
            "bodyLinks": [{"Target": ref(t), "Display": None, "Coordinates": None} for t in links] or None,
            "source": {"Origin": "repo", "Path": f"model/{package}/{local}.yaml", "Digest": "sha256:" + "3" * 64}}


def words(n):
    return " ".join(["word"] * n)


CORE = [definition("markitect/core/Kind", "markitect/core/Kind"), definition("markitect/core/Property", "markitect/core/Kind")]
VOCABULARY = [
    definition("vocab/Rule", "markitect/core/Kind", spec={"properties": [ref("vocab/owner"), ref("vocab/level")]}),
    definition("vocab/owner", "markitect/core/Property", spec={"type": "reference", "target": ref("vocab/Rule"), "minCount": {"integer": "0"}, "maxCount": {"integer": "1"}}),
    definition("vocab/level", "markitect/core/Property", spec={"name": "lvl", "type": "string", "minCount": {"integer": "0"}, "maxCount": {"integer": "1"}}),
]
CONTENT = [
    definition("a/x", "vocab/Rule", "Uses `fooBar`, `x` and `ok_code`. " + words(160), {"owner": ref("b/y")}, ["c/z"]),
    definition("b/y", "vocab/Rule", "Also `fooBar`. See [[a/x]].", links=["a/x"]),
    definition("c/z", "vocab/Rule", "Again `fooBar`. " + words(310), {"notes": [{"see": ref("a/x")}]}),
    definition("d/lonely", "vocab/Rule", "Alone with `ok_code`."),
]
IMPORTS = {
    "a/x": "imports:\n  - package: vocab\n    as: Vocab\n  - package: b\n  - package: c\n  - package: d  # unused\nid: x\n",
    "b/y": "imports: [{package: vocab, as: Vocab}, {package: a}]\nid: y\n",
    "c/z": "imports:\n- package: 'vocab'\n- package: \"a\"\nid: z\n",
    "d/lonely": "imports:\n  - package: vocab\nid: lonely\n",
    "vocab/Rule": "imports:\n  - package: markitect/core\nid: Rule\n",
    "vocab/owner": "imports:\n  - package: markitect/core\nid: owner\n",
    "vocab/level": "imports:\n  - package: markitect/core\nid: level\n",
}


def report(content=CONTENT, records=None, ownership=("b",), vocabulary=VOCABULARY):
    """A passed check report: one context per content Package, each with the vocabulary and imported Core."""
    contexts = []
    for package in sorted({d["identity"]["Package"] for d in content}):
        own = [d for d in content if d["identity"]["Package"] == package]
        entries = [{"definition": d, "canonicalModelDigest": LOCAL} for d in own + vocabulary]
        entries += [{"definition": d, "canonicalModelDigest": CORE_DIGEST} for d in CORE]
        contexts.append({"package": package, "modelID": "fixture", "canonicalModelDigest": LOCAL, "definitions": entries})
    if records is None:
        records = [{"Owner": "a", "Path": "src/a/x.go", "Form": "code", "Realizes": ["a/x"], "Why": "Implements."}]
    return {"status": "passed", "contexts": contexts, "stages": {"structural": {
        "model": {"models": [{"id": "fixture", "local": True, "definitions": len(content) + len(VOCABULARY), "canonicalModelDigest": LOCAL,
                              "source": {"Origin": "repo", "Path": "model.yaml"}},
                             {"id": "markitect-core", "local": False, "canonicalModelDigest": CORE_DIGEST,
                              "source": {"Origin": "repo", "Path": "deps/core/model.yaml"}}]},
        "registry": {"records": records},
        "execution": {"declarations": {"Ownership": {"Value": [{"Path": {"Value": "b/"}, "Owner": {"Value": p}} for p in ownership]}}}}}}


class ModelTests(unittest.TestCase):
    def test_contexts_join_into_local_definitions(self):
        model = ma.Model(report())
        self.assertEqual(len(model.local), 7)
        self.assertNotIn(("markitect/core", "Kind"), model.local)
        self.assertEqual(model.content(), [("a", "x"), ("b", "y"), ("c", "z"), ("d", "lonely")])

    def test_edges_cover_spec_references_and_links(self):
        edges = {(ma.text(s), ma.text(t), via, at) for s, t, via, at in ma.Model(report()).edges}
        self.assertIn(("a/x", "b/y", "spec", "/spec/owner"), edges)
        self.assertIn(("c/z", "a/x", "spec", "/spec/notes/0/see"), edges)
        self.assertIn(("b/y", "a/x", "link", "/bodyLinks/0"), edges)
        self.assertIn(("vocab/owner", "vocab/Rule", "spec", "/spec/target"), edges)
        self.assertFalse(any(at.startswith("/spec/minCount") for *_, at in edges))

    def test_failed_report_is_refused(self):
        with self.assertRaises(ma.InputError):
            ma.Model({"status": "failed", "contexts": []})


class GraphTests(unittest.TestCase):
    def test_cycles_coupling_and_cross_package_share(self):
        g = ma.package_graph(ma.Model(report()))
        self.assertEqual((g["contentDefinitions"], g["contentPackages"], g["links"]), (4, 4, 2))
        self.assertEqual(g["cycles"], [["a", "b", "c"]])
        self.assertEqual(g["packageEdges"], 4)
        self.assertEqual(g["crossPackageDefinitionEdges"], 4)
        self.assertEqual(g["definitionEdges"], 7)
        self.assertEqual(g["crossPackageShare"], 0.571)
        a = next(c for c in g["coupling"] if c["package"] == "a")
        self.assertEqual((a["afferent"], a["efferent"], a["instability"]), (2, 2, 0.5))
        d = next(c for c in g["coupling"] if c["package"] == "d")
        self.assertIsNone(d["instability"])

    def test_strongly_connected_handles_self_and_chains(self):
        self.assertEqual(ma.strongly_connected({1, 2, 3, 4}, {1: {2}, 2: {1, 3}, 3: {3}}), [[1, 2], [3], [4]])


class ReferenceTests(unittest.TestCase):
    def test_incoming_edges_instances_and_artifacts(self):
        model = ma.Model(report())
        r = ma.references(model, ("a", "x"))
        self.assertEqual({(i["from"], i["via"]) for i in r["incoming"]}, {("b/y", "link"), ("c/z", "spec")})
        self.assertEqual(r["artifacts"], [{"path": "src/a/x.go", "form": "code", "owner": "a"}])
        self.assertEqual(ma.references(model, ("vocab", "Rule"))["instances"], ["a/x", "b/y", "c/z", "d/lonely"])
        with self.assertRaises(ma.InputError):
            ma.references(model, ("a", "missing"))


class ImpactTests(unittest.TestCase):
    def test_changed_added_removed_and_their_referrers(self):
        new = copy.deepcopy(CONTENT[:3]) + [definition("e/new", "vocab/Rule", links=["c/z"])]
        new[1]["body"] = "Changed."
        new[2]["source"]["Digest"] = "sha256:" + "9" * 64  # provenance only: not a change
        r = ma.impact(ma.Model(report()), ma.Model(report(new)))
        self.assertEqual((r["changed"], r["added"], r["removed"]), (["b/y"], ["e/new"], ["d/lonely"]))
        self.assertEqual(r["affected"], ["a/x"])
        self.assertEqual(r["artifacts"], [{"path": "src/a/x.go", "realizes": "a/x"}])
        self.assertEqual(ma.impact(ma.Model(report()), ma.Model(report(new)), transitive=True)["affected"], ["a/x", "c/z"])

    def test_identical_reports_have_no_impact(self):
        r = ma.impact(ma.Model(report()), ma.Model(report()))
        self.assertEqual(r["changed"] + r["added"] + r["removed"] + r["affected"], [])

    def test_kind_change_reaches_instances(self):
        vocabulary = copy.deepcopy(VOCABULARY)
        vocabulary[0]["body"] = "A changed contract."
        r = ma.impact(ma.Model(report()), ma.Model(report(vocabulary=vocabulary)))
        self.assertEqual(r["changed"], ["vocab/Rule"])
        self.assertEqual(r["affected"], ["a/x", "b/y", "c/z", "d/lonely", "vocab/owner"])


class ImportReaderTests(unittest.TestCase):
    def test_forms(self):
        self.assertEqual(ma.read_imports(IMPORTS["a/x"]), [("vocab", "Vocab"), ("b", None), ("c", None), ("d", None)])
        self.assertEqual(ma.read_imports(IMPORTS["b/y"]), [("vocab", "Vocab"), ("a", None)])
        self.assertEqual(ma.read_imports(IMPORTS["c/z"]), [("vocab", None), ("a", None)])
        self.assertEqual(ma.read_imports("imports: []\nid: q\n"), [])
        self.assertEqual(ma.read_imports("id: q\nbody: |-\n  imports: [x]\n"), [])

    def test_unsupported_forms_are_reported_not_guessed(self):
        self.assertIsNone(ma.read_imports("imports: [\n  {package: a}]\n"))
        self.assertIsNone(ma.read_imports("imports:\n  - {package: a}\n"))


class LintTests(unittest.TestCase):
    def lint(self, **kwargs):
        with tempfile.TemporaryDirectory() as repo:
            for address, source in IMPORTS.items():
                package, local = ma.parse_identity(address)
                path = pathlib.Path(repo, "model", package, local + ".yaml")
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(source, encoding="utf-8")
            return ma.lint(ma.Model(report()), repo, **kwargs)

    def signals(self, result):
        return {(f["signal"], f["severity"], f["subject"]) for f in result["findings"]}

    def test_every_signal(self):
        result = self.lint(rules={"layers": [["a"], ["b", "c"]], "forbid": [{"from": "c", "to": "a", "reason": "c stays below a"}]})
        found = self.signals(result)
        self.assertIn(("size", "warning", "a/x"), found)
        self.assertIn(("size", "strong", "c/z"), found)
        self.assertIn(("isolated", "warning", "d/lonely"), found)
        self.assertEqual({s for sig, _, s in found if sig == "no-area"}, {"c", "d"})
        self.assertIn(("unused-import", "warning", "a/x"), found)
        self.assertEqual({s for sig, _, s in found if sig == "unset-field"}, {"vocab/Rule"})
        self.assertTrue(any("`lvl`" in f["message"] for f in result["findings"] if f["signal"] == "unset-field"))
        self.assertEqual({s for sig, _, s in found if sig == "undefined-term"}, {"fooBar"})
        self.assertIn(("import-cycle", "warning", "a <-> b <-> c"), found)
        self.assertEqual({s for sig, _, s in found if sig == "direction"}, {"b -> a", "c -> a"})
        self.assertEqual(result["notes"], [])

    def test_without_repo_import_signals_are_skipped_with_a_note(self):
        result = ma.lint(ma.Model(report()))
        self.assertFalse({"unused-import", "import-cycle"} & set(result["counts"]))
        self.assertEqual(len(result["notes"]), 1)

    def test_lint_never_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = pathlib.Path(tmp, "report.json")
            path.write_text(json.dumps(report()), encoding="utf-16")
            out = io.StringIO()
            with redirect_stdout(out):
                self.assertEqual(ma.main(["lint", str(path), "--json"]), 0)
            self.assertIn("undefined-term", json.loads(out.getvalue())["counts"])


class LockTests(unittest.TestCase):
    def test_lines_from_report_and_manifests(self):
        with tempfile.TemporaryDirectory() as repo:
            manifest = pathlib.Path(repo, "deps", "core", "model.yaml")
            manifest.parent.mkdir(parents=True)
            manifest.write_text('formatVersion: 1\nid: markitect-core\nversion: "1.1.0"\nroot: model\n', encoding="utf-8")
            text, notes = ma.lock_lines(report(), repo)
        self.assertEqual(text, f"formatVersion: 1\nmodels:\n  markitect-core:\n    version: 1.1.0\n    digest: {CORE_DIGEST}\n")
        self.assertEqual(notes, [])

    def test_missing_manifest_is_noted(self):
        text, notes = ma.lock_lines(report(), "does-not-exist")
        self.assertIn("version: UNKNOWN", text)
        self.assertEqual(len(notes), 1)


class CommandTests(unittest.TestCase):
    def run_main(self, *args):
        out = io.StringIO()
        with redirect_stdout(out):
            code = ma.main(list(args))
        return code, out.getvalue()

    def test_text_and_json_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = pathlib.Path(tmp, "report.json")
            path.write_text(json.dumps(report()), encoding="utf-8")
            code, out = self.run_main("graph", str(path))
            self.assertEqual(code, 0)
            self.assertIn("Cycles: 1", out)
            code, out = self.run_main("refs", str(path), "a/x", "--json")
            self.assertEqual(json.loads(out)["definition"], "a/x")
            code, out = self.run_main("impact", str(path), str(path))
            self.assertIn("Changed: 0", out)
            self.assertEqual(self.run_main("refs", str(path), "nope")[0], 2)


if __name__ == "__main__":
    unittest.main()
