from __future__ import annotations

import copy
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

from playground import manifest

BASE = {
    "schema": 1,
    "id": "conv-readinglog-001",
    "case": "readinglog",
    "method": "conventional",
    "agent": {"kind": "codex", "codexVersion": "0.162.0", "model": "gpt-6-luna",
              "effort": "high", "maxSubagents": 3},
    "limits": {"stationSeconds": 5400, "totalSeconds": 14400},
}
MARKITECT = {"sourceRepo": "C:/src/Markitect", "commit": "669cecd2"}
PLAYGROUND = Path(__file__).resolve().parents[1]


def make_case(playground: Path, name: str, stations: int) -> Path:
    """A minimal valid case folder: README, BACKLOG, STATIONS.json and its public checks."""
    folder = playground / "cases" / name
    (folder / "checks").mkdir(parents=True)
    (folder / "README.md").write_text("# case\n", encoding="utf-8")
    (folder / "BACKLOG.md").write_text("- X01\n", encoding="utf-8")
    plan = {"schema": 1, "case": name, "stations": [{"id": f"S{n}", "items": [f"X{n:02d}"]}
                                                     for n in range(1, stations + 1)]}
    (folder / "STATIONS.json").write_text(json.dumps(plan), encoding="utf-8")
    (folder / "checks" / f"{name}.py").write_text("def checks(ctx):\n    pass\n", encoding="utf-8")
    return folder


def variant(**changes):
    data = copy.deepcopy(BASE)
    for dotted, value in changes.items():
        target, *path = dotted.split("__")
        node = data
        for key in [target, *path][:-1]:
            node = node[key]
        last = [target, *path][-1]
        if value is ...:
            del node[last]
        else:
            node[last] = value
    return data


class ManifestTests(unittest.TestCase):
    def assertRejected(self, data, fragment):
        with self.assertRaises(manifest.ManifestError) as caught:
            manifest.validate(data)
        self.assertIn(fragment, str(caught.exception))

    def test_valid_manifest_gets_container_defaults(self):
        result = manifest.validate(BASE)
        self.assertEqual(result["container"], {"cpus": 4, "memory": "8g", "pidsLimit": 2048})
        self.assertNotIn("markitect", result)
        self.assertEqual(result["agent"]["maxSubagents"], 3)

    def test_partial_container_block_keeps_given_values(self):
        result = manifest.validate(variant(container={"cpus": 2.5, "memory": "4g"}))
        self.assertEqual(result["container"], {"cpus": 2.5, "memory": "4g", "pidsLimit": 2048})

    def test_ids_and_enums(self):
        self.assertRejected(variant(id="Conv_1"), "id")
        self.assertRejected(variant(id="-leading-dash"), "id")
        self.assertRejected(variant(id="a" * 64), "id")
        manifest.validate(variant(id="a" * 63))
        self.assertRejected(variant(case="todo"), "case")
        self.assertRejected(variant(method="vibes"), "method")
        self.assertRejected(variant(agent__kind="gemini"), "agent.kind")
        self.assertRejected(variant(schema=2), "schema")
        self.assertRejected(variant(schema=True), "schema")
        manifest.validate(variant(agent__kind="fake"))

    def assertRejectedIn(self, playground, data, fragment):
        with self.assertRaises(manifest.ManifestError) as caught:
            manifest.validate(data, playground=playground)
        self.assertIn(fragment, str(caught.exception))

    def test_cases_come_from_the_case_folders(self):
        self.assertEqual(manifest.validate(variant(case="readinglog2"))["case"], "readinglog2")
        self.assertFalse(hasattr(manifest, "CASES"))
        with tempfile.TemporaryDirectory() as folder:
            playground = Path(folder)
            (playground / "cases" / "common").mkdir(parents=True)
            make_case(playground, "todo", 3)
            self.assertEqual(manifest.validate(variant(case="todo"), playground=playground)["stations"], 3)
            self.assertRejectedIn(playground, BASE, "case: expected one of todo, got 'readinglog'")
            (playground / "cases" / "broken").mkdir()
            self.assertRejectedIn(playground, variant(case="todo"), "cases/broken: README.md is missing")

    def test_stations_default_to_every_wave_of_the_case(self):
        self.assertEqual(manifest.validate(BASE)["stations"], 4)
        self.assertEqual(manifest.validate(variant(case="readinglog2"))["stations"], 6)
        self.assertEqual(manifest.validate(variant(stations=2))["stations"], 2)
        self.assertEqual(manifest.validate(variant(case="readinglog2", stations=6))["stations"], 6)
        self.assertRejected(variant(stations=5), "stations: case readinglog has 4 stations, got 5")
        for bad in (0, -1, 1.5, True, "2"):
            with self.subTest(stations=bad):
                self.assertRejected(variant(stations=bad), "stations")
        normalized = manifest.validate(variant(stations=3))
        self.assertEqual(manifest.validate(normalized), normalized)  # a normalized manifest validates again

    def test_claude_kinds_need_both_versions_and_codex_gets_a_default(self):
        self.assertEqual(manifest.validate(BASE)["agent"]["claudeVersion"], manifest.DEFAULT_CLAUDE_VERSION)
        for kind in ("claude", "fake-claude"):
            with self.subTest(kind=kind):
                self.assertRejected(variant(agent__kind=kind), "claudeVersion")
                result = manifest.validate(variant(agent__kind=kind, agent__claudeVersion="2.1.296",
                                                   agent__model="claude-opus-5-5"))
                self.assertEqual((result["agent"]["kind"], result["agent"]["claudeVersion"],
                                  result["agent"]["codexVersion"]), (kind, "2.1.296", "0.162.0"))
                self.assertRejected(variant(agent__kind=kind, agent__claudeVersion="2.1.296",
                                            agent__codexVersion=...), "codexVersion")
        self.assertRejected(variant(agent__claudeVersion="latest"), "agent.claudeVersion")
        self.assertRejected(variant(agent__claudeVersion="2" * 57), "agent.claudeVersion")  # Docker tag length
        self.assertEqual(manifest.validate(variant(agent__claudeVersion="2.1.300"))["agent"]["claudeVersion"],
                         "2.1.300")

    def test_limits_must_be_positive_integers(self):
        self.assertRejected(variant(limits__stationSeconds=0), "limits.stationSeconds")
        self.assertRejected(variant(limits__totalSeconds=-5), "limits.totalSeconds")
        self.assertRejected(variant(limits__totalSeconds=10.5), "limits.totalSeconds")
        self.assertRejected(variant(limits__totalSeconds=True), "limits.totalSeconds")
        self.assertRejected(variant(agent__maxSubagents=0), "agent.maxSubagents")
        self.assertRejected(variant(container={"pidsLimit": 0}), "container.pidsLimit")
        self.assertRejected(variant(container={"cpus": 0}), "container.cpus")
        self.assertRejected(variant(container={"memory": "lots"}), "container.memory")

    def test_missing_and_unknown_fields_are_named(self):
        self.assertRejected(variant(limits=...), "limits")
        self.assertRejected(variant(agent__model=...), "model")
        self.assertRejected(variant(extra=1), "extra")
        self.assertRejected(variant(agent__sandbox="none"), "sandbox")
        self.assertRejected(variant(agent__codexVersion="latest"), "agent.codexVersion")

    def test_markitect_block_only_for_markitect_method(self):
        self.assertRejected(variant(method="markitect"), "markitect: required")
        self.assertRejected(variant(markitect=MARKITECT), "only allowed")
        result = manifest.validate(variant(method="markitect", markitect=MARKITECT))
        self.assertEqual(result["markitect"], MARKITECT)
        self.assertRejected(variant(method="markitect", markitect={**MARKITECT, "commit": "HEAD"}),
                            "markitect.commit")
        self.assertRejected(variant(method="markitect", markitect={**MARKITECT, "sourceRepo": " "}), "sourceRepo")

    @unittest.skipUnless(any((folder / ".git").exists() for folder in PLAYGROUND.resolve().parents),
                         "the playground is not inside a Git checkout here (e.g. mounted into the image)")
    def test_this_checkout_is_the_default_source_repo(self):
        result = manifest.validate(variant(method="markitect", markitect={"commit": "669cecd2"}))
        source = Path(result["markitect"]["sourceRepo"])
        self.assertTrue(source.is_absolute())
        self.assertTrue((source / ".git").exists())
        self.assertIn(source, PLAYGROUND.resolve().parents)
        self.assertEqual(manifest.validate(result), result)

    def test_source_repo_defaults_to_the_checkout_holding_the_playground(self):
        with tempfile.TemporaryDirectory() as folder:
            checkout = Path(folder) / "checkout"
            playground = checkout / "experiments" / "case-playground"
            (playground / "cases" / "common").mkdir(parents=True)
            make_case(playground, "readinglog", 4)
            data = variant(method="markitect", markitect={"commit": "669cecd2"})
            self.assertRejectedIn(playground, data, "no Git checkout holds the playground")
            (checkout / ".git").write_text("gitdir: elsewhere\n", encoding="utf-8")  # a worktree's .git file
            self.assertRejectedIn(playground, data, "not a Markitect checkout")
            (checkout / "go.mod").write_text("module example.com/other\n", encoding="utf-8")
            self.assertRejectedIn(playground, data, "set markitect.sourceRepo")
            (checkout / "go.mod").write_text("// product\nmodule github.com/Glacius-Labs/Markitect // root\n\n"
                                             "go 1.27.1\n", encoding="utf-8")
            self.assertEqual(manifest.validate(data, playground=playground)["markitect"]["sourceRepo"],
                             str(checkout.resolve()))
            given = manifest.validate(variant(method="markitect", markitect=MARKITECT), playground=playground)
            self.assertEqual(given["markitect"]["sourceRepo"], MARKITECT["sourceRepo"])  # the host resolves it

    def test_load_resolves_a_relative_source_repo_from_the_manifest_folder(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "studies" / "m.json"
            path.parent.mkdir()
            for given, expected in (("../Markitect", path.parent / ".." / "Markitect"),
                                    ("~/Markitect", Path.home() / "Markitect"),
                                    (str(Path(folder).resolve() / "abs"), Path(folder).resolve() / "abs")):
                with self.subTest(given=given):
                    path.write_text(json.dumps(variant(method="markitect",
                                                       markitect={"sourceRepo": given, "commit": "669cecd2"})),
                                    encoding="utf-8")
                    self.assertEqual(manifest.load(path)["markitect"]["sourceRepo"], str(expected))

    def test_claude_with_markitect_names_the_inner_roles_model(self):
        claude = {"agent__kind": "claude", "agent__claudeVersion": "2.1.296", "agent__model": "claude-opus-5-5",
                  "method": "markitect"}
        self.assertRejected(variant(**claude, markitect=MARKITECT), "innerEffort, innerModel")
        self.assertRejected(variant(**claude, markitect={**MARKITECT, "innerModel": "gpt-6-luna"}), "innerEffort")
        inner = {**MARKITECT, "innerModel": "gpt-6-luna", "innerEffort": "high"}
        self.assertEqual(manifest.validate(variant(**claude, markitect=inner))["markitect"], inner)
        self.assertEqual(manifest.validate(variant(method="markitect", markitect=inner))["markitect"], inner)
        self.assertRejected(variant(method="markitect", markitect={**inner, "innerEffort": "HIGH"}),
                            "markitect.innerEffort")

    def test_load_reads_utf8_json_and_reports_bad_files(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "m.json"
            path.write_text(json.dumps(BASE), encoding="utf-8")
            self.assertEqual(manifest.load(path)["id"], BASE["id"])
            path.write_text("{not json", encoding="utf-8")
            with self.assertRaises(manifest.ManifestError):
                manifest.load(path)
            with self.assertRaises(manifest.ManifestError):
                manifest.load(Path(folder) / "missing.json")
            self.assertTrue(issubclass(manifest.ManifestError, ValueError))

    def test_shipped_examples_are_valid(self):
        examples = Path(__file__).resolve().parents[1] / "examples"
        # The Markitect example relies on the sourceRepo default, which needs a checkout
        # around the playground; the tests also run on a copy mounted into the image.
        with mock.patch.object(manifest, "default_source_repo", return_value="/checkout"):
            for path in sorted(examples.glob("*.json")):
                if path.name.startswith("study-"):
                    continue  # study files; see test_study
                with self.subTest(example=path.name):
                    manifest.load(path)


if __name__ == "__main__":
    unittest.main()
