package projectrun

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

type freshReviewFixture struct {
	root          string
	host          Host
	store         *runStore
	dir           string
	base          *Project
	plan          PlanRecord
	run           RunReport
	initial       candidateData
	inventory     candidateData
	orders        candidateData
	merged        candidateData
	inventoryTask ManagerTask
	ordersTask    ManagerTask
}

func newFreshReviewFixture(t *testing.T) freshReviewFixture {
	t.Helper()
	root := makeProjectRunFixture(t)
	host := projectworkHost()
	baseRevision := identityHead(t, root)
	base, err := host.Load(root, baseRevision)
	if err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(RunsPath)), 0o700); err != nil {
		t.Fatal(err)
	}
	const runID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	dir, err := store.createRun(runID)
	if err != nil {
		t.Fatal(err)
	}
	fixture := freshReviewFixture{
		root: root, host: host, store: store, dir: dir, base: base,
		plan: PlanRecord{BaseRevision: baseRevision, InitialCandidateID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		run:  RunReport{ID: runID},
	}
	rootTask := ManagerTask{ManagerID: e2eManagerID("", "project-owner"), Goal: "Reconcile the project deliveries."}
	fixture.inventoryTask = ManagerTask{
		ID: "inventory-task", ManagerID: e2eManagerID("inventory", "inventory"), ParentTask: rootTask.ManagerID,
		Goal: "Implement the inventory artifact.", Artifacts: []string{"inventory-code"}, WrittenPaths: []string{"src/inventory/implementation.txt"},
	}
	fixture.ordersTask = ManagerTask{
		ID: "orders-task", ManagerID: e2eManagerID("orders", "orders"), ParentTask: rootTask.ManagerID,
		Goal: "Implement the orders artifact.", Artifacts: []string{"orders-code"}, WrittenPaths: []string{"src/orders/implementation.txt"},
	}
	fixture.plan.Managers = []ManagerTask{rootTask, fixture.inventoryTask, fixture.ordersTask}
	fixture.run.Tasks = append([]ManagerTask(nil), fixture.plan.Managers...)
	fixture.initial = candidateData{ID: fixture.plan.InitialCandidateID, Files: map[string]File{}}
	fixture.inventory = candidateData{
		ID: "cccccccccccccccccccccccccccccccc", Parents: []string{fixture.initial.ID},
		Files: map[string]File{"src/inventory/implementation.txt": {Path: "src/inventory/implementation.txt", Mode: "0644", Content: []byte("inventory implementation v2\n")}},
	}
	fixture.orders = candidateData{
		ID: "dddddddddddddddddddddddddddddddd", Parents: []string{fixture.initial.ID},
		Files: map[string]File{"src/orders/implementation.txt": {Path: "src/orders/implementation.txt", Mode: "0644", Content: []byte("orders implementation v2\n")}},
	}
	fixture.merged = candidateData{
		ID: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", Parents: []string{fixture.inventory.ID, fixture.orders.ID},
		Files: map[string]File{
			"src/inventory/implementation.txt": fixture.inventory.Files["src/inventory/implementation.txt"],
			"src/orders/implementation.txt":    fixture.orders.Files["src/orders/implementation.txt"],
		},
	}
	for _, candidate := range []candidateData{fixture.initial, fixture.inventory, fixture.orders, fixture.merged} {
		if err := store.writeCandidate(dir, candidate); err != nil {
			t.Fatal(err)
		}
	}
	fixture.initial, err = store.readCandidate(dir, fixture.initial.ID)
	if err != nil {
		t.Fatal(err)
	}
	fixture.inventory, err = store.readCandidate(dir, fixture.inventory.ID)
	if err != nil {
		t.Fatal(err)
	}
	fixture.orders, err = store.readCandidate(dir, fixture.orders.ID)
	if err != nil {
		t.Fatal(err)
	}
	fixture.merged, err = store.readCandidate(dir, fixture.merged.ID)
	if err != nil {
		t.Fatal(err)
	}
	return fixture
}

func (fixture *freshReviewFixture) addPassedReview(t *testing.T, manager ManagerTask, candidate candidateData) {
	t.Helper()
	project, err := projectForCandidate(fixture.host, fixture.root, fixture.base.Snapshot, candidate)
	if err != nil {
		t.Fatal(err)
	}
	state, err := reviewCandidateStateFor(fixture.host, fixture.root, fixture.plan, fixture.run, candidate)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := reviewScopeDigest(fixture.plan, project, manager, "work", fixture.run, state)
	if err != nil {
		t.Fatal(err)
	}
	fixture.run.Reviews = append(fixture.run.Reviews, ReviewRecord{
		ManagerID: manager.ManagerID, Phase: "work", Round: 1, CandidateID: candidate.ID,
		CandidateDigest: candidate.Digest, ScopeDigest: scope, Outcome: "pass",
	})
}

func TestFreshPassedReviewUsesItsOwnParentAfterSiblingMerge(t *testing.T) {
	fixture := newFreshReviewFixture(t)
	fixture.addPassedReview(t, fixture.inventoryTask, fixture.inventory)
	fixture.addPassedReview(t, fixture.ordersTask, fixture.orders)
	finalProject, err := projectForCandidate(fixture.host, fixture.root, fixture.base.Snapshot, fixture.merged)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range []ManagerTask{fixture.inventoryTask, fixture.ordersTask} {
		review, state, found, err := freshPassedReviewForScope(fixture.host, fixture.root, fixture.store, fixture.dir,
			fixture.plan, fixture.run, finalProject, fixture.merged, task, "work")
		if err != nil || !found || review.ManagerID != task.ManagerID || state.Candidate.ID != fixture.merged.ID {
			t.Fatalf("unchanged %s review did not survive sibling integration: review=%+v found=%t err=%v", task.ManagerID, review, found, err)
		}
	}
	corruptLedger := fixture.run
	corruptLedger.Reviews = append([]ReviewRecord(nil), fixture.run.Reviews...)
	corruptLedger.Reviews[0].CandidateDigest = "sha256:wrong"
	if _, _, _, err := freshPassedReviewForScope(fixture.host, fixture.root, fixture.store, fixture.dir,
		fixture.plan, corruptLedger, finalProject, fixture.merged, fixture.inventoryTask, "work"); err == nil {
		t.Fatal("freshness accepted a passed review whose retained candidate digest was changed")
	}

	changedInventory := fixture.merged
	changedInventory.ID = "ffffffffffffffffffffffffffffffff"
	changedInventory.Files = map[string]File{
		"src/inventory/implementation.txt": {Path: "src/inventory/implementation.txt", Mode: "0644", Content: []byte("inventory implementation v3\n")},
		"src/orders/implementation.txt":    fixture.merged.Files["src/orders/implementation.txt"],
	}
	if err := fixture.store.writeCandidate(fixture.dir, changedInventory); err != nil {
		t.Fatal(err)
	}
	changedInventory, err = fixture.store.readCandidate(fixture.dir, changedInventory.ID)
	if err != nil {
		t.Fatal(err)
	}
	changedProject, err := projectForCandidate(fixture.host, fixture.root, fixture.base.Snapshot, changedInventory)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, found, err := freshPassedReviewForScope(fixture.host, fixture.root, fixture.store, fixture.dir,
		fixture.plan, fixture.run, changedProject, changedInventory, fixture.inventoryTask, "work"); err != nil || found {
		t.Fatalf("changed owned bytes reused a prior pass: found=%t err=%v", found, err)
	}
	changedMode := changedInventory
	changedMode.ID = "11111111111111111111111111111111"
	changedMode.Files = map[string]File{
		"src/inventory/implementation.txt": {Path: "src/inventory/implementation.txt", Mode: "0755", Content: []byte("inventory implementation v2\n")},
		"src/orders/implementation.txt":    fixture.merged.Files["src/orders/implementation.txt"],
	}
	if err := fixture.store.writeCandidate(fixture.dir, changedMode); err != nil {
		t.Fatal(err)
	}
	changedMode, err = fixture.store.readCandidate(fixture.dir, changedMode.ID)
	if err != nil {
		t.Fatal(err)
	}
	modeProject, err := projectForCandidate(fixture.host, fixture.root, fixture.base.Snapshot, changedMode)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, found, err := freshPassedReviewForScope(fixture.host, fixture.root, fixture.store, fixture.dir,
		fixture.plan, fixture.run, modeProject, changedMode, fixture.inventoryTask, "work"); err != nil || found {
		t.Fatalf("changed owned mode reused a prior pass: found=%t err=%v", found, err)
	}
	changedTask := fixture.inventoryTask
	changedTask.Goal = "Implement the inventory artifact and its declared checks."
	if _, _, found, err := freshPassedReviewForScope(fixture.host, fixture.root, fixture.store, fixture.dir,
		fixture.plan, fixture.run, finalProject, fixture.merged, changedTask, "work"); err != nil || found {
		t.Fatalf("changed task contract reused a prior pass: found=%t err=%v", found, err)
	}
}

func TestFreshPassedReviewBindsDeletionPreimageAndSource(t *testing.T) {
	fixture := newFreshReviewFixture(t)
	const path = "src/orders/implementation.txt"
	prior, present := fixture.base.Snapshot.Files[path]
	if !present {
		t.Fatal("fixture source file is missing")
	}
	deleted := candidateData{
		ID: "22222222222222222222222222222222", Parents: []string{fixture.initial.ID},
		Files: map[string]File{path: {Path: path, Mode: fixture.base.Snapshot.Modes[path], Content: append([]byte(nil), prior...), Delete: true}},
	}
	if err := fixture.store.writeCandidate(fixture.dir, deleted); err != nil {
		t.Fatal(err)
	}
	deleted, err := fixture.store.readCandidate(fixture.dir, deleted.ID)
	if err != nil {
		t.Fatal(err)
	}
	deleteTask := fixture.ordersTask
	deleteTask.WrittenPaths = []string{path}
	fixture.addPassedReview(t, deleteTask, deleted)
	deleteProject, err := projectForCandidate(fixture.host, fixture.root, fixture.base.Snapshot, deleted)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, found, err := freshPassedReviewForScope(fixture.host, fixture.root, fixture.store, fixture.dir,
		fixture.plan, fixture.run, deleteProject, deleted, deleteTask, "work"); err != nil || !found {
		t.Fatalf("unchanged source-grounded deletion did not retain its review: found=%t err=%v", found, err)
	}

	for name, mutation := range map[string]struct {
		id     string
		mutate func(*candidateData)
	}{
		"preimage": {id: "33333333333333333333333333333333", mutate: func(candidate *candidateData) {
			candidate.Files[path] = File{Path: path, Mode: fixture.base.Snapshot.Modes[path], Content: []byte("other prior bytes"), Delete: true}
		}},
		"mode": {id: "55555555555555555555555555555555", mutate: func(candidate *candidateData) {
			candidate.Files[path] = File{Path: path, Mode: "0755", Content: append([]byte(nil), prior...), Delete: true}
		}},
	} {
		t.Run(name, func(t *testing.T) {
			changed := deleted
			changed.ID = mutation.id
			changed.Files = map[string]File{path: deleted.Files[path]}
			mutation.mutate(&changed)
			if err := fixture.store.writeCandidate(fixture.dir, changed); err != nil {
				t.Fatal(err)
			}
			changed, err = fixture.store.readCandidate(fixture.dir, changed.ID)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, found, err := freshPassedReviewForScope(fixture.host, fixture.root, fixture.store, fixture.dir,
				fixture.plan, fixture.run, deleteProject, changed, deleteTask, "work"); err != nil || found {
				t.Fatalf("changed deletion %s reused a prior pass: found=%t err=%v", name, found, err)
			}
		})
	}

	for name, candidateFixture := range map[string]struct {
		id   string
		file File
	}{
		"wrong source bytes": {id: "44444444444444444444444444444444", file: File{Path: path, Mode: fixture.base.Snapshot.Modes[path], Content: []byte("unrelated"), Delete: true}},
		"wrong source mode":  {id: "66666666666666666666666666666666", file: File{Path: path, Mode: "0755", Content: append([]byte(nil), prior...), Delete: true}},
		"missing source":     {id: "77777777777777777777777777777777", file: File{Path: "src/not-in-source.txt", Mode: "0644", Content: []byte("not grounded"), Delete: true}},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := candidateData{ID: candidateFixture.id, Parents: []string{fixture.initial.ID}, Files: map[string]File{candidateFixture.file.Path: candidateFixture.file}}
			if err := fixture.store.writeCandidate(fixture.dir, invalid); err != nil {
				t.Fatal(err)
			}
			invalid, err = fixture.store.readCandidate(fixture.dir, invalid.ID)
			if err != nil {
				t.Fatal(err)
			}
			fixture.run.Reviews = []ReviewRecord{{ManagerID: deleteTask.ManagerID, Phase: "work", CandidateID: invalid.ID,
				CandidateDigest: invalid.Digest, ScopeDigest: "sha256:tampered-review", Outcome: "pass"}}
			invalidTask := deleteTask
			invalidTask.WrittenPaths = []string{candidateFixture.file.Path}
			if _, _, _, err := freshPassedReviewForScope(fixture.host, fixture.root, fixture.store, fixture.dir,
				fixture.plan, fixture.run, deleteProject, invalid, invalidTask, "work"); err == nil {
				t.Fatal("review accepted a retained deletion whose prior bytes or mode did not match its source baseline")
			}
		})
	}
}

func TestReviewCandidateChangesGroundsDeletionWithPriorBytes(t *testing.T) {
	base := &Snapshot{Files: map[string][]byte{
		"src/deleted.txt": []byte("prior contents"),
		"src/changed.txt": []byte("before"),
	}, Modes: map[string]string{
		"src/deleted.txt": snapshot.RegularMode,
		"src/changed.txt": snapshot.RegularMode,
	}}
	candidate := candidateData{Files: map[string]File{
		"src/added.txt":   {Path: "src/added.txt", Mode: "0644", Content: []byte("new")},
		"src/changed.txt": {Path: "src/changed.txt", Mode: "0644", Content: []byte("after")},
		"src/deleted.txt": {Path: "src/deleted.txt", Mode: "0644", Content: []byte("prior contents"), Delete: true},
	}}
	changes, deleted, err := reviewCandidateChanges(base, candidate, []string{"src/added.txt", "src/changed.txt", "src/deleted.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 3 || changes[0].Path != "src/added.txt" || changes[0].Operation != "add" ||
		changes[1].Path != "src/changed.txt" || changes[1].Operation != "modify" ||
		changes[2].Path != "src/deleted.txt" || changes[2].Operation != "delete" {
		t.Fatalf("unexpected typed change manifest: %+v", changes)
	}
	if changes[2].PriorDigest != rawContentDigest([]byte("prior contents")) || changes[2].PriorMode != "0644" {
		t.Fatalf("deletion is not grounded in exact prior bytes and mode: %+v", changes[2])
	}
	if len(deleted) != 1 || deleted[0].Path != "src/deleted.txt" || string(deleted[0].Content) != "prior contents" ||
		deleted[0].Digest != rawContentDigest([]byte("prior contents")) || deleted[0].Mode != "0644" {
		t.Fatalf("deletion evidence does not carry the exact prior file: %+v", deleted)
	}
}

func TestReviewContextAndScopeBindDeletedPathOperationAndPreimage(t *testing.T) {
	const managerID = "shop/orders"
	const path = "src/orders/implementation.txt"
	prior := []byte("prior implementation")
	base := &Snapshot{Files: map[string][]byte{path: prior}, Modes: map[string]string{path: snapshot.RegularMode}}
	candidate := candidateData{ID: "candidate", Files: map[string]File{
		path: {Path: path, Mode: "0644", Content: append([]byte(nil), prior...), Delete: true},
	}}
	project := &Project{
		Config:   projectwork.Config{InventoryRoots: []string{"src"}},
		Snapshot: &snapshot.Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}},
		Report: projectmodel.Report{
			Managers:  []projectmodel.Manager{{ID: managerID}},
			Artifacts: []projectmodel.Artifact{{ID: "orders-implementation", Owner: managerID, Required: true, Paths: []string{path}}},
			Files:     []projectmodel.FileEntry{{Path: path, Owner: managerID, Class: "source", Artifacts: []string{"orders-implementation"}}},
		},
	}
	task := ManagerTask{ManagerID: managerID, Goal: "Assess order implementation.", Artifacts: []string{"orders-implementation"}, WrittenPaths: []string{path}}
	plan := PlanRecord{Goal: task.Goal, Managers: []ManagerTask{task}}
	state := reviewCandidateState{Base: base, Candidate: candidate}
	context, files, refs, err := buildReviewerContext(plan, project, task, "work", 1, candidate, BriefingContext{}, RunReport{}, state)
	if err != nil {
		t.Fatal(err)
	}
	if len(context.CandidateChanges) != 1 || context.CandidateChanges[0].Operation != "delete" ||
		context.CandidateChanges[0].PriorDigest != rawContentDigest(prior) || context.CandidateChanges[0].PriorMode != "0644" {
		t.Fatalf("review context omitted the typed, grounded deletion: %+v", context.CandidateChanges)
	}
	if len(files) != 1 || files[0].Path != path || string(files[0].Content) != string(prior) || files[0].Digest != rawContentDigest(prior) {
		t.Fatalf("review request does not carry exact non-empty prior bytes: %+v", files)
	}
	if len(refs) != 1 || refs[0].Operation != "delete" || refs[0].Digest != rawContentDigest(prior) ||
		!containsString(refs[0].Grounding, "operation:delete") || !containsString(refs[0].Grounding, "prior-file-bytes") {
		t.Fatalf("deleted path review ref is not explicitly grounded: %+v", refs)
	}
	if _, err := validateReviewFindings([]reviewFindingResponse{{Path: path, Expectation: "Retain required behavior.", Grounding: "operation:delete"}}, refs); err != nil {
		t.Fatalf("typed deletion reference cannot ground a finding: %v", err)
	}

	initialScope, err := reviewScopeDigest(plan, project, task, "work", RunReport{}, state)
	if err != nil {
		t.Fatal(err)
	}
	changedBase := &Snapshot{Files: map[string][]byte{path: []byte("different prior bytes")}, Modes: map[string]string{path: snapshot.RegularMode}}
	changedScope, err := reviewScopeDigest(plan, project, task, "work", RunReport{}, reviewCandidateState{Base: changedBase, Candidate: candidate})
	if err != nil {
		t.Fatal(err)
	}
	if changedScope == initialScope {
		t.Fatal("review freshness digest ignored changed prior bytes for a deletion")
	}
}

func TestReviewCandidateChangesUsesRetainedDeletionPreimageAndRequiresGrounding(t *testing.T) {
	candidate := candidateData{Files: map[string]File{
		"src/empty.txt": {Path: "src/empty.txt", Mode: "0644", Content: []byte{}, Delete: true},
	}}
	changes, deleted, err := reviewCandidateChanges(&Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}}, candidate, []string{"src/empty.txt"})
	if err != nil {
		t.Fatalf("valid empty-file deletion preimage rejected: %v", err)
	}
	if len(changes) != 1 || changes[0].Operation != "delete" || changes[0].PriorDigest != rawContentDigest(nil) ||
		len(deleted) != 1 || len(deleted[0].Content) != 0 || deleted[0].Mode != "0644" {
		t.Fatalf("empty prior file was not represented as a grounded deletion: changes=%+v artifacts=%+v", changes, deleted)
	}

	missing := candidateData{Files: map[string]File{"src/missing.txt": {Path: "src/missing.txt", Delete: true}}}
	if _, _, err := reviewCandidateChanges(&Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}}, missing, []string{"src/missing.txt"}); err == nil {
		t.Fatal("ungrounded deletion without prior bytes or mode was accepted")
	}
}
