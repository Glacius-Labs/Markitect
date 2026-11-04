"""The lasting store, the measuring and the blind material on small repositories, without an agent."""
import json
import os
import pathlib
import shutil
import subprocess
import tempfile
import unittest

import blind_read
import kept
import measure
import side_by_side

GO = shutil.which("go")


def git(*args, cwd):
    return subprocess.run(["git", "-c", "user.name=t", "-c", "user.email=t@example.com", *args], cwd=cwd, check=True, capture_output=True, text=True).stdout.strip()


def write(root, files):
    for name, text in files.items():
        p = pathlib.Path(root, name)
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(text, encoding="utf-8")


PROGRAM = {
    "realization/go.mod": "module demo\n\ngo 1.24\n",
    "realization/cmd/demo/main.go": 'package main\n\nimport "demo/a"\n\nfunc main() { a.Run() }\n',
    "realization/a/a.go": 'package a\n\nimport "demo/b"\n\nfunc Run() { b.Do() }\n',
    "realization/b/b.go": 'package b\n\nimport "demo/c"\n\nfunc Do() { c.Do() }\n\nfunc unused() {}\n',
    "realization/c/c.go": "package c\n\nfunc Do() {}\n",
    "realization/b/b_test.go": 'package b\n\nimport "testing"\n\nfunc TestDo(t *testing.T) { Do() }\n',
    "task.md": "Build it.\n",
    ".markitect/manifest.yaml": "libraries: {}\n",
    "CLAUDE.md": "notes\n",
}
TABLE = """An earlier table does not count:

| Code | Why | What |
| --- | --- | --- |
| `early` | no | `x` |

| Module | Model | May depend on |
| --- | --- | --- |
| `a` | [[a]] | `b` |
| `b` | [[b]] | nothing |
| `c` | [[c]] | nothing |
| `d` | [[d]] | nothing |

Nor does a later one:

| Code | Why | What |
| --- | --- | --- |
| `b` | no | `x` |
"""


class Store(unittest.TestCase):
    def setUp(self):
        self.tmp = pathlib.Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, self.tmp, ignore_errors=True)
        self.addCleanup(os.environ.pop, kept.TOKEN_VARIABLE, None)
        os.environ.pop(kept.TOKEN_VARIABLE, None)
        self.store = self.tmp / "store"
        self.store.mkdir()

    def run_dir(self, files=PROGRAM, transcript='{"type":"assistant","message":{"model":"claude-haiku-5-5"}}\n'):
        repo = self.tmp / "repo"
        shutil.rmtree(repo, ignore_errors=True)
        repo.mkdir()
        write(repo, files)
        git("init", "-q", "-b", "result", cwd=repo)
        git("add", "-A", cwd=repo)
        git("commit", "-q", "-m", "x", cwd=repo)
        commit = git("rev-parse", "HEAD", cwd=repo)
        run = self.tmp / "run"
        shutil.rmtree(run, ignore_errors=True)
        (run / "variant" / "transcripts" / "s" / "subagents").mkdir(parents=True)
        git("bundle", "create", "-q", str(run / "variant" / "result.bundle"), "result", cwd=repo)
        (run / "suite.json").write_text(json.dumps({"delivered": True, "wallSeconds": 60.5, "result": {"commit": commit}}), encoding="utf-8")
        (run / "variant" / "run.json").write_text(json.dumps({"requestedModel": "claude-haiku-5-5", "usage": {"turns": 3, "costUsd": 1.5}}), encoding="utf-8")
        (run / "variant" / "transcripts" / "s" / "lead.jsonl").write_text(transcript, encoding="utf-8")
        (run / "variant" / "transcripts" / "s" / "subagents" / "one.jsonl").write_text(transcript, encoding="utf-8")
        return run, commit

    def test_a_launch_is_kept_whole_and_never_changes(self):
        run, commit = self.run_dir()
        r = kept.add(self.store, "demo", "p", "W0", "K", run, weekly=(19, 20), note="n")
        self.assertEqual(r["commit"], commit)
        self.assertEqual(r["modelsAnswered"], {"claude-haiku-5-5": 2})
        self.assertEqual(r["subagents"], 1)
        self.assertEqual(r["relativeCost"], 1.5)
        self.assertNotIn("costUsd", r["usage"])
        self.assertEqual(r["weekly"], {"before": 19, "after": 20})
        self.assertEqual(kept.check(self.store), [])
        with self.assertRaisesRegex(kept.KeptError, "never changes"):
            kept.add(self.store, "demo", "p", "W0", "K", run)
        with open(self.store / "demo" / "repo.bundle", "ab") as f:
            f.write(b"x")
        self.assertEqual(kept.check(self.store), ["demo: repo.bundle changed"])

    def test_a_commands_result_is_kept(self):
        run, _ = self.run_dir()
        (run / "variant" / "results").mkdir()
        (run / "variant" / "results" / "r.json").write_text('{"state": "findings"}', encoding="utf-8")
        kept.add(self.store, "demo", "p", "W0", "default", run)
        self.assertEqual(json.loads((self.store / "demo" / "records" / "results" / "r.json").read_text(encoding="utf-8"))["state"], "findings")

    def test_a_commands_cost_is_the_sum_of_its_sessions(self):
        lines = [json.dumps({"type": "assistant", "message": {"model": "claude-sonnet-5-5"}}), json.dumps({"type": "result", "total_cost_usd": 1.25})]
        run, _ = self.run_dir(transcript="\n".join(lines) + "\n")
        (run / "variant" / "run.json").write_text(json.dumps({"model": "claude-sonnet-5-5", "tokens": {"inputTokens": 5}}), encoding="utf-8")
        r = kept.add(self.store, "command", "p", "W0", "D", run)
        # The subagent's transcript does not count twice.
        self.assertEqual(r["relativeCost"], 1.25)
        self.assertEqual(r["requestedModel"], "claude-sonnet-5-5")
        self.assertEqual(r["usage"], {"inputTokens": 5})

    def test_a_changed_record_and_a_deleted_launch_show(self):
        run, _ = self.run_dir()
        kept.add(self.store, "one", "p", "W0", "K", run)
        kept.add(self.store, "two", "p", "W0", "K", run)
        with open(self.store / "one" / "launch.json", "a", encoding="utf-8") as f:
            f.write(" ")
        shutil.rmtree(self.store / "two")
        self.assertEqual(kept.check(self.store), ["two: the launch is gone", "one: launch.json changed"])

    def test_a_credential_in_the_evidence_keeps_nothing(self):
        run, _ = self.run_dir(transcript="sk-ant-" + "a" * 30 + "\n")
        with self.assertRaisesRegex(kept.KeptError, "holds a credential") as caught:
            kept.add(self.store, "demo", "p", "W0", "K", run)
        self.assertNotIn("sk-ant", str(caught.exception))
        self.assertEqual(list(self.store.iterdir()), [])

    def test_the_tokens_value_keeps_nothing(self):
        os.environ[kept.TOKEN_VARIABLE] = "a-value-long-enough-to-find"
        run, _ = self.run_dir(transcript="a-value-long-enough-to-find\n")
        with self.assertRaisesRegex(kept.KeptError, "holds a credential"):
            kept.add(self.store, "demo", "p", "W0", "K", run)

    def test_an_undelivered_run_is_not_kept(self):
        run, _ = self.run_dir()
        (run / "suite.json").write_text(json.dumps({"delivered": False}), encoding="utf-8")
        with self.assertRaisesRegex(kept.KeptError, "delivered no result"):
            kept.add(self.store, "demo", "p", "W0", "K", run)

    def test_blind_material_names_nothing_about_how_it_was_made(self):
        run, _ = self.run_dir()
        kept.add(self.store, "first", "p", "W0", "K", run)
        kept.add(self.store, "second", "p", "W0", "S", run)
        requirements = self.tmp / "req.md"
        requirements.write_text("the requirements", encoding="utf-8")
        out = self.tmp / "blind"
        labels = blind_read.prepare(self.store, ["first", "second"], requirements, out)
        self.assertEqual(labels, ["A", "B"])
        key = json.loads((out / "key.json").read_text(encoding="utf-8"))
        self.assertEqual(sorted(v["name"] for v in key.values()), ["first", "second"])
        for label in labels:
            folder = out / "material" / label
            self.assertEqual(sorted(p.name for p in folder.iterdir()), sorted(["requirements.md", "rubric.md", key[label]["token"]]))
            names = {p.relative_to(folder / key[label]["token"]).as_posix() for p in (folder / key[label]["token"]).rglob("*") if p.is_file()}
            self.assertIn("realization/a/a.go", names)
            for hidden in ("task.md", "CLAUDE.md", ".markitect/manifest.yaml"):
                self.assertNotIn(hidden, names)
            self.assertFalse(any(name.startswith(".git") for name in names))
        with self.assertRaisesRegex(blind_read.ReadError, "exists"):
            blind_read.prepare(self.store, ["first"], requirements, out)

    def read_material(self):
        root = self.tmp / "blind"
        for label in "AB":
            (root / "material" / label).mkdir(parents=True)
        (root / "key.json").write_text(json.dumps({"A": {"name": "one", "token": "t1"}, "B": {"name": "two", "token": "t2"}}), encoding="utf-8")
        return root

    def fake_session(self, answers, weekly=0.2):
        """reviewing.session with canned answers, each session's stream holding a weekly figure."""
        import reviewing

        started = []
        readings = iter(weekly if isinstance(weekly, list) else [weekly] * 9)

        def session(name, prompt, token, material, out, model, box, reviewer, deadline):
            started.append(token)
            (out / "sessions").mkdir(parents=True)
            event = {"type": "rate_limit_event", "rate_limit_info": {"status": "allowed", "unifiedWindows": {"seven_day": {"utilization": next(readings)}}}}
            (out / "sessions" / "s.jsonl").write_text(json.dumps(event) + "\n", encoding="utf-8")
            answer = answers.pop(0)
            return {"valid": answer is not None, "answer": answer, "reported": {"usage": {"turns": 0 if answer == "no work" else 3}}, "wallSeconds": 5}

        original = reviewing.session
        reviewing.session = session
        self.addCleanup(setattr, reviewing, "session", original)
        return started

    def good_answer(self):
        part = {"score": 3, "good": [], "weak": [], "files": []}
        return {"architecture": part, "naming": part, "cleanliness": part, "tests": part, "takingOver": {"wouldMaintain": "yes"}}

    def test_a_failed_read_stays_and_a_second_run_starts_only_what_lacks_one(self):
        root = self.read_material()
        started = self.fake_session([None, self.good_answer(), self.good_answer()])
        results = blind_read.run(root, "claude:m", None)
        self.assertEqual({k: v["valid"] for k, v in results.items()}, {"A": False, "B": True})
        self.assertEqual(started, ["t1", "t2"])
        results = blind_read.run(root, "claude:m", None)
        self.assertEqual({k: v["valid"] for k, v in results.items()}, {"A": True, "B": True})
        self.assertEqual(started, ["t1", "t2", "t1"])
        self.assertTrue((root / "reads" / "A" / "a1" / "read.json").is_file())
        self.assertTrue((root / "reads" / "A" / "a2" / "read.json").is_file())
        self.assertEqual(results["A"]["weeklyReading"], 20.0)

    def test_a_killed_attempt_does_not_wedge_the_next_start(self):
        root = self.read_material()
        (root / "reads" / "A" / "a1").mkdir(parents=True)
        started = self.fake_session([self.good_answer(), self.good_answer()])
        results = blind_read.run(root, "claude:m", None)
        self.assertEqual(len(started), 2)
        self.assertTrue(results["A"]["valid"])
        self.assertTrue((root / "reads" / "A" / "a2" / "read.json").is_file())

    def test_a_session_that_did_no_work_stops_the_reads(self):
        root = self.read_material()
        started = self.fake_session(["no work", self.good_answer()])
        with self.assertRaisesRegex(blind_read.ReadError, "reports no turns"):
            blind_read.run(root, "claude:m", None)
        self.assertEqual(len(started), 1)

    def test_the_weekly_limit_counts_from_the_first_start(self):
        root = self.read_material()
        started = self.fake_session([None, self.good_answer(), self.good_answer()], weekly=[0.2, 0.25, 0.3])
        self.assertFalse(blind_read.run(root, "claude:m", None, stop_after=8)["A"]["valid"])
        with self.assertRaisesRegex(blind_read.ReadError, "rose from 20.0 to 30.0"):
            blind_read.run(root, "claude:m", None, stop_after=8)
        self.assertEqual(len(started), 3)

    def test_the_reads_stop_at_the_weekly_limit(self):
        root = self.read_material()
        started = self.fake_session([self.good_answer(), self.good_answer()], weekly=[0.2, 0.3])
        with self.assertRaisesRegex(blind_read.ReadError, "rose from 20.0 to 30.0"):
            blind_read.run(root, "claude:m", None, stop_after=8)
        self.assertEqual(len(started), 2)

    def test_the_run_command_takes_the_suites_series_by_default(self):
        import sandbox as sandboxing

        root = self.read_material()
        self.fake_session([self.good_answer(), self.good_answer()])
        seen = {}

        class Box:
            def __init__(self, engine, image, series, code, login, provider):
                seen.update(series=series, login=login, provider=provider)

        original = sandboxing.Sandbox
        sandboxing.Sandbox = Box
        self.addCleanup(setattr, sandboxing, "Sandbox", original)
        code = blind_read.main(["run", "--material", str(root), "--sandbox", "sha256:x", "--variant-code", "code"])
        self.assertEqual(code, 0)
        self.assertIn("limits", seen["series"])
        self.assertEqual(seen["provider"], "claude")
        self.assertIsInstance(seen["login"], str)

    def test_a_reads_answer_needs_its_scores(self):
        part = {"score": 3, "good": [], "weak": [], "files": []}
        answer = {"architecture": part, "naming": part, "cleanliness": part, "tests": part, "takingOver": {"wouldMaintain": "yes"}}
        self.assertIsNone(blind_read.valid_answer(answer))
        self.assertIn("naming", blind_read.valid_answer({**answer, "naming": {"score": 9}}))
        self.assertEqual(blind_read.valid_answer(None), "no answer")

    def test_duplication_finds_a_copy_and_a_renamed_copy(self):
        lines = [f"call{i}(x, {i})" for i in range(14)]
        self.assertEqual(measure.repeated_share({"a": lines, "b": lines}, 6), 1.0)
        self.assertEqual(measure.repeated_share({"a": lines, "b": [f"other{i}(y, {i + 9})" for i in range(14)]}, 6), 0.0)
        self.assertEqual(measure.repeated_share({"a": lines, "b": [f"other{i}(y, {i + 9})" for i in range(14)]}, 12, measure.renamed), 1.0)

    def test_blank_lines_comments_and_brackets_do_not_count(self):
        self.assertIsNone(measure.significant("   "))
        self.assertIsNone(measure.significant("// a comment"))
        self.assertIsNone(measure.significant("}"))
        self.assertEqual(measure.significant("  x := 1  "), "x := 1")

    def test_a_cycle_is_found(self):
        self.assertEqual(measure.cycles({"a": ["b"], "b": ["a"], "c": ["a"]}), [["a", "b"]])
        self.assertEqual(measure.cycles({"a": ["b"], "b": []}), [])
        self.assertEqual(measure.longest_chain({"a": ["b"], "b": ["c"], "c": []}), 3)

    def test_the_spec_table_and_what_breaks_it(self):
        page = self.tmp / "architecture.md"
        page.write_text(TABLE, encoding="utf-8")
        table = measure.read_table(page)
        self.assertEqual(table, {"a": ["b"], "b": [], "c": [], "d": []})
        page.write_text(TABLE.replace("| `d` | [[d]] | nothing |", "| `a` | [[d]] | nothing |"), encoding="utf-8")
        with self.assertRaisesRegex(measure.MeasureError, "twice"):
            measure.read_table(page)
        edges = {"cmd/demo": ["a"], "a": ["b"], "b": ["c"], "c": [], "helpers": []}
        spec = measure.against_spec(table, edges, {"c": ["a"]}, {k: 1 for k in edges})
        self.assertEqual(spec["found"], 3)
        self.assertEqual(spec["missing"], ["d"])
        self.assertEqual(spec["extraPackages"], ["helpers"])
        self.assertEqual(spec["forbiddenImports"], [{"from": "b", "to": "c"}])
        self.assertEqual(spec["forbiddenImportsInTests"], 1)

    @unittest.skipUnless(GO, "needs Go")
    def test_measuring_a_kept_result(self):
        run, _ = self.run_dir()
        kept.add(self.store, "demo", "p", "W0", "K", run)
        page = self.tmp / "architecture.md"
        page.write_text(TABLE, encoding="utf-8")
        original = measure.GO_TOOLCHAIN
        probe = subprocess.run(["go", "env", "GOVERSION"], capture_output=True, text=True, env={**os.environ, "GOTOOLCHAIN": original, "GOPROXY": "off"})
        if probe.returncode != 0:
            self.skipTest(f"{original} is not in the module cache")
        m = measure.measure(self.store / "demo", page)
        self.assertEqual(m["root"], "realization")
        self.assertEqual(m["structure"]["packages"], 4)
        self.assertEqual(m["structure"]["cycles"], [])
        self.assertEqual(m["testFunctions"], 1)
        self.assertTrue(m["linters"]["builds"])
        self.assertIsNone(m["linters"]["staticcheck"])
        self.assertEqual(m["structure"]["linesPerPackage"]["count"], 4)
        self.assertEqual(m["againstSpec"]["missing"], ["d"])
        self.assertEqual(m["againstSpec"]["forbiddenImports"], [{"from": "b", "to": "c"}])

    def test_the_page_holds_the_key_back_and_gives_no_dollar_amount(self):
        figure = lambda name, n: {"lines": {"program": n, "tests": 1, "testShare": 0.5}, "testFunctions": 1, "commits": 2,
                                  "files": {"count": 1, "over": 0, "max": 9}, "functions": {"count": 1, "over": 0, "max": 9},
                                  "structure": {"packages": 1, "largestPackages": [{"package": "a", "lines": 9}], "imports": 0, "fanOut": [], "longestChain": 1, "cycles": []},
                                  "duplication": {"exact": 0.1, "renamed": 0.2, "testsExact": 0.0},
                                  "linters": {"builds": True, "deadcode": None, "staticcheck": {"findings": 3}, "vet": 0, "gofmt": 0},
                                  "againstSpec": {"modules": 2, "found": 1, "missing": ["d"], "notAtTheirPath": [], "extraPackages": [], "forbiddenImports": [{"from": "b", "to": "c"}], "forbiddenImportsInTests": 0}}
        launch = lambda cost, note: {"wallSeconds": 600, "relativeCost": cost, "hiddenCases": {"planned": 5, "passed": 5}, "weekly": {"before": 1, "after": 2}, "subagents": 0,
                                     "note": note, "modelsAnswered": {"claude-haiku-5-5": 1}}
        part = {"score": 4, "good": ["clear"], "weak": ["long"], "files": []}
        answer = {"architecture": part, "naming": part, "cleanliness": part, "tests": part, "takingOver": {"wouldMaintain": "yes", "why": "fine", "changeFirst": ["split"]}}
        key = {"A": {"name": "secret-arm-one"}, "B": {"name": "secret-arm-two"}}
        text = side_by_side.page("A page", ["A", "B"], {"A": launch(2.0, "K on Haiku"), "B": launch(1.0, "the reference")}, {"A": figure("a", 10), "B": figure("b", 20)},
                                 {"A": {"answer": answer}, "B": None}, key, "B", "My note.")
        before, _, after = text.partition("## The key")
        self.assertIn("2.00 ×", before)
        self.assertIn("My note.", before)
        self.assertIn("No valid read.", before)
        self.assertIn("forbidden `b` to `c`", before)
        self.assertIn("not run", before)
        self.assertNotIn("Subagents", before)
        self.assertIn("Builds", before)
        for hidden in ("secret-arm", "K on Haiku", "claude-haiku", "Subagents"):
            self.assertNotIn(hidden, before)
            self.assertIn(hidden, after)
        self.assertNotIn("$", text)


if __name__ == "__main__":
    unittest.main()
