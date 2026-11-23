package lang

import (
	"reflect"
	"testing"
)

// kind is a type of module t with a reference property Peer to its own objects.
func kind(extra string) string {
	return "# Kind\n\nA thing.\n\n## Peer\nType: [Kind](kind.md)\n\nThe things it is next to.\n" + extra
}

func TestModuleCycleThroughReferences(t *testing.T) {
	files := map[string]string{
		"types/kind.md": kind(""),
		"a/x.md":        "# X\n\nType: [Kind](../types/kind.md)\n\n## Peer\n[Y](../b/y.md)\n",
		"b/y.md":        "# Y\n\nType: [Kind](../types/kind.md)\n\n## Peer\n[X](../a/x.md)\n",
	}
	expect(t, checkFiles(t, files),
		"a/x.md:6: module.cycle: module cycle a -> b -> a: a/x.md:6 refers to b/y.md, b/y.md:6 refers to a/x.md")
	// One of the two references turned round: no cycle.
	files["b/y.md"] = "# Y\n\nType: [Kind](../types/kind.md)\n\n## Peer\nNobody yet.\n"
	expect(t, checkFiles(t, files))
}

func TestModuleCycleThroughATypeLineAndAReference(t *testing.T) {
	files := map[string]string{
		"b/thing.md": "# Thing\n\nA thing.\n\n## Peer\nType: [Thing](thing.md)\n\nWhom it knows.\n",
		"a/x.md":     "# X\n\nType: [Thing](../b/thing.md)\n\n## Peer\nNobody.\n",
		"b/y.md":     "# Y\n\nType: [Thing](thing.md)\n\n## Peer\n[X](../a/x.md)\n",
	}
	expect(t, checkFiles(t, files),
		"a/x.md:3: module.cycle: module cycle a -> b -> a: a/x.md:3 is of type b/thing.md, b/y.md:6 refers to a/x.md")
}

func TestModuleCycleThroughExtends(t *testing.T) {
	expect(t, checkFiles(t, map[string]string{
		"a/base.md":    "# Base\n\nA base.\n",
		"a/derived.md": "# Derived\n\nExtends: [Other](../b/other.md)\n\nA type.\n",
		"b/other.md":   "# Other\n\nExtends: [Base](../a/base.md)\n\nAnother.\n",
	}),
		"a/derived.md:3: module.cycle: module cycle a -> b -> a: a/derived.md:3 extends b/other.md, b/other.md:3 extends a/base.md")
}

func TestModuleCycleThroughAPropertyTypeLine(t *testing.T) {
	expect(t, checkFiles(t, map[string]string{
		"a/order.md":  "# Order\n\nAn order.\n\n## Lines\nType: [Line](../b/line.md)\n",
		"b/line.md":   "# Line\n\nA line.\n\n## Order\nType: [Order](../a/order.md)\n",
		"b/values.md": "# Values\n\nAnother.\n\n## State\nType: `a`, `b`\n",
	}),
		"a/order.md:6: module.cycle: module cycle a -> b -> a: a/order.md:6 declares a reference to b/line.md, b/line.md:6 declares a reference to a/order.md")
}

func TestModuleCycleBetweenAParentAndItsChild(t *testing.T) {
	files := map[string]string{
		"shop/kind.md":      "# Kind\n\nA kind.\n",
		"shop/fines/law.md": "# Law\n\nA law.\n",
		"shop/borrow.md":    "# Borrow\n\nType: [Law](fines/law.md)\n",
		"shop/fines/old.md": "# Old\n\nType: [Kind](../kind.md)\n",
	}
	expect(t, checkFiles(t, files),
		"shop/borrow.md:3: module.cycle: module cycle shop -> shop/fines -> shop: shop/borrow.md:3 is of type shop/fines/law.md, shop/fines/old.md:3 is of type shop/kind.md")
	// Only the child's way up: no cycle.
	files["shop/borrow.md"] = "# Borrow\n\nNo type.\n"
	expect(t, checkFiles(t, files))
	// A cycle through the spec folder.
	expect(t, checkFiles(t, map[string]string{
		"kind.md":  "# Kind\n\nA kind.\n",
		"x.md":     "# X\n\nType: [Law](a/law.md)\n",
		"a/law.md": "# Law\n\nA law.\n",
		"a/y.md":   "# Y\n\nType: [Kind](../kind.md)\n",
	}),
		"x.md:3: module.cycle: module cycle (spec folder) -> a -> (spec folder): x.md:3 is of type a/law.md, a/y.md:3 is of type kind.md")
}

func TestModuleCycleOfThreeModules(t *testing.T) {
	files := map[string]string{
		"a/t.md": "# T\n\nA.\n",
		"b/t.md": "# T\n\nB.\n",
		"c/t.md": "# T\n\nC.\n",
		"a/x.md": "# X\n\nType: [T](../b/t.md)\n",
		"b/y.md": "# Y\n\nType: [T](../c/t.md)\n",
		"c/z.md": "# Z\n\nType: [T](../a/t.md)\n",
	}
	expect(t, checkFiles(t, files),
		"a/x.md:3: module.cycle: module cycle a -> b -> c -> a: a/x.md:3 is of type b/t.md, b/y.md:3 is of type c/t.md, c/z.md:3 is of type a/t.md")
}

func TestTwoSeparateModuleCycles(t *testing.T) {
	files := map[string]string{
		"a/t.md": "# T\n\nA.\n",
		"b/t.md": "# T\n\nB.\n",
		"c/t.md": "# T\n\nC.\n",
		"d/t.md": "# T\n\nD.\n",
		"a/x.md": "# X\n\nType: [T](../b/t.md)\n",
		"b/y.md": "# Y\n\nType: [T](../a/t.md)\n",
		// c and d are one group, and a depends on them without being in it.
		"c/z.md": "# Z\n\nType: [T](../d/t.md)\n",
		"d/w.md": "# W\n\nType: [T](../c/t.md)\n",
		"a/v.md": "# V\n\nType: [T](../c/t.md)\n",
	}
	expect(t, checkFiles(t, files),
		"a/x.md:3: module.cycle: module cycle a -> b -> a: a/x.md:3 is of type b/t.md, b/y.md:3 is of type a/t.md",
		"c/z.md:3: module.cycle: module cycle c -> d -> c: c/z.md:3 is of type d/t.md, d/w.md:3 is of type c/t.md")
}

func TestOneReportForAGroupWithSeveralCycles(t *testing.T) {
	files := map[string]string{
		"a/t.md": "# T\n\nA.\n",
		"b/t.md": "# T\n\nB.\n",
		"c/t.md": "# T\n\nC.\n",
		"d/t.md": "# T\n\nD.\n",
		// a -> b -> c -> a is the shortest cycle through a; c <-> d and b -> d make
		// more of them, and d belongs to the group.
		"a/x.md": "# X\n\nType: [T](../b/t.md)\n",
		"b/y.md": "# Y\n\nType: [T](../c/t.md)\n\nA [d](../d/t.md).\n",
		"c/z.md": "# Z\n\nType: [T](../a/t.md)\n",
		"c/w.md": "# W\n\nType: [T](../d/t.md)\n",
		"d/v.md": "# V\n\nType: [T](../c/t.md)\n",
		"b/u.md": "# U\n\nType: [T](../d/t.md)\n",
	}
	expect(t, checkFiles(t, files),
		"a/x.md:3: module.cycle: module cycle a -> b -> c -> a: a/x.md:3 is of type b/t.md, b/y.md:3 is of type c/t.md, c/z.md:3 is of type a/t.md")
	// The first link of a dependency is the first by file and line.
	files["a/x.md"] = "# X\n\nType: [T](../c/t.md)\n"
	files["a/w.md"] = "# W\n\nType: [T](../b/t.md)\n"
	expect(t, checkFiles(t, files),
		"a/x.md:3: module.cycle: module cycle a -> c -> a: a/x.md:3 is of type c/t.md, c/z.md:3 is of type a/t.md")
}

func TestMentionsAndReadmesCloseNoCycle(t *testing.T) {
	files := map[string]string{
		"types/kind.md": kind(""),
		"b/y.md":        "# Y\n\nType: [Kind](../types/kind.md)\n\n## Peer\nNobody.\n",
		// a refers to b by links that are mentions: in a text property, before the
		// first section, in a section the type does not declare, and by a README.
		"a/x.md":      "# X\n\nType: [Kind](../types/kind.md)\n\nSee [Y](../b/y.md).\n\n## Peer\nNobody.\n\n## Remarks\n[Y](../b/y.md)\n",
		"a/README.md": "# A\n\nSee [Y](../b/y.md).\n\n## Contents\n- [X](x.md)\n",
		"b/README.md": "# B\n\nSee [X](../a/x.md).\n\n## Contents\n- [Y](y.md)\n",
	}
	// b depends on a through nothing but links of this kind, and a on b.
	files["b/y.md"] = "# Y\n\nType: [Kind](../types/kind.md)\n\nSee [X](../a/x.md).\n\n## Peer\nNobody.\n\n## Remarks\n[X](../a/x.md)\n"
	expect(t, checkFiles(t, files))
	// A text property of the type that is no reference: the link is a mention.
	files["types/kind.md"] = "# Kind\n\nA thing.\n\n## Peer\nWhom it knows.\n"
	files["a/x.md"] = "# X\n\nType: [Kind](../types/kind.md)\n\n## Peer\n[Y](../b/y.md)\n"
	files["b/y.md"] = "# Y\n\nType: [Kind](../types/kind.md)\n\n## Peer\n[X](../a/x.md)\n"
	expect(t, checkFiles(t, files))
}

func TestBrokenDependenciesInCycles(t *testing.T) {
	// A link that points nowhere is no dependency. A link to what the other module
	// does not show is a dependency, and an error of its own besides.
	files := map[string]string{
		"b/kind.md": "# Kind\n\nA kind.\n",
		"b/y.md":    "# Y\n\nA document.\n",
		"a/x.md":    "# X\n\nType: [Kind](../b/kind.md)\n",
		"b/z.md":    "# Z\n\nType: [Ghost](../a/ghost.md)\n",
	}
	expect(t, checkFiles(t, files), `b/z.md:3: type.unresolved: no document at "../a/ghost.md"`)
	files["a/internal.md"] = "# Internal\n\nA kind.\n"
	files["a/README.md"] = "# A\n\n## Contents\n- [X](x.md)\n"
	files["b/z.md"] = "# Z\n\nType: [Internal](../a/internal.md)\n"
	expect(t, checkFiles(t, files),
		"a/x.md:3: module.cycle: module cycle a -> b -> a: a/x.md:3 is of type b/kind.md, b/z.md:3 is of type a/internal.md",
		`b/z.md:3: link.internal: "a/internal.md" is internal to a: a/README.md does not list it under Contents`)
}

func TestModuleCycleIsDeterministic(t *testing.T) {
	files := withReadmes(map[string]string{
		"a/t.md": "# T\n\nA.\n",
		"b/t.md": "# T\n\nB.\n",
		"c/t.md": "# T\n\nC.\n",
		"d/t.md": "# T\n\nD.\n",
		"a/1.md": "# 1\n\nType: [T](../b/t.md)\n",
		"a/2.md": "# 2\n\nType: [T](../c/t.md)\n",
		"b/1.md": "# 1\n\nType: [T](../d/t.md)\n",
		"c/1.md": "# 1\n\nType: [T](../d/t.md)\n",
		"d/1.md": "# 1\n\nType: [T](../a/t.md)\n",
		"d/2.md": "# 2\n\nType: [T](../a/t.md)\n",
	})
	want := lines(t, specOf(t, files))
	for i := 0; i < 30; i++ {
		if got := lines(t, specOf(t, files)); !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d: got %v, want %v", i, got, want)
		}
	}
	// Of two cycles of the same length the first by path counts: through b.
	expect(t, want,
		"a/1.md:3: module.cycle: module cycle a -> b -> d -> a: a/1.md:3 is of type b/t.md, b/1.md:3 is of type d/t.md, d/1.md:3 is of type a/t.md")
}

func TestAMentionThatWouldCloseACycle(t *testing.T) {
	// a depends on b by a type line; b only mentions a, in a text property, in the
	// text before its sections and in a section the type does not declare.
	files := map[string]string{
		"b/t.md": "# T\n\nA thing.\n\n## Notes\nText.\n",
		"a/x.md": "# X\n\nType: [T](../b/t.md)\n\n## Notes\nText.\n",
		"b/y.md": "# Y\n\nType: [T](t.md)\n\nSee [X](../a/x.md).\n\n## Notes\n[X](../a/x.md)\n\n## Remarks\n[X](../a/x.md)\n",
	}
	expect(t, checkFiles(t, files))
	// The same link in a reference property closes it.
	files["b/t.md"] = "# T\n\nA thing.\n\n## Notes\nType: [T](t.md)\n"
	files["a/x.md"] = "# X\n\nType: [T](../b/t.md)\n\n## Notes\n[Y](../b/y.md)\n"
	files["b/y.md"] = "# Y\n\nType: [T](t.md)\n\n## Notes\n[X](../a/x.md)\n"
	expect(t, checkFiles(t, files),
		"a/x.md:3: module.cycle: module cycle a -> b -> a: a/x.md:3 is of type b/t.md, b/y.md:6 refers to a/x.md")
}
