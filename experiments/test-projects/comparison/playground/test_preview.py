"""Source refusal and packet-boundary checks; never runs a target application."""

import copy
import contextlib
import io
import json
import unittest
from unittest.mock import patch

import preview


REVISION = "a" * 40


class Sources:
    def __init__(self):
        self.blobs = {}
        self.files = []
        for name, target, arms, content in [
            ("requirements", "requirements.md", ["ordinary"], b"Visible requirements"),
            ("prompt", "implementation-prompt.md", ["ordinary"], b"Restore conformance"),
            ("model", ".markitect/model.yaml", ["markitect"], b"Current stage"),
            ("lock", ".markitect/model-lock.yaml", ["markitect"], b"Fixed dependencies"),
            ("app", "realization/app.py", ["ordinary", "markitect"], b"normalized guard\n"),
        ]:
            self.blobs[name] = content
            self.files.append({"source": name, "target": target, "arms": arms,
                               "revision": "packet", "sha256": preview.digest(content)})
        self.scenarios = {
            "formatVersion": 1,
            "sourceSets": {"baseline": {"files": self.files}},
            "scenarios": [{"id": "current", "sourceSet": "baseline",
                           "applicationTargets": ["realization/app.py"],
                           "profile": "matched", "writerTask": "Restore conformance"}],
        }
        self.factors = {"formatVersion": 1, "profiles": {
            "matched": {"actualBindings": {"model": None, "finiteGrant": None}}}}

    def read(self, revision, source):
        if source == preview.ROOT + "scenarios.yaml":
            return json.dumps(self.scenarios).encode()
        if source == preview.ROOT + "factors.yaml":
            return json.dumps(self.factors).encode()
        return self.blobs[source]

    def seed(self):
        self.scenarios["sourceSets"]["baseline"]["patches"] = [{
            "target": "realization/app.py", "arms": ["ordinary", "markitect"],
            "before": "normalized", "after": "raw-input",
            "beforeSha256": preview.digest(self.blobs["app"]),
            "afterSha256": preview.digest(b"raw-input guard\n"),
        }]


class PacketBoundaries(unittest.TestCase):
    def test_same_seed_bytes_both_arms_without_exposing_patch_in_inventory(self):
        sources = Sources()
        sources.seed()
        result = preview.inspect(sources, REVISION, "current")
        self.assertTrue(result["sourcePrepared"])
        self.assertFalse(result["runReady"])
        app = lambda arm: next(item for item in result["arms"][arm]
                               if item["target"] == "realization/app.py")
        self.assertEqual(app("ordinary"), app("markitect"))
        self.assertNotIn("raw-input", json.dumps(result))
        self.assertNotIn("beforeSha256", json.dumps(result))

    def test_corrupted_source_or_patch_refuses_instead_of_claiming_prepared(self):
        for failure in ("source", "before", "after"):
            with self.subTest(failure=failure):
                sources = Sources()
                sources.seed()
                if failure == "source":
                    sources.blobs["app"] = b"unexpected bytes"
                else:
                    key = failure + "Sha256"
                    sources.scenarios["sourceSets"]["baseline"]["patches"][0][key] = "0" * 64
                with self.assertRaises(ValueError):
                    preview.inspect(sources, REVISION, "current")

    def test_missing_hash_target_or_actual_prior_result_is_unbound(self):
        for failure in ("hash", "target", "prior-result"):
            with self.subTest(failure=failure):
                sources = Sources()
                if failure == "hash":
                    sources.files[0]["sha256"] = None
                elif failure == "target":
                    sources.files.pop()
                else:
                    sources.scenarios["sourceSets"]["baseline"]["sourceBlockers"] = [
                        "UNBOUND actual prior application result"]
                result = preview.inspect(sources, REVISION, "current")
                self.assertFalse(result["sourcePrepared"])
                self.assertEqual(result["sourceStatus"], "UNBOUND")
                self.assertTrue(result["sourceBlockers"])

    def test_current_snapshot_does_not_fetch_future_intent(self):
        sources = Sources()
        future = copy.deepcopy(sources.files[0])
        future["source"] = "future-not-present"
        sources.scenarios["sourceSets"]["later"] = {"files": [future]}
        self.assertTrue(preview.inspect(sources, REVISION, "current")["sourcePrepared"])

    def test_seed_applied_to_only_one_arm_refuses(self):
        sources = Sources()
        sources.seed()
        sources.scenarios["sourceSets"]["baseline"]["patches"][0]["arms"] = ["ordinary"]
        with self.assertRaises(ValueError):
            preview.inspect(sources, REVISION, "current")

    def test_duplicate_targets_and_cycles_refuse(self):
        sources = Sources()
        sources.files.append(copy.deepcopy(sources.files[0]))
        with self.assertRaises(ValueError):
            preview.inspect(sources, REVISION, "current")
        sources = Sources()
        sources.scenarios["sourceSets"]["baseline"]["base"] = "baseline"
        with self.assertRaises(ValueError):
            preview.inspect(sources, REVISION, "current")

    def test_cli_never_calls_unbound_run_ready(self):
        with patch.object(preview, "GitSources", return_value=Sources()):
            with contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(preview.main(["--revision", REVISION,
                                               "--scenario", "current", "--require-ready"]), 3)

    def test_unsafe_paths_and_symbolic_revisions_refuse_before_git(self):
        for value in ("../app.py", "/app.py", "a/./b", "a\\b", "a\nb"):
            with self.subTest(path=value), self.assertRaises(ValueError):
                preview.path(value)
        with patch.object(preview.subprocess, "run") as git:
            with self.assertRaises(ValueError):
                preview.GitSources(".").read("main", "app.py")
            git.assert_not_called()

    def test_symlink_blob_mode_is_refused(self):
        commit = type("Response", (), {"stdout": b"commit\n"})()
        response = type("Response", (), {"stdout": b"120000 blob abc\tapp.py\0"})()
        with patch.object(preview.subprocess, "run", side_effect=[commit, response]):
            with self.assertRaises(ValueError):
                preview.GitSources(".").read(REVISION, "app.py")

    def test_full_sha_that_is_not_a_commit_refuses(self):
        response = type("Response", (), {"stdout": b"tag\n"})()
        with patch.object(preview.subprocess, "run", return_value=response):
            with self.assertRaises(ValueError):
                preview.GitSources(".").read(REVISION, "app.py")


if __name__ == "__main__":
    unittest.main()
