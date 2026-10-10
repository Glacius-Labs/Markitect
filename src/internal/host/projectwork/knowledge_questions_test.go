package projectwork

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"go.yaml.in/yaml/v3"
)

// KG-01: the knowledge questions answered against the shipped Shop example.
// A gap or partial row pins what main lacks, so it fails once main answers it
// and the row must be updated deliberately.
const (
	questionAnswered     = "answered"
	questionPartial      = "partial"
	questionGap          = "gap"
	questionOutsideModel = "outside-model"
)

const (
	shopOrders       = "commerce.sales.orders"
	shopReservations = "commerce.sales.inventory.reservations"
)

var (
	shopRootManager        = shopID("Manager", "", "shop")
	shopCommerceManager    = shopID("Manager", "commerce", "commerce")
	shopSalesManager       = shopID("Manager", "commerce.sales", "sales")
	shopOrdersManager      = shopID("Manager", shopOrders, "orders")
	shopInventoryManager   = shopID("Manager", "commerce.sales.inventory", "inventory")
	shopEngineeringManager = shopID("Manager", "engineering", "engineering")

	shopTransaction         = shopID("Statement", "commerce", "cancellation-transaction")
	shopIntegration         = shopID("Statement", "commerce.sales", "cancellation-integration")
	shopReleaseReservation  = shopID("Statement", shopReservations, "release-reservation")
	shopReservation         = shopID("Statement", shopReservations, "reservation")
	shopCancelBeforeShipped = shopID("Statement", shopOrders, "cancel-before-shipped")
	shopCancelOrder         = shopID("Statement", shopOrders, "cancel-order")
	shopOrder               = shopID("Statement", shopOrders, "order")
	shopRequiresReservation = shopID("Statement", shopOrders, "requires-active-reservation")

	shopOrderLifecycle       = shopID("Artifact", shopOrders, "order-lifecycle")
	shopReservationLifecycle = shopID("Artifact", shopReservations, "reservation-lifecycle")

	shopCancellationTests = shopID("Check", "engineering", "cancellation-tests")
)

type knowledgeQuestion struct {
	id       string
	question string
	status   string
	source   string
	run      func(t *testing.T, shop shopFixture)
}

type shopFixture struct {
	files   map[string]string
	project *Project
}

func TestShopKnowledgeQuestions(t *testing.T) {
	files := readShopExample(t)
	shop := shopFixture{files: files, project: loadShopSnapshot(t, files)}
	base := shop.project.Report
	// Impact routes everything when scope is unknown, which would make every
	// must-contain check below vacuous.
	if base.Status != "succeeded" || len(base.Findings) != 0 || len(base.Unknown) != 0 {
		t.Fatalf("shipped Shop report = status %q, findings %+v, unknown %v; want a clean baseline", base.Status, base.Findings, base.Unknown)
	}
	questions := []knowledgeQuestion{
		{"Q01", "Who owns a declared file and the definition that claims it?", questionAnswered,
			"Report.Files owner, artifacts, statements and checks; Statement.Owner", askShopFileOwnership},
		{"Q02", "Which consumers depend on a changed contract, and why?", questionPartial,
			"Impact routes consumers, their paths, checks and owners; it returns no reason per element", askShopContractConsumers},
		{"Q03", "What changes when the relation itself is removed?", questionAnswered,
			"Impact evaluates the union of base and candidate edges", askShopRemovedRelation},
		{"Q04", "Which decision supports this rule?", questionAnswered,
			"Report.Decisions by subject and the owning Manager's Context (DEC-022); the Shop declares none", askShopSupportingDecision},
		{"Q05", "Which accepted work led to an applied change?", questionOutsideModel,
			"Host run, plan and apply records, not the project model", askShopAppliedWork},
		{"Q06", "Which required artifact or check is missing?", questionAnswered,
			"Report finding coverage.required-artifact-missing and Report.Status; an undefined Check fails the model compile", askShopMissingArtifact},
		{"Q07", "Was a concept renamed or replaced?", questionGap,
			"Impact lists the old and new identities as separate changes; no continuity fact", askShopRename},
		{"Q08", "Why is a relationship visible to one scope but hidden from another?", questionPartial,
			"Context applies ownership, public and direct-use rules; it returns no visibility reason", askShopScopeVisibility},
		{"Q09", "Which supplied claims conflict?", questionOutsideModel,
			"Brownfield claims live in projectadoption, not the project model", askShopClaimConflicts},
		{"Q10", "Is evidence current, stale, historical, partial, or unknown?", questionOutsideModel,
			"Run and verify records, not the project model", askShopEvidenceState},
		{"Q11", "Why did Context include a contract?", questionAnswered,
			"Context.Contracts plus the own Statement.Uses and Requires that name them", askShopContractReason},
		{"Q12", "Does the model declare full coverage for this obligation?", questionAnswered,
			"Config.CoverageMode and Artifact realizes, paths and checks; no completeness certificate", askShopCoverageDeclaration},
	}
	counts := map[string]int{}
	for i, question := range questions {
		if want := fmt.Sprintf("Q%02d", i+1); question.id != want {
			t.Fatalf("question %d has id %s, want %s", i, question.id, want)
		}
		if question.question == "" || question.source == "" {
			t.Fatalf("%s must name its question and source", question.id)
		}
		switch question.status {
		case questionAnswered, questionPartial, questionGap, questionOutsideModel:
		default:
			t.Fatalf("%s has unknown status %q", question.id, question.status)
		}
		counts[question.status]++
		t.Run(question.id, func(t *testing.T) { question.run(t, shop) })
	}
	want := map[string]int{questionAnswered: 6, questionPartial: 2, questionGap: 1, questionOutsideModel: 3}
	if !reflect.DeepEqual(counts, want) {
		t.Fatalf("knowledge question statuses = %v, want %v; change a status only together with its row", counts, want)
	}
}

func askShopFileOwnership(t *testing.T, shop shopFixture) {
	report := shop.project.Report
	order := shopFile(t, report, "src/shop/orders/order.py")
	if order.Owner != shopOrdersManager || !order.Exists || order.Class != "project-file" {
		t.Fatalf("order.py = owner %s, exists %v, class %q; want orders through its longest selector src/shop/orders/", order.Owner, order.Exists, order.Class)
	}
	assertSameSet(t, "order.py artifacts", order.Artifacts, shopOrderLifecycle)
	assertSameSet(t, "order.py statements", order.Statements, shopOrder, shopCancelOrder, shopCancelBeforeShipped, shopRequiresReservation)
	assertSameSet(t, "order.py checks", order.Checks, shopCancellationTests)
	// sales and orders both own docs/cancellation.md exactly; the deeper namespace wins.
	document := shopFile(t, report, "docs/cancellation.md")
	if document.Owner != shopOrdersManager {
		t.Fatalf("docs/cancellation.md owner = %s, want %s", document.Owner, shopOrdersManager)
	}
	assertSameSet(t, "docs/cancellation.md artifacts", document.Artifacts, shopOrderLifecycle)
	if owner := shopArtifact(t, report, shopOrderLifecycle).Owner; owner != shopOrdersManager {
		t.Fatalf("order-lifecycle owner = %s, want %s", owner, shopOrdersManager)
	}
	if owner := shopStatement(t, report, shopCancelOrder).Owner; owner != shopOrdersManager {
		t.Fatalf("cancel-order owner = %s, want %s", owner, shopOrdersManager)
	}
	// The reservations namespace declares no Manager; the nearest namespace ancestor owns its statements.
	if owner := shopStatement(t, report, shopReleaseReservation).Owner; owner != shopInventoryManager {
		t.Fatalf("release-reservation owner = %s, want %s", owner, shopInventoryManager)
	}
}

func askShopContractConsumers(t *testing.T, shop shopFixture) {
	candidate := shop.candidate(t, func(files map[string]string) {
		replaceInShop(t, files, ".markitect/model/commerce/sales/inventory/reservations/release-reservation.yaml",
			"repeating the operation does not release quantity twice.", "a repeated release is a no-op and never releases quantity twice.", 1)
	})
	impact := projectmodel.Impact(shop.project.Report, candidate.Report)
	assertResolvedImpact(t, impact)
	assertSameSet(t, "changed definitions", impact.ChangedDefinitions, shopReleaseReservation)
	var consumers []string
	for _, statement := range shop.project.Report.Statements {
		if slices.Contains(statement.Uses, shopReleaseReservation) || slices.Contains(statement.Requires, shopReleaseReservation) {
			consumers = append(consumers, statement.ID)
		}
	}
	assertSameSet(t, "direct consumers of release-reservation", consumers, shopTransaction, shopIntegration, shopCancelOrder)
	assertContains(t, "affected statements", impact.AffectedStatements, shopReleaseReservation, shopTransaction, shopIntegration, shopCancelOrder)
	// cancellation-transaction has no realizing Artifact; the other consumers and the contract bring their declared paths.
	assertContains(t, "impact files", impact.Files, "src/shop/inventory/", "src/shop/orders/", "docs/cancellation.md",
		"src/shop/__init__.py", "src/shop/commerce/__init__.py", "src/shop/commerce/cancellation.py")
	assertContains(t, "impact checks", impact.Checks, shopCancellationTests)
	assertContains(t, "impact managers", impact.Managers, shopInventoryManager, shopCommerceManager, shopSalesManager, shopOrdersManager)
	// Gap: nothing names the edge that put a consumer into the impact.
	assertChangeImpactShape(t)
	for _, finding := range impact.Findings {
		if slices.Contains(consumers, finding.Subject) {
			t.Fatalf("impact finding %+v now explains a consumer; re-assess Q02", finding)
		}
	}
}

func askShopRemovedRelation(t *testing.T, shop shopFixture) {
	candidate := shop.candidate(t, func(files map[string]string) {
		replaceInShop(t, files, ".markitect/model/commerce/sales/orders/cancel-order.yaml",
			"      name: requires-active-reservation\n    - namespace: commerce.sales.inventory.reservations\n      name: release-reservation\n",
			"      name: requires-active-reservation\n", 1)
	})
	cancel := shopStatement(t, candidate.Report, shopCancelOrder)
	assertSameSet(t, "candidate cancel-order requires", cancel.Requires, shopCancelBeforeShipped, shopRequiresReservation)
	if !slices.Contains(cancel.Uses, shopReleaseReservation) {
		t.Fatalf("candidate dropped the uses edge too: %v", cancel.Uses)
	}
	impact := projectmodel.Impact(shop.project.Report, candidate.Report)
	assertResolvedImpact(t, impact)
	assertSameSet(t, "changed definitions", impact.ChangedDefinitions, shopCancelOrder)
	assertContains(t, "affected statements", impact.AffectedStatements, shopCancelOrder, shopReleaseReservation)
	// The removed requires edge still brings the former contract's unchanged realization and check.
	assertContains(t, "impact files", impact.Files, "src/shop/inventory/")
	assertContains(t, "impact checks", impact.Checks, shopCancellationTests)
	assertContains(t, "impact managers", impact.Managers, shopOrdersManager, shopInventoryManager)
}

func askShopSupportingDecision(t *testing.T, shop shopFixture) {
	if len(shop.project.Report.Decisions) != 0 {
		t.Fatalf("the Shop now declares Decisions %+v; re-assess Q04", shop.project.Report.Decisions)
	}
	// Candidate, not the shipped Shop: inventory records why release is idempotent.
	const decisionPath = ".markitect/model/commerce/sales/inventory/release-once.yaml"
	candidate := shop.candidate(t, func(files map[string]string) {
		files[decisionPath] = `apiVersion: project.markitect.example.org/v1alpha1
kind: Decision
metadata:
  name: release-once
  namespace: commerce.sales.inventory
purpose: Records why a reservation is released at most once.
spec:
  subject:
    namespace: commerce.sales.inventory.reservations
    name: release-reservation
  decision: Repeating a release never releases quantity twice.
  reason: A double release would overstate available stock.
  actor:
    namespace: commerce.sales.inventory
    name: inventory
`
		replaceInShop(t, files, ManifestPath, "  - .markitect/model/commerce/sales/inventory/manager.yaml\n",
			"  - .markitect/model/commerce/sales/inventory/manager.yaml\n  - "+decisionPath+"\n", 1)
	})
	decision := shopID("Decision", "commerce.sales.inventory", "release-once")
	var supporting []string
	for _, d := range candidate.Report.Decisions {
		if d.Subject == shopReleaseReservation {
			supporting = append(supporting, d.ID)
			if d.Owner != shopInventoryManager || d.Actor != shopInventoryManager || d.Reason == "" {
				t.Fatalf("decision = %+v, want owner and actor inventory with a reason", d)
			}
		}
	}
	assertSameSet(t, "decisions supporting release-reservation", supporting, decision)
	var visible []string
	for _, d := range shopContext(t, candidate.Report, shopInventoryManager).Decisions {
		visible = append(visible, d.ID)
	}
	assertSameSet(t, "inventory's decisions", visible, decision)
	if others := shopContext(t, candidate.Report, shopOrdersManager).Decisions; len(others) != 0 {
		t.Fatalf("orders sees inventory's decisions: %+v", others)
	}
	// Recording the decision routes its subject and owner, not the whole project.
	impact := projectmodel.Impact(shop.project.Report, candidate.Report)
	assertResolvedImpact(t, impact)
	assertSameSet(t, "changed definitions", impact.ChangedDefinitions, decision)
	assertContains(t, "affected statements", impact.AffectedStatements, shopReleaseReservation)
	assertContains(t, "impact managers", impact.Managers, shopInventoryManager)
}

func askShopAppliedWork(t *testing.T, _ shopFixture) {
	assertNoModelField(t, "Run", "Plan", "Apply", "Applied", "Accept", "Receipt")
}

func askShopMissingArtifact(t *testing.T, shop shopFixture) {
	candidate := shop.candidate(t, func(files map[string]string) {
		removed := 0
		for path := range files {
			if strings.HasPrefix(path, "src/shop/orders/") {
				delete(files, path)
				removed++
			}
		}
		if removed == 0 {
			t.Fatal("the Shop has no files under src/shop/orders/")
		}
	})
	report := candidate.Report
	if report.Status != "incomplete" {
		t.Fatalf("status without src/shop/orders/ = %q, want incomplete", report.Status)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings without src/shop/orders/ = %+v, want one missing order-lifecycle path", report.Findings)
	}
	finding := report.Findings[0]
	if finding.Code != "coverage.required-artifact-missing" || finding.Subject != shopOrderLifecycle || finding.Severity != "incomplete" || !strings.Contains(finding.Message, "src/shop/orders/") {
		t.Fatalf("finding = %+v, want coverage.required-artifact-missing for order-lifecycle at src/shop/orders/", finding)
	}
	// A declared but undefined Check is no finding: the model does not compile.
	withoutCheck := shop.edited(t, func(files map[string]string) {
		replaceInShop(t, files, ManifestPath, "  - .markitect/model/engineering/checks.yaml\n", "", 1)
		delete(files, ".markitect/model/engineering/checks.yaml")
	})
	if _, err := FromSnapshot(t.TempDir(), mapSnapshot(withoutCheck)); err == nil || !strings.Contains(err.Error(), shopCancellationTests) {
		t.Fatalf("load without cancellation-tests error = %v, want it to name the missing Check", err)
	}
}

func askShopRename(t *testing.T, shop shopFixture) {
	renamed := shopID("Statement", shopOrders, "cancel-unshipped-order")
	candidate := shop.candidate(t, func(files map[string]string) {
		const oldPath = ".markitect/model/commerce/sales/orders/cancel-order.yaml"
		const newPath = ".markitect/model/commerce/sales/orders/cancel-unshipped-order.yaml"
		files[newPath] = files[oldPath]
		delete(files, oldPath)
		replaceInShop(t, files, ManifestPath, "orders/cancel-order.yaml", "orders/cancel-unshipped-order.yaml", 1)
		for _, path := range []string{newPath, ".markitect/model/commerce/architecture.yaml", ".markitect/model/commerce/sales/integration.yaml",
			".markitect/model/commerce/sales/orders/artifacts.yaml", ".markitect/model/engineering/checks.yaml"} {
			replaceInShop(t, files, path, "name: cancel-order\n", "name: cancel-unshipped-order\n", 1)
		}
		for path, content := range files {
			if strings.HasPrefix(path, ModelRoot+"/") && strings.Contains(content, "name: cancel-order\n") {
				t.Fatalf("rename missed a reference in %s", path)
			}
		}
	})
	if candidate.Report.Status != "succeeded" || len(candidate.Report.Findings) != 0 {
		t.Fatalf("renamed Shop = status %q, findings %+v", candidate.Report.Status, candidate.Report.Findings)
	}
	before, after := shopStatement(t, shop.project.Report, shopCancelOrder), shopStatement(t, candidate.Report, renamed)
	if before.Description != after.Description || !slices.Equal(before.Uses, after.Uses) || !slices.Equal(before.Requires, after.Requires) {
		t.Fatalf("rename changed content: %+v versus %+v", before, after)
	}
	impact := projectmodel.Impact(shop.project.Report, candidate.Report)
	assertResolvedImpact(t, impact)
	// Every definition naming the concept changes; the old and new identities stand side by side.
	assertSameSet(t, "changed definitions", impact.ChangedDefinitions,
		shopCancelOrder, renamed, shopTransaction, shopIntegration, shopOrderLifecycle, shopCancellationTests)
	// Gap: no fact connects the two identities, so the honest answer is
	// "unknown, shown as removed plus added", even with identical content.
	assertChangeImpactShape(t)
	var codes []string
	for _, finding := range impact.Findings {
		codes = append(codes, finding.Code)
	}
	assertSameSet(t, "impact finding codes", uniqueSorted(codes), "impact.manager-routing", "impact.statement-change")
}

func askShopScopeVisibility(t *testing.T, shop shopFixture) {
	report := shop.project.Report
	orders := shopContext(t, report, shopOrdersManager)
	assertSameSet(t, "orders contracts", statementIDs(orders.Contracts), shopReservation, shopReleaseReservation)
	assertSameSet(t, "orders artifacts", artifactIDs(orders.Artifacts), shopOrderLifecycle)
	assertSameSet(t, "orders checks", checkIDs(orders.Checks))
	// inventory's statements use no orders statement, so none is visible there.
	assertSameSet(t, "inventory contracts", statementIDs(shopContext(t, report, shopInventoryManager).Contracts))
	// Own checks select contracts too: change-delivery uses cancellation-transaction,
	// and engineering's own Check exercises cancel-order and release-reservation.
	engineering := shopContext(t, report, shopEngineeringManager)
	assertSameSet(t, "engineering checks", checkIDs(engineering.Checks), shopCancellationTests)
	assertSameSet(t, "engineering contracts", statementIDs(engineering.Contracts), shopTransaction, shopCancelOrder, shopReleaseReservation)

	// Candidate, not the shipped Shop: order becomes private and commerce gets
	// instructions, because the Shop has neither a private statement nor a
	// child Manager with instructions.
	candidate := shop.candidate(t, func(files map[string]string) {
		replaceInShop(t, files, ".markitect/model/commerce/sales/orders/order.yaml", "  public: true\n", "  public: false\n", 1)
		replaceInShop(t, files, ".markitect/model/commerce/manager.yaml", "    - docs/\n", "    - docs/\n  instructions: Coordinate the cross-slice cancellation.\n", 1)
	})
	if candidate.Report.Status != "succeeded" || len(candidate.Report.Findings) != 0 {
		t.Fatalf("visibility candidate = status %q, findings %+v", candidate.Report.Status, candidate.Report.Findings)
	}
	assertSameSet(t, "cancel-order uses in the report", shopStatement(t, candidate.Report, shopCancelOrder).Uses, shopOrder, shopReservation, shopReleaseReservation)
	var seen []string
	for _, contract := range shopContext(t, candidate.Report, shopCommerceManager).Contracts {
		if contract.ID == shopCancelOrder {
			seen = contract.Uses
		}
	}
	assertSameSet(t, "cancel-order uses as commerce sees them", seen, shopReservation, shopReleaseReservation)
	if !slices.Contains(statementIDs(shopContext(t, candidate.Report, shopOrdersManager).Statements), shopOrder) {
		t.Fatal("orders lost its own private statement")
	}
	if shopManager(t, candidate.Report, shopCommerceManager).Instructions == "" {
		t.Fatal("candidate commerce instructions are missing from the report")
	}
	root := shopContext(t, candidate.Report, shopRootManager)
	assertSameSet(t, "root children", managerIDs(root.Children), shopCommerceManager, shopEngineeringManager)
	for _, child := range root.Children {
		if child.Instructions != "" {
			t.Fatalf("child %s carries instructions into its parent's context", child.ID)
		}
	}
	// Gap: Context says what is visible, never why something is hidden.
	assertFieldNames(t, projectmodel.ManagerContext{}, "Manager", "Statements", "Contracts", "Artifacts", "Checks", "Decisions", "Children", "Findings")
}

func askShopClaimConflicts(t *testing.T, _ shopFixture) {
	assertNoModelField(t, "Claim", "Conflict", "Adoption", "Brownfield")
}

func askShopEvidenceState(t *testing.T, _ shopFixture) {
	assertNoModelField(t, "Evidence", "Verif", "Stale", "Fresh", "Histor", "Run")
}

func askShopContractReason(t *testing.T, shop shopFixture) {
	orders := shopContext(t, shop.project.Report, shopOrdersManager)
	assertSameSet(t, "orders contracts", statementIDs(orders.Contracts), shopReservation, shopReleaseReservation)
	// One more lookup over the own statements' edges names the reason.
	reasons := map[string][]string{}
	for _, contract := range orders.Contracts {
		for _, own := range orders.Statements {
			if slices.Contains(own.Uses, contract.ID) {
				reasons[contract.ID] = append(reasons[contract.ID], own.Name+" uses")
			}
			if slices.Contains(own.Requires, contract.ID) {
				reasons[contract.ID] = append(reasons[contract.ID], own.Name+" requires")
			}
		}
	}
	assertSameSet(t, "release-reservation reasons", reasons[shopReleaseReservation], "cancel-order uses", "cancel-order requires")
	assertSameSet(t, "reservation reasons", reasons[shopReservation], "cancel-order uses", "requires-active-reservation uses")
	// No transitive closure. Every foreign statement orders reaches is also used
	// directly, so commerce shows it: cancel-order reaches order, reservation and
	// requires-active-reservation, which commerce's own statement never names.
	commerce := shopContext(t, shop.project.Report, shopCommerceManager)
	assertSameSet(t, "commerce contracts", statementIDs(commerce.Contracts), shopCancelOrder, shopCancelBeforeShipped, shopReleaseReservation)
}

func askShopCoverageDeclaration(t *testing.T, shop shopFixture) {
	report := shop.project.Report
	if shop.project.Config.CoverageMode != "full" {
		t.Fatalf("coverageMode = %q, want full", shop.project.Config.CoverageMode)
	}
	realizers := func(statement string) []string {
		var ids []string
		for _, artifact := range report.Artifacts {
			if slices.Contains(artifact.Realizes, statement) {
				ids = append(ids, artifact.ID)
			}
		}
		return ids
	}
	assertSameSet(t, "release-reservation realizers", realizers(shopReleaseReservation), shopReservationLifecycle)
	reservation := shopArtifact(t, report, shopReservationLifecycle)
	assertSameSet(t, "reservation-lifecycle paths", reservation.Paths, "src/shop/inventory/")
	assertSameSet(t, "reservation-lifecycle checks", reservation.Checks, shopCancellationTests)
	assertSameSet(t, "cancel-order realizers", realizers(shopCancelOrder), shopOrderLifecycle)
	lifecycle := shopArtifact(t, report, shopOrderLifecycle)
	assertSameSet(t, "order-lifecycle paths", lifecycle.Paths, "src/shop/orders/", "docs/cancellation.md")
	assertSameSet(t, "order-lifecycle checks", lifecycle.Checks, shopCancellationTests)
	if !reservation.Required || !lifecycle.Required {
		t.Fatalf("required = reservation-lifecycle %v, order-lifecycle %v; want both", reservation.Required, lifecycle.Required)
	}
	// A declaration, not a certificate: cancellation-transaction has no
	// realizing Artifact, yet the full-coverage report stays clean.
	assertSameSet(t, "cancellation-transaction realizers", realizers(shopTransaction))
	if report.Status != "succeeded" || len(report.Findings) != 0 {
		t.Fatalf("report = status %q, findings %+v", report.Status, report.Findings)
	}
	if limitation := shopCheck(t, report, shopCancellationTests).Limitation; !strings.Contains(limitation, "completeness of the project model") {
		t.Fatalf("cancellation-tests limitation = %q, want it to disclaim model completeness", limitation)
	}
}

func TestShopModelEdgesMatchYAMLLists(t *testing.T) {
	files := readShopExample(t)
	report := loadShopSnapshot(t, files).Report
	type reference struct {
		Namespace string `yaml:"namespace"`
		Name      string `yaml:"name"`
	}
	type definition struct {
		Kind     string    `yaml:"kind"`
		Metadata reference `yaml:"metadata"`
		Spec     struct {
			Uses     []reference `yaml:"uses"`
			Requires []reference `yaml:"requires"`
			Realizes []reference `yaml:"realizes"`
			Checks   []reference `yaml:"checks"`
		} `yaml:"spec"`
	}
	var declared []string
	for path, content := range files {
		if !strings.HasPrefix(path, ModelRoot+"/") || !strings.HasSuffix(path, ".yaml") {
			continue
		}
		var d definition
		if err := yaml.Unmarshal([]byte(content), &d); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		source := shopID(d.Kind, d.Metadata.Namespace, d.Metadata.Name)
		for _, list := range []struct {
			relation, target string
			refs             []reference
		}{{"uses", "Statement", d.Spec.Uses}, {"requires", "Statement", d.Spec.Requires}, {"realizes", "Statement", d.Spec.Realizes}, {"checks", "Check", d.Spec.Checks}} {
			for _, ref := range list.refs {
				declared = append(declared, d.Kind+"."+list.relation+" "+source+" -> "+shopID(list.target, ref.Namespace, ref.Name))
			}
		}
	}
	var projected []string
	add := func(label, source string, targets []string) {
		for _, target := range targets {
			projected = append(projected, label+" "+source+" -> "+target)
		}
	}
	for _, statement := range report.Statements {
		add("Statement.uses", statement.ID, statement.Uses)
		add("Statement.requires", statement.ID, statement.Requires)
	}
	for _, artifact := range report.Artifacts {
		add("Artifact.realizes", artifact.ID, artifact.Realizes)
		add("Artifact.checks", artifact.ID, artifact.Checks)
	}
	for _, check := range report.Checks {
		add("Check.uses", check.ID, check.Uses)
	}
	declaredCounts, projectedCounts := edgeCounts(declared), edgeCounts(projected)
	for _, label := range []string{"Statement.uses", "Statement.requires", "Artifact.realizes", "Artifact.checks", "Check.uses"} {
		if declaredCounts[label] == 0 {
			t.Fatalf("the Shop YAML declares no %s edge; the comparison would be vacuous", label)
		}
	}
	if !reflect.DeepEqual(declaredCounts, projectedCounts) {
		t.Fatalf("edge counts: YAML %v, report %v", declaredCounts, projectedCounts)
	}
	assertSameSet(t, "report edges", projected, declared...)
}

func edgeCounts(edges []string) map[string]int {
	counts := map[string]int{}
	for _, edge := range edges {
		counts[edge[:strings.Index(edge, " ")]]++
	}
	return counts
}

func readShopExample(t *testing.T) map[string]string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate the Shop example")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "examples", "project-world")
	files := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok := files[ManifestPath]; !ok {
		t.Fatalf("Shop example lacks %s", ManifestPath)
	}
	return files
}

func loadShopSnapshot(t *testing.T, files map[string]string) *Project {
	t.Helper()
	project, err := FromSnapshot(t.TempDir(), mapSnapshot(files))
	if err != nil {
		t.Fatalf("load Shop snapshot: %v", err)
	}
	return project
}

// candidate loads a copy of the Shop after edit changed its files.
func (shop shopFixture) candidate(t *testing.T, edit func(files map[string]string)) *Project {
	t.Helper()
	return loadShopSnapshot(t, shop.edited(t, edit))
}

func (shop shopFixture) edited(t *testing.T, edit func(files map[string]string)) map[string]string {
	t.Helper()
	files := make(map[string]string, len(shop.files))
	for path, content := range shop.files {
		files[path] = content
	}
	edit(files)
	return files
}

func replaceInShop(t *testing.T, files map[string]string, path, old, replacement string, count int) {
	t.Helper()
	content, ok := files[path]
	if !ok {
		t.Fatalf("Shop file %s is missing", path)
	}
	if got := strings.Count(content, old); got != count {
		t.Fatalf("Shop file %s contains %q %d times, want %d", path, old, got, count)
	}
	files[path] = strings.ReplaceAll(content, old, replacement)
}

func shopID(kind, namespace, name string) string {
	return core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: kind, Namespace: namespace, Name: name}.Key()
}

func shopContext(t *testing.T, report projectmodel.Report, managerID string) projectmodel.ManagerContext {
	t.Helper()
	context, err := projectmodel.Context(report, managerID)
	if err != nil {
		t.Fatalf("context for %s: %v", managerID, err)
	}
	return context
}

func shopManager(t *testing.T, report projectmodel.Report, id string) projectmodel.Manager {
	t.Helper()
	for _, manager := range report.Managers {
		if manager.ID == id {
			return manager
		}
	}
	t.Fatalf("Manager %s is missing from the report", id)
	return projectmodel.Manager{}
}

func shopStatement(t *testing.T, report projectmodel.Report, id string) projectmodel.Statement {
	t.Helper()
	for _, statement := range report.Statements {
		if statement.ID == id {
			return statement
		}
	}
	t.Fatalf("Statement %s is missing from the report", id)
	return projectmodel.Statement{}
}

func shopArtifact(t *testing.T, report projectmodel.Report, id string) projectmodel.Artifact {
	t.Helper()
	for _, artifact := range report.Artifacts {
		if artifact.ID == id {
			return artifact
		}
	}
	t.Fatalf("Artifact %s is missing from the report", id)
	return projectmodel.Artifact{}
}

func shopCheck(t *testing.T, report projectmodel.Report, id string) projectmodel.Check {
	t.Helper()
	for _, check := range report.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("Check %s is missing from the report", id)
	return projectmodel.Check{}
}

func shopFile(t *testing.T, report projectmodel.Report, path string) projectmodel.FileEntry {
	t.Helper()
	for _, entry := range report.Files {
		if entry.Path == path {
			return entry
		}
	}
	t.Fatalf("file %s is missing from the report", path)
	return projectmodel.FileEntry{}
}

func statementIDs(values []projectmodel.Statement) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids
}

func artifactIDs(values []projectmodel.Artifact) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids
}

func checkIDs(values []projectmodel.Check) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids
}

func managerIDs(values []projectmodel.Manager) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids
}

func assertSameSet(t *testing.T, label string, got []string, want ...string) {
	t.Helper()
	sortedGot, sortedWant := slices.Clone(got), slices.Clone(want)
	sort.Strings(sortedGot)
	sort.Strings(sortedWant)
	if !slices.Equal(sortedGot, sortedWant) {
		t.Fatalf("%s = %v, want %v", label, sortedGot, sortedWant)
	}
}

func assertContains(t *testing.T, label string, got []string, want ...string) {
	t.Helper()
	var missing []string
	for _, value := range want {
		if !slices.Contains(got, value) {
			missing = append(missing, value)
		}
	}
	if len(missing) != 0 {
		t.Fatalf("%s lacks %v; got %v", label, missing, got)
	}
}

func assertResolvedImpact(t *testing.T, impact projectmodel.ChangeImpact) {
	t.Helper()
	if len(impact.Unknown) != 0 {
		t.Fatalf("impact has unknown scope %v; its route-everything fallback would make must-contain checks vacuous", impact.Unknown)
	}
}

// assertChangeImpactShape pins that Impact carries no per-element reason.
func assertChangeImpactShape(t *testing.T) {
	t.Helper()
	assertFieldNames(t, projectmodel.ChangeImpact{}, "APIVersion", "BaseDigest", "CandidateDigest", "Digest",
		"ChangedDefinitions", "AffectedStatements", "Managers", "Files", "Checks", "Unknown", "Findings")
	assertFieldNames(t, projectmodel.Finding{}, "Code", "Subject", "Message", "Severity")
}

func assertFieldNames(t *testing.T, value any, want ...string) {
	t.Helper()
	typ := reflect.TypeOf(value)
	var got []string
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).IsExported() {
			got = append(got, typ.Field(i).Name)
		}
	}
	assertSameSet(t, typ.Name()+" fields (re-assess the row if this changed)", got, want...)
}

// assertNoModelField fails when any type reachable from Report, ChangeImpact or
// ManagerContext gains an exported field whose name mentions one of terms.
func assertNoModelField(t *testing.T, terms ...string) {
	t.Helper()
	seen := map[reflect.Type]bool{}
	var visit func(reflect.Type)
	visit = func(typ reflect.Type) {
		for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array || typ.Kind() == reflect.Map {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}
		seen[typ] = true
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}
			for _, term := range terms {
				if strings.Contains(field.Name, term) {
					t.Fatalf("%s.%s mentions %q; the model may now answer this question, re-assess the row", typ.Name(), field.Name, term)
				}
			}
			visit(field.Type)
		}
	}
	for _, value := range []any{projectmodel.Report{}, projectmodel.ChangeImpact{}, projectmodel.ManagerContext{}} {
		visit(reflect.TypeOf(value))
	}
	if len(seen) < 3 {
		t.Fatalf("visited only %d model types", len(seen))
	}
}
