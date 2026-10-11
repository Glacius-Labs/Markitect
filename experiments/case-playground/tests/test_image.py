"""Image pins, the image inventory and `host image-key` (image.py), without Docker."""

import contextlib
import io
import json
import os
import re
import subprocess
import tempfile
import unittest
from datetime import datetime, timezone
from pathlib import Path
from unittest import mock

from playground import host, image

PLAYGROUND = Path(__file__).resolve().parents[1]
DOCKERFILE = PLAYGROUND / "container" / "Dockerfile"
README = PLAYGROUND / "README.md"
NPM = ("@anthropic-ai/claude-code@2.1.296", "@openai/codex@0.162.0", "@openai/codex-linux-x64@0.162.0-linux-x64")
DPKG = ("bash 5.2.15-2+b8 amd64", "git 1:2.39.5-0+deb12u2 amd64", "python3 3.11.2-1+b1 amd64")
OTHER_DIGEST = "sha256:" + "ab" * 32


def sample_inventory(pinned: dict, *, npm=NPM, dpkg=DPKG, node: str = "v22.23.3") -> str:
    """A valid inventory for the pins `pinned` (as image.pins returns them)."""
    lines = [image.HEADER, f"base {pinned['base']}", f"snapshot {pinned['snapshot']}", f"node {node}"]
    lines += [f"dpkg {entry}" for entry in dpkg] + [f"npm {entry}" for entry in npm]
    return "\n".join(lines) + "\n"


def inventory_script() -> str:
    """The Python program of the Dockerfile's last step (its heredoc)."""
    text = DOCKERFILE.read_text(encoding="utf-8")
    return text.split("<<'PY'\n", 1)[1].split("\nPY\n", 1)[0] + "\n"


class DockerfileTests(unittest.TestCase):
    def test_the_dockerfile_pins_the_base_by_digest_and_the_packages_by_a_snapshot(self):
        pinned = image.pins(DOCKERFILE)
        self.assertRegex(pinned["base"], r"^node:22-bookworm-slim@sha256:[0-9a-f]{64}$")
        self.assertRegex(pinned["snapshot"], r"^\d{8}T\d{6}Z$")
        moment = datetime.strptime(pinned["snapshot"], "%Y%m%dT%H%M%SZ").replace(tzinfo=timezone.utc)
        self.assertEqual(pinned["sourceDateEpoch"], int(moment.timestamp()))
        self.assertEqual(pinned["dockerfileSha256"], image.sha256(DOCKERFILE.read_bytes()))
        text = DOCKERFILE.read_text(encoding="utf-8")
        self.assertIn("\nFROM ${BASE_IMAGE}\nARG BASE_IMAGE\n", text)  # the last step reads it from its environment
        for needle in ("http://snapshot.debian.org/archive/debian/${DEBIAN_SNAPSHOT}",
                       "http://snapshot.debian.org/archive/debian-security/${DEBIAN_SNAPSHOT}",
                       "Suites: bookworm bookworm-updates", "Suites: bookworm-security",
                       "Acquire::Check-Valid-Until \"false\";", "Acquire::Retries \"5\";",
                       "for attempt in 1 2 3; do", "rm -f /etc/apt/sources.list",
                       'npm install -g "@openai/codex@${CODEX_VERSION}"',
                       'npm install -g "@anthropic-ai/claude-code@${CLAUDE_VERSION}"'):
            self.assertIn(needle, text)
        self.assertNotIn("deb.debian.org", text)

    def test_the_last_step_writes_the_inventory_without_anything_docker_would_expand(self):
        text = DOCKERFILE.read_text(encoding="utf-8")
        last = text.rsplit("\nRUN ", 1)[1]
        self.assertTrue(last.startswith("python3 -I -B - <<'PY'\n"), last[:80])
        self.assertEqual(last.split("\nPY\n", 1)[1].strip(), "")  # nothing after it
        script = inventory_script()
        self.assertNotIn("${", script)
        self.assertIn(image.INVENTORY_IN_IMAGE, script)
        self.assertIn(f'"{image.HEADER}"', script)
        compile(script, "inventory", "exec")

    def test_the_inventory_step_lists_both_cli_trees_and_the_installed_packages(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp) / "node_modules"

            def package(folder: Path, name: str, version: str | None) -> None:
                folder.mkdir(parents=True)
                data = {"name": name} if version is None else {"name": name, "version": version}
                (folder / "package.json").write_text(json.dumps(data), encoding="utf-8")
            codex = root / "@openai" / "codex"
            package(codex, "@openai/codex", "0.162.0")
            package(codex / "node_modules" / "@openai" / "codex-linux-x64", "@openai/codex", "0.162.0-linux-x64")
            package(codex / "node_modules" / "left-pad", "left-pad", "1.3.0")
            package(codex / "node_modules" / "left-pad" / "node_modules" / "tiny", "tiny", "2.0.0")
            package(codex / "node_modules" / "broken", "broken", None)
            (codex / "node_modules" / ".bin").mkdir()
            (codex / "node_modules" / ".package-lock.json").write_text("{}", encoding="utf-8")
            claude = root / "@anthropic-ai" / "claude-code"
            package(claude, "@anthropic-ai/claude-code", "2.1.296")
            package(claude / "node_modules" / "left-pad", "left-pad", "1.3.0")
            package(root / "npm", "npm", "10.9.0")  # not a CLI tree: never listed
            target = Path(temp) / "inventory.txt"
            seen = []

            def run(cmd, **kwargs):
                seen.append(list(cmd))
                answers = {"npm": f"{root}\n", "node": "v22.23.3\n",
                           "dpkg-query": "installed python3 3.11.2-1+b1 amd64\nconfig-files old 1.0 amd64\n"
                                         "installed bash 5.2.15-2+b8 amd64\n"}
                return subprocess.CompletedProcess(cmd, 0, stdout=answers[cmd[0]], stderr="")
            script = inventory_script().replace(image.INVENTORY_IN_IMAGE, target.as_posix())
            pinned = image.pins(DOCKERFILE)
            with mock.patch.object(subprocess, "run", run), \
                    mock.patch.dict(os.environ, {"BASE_IMAGE": pinned["base"], "DEBIAN_SNAPSHOT": pinned["snapshot"]}):
                exec(compile(script, "inventory", "exec"), {"__name__": "inventory"})
            text = target.read_text(encoding="utf-8")
        dpkg_call = next(call for call in seen if call[0] == "dpkg-query")
        self.assertEqual(dpkg_call[-1], "${db:Status-Status} ${Package} ${Version} ${Architecture}\n")
        self.assertEqual(text.splitlines(), [
            image.HEADER, f"base {pinned['base']}", f"snapshot {pinned['snapshot']}", "node v22.23.3",
            "dpkg bash 5.2.15-2+b8 amd64", "dpkg python3 3.11.2-1+b1 amd64",
            "npm @anthropic-ai/claude-code@2.1.296", "npm @openai/codex-linux-x64@0.162.0-linux-x64",
            "npm @openai/codex@0.162.0", "npm broken@", "npm left-pad@1.3.0", "npm tiny@2.0.0"])
        with self.assertRaisesRegex(image.ImageError, "1 entry of the CLI dependency trees is not an exact version: "
                                                      "broken@$"):
            image.validate(text, pinned)  # a package without a version fails the build
        image.validate(text.replace("npm broken@\n", ""), pinned)

    def dockerfile(self, folder: Path, text: str) -> Path:
        path = folder / "Dockerfile"
        path.write_text(text, encoding="utf-8", newline="\n")
        return path

    def test_a_dockerfile_without_its_pins_is_refused(self):
        real = DOCKERFILE.read_text(encoding="utf-8")
        base = image.pins(DOCKERFILE)["base"]
        snapshot = image.pins(DOCKERFILE)["snapshot"]
        broken = {
            "not pinned by digest": real.replace(f"ARG BASE_IMAGE={base}", "ARG BASE_IMAGE=node:22-bookworm-slim"),
            "not pinned by digest (": real.replace(base, base[:-1]),
            "has no `ARG BASE_IMAGE": real.replace(f"ARG BASE_IMAGE={base}\n", ""),
            "exactly one FROM line": real.replace("FROM ${BASE_IMAGE}", "FROM node:22-bookworm-slim"),
            "exactly one FROM line,": real.replace("FROM ${BASE_IMAGE}\n", "FROM ${BASE_IMAGE}\nFROM ${BASE_IMAGE}\n"),
            "must come before": real.replace(f"ARG BASE_IMAGE={base}\nFROM ${{BASE_IMAGE}}\n",
                                             f"FROM ${{BASE_IMAGE}}\nARG BASE_IMAGE={base}\n"),
            "sets BASE_IMAGE more than once": real.replace(f"ARG BASE_IMAGE={base}\n",
                                                           f"ARG BASE_IMAGE={base}\nARG BASE_IMAGE={base}\n"),
            "has no `ARG DEBIAN_SNAPSHOT": real.replace(f"ARG DEBIAN_SNAPSHOT={snapshot}\n", ""),
            "(found '2026-10-10')": real.replace(f"ARG DEBIAN_SNAPSHOT={snapshot}", "ARG DEBIAN_SNAPSHOT=2026-10-10"),
            "is not a time": real.replace(f"ARG DEBIAN_SNAPSHOT={snapshot}", "ARG DEBIAN_SNAPSHOT=20261310T000000Z"),
            "does not take its Debian packages from": real.replace("snapshot.debian.org/archive/debian-security/",
                                                                   "deb.debian.org/debian-security/"),
        }
        with tempfile.TemporaryDirectory() as temp:
            for fragment, text in broken.items():
                with self.subTest(fragment=fragment):
                    self.assertNotEqual(text, real)
                    with self.assertRaises(image.ImageError) as caught:
                        image.pins(self.dockerfile(Path(temp), text))
                    self.assertIn(fragment, str(caught.exception))
            with self.assertRaisesRegex(image.ImageError, "cannot read"):
                image.pins(Path(temp) / "missing" / "Dockerfile")


class InventoryTests(unittest.TestCase):
    pinned = {"base": "node:22-bookworm-slim@" + OTHER_DIGEST, "snapshot": "20261010T203349Z"}

    def test_a_valid_inventory_parses(self):
        parsed = image.parse(sample_inventory(self.pinned))
        self.assertEqual((parsed["base"], parsed["snapshot"], parsed["node"]),
                         (self.pinned["base"], "20261010T203349Z", "v22.23.3"))
        self.assertEqual(parsed["dpkg"], list(DPKG))
        self.assertIn(("@openai/codex-linux-x64", "0.162.0-linux-x64"), parsed["npm"])
        self.assertEqual(image.check(sample_inventory(self.pinned), self.pinned, None), parsed)

    def test_anything_else_is_not_an_inventory(self):
        good = sample_inventory(self.pinned)
        cases = {
            "the first line is not": good.replace(image.HEADER, "inventory 2"),
            "unknown entry 'pip'": good + "pip requests==2.0\n",
            "a second base entry": good + f"base {self.pinned['base']}\n",
            "`dpkg` takes name, version and architecture": good + "dpkg bash 5.2\n",
            "`npm` takes <name>@<version>": good + "npm left-pad\n",
            "`npm` takes <name>@<version>:": good + "npm @1.0.0\n",
            "`node` takes one value": good.replace("node v22.23.3", "node v22 v23"),
            "no node entry": good.replace("node v22.23.3\n", ""),
            "no npm entry": "".join(line + "\n" for line in good.splitlines() if not line.startswith("npm ")),
        }
        for fragment, text in cases.items():
            with self.subTest(fragment=fragment), self.assertRaises(image.ImageError) as caught:
                image.parse(text)
            self.assertIn(fragment, str(caught.exception))

    def test_every_npm_entry_must_be_an_exact_version(self):
        for version in ("^1.2.3", "~1.2.3", "latest", "", "1.2", "1.2.x", "*", "git+https://github.com/a/b.git",
                        "file:../x", "1.0.0-", "01.2.3", ">=1.0.0"):
            with self.subTest(version=version):
                text = sample_inventory(self.pinned, npm=(*NPM, f"left-pad@{version}"))
                with self.assertRaises(image.ImageError) as caught:
                    image.check(text, self.pinned, None)
                self.assertIn("not an exact version", str(caught.exception))
                self.assertIn(f"left-pad@{version}", str(caught.exception))
        for version in ("1.2.3", "0.162.0-linux-x64", "1.0.0-beta.1", "1.0.0+build.5", "10.20.30"):
            with self.subTest(version=version):
                image.check(sample_inventory(self.pinned, npm=(*NPM, f"left-pad@{version}")), self.pinned, None)

    def test_the_inventory_must_name_the_dockerfiles_pins(self):
        other = {**self.pinned, "snapshot": "20261011T000000Z"}
        with self.assertRaisesRegex(image.ImageError, "its snapshot is 20261011T000000Z, the Dockerfile pins "
                                                      "20261010T203349Z"):
            image.check(sample_inventory(other), self.pinned, None)
        with self.assertRaisesRegex(image.ImageError, "its base is"):
            image.check(sample_inventory({**self.pinned, "base": "node:22@" + "sha256:" + "cd" * 32}),
                        self.pinned, None)

    def test_the_built_inventory_must_equal_the_committed_one(self):
        built = sample_inventory(self.pinned)
        image.check(built, self.pinned, built)
        image.check(built, self.pinned, built.replace("\n", "\r\n") + "\n\n")  # line endings and blank lines
        changed = built.replace("python3 3.11.2-1+b1", "python3 3.11.2-1+b2")
        with self.assertRaises(image.ImageError) as caught:
            image.check(changed, self.pinned, built)
        message = str(caught.exception)
        self.assertIn("differs from the committed inventory in 2 entries", message)
        self.assertIn("- dpkg python3 3.11.2-1+b1 amd64", message)
        self.assertIn("+ dpkg python3 3.11.2-1+b2 amd64", message)
        extra = sample_inventory(self.pinned, npm=(*NPM, "left-pad@1.3.0"))
        with self.assertRaisesRegex(image.ImageError, r"in 1 entry: \+ npm left-pad@1\.3\.0$"):
            image.check(extra, self.pinned, built)
        many = sample_inventory(self.pinned, npm=(*NPM, *(f"pkg{n}@1.0.{n}" for n in range(30))))
        with self.assertRaisesRegex(image.ImageError, r"\(and 10 more\)$"):
            image.check(many, self.pinned, built)
        self.assertEqual(image.differences(built, "\n".join(reversed(built.splitlines()))),
                         ["the same entries in another order or repeated"])

    def test_a_committed_inventory_that_is_itself_wrong_is_reported(self):
        built = sample_inventory(self.pinned)
        for committed, fragment in ((sample_inventory(self.pinned, npm=(*NPM, "left-pad@^1.0.0")), "not an exact"),
                                    (sample_inventory({**self.pinned, "snapshot": "20250101T000000Z"}), "snapshot"),
                                    ("garbage\n", "not an image inventory")):
            with self.subTest(fragment=fragment), self.assertRaises(image.ImageError) as caught:
                image.check(built, self.pinned, committed)
            self.assertTrue(str(caught.exception).startswith("the committed inventory: "), caught.exception)
            self.assertIn(fragment, str(caught.exception))


class KeyTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name) / "root"
        self.container = self.root / "container"
        (self.container / "inventory").mkdir(parents=True)
        self.dockerfile = self.container / "Dockerfile"
        self.dockerfile.write_bytes(DOCKERFILE.read_bytes())
        self.pinned = image.pins(self.dockerfile)
        self.inventory = self.container / "inventory" / image.inventory_name("0.162.0", "2.1.296")
        self.inventory.write_text(sample_inventory(self.pinned), encoding="utf-8", newline="\n")

    def key(self) -> str:
        return image.key(self.container)[0]

    def edit(self, old: str, new: str) -> None:
        text = self.dockerfile.read_text(encoding="utf-8")
        self.assertIn(old, text)
        self.dockerfile.write_text(text.replace(old, new), encoding="utf-8", newline="\n")

    def test_the_key_is_stable(self):
        first, files = image.key(self.container)
        self.assertRegex(first, r"^playground-image-[0-9a-f]{64}$")
        self.assertEqual(files, [self.inventory])
        os.utime(self.dockerfile, (1, 1))
        os.utime(self.inventory, (1, 1))
        self.assertEqual(self.key(), first)  # only content counts, never times

    def test_the_key_changes_with_every_input(self):
        seen = {self.key()}

        def changed(label: str) -> None:
            key = self.key()
            self.assertNotIn(key, seen, label)
            seen.add(key)
        self.edit(self.pinned["base"].split("@")[1], OTHER_DIGEST)
        changed("base digest")
        self.edit(self.pinned["snapshot"], "20261011T000000Z")
        changed("snapshot")
        self.edit("# One fresh container per run.", "# One fresh container per run, always.")
        changed("Dockerfile")
        self.inventory.write_text(self.inventory.read_text(encoding="utf-8") + "npm extra@1.0.0\n", encoding="utf-8")
        changed("inventory content")
        self.inventory.rename(self.inventory.with_name(image.inventory_name("0.163.0", "2.1.296")))
        changed("inventory name")
        (self.container / "inventory" / image.inventory_name("0.164.0", "2.1.296")).write_text("x", encoding="utf-8")
        changed("another inventory")
        for path in (self.container / "inventory").iterdir():
            path.unlink()
        key, files = image.key(self.container)
        self.assertEqual(files, [])
        self.assertNotIn(key, seen, "no inventory")

    def test_an_unpinned_dockerfile_has_no_key(self):
        self.edit(self.pinned["base"], "node:22-bookworm-slim")
        with self.assertRaisesRegex(image.ImageError, "not pinned by digest"):
            image.key(self.container)

    def image_key_command(self) -> tuple[int, str, str]:
        out, err = io.StringIO(), io.StringIO()
        with mock.patch.object(host, "ROOT", self.root), contextlib.redirect_stdout(out), \
                contextlib.redirect_stderr(err):
            code = host.main(["image-key"])
        return code, out.getvalue(), err.getvalue()

    def test_host_image_key_prints_the_key(self):
        code, stdout, stderr = self.image_key_command()
        self.assertEqual((code, stdout, stderr), (0, self.key() + "\n", ""))
        self.inventory.unlink()
        code, stdout, stderr = self.image_key_command()
        self.assertEqual((code, stdout), (0, self.key() + "\n"))
        self.assertIn("warning: no committed image inventory", stderr)
        self.edit(self.pinned["base"], "node:22-bookworm-slim")
        code, stdout, stderr = self.image_key_command()
        self.assertEqual((code, stdout), (2, ""))
        self.assertIn("not pinned by digest", stderr)
        self.assertIn("Updating the image pins", stderr)


class ReadmeTests(unittest.TestCase):
    def test_the_readme_says_how_to_move_the_pins(self):
        text = README.read_text(encoding="utf-8")
        self.assertIn("\n### Updating the image pins\n", text)
        section = text.split("\n### Updating the image pins\n", 1)[1].split("\n#", 1)[0]
        for needle in ("docker buildx imagetools inspect node:22-bookworm-slim", "snapshot.debian.org",
                       "`ARG BASE_IMAGE=", "`ARG DEBIAN_SNAPSHOT=", "SOURCE_DATE_EPOCH", "image-inventory.txt",
                       "container/inventory/codex-<v>-claude-<v>.txt", "python3 -m playground host image-key",
                       "python3 tests/smoke_docker.py --keep", "Acquire::Retries", "exit 11"):
            self.assertIn(needle, section)
        digest = image.pins(DOCKERFILE)["base"].split("@", 1)[1]
        self.assertNotIn(digest, text)  # the Dockerfile is the one place for the pins
        self.assertTrue(re.search(r"\| `playground/image\.py` \|", text))


if __name__ == "__main__":
    unittest.main()
