package projectapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

func TestExploreOperationPreviewsAndWritesWithExpectedDigest(t *testing.T) {
	root := makePlanFixture(t)
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Report.Managers) == 0 {
		t.Fatal("fixture has no Managers")
	}
	record := projectexplore.Record{
		APIVersion: projectexplore.APIVersion, ID: "facade-exploration", Status: projectexplore.StatusActive,
		Request:   "Implement a bounded change.",
		Scopes:    []projectexplore.Scope{{ID: "bounded", Name: "Bounded", Goal: "Implement the bounded change.", Operation: "apply", ManagerIDs: []string{project.Report.Managers[0].ID}}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{},
		Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{},
	}
	operations := Operations{Host: projectrun.Host{Load: projectwork.Load}}
	selection := Selection{Root: root}
	preview, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Plan == nil || preview.Plan.Digest == "" || preview.Persisted != nil {
		t.Fatalf("preview result = %#v", preview)
	}
	if _, err := projectexplore.Load(root, record.ID); err == nil {
		t.Fatal("preview persisted the record")
	}
	if record.Digest != "" || record.CreatedAgainst != "" {
		t.Fatalf("preview mutated caller-owned record: %#v", record)
	}
	if _, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record, Write: true, ExpectedDigest: "sha256:stale"}); err == nil {
		t.Fatal("write with a stale expected digest succeeded")
	}
	if _, err := projectexplore.Load(root, record.ID); err == nil {
		t.Fatal("stale digest write persisted the record")
	}
	written, err := operations.Explore(ExploreOperation{Selection: selection, Record: &record, Write: true, ExpectedDigest: preview.Plan.Digest})
	if err != nil {
		t.Fatalf("write exact preview: %v", err)
	}
	if written.Persisted == nil || written.Persisted.ID != record.ID || written.Plan == nil || written.Plan.Digest != preview.Plan.Digest {
		t.Fatalf("write result = %#v", written)
	}
	loaded, err := projectexplore.Load(root, record.ID)
	if err != nil || loaded.Digest != written.Persisted.Digest {
		t.Fatalf("loaded record = %#v err=%v", loaded, err)
	}
	listed, err := operations.Explore(ExploreOperation{Selection: selection})
	if err != nil || len(listed.Records) != 1 || listed.Records[0].ID != record.ID {
		t.Fatalf("list result = %#v err=%v", listed, err)
	}
	read, err := operations.Explore(ExploreOperation{Selection: selection, ExplorationID: record.ID})
	if err != nil || read.Record == nil || read.Record.Digest != loaded.Digest {
		t.Fatalf("read result = %#v err=%v", read, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".markitect", "state", "explorations", record.ID+".json")); err != nil {
		t.Fatalf("expected durable exploration file: %v", err)
	}
}

func TestReadinessRequiresExplicitAcknowledgementForWrite(t *testing.T) {
	loads := 0
	operations := Operations{Host: projectrun.Host{Load: func(string, string) (*projectrun.Project, error) {
		loads++
		return nil, nil
	}}}
	_, err := operations.Readiness(ReadinessOperation{Selection: Selection{Root: t.TempDir()}, Write: true})
	if err == nil || err.Error() != "project readiness --write requires --acknowledge-structure" {
		t.Fatalf("Readiness write without acknowledgement error = %v", err)
	}
	if loads != 0 {
		t.Fatalf("invalid write reached project Host %d times", loads)
	}
}

func TestExplorationOperationJSONUsesCamelCaseAndTypedRecord(t *testing.T) {
	record := &projectexplore.Record{ID: "typed-record"}
	payload, err := json.Marshal(ExploreOperation{
		Selection: Selection{Root: "repo"}, ExplorationID: "typed-record", Record: record, ExpectedDigest: "sha256:expected",
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(payload)
	for _, field := range []string{`"selection":{"root":"repo","revision":""}`, `"explorationId"`, `"expectedDigest"`, `"record":{"apiVersion"`} {
		if !strings.Contains(encoded, field) {
			t.Fatalf("ExploreOperation JSON %s omits %s", encoded, field)
		}
	}
	if strings.Contains(encoded, `"input"`) {
		t.Fatalf("ExploreOperation JSON retained byte input: %s", encoded)
	}

	acknowledgement := &StructureAcknowledgementInput{Actor: "caller", Authority: "owner", Provenance: "request"}
	readiness, err := json.Marshal(ReadinessOperation{ExplorationID: "typed-record", ScopeID: "scope", Acknowledgement: acknowledgement})
	if err != nil {
		t.Fatal(err)
	}
	encoded = string(readiness)
	for _, field := range []string{`"explorationId"`, `"scopeId"`, `"acknowledgement"`, `"recordedAt"`} {
		if !strings.Contains(encoded, field) {
			t.Fatalf("ReadinessOperation JSON %s omits %s", encoded, field)
		}
	}
}
