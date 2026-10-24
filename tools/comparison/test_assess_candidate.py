"""Census, export and seed classification on real Git commits; no engine."""

import json
from pathlib import Path
import subprocess
import tempfile
import unittest

import assess_candidate


def commit_all(repo, message):
    subprocess.run(["git", "-C", str(repo), "add", "-A"], check=True)
    subprocess.run(["git", "-C", str(repo), "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", message], check=True)
    return subprocess.run(["git", "-C", str(repo), "rev-parse", "HEAD"], check=True, capture_output=True, text=True).stdout.strip()


def write(repo, path, data):
    target = repo / path
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_bytes(data)


class CensusTests(unittest.TestCase):
    def test_changes_are_classified_and_exports_exclude_arm_markers(self):
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp) / "repo"
            subprocess.run(["git", "init", "-q", str(repo)], check=True)
            write(repo, "requirements.md", b"limit 5\n")
            write(repo, ".markitect/model/lending/loan-policy.yaml", b"maxOpenLoans: 5\n")
            write(repo, ".markitect/registry/registry.yaml", b"artifacts: []\n")
            write(repo, "realization/go.mod", b"module library\n")
            write(repo, "realization/lending/policy.go", b"package lending\nconst Max = 5\n")
            start = commit_all(repo, "start")
            write(repo, "realization/lending/policy.go", b"package lending\nconst Max = 3\n")
            write(repo, "realization/README.md", b"limit 3\n")
            write(repo, ".markitect/registry/registry.yaml", b"artifacts: [x]\n")
            write(repo, "realization/library", b"\x7fELF binary")
            write(repo, "requirements.md", b"edited by the writer\n")
            end = commit_all(repo, "end")
            census = assess_candidate.census(repo, start, end)
            paths = {k: [e["path"] for e in v] for k, v in census.items()}
            self.assertEqual(sorted(paths["comparable"]), ["realization/README.md", "realization/lending/policy.go"])
            self.assertEqual(paths["registry"], [".markitect/registry/registry.yaml"])
            self.assertEqual(paths["build_residue"], ["realization/library"])
            self.assertEqual(paths["protected"], ["requirements.md"])
            policy = next(e for e in census["comparable"] if e["path"].endswith("policy.go"))
            self.assertEqual((policy["status"], policy["added"], policy["removed"]), ("M", "1", "1"))

            blind = Path(tmp) / "blind"
            assess_candidate.export(repo, end, blind, (".markitect", "requirements.md"))
            names = sorted(p.relative_to(blind).as_posix() for p in blind.rglob("*") if p.is_file())
            self.assertEqual(names, ["realization/README.md", "realization/go.mod", "realization/lending/policy.go", "realization/library"])
            self.assertEqual((blind / "realization/lending/policy.go").read_bytes(), b"package lending\nconst Max = 3\n")

    def test_scope_patterns_and_seed_lines(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write(root, "a.go", b'package a\nimport "net/http"\nvar t = time.Now()\n')
            write(root, "a_test.go", b"package a\nvar u = time.Now()\n")
            found = assess_candidate.scope_findings(root)
            self.assertEqual(found, {"machine_clock": ["a.go"], "network": ["a.go"]})
            seeds = root / "seed.json"
            seeds.write_text(json.dumps([{"path": "a.go", "line": "var t = time.Now()"}, {"path": "a.go", "line": "gone"}]), encoding="utf-8")
            result = assess_candidate.seed_check(root, seeds)
            self.assertEqual(result["seeded_lines"], 2)
            self.assertEqual([r["line"] for r in result["remaining"]], ["var t = time.Now()"])


if __name__ == "__main__":
    unittest.main()
