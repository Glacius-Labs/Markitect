package bootstrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
	"go.yaml.in/yaml/v3"
)

// The standard Libraries under libraries/ must compile on the current Core
// and be usable by a project: an Operation with typed arguments, refusals and
// permissions, a project Kind implementing Api.Operation, the value Kinds, the
// Project anchor and Rules, and the architecture Rules.
func TestStandardLibrariesCompileForAProject(t *testing.T) {
	repo := testkit.NewRepo(t)
	repo.Write(".markitect/model.yaml", "formatVersion: 1\nid: library-consumer\nroot: model\ndependencies:\n  markitect-core: \"1.2.0\"\n  markitect-values: \"1.0.0\"\n  markitect-api: \"1.0.0\"\n  markitect-practice: \"1.0.0\"\n  markitect-architecture: \"1.0.0\"\n")
	repo.Write(".markitect/model-lock.yaml", "formatVersion: 1\nmodels: {}\n")
	repo.Write(".markitect/execution.yaml", "apiVersion: markitect.execution/v2\nownership:\n  - {path: ./, owner: shop}\n")
	repo.Write(".markitect/registry/shop/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts: []\n")
	repo.Write(".markitect/ignore.yaml", "apiVersion: markitect.ignore/v1\nfiles: []\n")
	for name, body := range map[string]string{
		"customer-id":       "imports:\n  - {package: markitect/values, as: Values}\nid: customer-id\nkind: Values.Scalar\nbody: The identifier of a customer.\nspec:\n  base: string\n",
		"order-status":      "imports:\n  - {package: markitect/values, as: Values}\nid: order-status\nkind: Values.Enumeration\nbody: The states of an order.\nspec:\n  members:\n    - {value: open}\n    - {value: shipped, meaning: Handed to the carrier.}\n",
		"cents":             "imports:\n  - {package: markitect/values, as: Values}\nid: cents\nkind: Values.Unit\nbody: Hundredths of the currency.\nspec:\n  symbol: ct\n",
		"amount":            "imports:\n  - {package: markitect/values, as: Values}\nid: amount\nkind: Values.Scalar\nbody: A non-negative amount of money.\nspec:\n  base: integer\n  lowest: 0\n  highest: 100000000\n  measuredIn: cents\n",
		"out-of-stock":      "imports:\n  - {package: markitect/api, as: Api}\nid: out-of-stock\nkind: Api.RefusalCode\nbody: Not enough stock.\nspec:\n  code: out_of_stock\n",
		"clerk":             "imports:\n  - {package: markitect/api, as: Api}\nid: clerk\nkind: Api.Role\nbody: Shop staff.\nspec: {}\n",
		"place-order":       "imports:\n  - {package: markitect/api, as: Api}\nid: place-order\nkind: Api.Operation\nbody: Places an order for an existing customer.\nspec:\n  wireName: placeOrder\n  mutation: true\n  arguments:\n    - {name: customerId, fieldType: customer-id}\n  refusals:\n    - {refusal: out-of-stock, when: \"the stock is lower than the quantity\"}\n  permissions:\n    - {role: clerk, scope: all}\n",
		"audited-operation": "imports:\n  - {package: markitect/core, as: Core}\n  - {package: markitect/api, as: Api}\nid: AuditedOperation\nkind: Core.Kind\nbody: An operation that also names its audit target.\nspec:\n  implements: [Api.Operation]\n  properties: [Api.wireName, Api.mutation, Api.arguments, Api.refusals, Api.results, Api.permissions, auditTarget]\n",
		"audit-target":      "imports:\n  - {package: markitect/core, as: Core}\nid: auditTarget\nkind: Core.Property\nbody: The record an audit entry names.\nspec:\n  valueType: string\n  minCount: 1\n  maxCount: 1\n",
		"cancel-order":      "id: cancel-order\nkind: AuditedOperation\nbody: Cancels an open order.\nspec:\n  wireName: cancelOrder\n  mutation: true\n  arguments:\n    - {name: orderId, fieldType: customer-id, refersTo: customer-id}\n  results:\n    - {name: status, fieldType: order-status}\n    - {name: refund, fieldType: amount, nullable: true}\n  permissions:\n    - {role: clerk, scope: all}\n  auditTarget: order\n",
		"shop":              "imports:\n  - {package: markitect/practice, as: Practice}\nid: shop\nkind: Practice.Project\nbody: The shop service.\nspec:\n  language: Go 1.27\n  moduleRoot: .\n  testCommand: go test ./...\n",
		"handler-tests":     "imports:\n  - {package: markitect/practice, as: Practice}\nid: handler-tests\nkind: Practice.TestObligation\nbody: Every operation has a test per refusal.\nspec:\n  appliesTo: [shop]\n",
		"security-duty":     "imports:\n  - {package: markitect/core, as: Core}\n  - {package: markitect/practice, as: Practice}\nid: SecurityObligation\nkind: Core.Kind\nbody: A duty about security.\nspec:\n  implements: [Practice.Rule]\n  properties: [Practice.appliesTo]\n",
		"authorization":     "id: authorization\nkind: SecurityObligation\nbody: Every call is checked against the operation's permissions.\nspec:\n  appliesTo: [shop]\n",
		"shop-interface":    "imports:\n  - {package: markitect/architecture, as: Architecture}\nid: shop-interface\nkind: Architecture.Interface\nbody: What other services may call.\nspec:\n  appliesTo: [.]\n  operations: [place-order, cancel-order]\n  types: [amount, order-status]\n",
		"shop-dependencies": "imports:\n  - {package: markitect/architecture, as: Architecture}\nid: shop-dependencies\nkind: Architecture.DependencyRule\nbody: The shop depends on nothing else.\nspec:\n  appliesTo: [shop]\n  mayDependOn: []\n",
		"auth-placement":    "imports:\n  - {package: markitect/architecture, as: Architecture}\nid: auth-placement\nkind: Architecture.Placement\nbody: Authorization runs before any handler.\nspec:\n  appliesTo: [shop]\n  concern: authorization\n  realizedIn: shop\n",
	} {
		repo.Write(".markitect/model/shop/"+name+".yaml", body)
	}
	repo.Commit("library consumer fixture")

	var dependencies []string
	for _, manifest := range []string{"../../testdata/core/model.yaml", "../../libraries/values/model.yaml", "../../libraries/api/model.yaml", "../../libraries/practice/model.yaml", "../../libraries/architecture/model.yaml"} {
		absolute, err := filepath.Abs(manifest)
		if err != nil {
			t.Fatal(err)
		}
		dependencies = append(dependencies, "--dependency", absolute)
	}

	// The first check reports each dependency's digest; the lock is empty.
	first := checkReport(t, repo.Dir, 1, dependencies)
	digests := map[string]string{}
	if err := collectDigests(first, digests); err != nil {
		t.Fatalf("unreadable check report: %v\n%s", err, first)
	}
	var lock strings.Builder
	lock.WriteString("formatVersion: 1\nmodels:\n")
	for _, model := range []struct{ id, version string }{{"markitect-core", "1.2.0"}, {"markitect-values", "1.0.0"}, {"markitect-api", "1.0.0"}, {"markitect-practice", "1.0.0"}, {"markitect-architecture", "1.0.0"}} {
		digest, found := digests[model.id]
		if !found {
			t.Fatalf("Library %s did not compile: %s", model.id, first)
		}
		fmt.Fprintf(&lock, "  %s:\n    version: %s\n    digest: %s\n", model.id, model.version, digest)
	}
	repo.Write(".markitect/model-lock.yaml", lock.String())
	repo.Commit("lock the Libraries")
	checkReport(t, repo.Dir, 0, dependencies)

	// The Library targets constrain: each broken use must fail with its finding.
	for _, broken := range []struct{ file, body, code string }{
		{"auth-placement", "imports:\n  - {package: markitect/architecture, as: Architecture}\nid: auth-placement\nkind: Architecture.Placement\nbody: A Project is not a Rule.\nspec:\n  appliesTo: [shop]\n  concern: shop\n  realizedIn: shop\n", "value.reference_kind"},
		{"shop-interface", "imports:\n  - {package: markitect/architecture, as: Architecture}\nid: shop-interface\nkind: Architecture.Interface\nbody: A Role is not an Operation.\nspec:\n  appliesTo: [.]\n  operations: [clerk]\n", "value.reference_kind"},
		{"security-duty", "imports:\n  - {package: markitect/core, as: Core}\n  - {package: markitect/practice, as: Practice}\nid: SecurityObligation\nkind: Core.Kind\nbody: A Rule without appliesTo.\nspec:\n  implements: [Practice.Rule]\n  properties: []\n", "contract.implements_missing"},
	} {
		path := ".markitect/model/shop/" + broken.file + ".yaml"
		original, err := os.ReadFile(filepath.Join(repo.Dir, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		repo.Write(path, broken.body)
		repo.Commit("break " + broken.file)
		if report := checkReport(t, repo.Dir, 1, dependencies); !strings.Contains(string(report), `"`+broken.code+`"`) {
			t.Fatalf("broken %s did not report %s:\n%s", broken.file, broken.code, report)
		}
		repo.Write(path, string(original))
		repo.Commit("restore " + broken.file)
	}

	// Each shipped Library lock must pin exactly its manifest's dependencies,
	// with their versions and current digests.
	manifests, err := filepath.Glob("../../libraries/*/model.yaml")
	if err != nil || len(manifests) == 0 {
		t.Fatalf("no Library manifests: %v", err)
	}
	for _, manifest := range manifests {
		var declared struct {
			Dependencies map[string]string `yaml:"dependencies"`
		}
		var locked struct {
			Models map[string]struct{ Version, Digest string } `yaml:"models"`
		}
		readYAML(t, manifest, &declared)
		readYAML(t, filepath.Join(filepath.Dir(manifest), "model-lock.yaml"), &locked)
		if len(locked.Models) != len(declared.Dependencies) {
			t.Fatalf("%s locks %d models for %d dependencies", manifest, len(locked.Models), len(declared.Dependencies))
		}
		for id, version := range declared.Dependencies {
			entry, found := locked.Models[id]
			if !found || entry.Version != version || entry.Digest != digests[id] {
				t.Fatalf("%s: lock entry for %s is %+v, want version %s and digest %s", manifest, id, entry, version, digests[id])
			}
		}
	}
}

func readYAML(t *testing.T, path string, into any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(data, into); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

func checkReport(t *testing.T, dir string, want int, dependencies []string) json.RawMessage {
	t.Helper()
	out := invoke(t, dir, want, append([]string{"check"}, dependencies...)...)
	return json.RawMessage(out)
}

// collectDigests records the canonical digest of every model object that
// names one, keyed by model id.
func collectDigests(report json.RawMessage, digests map[string]string) error {
	var tree any
	if err := json.Unmarshal(report, &tree); err != nil {
		return err
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
	return nil
}
