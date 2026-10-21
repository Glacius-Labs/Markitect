package apply

import (
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

// valuesModel is a tiny model whose Definition widget uses every value kind,
// a final decisive sentence and body links through an import alias.
var valuesModel = map[string]string{
	"model.yaml":      "formatVersion: 1\nid: values-example\nroot: model\ndependencies:\n  markitect-core: \"1.0.0\"\n",
	"model-lock.yaml": "formatVersion: 1\nmodels:\n  markitect-core:\n    version: \"1.0.0\"\n    digest: \"sha256:562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1\"\n",
	"model/values/gadget.yaml": `imports:
  - package: markitect/core
    as: Core
id: Gadget
kind: Core.Kind
body: A configurable gadget.
spec:
  properties: [enabled, ratio, size, labels, mode, peer]
`,
	"model/values/enabled.yaml": "imports:\n  - package: markitect/core\n    as: Core\nid: enabled\nkind: Core.Property\nbody: Whether it runs.\nspec:\n  type: boolean\n  minCount: 1\n  maxCount: 1\n",
	"model/values/ratio.yaml":   "imports:\n  - package: markitect/core\n    as: Core\nid: ratio\nkind: Core.Property\nbody: Its ratio.\nspec:\n  type: number\n  minCount: 0\n  maxCount: 1\n",
	"model/values/size.yaml":    "imports:\n  - package: markitect/core\n    as: Core\nid: size\nkind: Core.Property\nbody: Its size.\nspec:\n  type: object\n  minCount: 0\n  maxCount: 1\n  properties: [width, height]\n",
	"model/values/width.yaml":   "imports:\n  - package: markitect/core\n    as: Core\nid: width\nkind: Core.Property\nbody: Width.\nspec:\n  type: integer\n  minCount: 0\n  maxCount: 1\n",
	"model/values/height.yaml":  "imports:\n  - package: markitect/core\n    as: Core\nid: height\nkind: Core.Property\nbody: Height.\nspec:\n  type: integer\n  minCount: 0\n  maxCount: 1\n",
	"model/values/labels.yaml":  "imports:\n  - package: markitect/core\n    as: Core\nid: labels\nkind: Core.Property\nbody: Its labels.\nspec:\n  type: string\n  minCount: 0\n  maxCount: -1\n",
	"model/values/mode.yaml":    "imports:\n  - package: markitect/core\n    as: Core\nid: mode\nkind: Core.Property\nbody: Its mode.\nspec:\n  type: enum\n  values: [fast, safe]\n  minCount: 0\n  maxCount: 1\n",
	"model/values/peer.yaml":    "imports:\n  - package: markitect/core\n    as: Core\nid: peer\nkind: Core.Property\nbody: Its peer.\nspec:\n  type: reference\n  target: Gadget\n  minCount: 0\n  maxCount: 1\n",
	"model/values/other.yaml":   "id: other\nkind: Gadget\nbody: Another gadget.\nspec:\n  enabled: true\n",
	"model/values/widget.yaml": `imports:
  - package: values
    as: V
id: widget
kind: Gadget
body: >-
  The widget works with [[V.other|its peer]] and [[other]]. It never runs
  above the ratio, and it stops when disabled.
spec:
  enabled: false
  ratio: 1.5
  size: {width: 3, height: 40}
  labels: [north, "south side"]
  mode: safe
  peer: other
`,
}

func sourceOf(t *testing.T, origin string, files map[string]string) model.SourceModel {
	t.Helper()
	src := model.SourceModel{Origin: origin, ManifestPath: "model.yaml"}
	dirs := map[string]bool{}
	var names []string
	for name := range files {
		names = append(names, name)
		for d := path.Dir(name); d != "."; d = path.Dir(d) {
			dirs[d] = true
		}
	}
	for d := range dirs {
		src.Entries = append(src.Entries, model.Entry{Path: d, Kind: model.Directory})
	}
	slices.Sort(names)
	for _, name := range names {
		src.Entries = append(src.Entries, model.Entry{Path: name, Kind: model.Regular, Bytes: []byte(files[name])})
	}
	return src
}

func coreSource(t *testing.T) model.SourceModel {
	t.Helper()
	dir := "../../docs/design/canonical-model/minimal-language/exchange-example/dependencies/core"
	files := map[string]string{}
	err := fs.WalkDir(os.DirFS(dir), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(dir + "/" + p)
		files[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return sourceOf(t, "opaque:core", files)
}

// Every value kind is stated exactly as the model gives it: booleans, exact
// decimals, objects, lists with their quoting, enums and references, and each
// body link with the identity its alias resolves to. The expected lines are
// written from the fixture above, not by the renderer; removing or changing
// any of them fails the same check.
func TestDefinitionsStateEveryValueKindExactly(t *testing.T) {
	checked := model.Check(model.Input{Root: sourceOf(t, "opaque:values", valuesModel), Dependencies: []model.SourceModel{coreSource(t)}})
	if checked.Status != model.Passed {
		t.Fatalf("fixture model: %s %+v", checked.Status, checked.Diagnostics)
	}
	widget, found := checkedDefinition(checked, model.Identity{Package: "values", ID: "widget"})
	if !found {
		t.Fatal("no widget")
	}
	var b strings.Builder
	writeDefinition(&b, organization.ContextDefinition{Definition: widget}, false, organization.Manager{Package: "reader"}, nil)
	text := b.String()
	want := []string{
		"values/widget (values/Gadget): widget (owned by values)\n",
		"It never runs above the ratio, and it stops when disabled.\n",
		"  links: [[V.other|its peer]] = values/other; [[other]] = values/other\n",
		"  enabled: false\n",
		"  ratio: 1.5\n",
		"  size: {height: 40, width: 3}\n",
		"  labels: [\"north\", \"south side\"]\n",
		"  mode: \"safe\"\n",
		"  peer: values/other\n",
	}
	lacks := func(text string) []string {
		var out []string
		for _, w := range want {
			if !strings.Contains(text, w) {
				out = append(out, w)
			}
		}
		return out
	}
	if gaps := lacks(text); len(gaps) > 0 {
		t.Fatalf("the Definition lost %q:\n%s", gaps, text)
	}
	for _, w := range want {
		if len(lacks(strings.Replace(text, w, "", 1))) == 0 {
			t.Errorf("the check misses a dropped %q", w)
		}
	}
	for _, tc := range []struct{ value, plain string }{
		{"15e-1", "1.5"}, {"25e2", "2500"}, {"-5e-3", "-0.005"}, {"0e0", "0"}, {"7e0", "7"}, {"1e-40", "1e-40"},
	} {
		coef, exp, _ := strings.Cut(tc.value, "e")
		if got := decimalText(model.Decimal{Coefficient: coef, Exponent: exp}); got != tc.plain {
			t.Errorf("decimal %s rendered %q, want %q", tc.value, got, tc.plain)
		}
	}
}
