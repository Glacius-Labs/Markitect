package cockpit

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestModelIsServedFromTheProjectObservation(t *testing.T) {
	release := make(chan struct{})
	srv, err := New(Config{States: []string{t.TempDir()}, Project: func(context.Context) (ProjectInfo, error) {
		<-release
		return ProjectInfo{Repository: "repo", ModelID: "markitect", Status: "passed", Head: "abc", Model: &ModelView{
			Models: []ModelScope{{ID: "markitect", Local: true, Status: "passed", Packages: []string{"product"},
				Definitions: []DefinitionView{{Package: "product", ID: "rule", Kind: "vocabulary/Rule", Body: "Follows [[other]].", Links: []LinkView{{Package: "product", ID: "other"}}, Spec: []FieldView{}}}}},
			Registry: []RegistryEntry{{Shard: ".markitect/registry/product/registry.yaml", Path: "main.go", Form: "code", Realizes: []string{"product/rule"}, Why: "Implements it."}},
		}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	rec := get(t, h, "", "/api/model", nil)
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "being observed") {
		t.Fatalf("model while observing = %d %s", rec.Code, rec.Body)
	}
	close(release)
	for i := 0; ; i++ {
		rec = get(t, h, "", "/api/model", nil)
		if rec.Code == http.StatusOK {
			break
		}
		if i == 200 {
			t.Fatalf("the model never arrived: %d %s", rec.Code, rec.Body)
		}
		time.Sleep(5 * time.Millisecond)
	}
	var got struct {
		Head  string    `json:"head"`
		Model ModelView `json:"model"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Head != "abc" || len(got.Model.Models) != 1 || got.Model.Models[0].Definitions[0].Links[0].ID != "other" || got.Model.Registry[0].Realizes[0] != "product/rule" {
		t.Fatalf("model = %s, %v", rec.Body, err)
	}
	// The project summary stays small: the model is not part of it.
	if rec := get(t, h, "", "/api/project", nil); strings.Contains(rec.Body.String(), "definitions\":[") || strings.Contains(rec.Body.String(), "registry") {
		t.Fatalf("the project summary carries the model: %s", rec.Body)
	}
}

func TestModelNeedsARepository(t *testing.T) {
	srv, err := New(Config{States: []string{t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	if rec := get(t, srv.Handler(), "", "/api/model", nil); rec.Code != http.StatusNotImplemented {
		t.Fatalf("model without a repository = %d", rec.Code)
	}
}
