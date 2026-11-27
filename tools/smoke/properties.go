package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// propertyTypes drives the property types and the module cycles over the small
// project: a type with a reference property and a property of fixed values, an
// object that follows them and one that does not, a derived type that types a
// property again, show of the type, and two modules that depend on each other.
// It leaves the project as it found it, total documents.
func (s *smoke) propertyTypes(project string, total int) error {
	markitect := func(want int, args ...string) (string, error) { return s.exit(project, want, args...) }
	model := func(name string) string { return "docs/model/" + name }
	remove := func(names ...string) error {
		for _, name := range names {
			if err := os.RemoveAll(filepath.Join(project, filepath.FromSlash(model(name)))); err != nil {
				return err
			}
		}
		return nil
	}
	write := func(files map[string]string) error {
		for name, content := range files {
			if err := writeFile(project, model(name), content); err != nil {
				return err
			}
		}
		return nil
	}

	// A type that says what its properties are, and an object that follows it.
	if err := write(map[string]string{
		"domain/site.md":      "# Site\n\nA site people open.\n\n## Pages\nType: [Operation](operation.md)\n\nThe operations it offers.\n\n## State\nType: `draft`, `live`\n\nWhere it stands.\n",
		"domain/shop-site.md": "# Shop site\n\nType: [Site](site.md)\n\n## Pages\n[Borrow](borrow.md)\n\n## State\nlive\n",
	}); err != nil {
		return err
	}
	if out, err := markitect(0, "check"); err != nil {
		return err
	} else if out != fmt.Sprintf("0 errors in %d documents\n", total+2) {
		return fmt.Errorf("check with property types that are followed: %s", tail(out))
	}
	if out, err := markitect(0, "show", "domain/site"); err != nil {
		return err
	} else if err := requireContains(out, "Sections:\n  Pages: reference to domain/operation.md\n  State: one of `draft`, `live`\n", "Documents:\n  domain/shop-site.md\n"); err != nil {
		return err
	}
	if out, err := markitect(0, "show", "domain/site", "--json"); err != nil {
		return err
	} else if err := requireContains(out, `"references": [`, `"path": "domain/operation.md"`, `"values": [`, `"live"`); err != nil {
		return err
	}
	if out, err := markitect(0, "show", "domain/shop-site"); err != nil {
		return err
	} else if err := requireContains(out, "Sections:\n  Pages\n  State\n"); err != nil {
		return err
	}

	// An object that does not follow it, and a derived type that types a property
	// again, are found.
	if err := write(map[string]string{
		"domain/shop-site.md": "# Shop site\n\nType: [Site](site.md)\n\n## Pages\n[Borrow](borrow.md), [Member](member.md)\n\n## State\nwip\n",
		"domain/retyped.md":   "# Retyped\n\nExtends: [Site](site.md)\n\n## State\nType: `a`\n",
	}); err != nil {
		return err
	}
	if out, err := markitect(1, "check"); err != nil {
		return err
	} else if err := requireContains(out,
		"domain/retyped.md:6: property.retyped: property \"State\" is one of `draft`, `live` in domain/site.md, not one of `a`\n",
		"domain/shop-site.md:6: reference.type: section \"Pages\" refers to documents of domain/operation.md or a type that extends it, but \"domain/member.md\" is a document of core/concept.md\n",
		"domain/shop-site.md:9: property.value: the first line of section \"State\" is \"wip\", not one of `draft`, `live`\n",
		fmt.Sprintf("3 errors in %d documents\n", total+3)); err != nil {
		return err
	}
	if err := remove("domain/site.md", "domain/shop-site.md", "domain/retyped.md"); err != nil {
		return err
	}

	// Two modules that each have a type of the other's: a cycle, reported once,
	// with the link to turn round.
	if err := write(map[string]string{
		"left/README.md":  "# Left\n\n## Contents\n- [A](a.md)\n- [T](t.md)\n",
		"left/t.md":       "# T\n\nA thing.\n",
		"left/a.md":       "# A\n\nType: [T](../right/t.md)\n",
		"right/README.md": "# Right\n\n## Contents\n- [B](b.md)\n- [T](t.md)\n",
		"right/t.md":      "# T\n\nAnother thing.\n",
		"right/b.md":      "# B\n\nType: [T](../left/t.md)\n",
	}); err != nil {
		return err
	}
	if out, err := markitect(1, "check"); err != nil {
		return err
	} else if err := requireContains(out,
		"left/a.md:3: module.cycle: module cycle left -> right -> left: left/a.md:3 is of type right/t.md, right/b.md:3 is of type left/t.md\n",
		fmt.Sprintf("1 error in %d documents\n", total+6)); err != nil {
		return err
	}
	if err := remove("left", "right"); err != nil {
		return err
	}
	if out, err := markitect(0, "check"); err != nil {
		return err
	} else if out != fmt.Sprintf("0 errors in %d documents\n", total) {
		return fmt.Errorf("check after the property types are removed: %s", tail(out))
	}
	return nil
}
