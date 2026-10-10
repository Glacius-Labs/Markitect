"""Focused regression tests for the standalone documentation link checker."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("check-docs.py")
SPEC = importlib.util.spec_from_file_location("check_docs", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
check_docs = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(check_docs)


class DocumentationCheckerTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.repo = Path(self.temp.name)
        (self.repo / "docs").mkdir()

    def tearDown(self) -> None:
        self.temp.cleanup()

    def write(self, relative: str, content: str = "") -> Path:
        path = self.repo / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
        return path

    def check(self, source: Path) -> list[str]:
        checker = check_docs.DocumentationChecker(self.repo)
        checker.check_file(source)
        return checker.errors

    def test_inline_angle_bare_balanced_and_reference_destinations(self) -> None:
        source = """[angle](<space%20name.md> "title")
[balanced](folder_(one).md)
[by reference][target]
[collapsed][]
[target]: target.md 'optional title'
[collapsed]: other.md
"""
        destinations = check_docs.extract_destinations(source)
        self.assertEqual(
            {dest for dest, _ in destinations},
            {"space%20name.md", "folder_(one).md", "target.md", "other.md"},
        )

    def test_code_comments_and_external_links_are_ignored(self) -> None:
        markdown = """`[inline](missing-inline.md)`

```md
[fenced](missing-fenced.md)
```

<!-- [comment](missing-comment.md) -->
[external](https://example.invalid/path)
"""
        self.assertEqual(check_docs.extract_destinations(markdown), [("https://example.invalid/path", 8)])
        source = self.write("docs/index.md", markdown)
        self.assertEqual(self.check(source), [])

    def test_fence_info_text_does_not_close_code_block(self) -> None:
        markdown = """```md
```text
[ignored](missing.md)
```
"""
        source = self.write("docs/index.md", markdown)
        self.assertEqual(self.check(source), [])

    def test_comment_opener_inside_inline_code_does_not_hide_real_link(self) -> None:
        source = self.write("docs/index.md", "`<!--` [real](missing.md) -->\n")
        errors = self.check(source)
        self.assertTrue(any("missing local target 'missing.md'" in error for error in errors), errors)

    def test_longer_backtick_run_inside_code_does_not_expose_link(self) -> None:
        source = self.write("docs/index.md", "`start `` [ignored](missing.md) end`\n")
        self.assertEqual(self.check(source), [])

    def test_reports_missing_target_and_anchor(self) -> None:
        source = self.write("docs/index.md", "[target](absent.md)\n[anchor](present.md#absent)\n")
        self.write("docs/present.md", "# Present\n")
        errors = self.check(source)
        self.assertTrue(any("missing local target 'absent.md'" in error for error in errors), errors)
        self.assertTrue(any("missing anchor #absent" in error for error in errors), errors)

    def test_percent_escaped_path_and_duplicate_unicode_heading_anchors(self) -> None:
        source = self.write("docs/index.md", "[space](space%20name.md#café--tea-1)\n")
        self.write("docs/space name.md", "# Café & Tea\n# Café & Tea\n")
        self.assertEqual(self.check(source), [])

    def test_heading_links_keep_inline_code_content(self) -> None:
        source = self.write("docs/index.md", "[command](target.md#run-project-check)\n")
        self.write("docs/target.md", "# Run `project check`\n")
        self.assertEqual(self.check(source), [])

    def test_heading_code_content_survives_longer_inner_tick_run(self) -> None:
        source = self.write("docs/index.md", "[command](target.md#run-fast-literal-command)\n")
        self.write("docs/target.md", "# Run `fast ``literal`` command`\n")
        self.assertEqual(self.check(source), [])

    def test_comment_opener_inside_heading_code_preserves_heading_content(self) -> None:
        source = self.write("docs/index.md", "[command](target.md#run----tool)\n")
        self.write("docs/target.md", "# Run `<!--` tool\n")
        self.assertEqual(self.check(source), [])

    def test_multiline_code_span_preserves_diagnostic_line_number(self) -> None:
        source = self.write(
            "docs/index.md",
            "`code span starts\nwith [ignored](fake.md)\nand ends`\n[real](missing.md)\n",
        )
        errors = self.check(source)
        self.assertEqual(len(errors), 1, errors)
        self.assertTrue(errors[0].startswith("docs/index.md:4:"), errors)

    def test_duplicate_heading_suffixes_do_not_collide_and_lower_is_preserved(self) -> None:
        anchors = check_docs.heading_anchors("# Title\n# Title\n# Title-1\n# Straße\n")
        self.assertTrue({"title", "title-1", "title-1-1", "straße"}.issubset(anchors), anchors)
        self.assertNotIn("strasse", anchors)

    def test_reference_shortcut_and_explicit_html_id(self) -> None:
        source = self.write(
            "docs/index.md",
            "[Page](target.md#Custom-ID)\n\n[target]: target.md\n",
        )
        self.write("docs/target.md", '<span id="Custom-ID"></span>\n')
        self.assertEqual(self.check(source), [])

    def test_data_id_does_not_create_an_explicit_anchor(self) -> None:
        source = self.write("docs/index.md", "[target](target.md#fake)\n")
        self.write("docs/target.md", '<span data-id="fake"></span>\n')
        errors = self.check(source)
        self.assertTrue(any("missing anchor #fake" in error for error in errors), errors)

    def test_id_text_inside_quoted_attribute_value_is_not_an_anchor(self) -> None:
        source = self.write("docs/index.md", "[target](target.md#fake)\n")
        self.write("docs/target.md", '<span title="id=fake"></span>\n')
        errors = self.check(source)
        self.assertTrue(any("missing anchor #fake" in error for error in errors), errors)

    def test_html_id_literal_inside_heading_code_is_not_an_anchor(self) -> None:
        source = self.write("docs/index.md", "[target](target.md#fake)\n")
        self.write("docs/target.md", '# Example `<span id="fake"></span>`\n')
        errors = self.check(source)
        self.assertTrue(any("missing anchor #fake" in error for error in errors), errors)

    def test_directory_targets_and_readme_anchors(self) -> None:
        source = self.write("docs/index.md", "[dir](section/)\n[section](section/#overview)\n")
        self.write("docs/section/README.md", "# Overview\n")
        self.assertEqual(self.check(source), [])

    def test_directory_fragment_requires_readme_or_index(self) -> None:
        source = self.write("docs/index.md", "[directory](empty/#missing)\n")
        (self.repo / "docs/empty").mkdir()
        errors = self.check(source)
        self.assertTrue(any("directory fragment target has no README.md or index.md" in error for error in errors), errors)

    def test_directory_fragment_rejects_readme_symlink_escape(self) -> None:
        source = self.write("docs/index.md", "[directory](section/#outside)\n")
        section = self.repo / "docs/section"
        section.mkdir()
        with tempfile.TemporaryDirectory() as outside_dir:
            outside_readme = Path(outside_dir) / "README.md"
            outside_readme.write_text("# Outside\n", encoding="utf-8")
            try:
                (section / "README.md").symlink_to(outside_readme)
            except (OSError, NotImplementedError) as exc:
                self.skipTest(f"file symlinks are unavailable: {exc}")
            errors = self.check(source)
        self.assertTrue(any("directory fragment document escapes repository" in error for error in errors), errors)

    def test_rejects_case_mismatch_and_repository_escape(self) -> None:
        source = self.write("docs/index.md", "[case](Target.md)\n[escape](../../outside.md)\n")
        self.write("docs/target.md", "# Target\n")
        errors = self.check(source)
        self.assertTrue(any("path case mismatch" in error for error in errors), errors)
        self.assertTrue(any("escapes repository" in error for error in errors), errors)

    def test_rejects_intermediate_symlink_escape(self) -> None:
        source = self.write("docs/index.md", "[outside](linked/secret.md)\n")
        with tempfile.TemporaryDirectory() as outside_dir:
            outside = Path(outside_dir)
            (outside / "secret.md").write_text("# Outside\n", encoding="utf-8")
            link = self.repo / "docs/linked"
            try:
                link.symlink_to(outside, target_is_directory=True)
            except (OSError, NotImplementedError) as exc:
                self.skipTest(f"directory symlinks are unavailable: {exc}")
            errors = self.check(source)
        self.assertTrue(any("escapes repository" in error for error in errors), errors)

    def test_rejects_malformed_percent_and_backslash_paths(self) -> None:
        source = self.write(
            "docs/index.md",
            "[percent](bad%2.md)\n[slash](folder\\file.md)\n[drive](C:/outside.md)\n",
        )
        errors = self.check(source)
        self.assertTrue(any("malformed percent escape" in error for error in errors), errors)
        self.assertTrue(any("backslash in local URL path" in error for error in errors), errors)
        self.assertTrue(any("absolute drive path is outside repository" in error for error in errors), errors)


if __name__ == "__main__":
    unittest.main()
