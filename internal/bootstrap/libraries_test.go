package bootstrap

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// The standard Libraries under libraries/ must compile on the current Core
// and be usable by a project: an Operation with typed arguments, refusals and
// permissions, and a project Scalar as an argument type.
func TestStandardLibrariesCompileForAProject(t *testing.T) {
	repo := testkit.NewRepo(t)
	repo.Write(".markitect/model.yaml", "formatVersion: 1\nid: library-consumer\nroot: model\ndependencies:\n  markitect-core: \"1.2.0\"\n  markitect-values: \"1.0.0\"\n  markitect-api: \"1.0.0\"\n")
	repo.Write(".markitect/model-lock.yaml", "formatVersion: 1\nmodels: {}\n")
	repo.Write(".markitect/execution.yaml", "apiVersion: markitect.execution/v2\nownership:\n  - {path: ./, owner: shop}\n")
	repo.Write(".markitect/registry/shop/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts: []\n")
	repo.Write(".markitect/ignore.yaml", "apiVersion: markitect.ignore/v1\nfiles: []\n")
	repo.Write(".markitect/model/shop/customer-id.yaml", "imports:\n  - {package: markitect/values, as: Values}\nid: customer-id\nkind: Values.Scalar\nbody: The identifier of a customer.\nspec:\n  base: string\n")
	repo.Write(".markitect/model/shop/out-of-stock.yaml", "imports:\n  - {package: markitect/api, as: Api}\nid: out-of-stock\nkind: Api.RefusalCode\nbody: Not enough stock.\nspec:\n  code: out_of_stock\n")
	repo.Write(".markitect/model/shop/clerk.yaml", "imports:\n  - {package: markitect/api, as: Api}\nid: clerk\nkind: Api.Role\nbody: Shop staff.\nspec: {}\n")
	repo.Write(".markitect/model/shop/place-order.yaml", "imports:\n  - {package: markitect/api, as: Api}\nid: place-order\nkind: Api.Operation\nbody: Places an order for an existing customer.\nspec:\n  wireName: placeOrder\n  mutation: true\n  arguments:\n    - {name: customerId, fieldType: customer-id}\n  refusals:\n    - {refusal: out-of-stock, when: \"the stock is lower than the quantity\"}\n  permissions:\n    - {role: clerk, scope: all}\n")
	repo.Commit("library consumer fixture")

	var dependencies []string
	for _, manifest := range []string{"../../testdata/core/model.yaml", "../../libraries/values/model.yaml", "../../libraries/api/model.yaml"} {
		absolute, err := filepath.Abs(manifest)
		if err != nil {
			t.Fatal(err)
		}
		dependencies = append(dependencies, "--dependency", absolute)
	}

	// The first check reports each dependency's digest; the lock is empty.
	first := checkReport(t, repo.Dir, 1, dependencies)
	digests := map[string]string{}
	collectDigests(first, digests)
	var lock strings.Builder
	lock.WriteString("formatVersion: 1\nmodels:\n")
	for _, model := range []struct{ id, version string }{{"markitect-core", "1.2.0"}, {"markitect-values", "1.0.0"}, {"markitect-api", "1.0.0"}} {
		digest, found := digests[model.id]
		if !found {
			t.Fatalf("Library %s did not compile: %s", model.id, first)
		}
		fmt.Fprintf(&lock, "  %s:\n    version: %s\n    digest: %s\n", model.id, model.version, digest)
	}
	repo.Write(".markitect/model-lock.yaml", lock.String())
	repo.Commit("lock the Libraries")
	second := checkReport(t, repo.Dir, 0, dependencies)
	if !strings.Contains(string(second), `"status": "passed"`) {
		t.Fatalf("project using the Libraries did not pass: %s", second)
	}
}

func checkReport(t *testing.T, dir string, want int, dependencies []string) json.RawMessage {
	t.Helper()
	out := invoke(t, dir, want, append([]string{"check"}, dependencies...)...)
	return json.RawMessage(out)
}

// collectDigests records the canonical digest of every model object that
// names one, keyed by model id.
func collectDigests(report json.RawMessage, digests map[string]string) {
	var tree any
	if err := json.Unmarshal(report, &tree); err != nil {
		return
	}
	var walk func(any)
	walk = func(node any) {
		switch value := node.(type) {
		case map[string]any:
			id, _ := value["id"].(string)
			digest, _ := value["canonicalModelDigest"].(string)
			if id != "" && strings.HasPrefix(digest, "sha256:") {
				digests[id] = digest
			}
			for _, child := range value {
				walk(child)
			}
		case []any:
			for _, child := range value {
				walk(child)
			}
		}
	}
	walk(tree)
}
