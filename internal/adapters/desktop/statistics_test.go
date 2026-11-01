package desktop

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestStatisticsFollowTheModelThroughItsCommits(t *testing.T) {
	files := library(t)
	s, repos, root, id := modelServer(t, files)
	repos.commit(root, "c1", false)
	files[".markitect/model/loans/return.md"] = "---\nkind: std/Feature\n---\n# Return a book\n"
	repos.commit(root, "c2", true)
	files[".markitect/model/broken.md"] = "no front matter\n"

	get := func() []statPoint {
		w := call(s, "GET", "/api/projects/"+id+"/statistics", "")
		var v struct {
			Model []statPoint `json:"model"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || w.Code != http.StatusOK {
			t.Fatalf("%d %s", w.Code, w.Body)
		}
		return v.Model
	}
	pts := get()
	if len(pts) != 3 {
		t.Fatalf("points %+v", pts)
	}
	c1, c2, now := pts[0], pts[1], pts[2]
	if c1.Commit != "c1" || c2.Commit != "c2" || !c2.Landed || c1.Landed || now.Commit != "" || now.When.IsZero() {
		t.Errorf("order and marks %+v", pts)
	}
	if !c1.Initialized || c1.Definitions != 5 || c2.Definitions != 6 || c1.Packages != 3 || c1.Errors != 0 {
		t.Errorf("counts %+v %+v", c1, c2)
	}
	if now.Errors == 0 {
		t.Errorf("the working tree's broken file is not counted: %+v", now)
	}
	// A commit's numbers are read once.
	reads := repos.reads
	get()
	if repos.reads != reads {
		t.Errorf("the commits were read again: %d → %d", reads, repos.reads)
	}
}

func TestStatisticsOfAProjectWithoutCommits(t *testing.T) {
	s, _, _, id := modelServer(t, library(t))
	w := call(s, "GET", "/api/projects/"+id+"/statistics", "")
	var v struct {
		Model []statPoint `json:"model"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || len(v.Model) != 1 || v.Model[0].Definitions != 5 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}
