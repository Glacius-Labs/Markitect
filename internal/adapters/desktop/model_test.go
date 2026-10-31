package desktop

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

// stdLock is the lock for std alone.
func stdLock(t *testing.T) string {
	t.Helper()
	files, ok := libraries.Files("std")
	if !ok {
		t.Fatal("no Library std")
	}
	lib := model.LibrarySource{Name: "std", Version: libraries.Version}
	for _, f := range files {
		lib.Files = append(lib.Files, model.File{Path: f.Path, Data: f.Data})
	}
	return string(model.RenderLock([]model.LibrarySource{lib}))
}

// library is a small project: a Rule at Root, members with a public and an
// internal Definition, and loans, which depends on members.
func library(t *testing.T) map[string]string {
	return map[string]string{
		".markitect/manifest.yaml":           "libraries:\n  std: 1.0.0\n",
		".markitect/lock.yaml":               stdLock(t),
		".markitect/execution.yaml":          "checks: []\n",
		".markitect/ignore.md":               "| Path | Why |\n|---|---|\n| LICENSE | Legal text |\n",
		".markitect/model/project.md":        "---\nkind: std/Project\n---\n# Library\n\nMembers borrow books; see [[members]] and [[loans]].\n",
		".markitect/model/policy.md":         "---\nkind: std/Rule\n---\n# Loan limit\n\nA member holds at most three loans.\n",
		".markitect/model/members/member.md": "---\nkind: std/Concept\n---\n# Member\n\nSomeone who may borrow.\n",
		".markitect/model/members/card.md":   "---\nkind: std/Concept\ninternal: true\n---\n# Card\n\nThe [[members/member]]'s card.\n",
		".markitect/model/loans/loan.md":     "---\nkind: std/Feature\n---\n# Borrow a book\n\nA [[members/member|member]] borrows a book.\n",
		".markitect/registry.md":             "| File | Realizes | Why |\n|---|---|---|\n| README.md | [[project]] | Overview |\n",
		".markitect/registry/members.md":     "---\npaths:\n  - src/members/**\n---\n| File | Realizes | Why |\n|---|---|---|\n| `src/members/member.go` | [[members/member]] [[members/card]] | Member type |\n",
		".markitect/registry/loans.md":       "---\npaths:\n  - src/loans/**\n---\n| File | Realizes | Why |\n|---|---|---|\n| `src/loans/loan.go` | [[loans/loan]] | Borrowing |\n",
		"README.md":                          "# Library\n",
		"LICENSE":                            "MIT\n",
		"src/members/member.go":              "package members\n",
		"src/loans/loan.go":                  "package loans\n",
	}
}

// modelServer serves one project with the given working tree.
func modelServer(t *testing.T, files map[string]string) (*server, *fakeRepos, string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "library")
	s, repos := testServer(t, root)
	repos.repos[root] = memRepo{root: root, files: files}
	w := call(s, "POST", "/api/projects", `{"action":"open","path":`+jsonString(root)+`}`)
	if w.Code != http.StatusOK {
		t.Fatalf("open: %d %s", w.Code, w.Body)
	}
	id := decode(t, w)["project"].(map[string]any)["id"].(string)
	return s, repos, root, id
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func getModel(t *testing.T, s *server, id, query string, edit ...func(*http.Request)) (modelView, *httpResult) {
	t.Helper()
	w := call(s, "GET", "/api/projects/"+id+"/model"+query, "", edit...)
	res := &httpResult{code: w.Code, etag: w.Header().Get("ETag"), body: w.Body.String()}
	var v modelView
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
	}
	return v, res
}

type httpResult struct {
	code int
	etag string
	body string
}

func (v modelView) definition(id string) definitionView {
	for _, d := range v.Definitions {
		if d.ID == id {
			return d
		}
	}
	return definitionView{}
}

func (v modelView) pkg(path string) packageView {
	for _, p := range v.Packages {
		if p.Path == path {
			return p
		}
	}
	return packageView{}
}

func TestTheModelViewSaysWhatTheModelCommandsSay(t *testing.T) {
	files := library(t)
	s, repos, root, id := modelServer(t, files)
	repos.commit(root, "c1", false)
	files[".markitect/model/loans/loan.md"] = "---\nkind: std/Feature\n---\n# Borrow a book\n\nA [[members/member|member]] borrows a book within the [[policy]].\n"

	v, res := getModel(t, s, id, "")
	if res.code != http.StatusOK {
		t.Fatalf("%d %s", res.code, res.body)
	}
	if !v.Initialized || len(v.Errors) != 0 {
		t.Fatalf("initialized %v, errors %+v", v.Initialized, v.Errors)
	}
	if v.Base == nil || v.Base.Kind != baseCommit || v.Base.Commit != "c1" || v.Bases.Landing != "" {
		t.Fatalf("base = %+v, bases = %+v", v.Base, v.Bases)
	}

	member := v.definition("members/member")
	if member.Title != "Member" || member.Kind != "std/Concept" || member.Package != "members" || !strings.Contains(member.Text, "Someone who may borrow.") {
		t.Fatalf("member = %+v", member)
	}
	var from []string
	for _, r := range member.ReferencedBy {
		from = append(from, r.From+" "+r.Via)
	}
	if strings.Join(from, ",") != "loans/loan link,members/card link" {
		t.Errorf("member referenced by %v", from)
	}
	if len(member.RealizedBy) != 1 || member.RealizedBy[0].File != "src/members/member.go" || member.RealizedBy[0].Reason != "Member type" {
		t.Errorf("member realized by %+v", member.RealizedBy)
	}
	// The members Manager knows it as its own, the loans Manager through its
	// link; Root sees it in the table of contents of its child members.
	if got := member.SeenBy; strings.Join(got.Full, ",") != "loans,members" || !slices.Equal(got.Contents, []string{model.RootPackage}) {
		t.Errorf("member seen by %+v", got)
	}
	card := v.definition("members/card")
	if got := card.SeenBy; strings.Join(got.Full, ",") != "members" || len(got.Contents) != 0 {
		t.Errorf("the internal card is seen by %+v", got)
	}
	// Root's Rule reaches the whole subtree.
	if got := v.definition("policy").SeenBy; strings.Join(got.Full, ",") != ",loans,members" {
		t.Errorf("policy seen by %+v", got)
	}
	if v.definition("std/Concept").Library != "std" {
		t.Error("the Library Kinds are not in the view")
	}

	members := v.pkg("members")
	if members.Name != "members" || strings.Join(members.Own, ",") != "members/card,members/member" || strings.Join(members.Patterns, ",") != "src/members/**" || members.Files != 1 || members.Afferent != 1 {
		t.Errorf("members = %+v", members)
	}
	if rootPkg := v.pkg(""); rootPkg.Name != "Root" || strings.Join(rootPkg.Children, ",") != "loans,members" {
		t.Errorf("root = %+v", rootPkg)
	}
	if len(v.Edges) != 1 || v.Edges[0].From != "loans" || v.Edges[0].To != "members" || v.Edges[0].Cycle || v.Edges[0].Uses[0].From != "loans/loan" {
		t.Errorf("edges = %+v", v.Edges)
	}

	// The loan changed since the commit, so its Manager has it on its
	// checklist, and Root integrates.
	if len(v.Diff.Definitions) != 1 || v.Diff.Definitions[0].ID != "loans/loan" || v.Diff.Definitions[0].Change != "modified" || !strings.Contains(v.Diff.Definitions[0].BaseText, "borrows a book.") {
		t.Fatalf("diff = %+v", v.Diff.Definitions)
	}
	if v.definition("loans/loan").Change != "modified" || v.definition("members/member").Change != "" {
		t.Error("the Definitions do not carry their change")
	}
	if v.Impact == nil || strings.Join(v.Impact.Packages, ",") != ",loans" {
		t.Fatalf("impact = %+v (%s)", v.Impact, v.ImpactNote)
	}
	// The policy was unused at the commit; the loan now links it.
	var gone []string
	for _, l := range v.Diff.Lint {
		gone = append(gone, l.Change+" "+l.Code+" "+l.Package)
	}
	if !slices.Contains(gone, "removed lint.unused ") {
		t.Errorf("lint diff = %v", gone)
	}
	for _, l := range v.Lint {
		if l.Code == "lint.unused" && l.Definition == "policy" {
			t.Error("the policy is still unused")
		}
	}
}

func TestTheModelViewAnswersAgainOnlyWhenSomethingChanged(t *testing.T) {
	files := library(t)
	s, repos, root, id := modelServer(t, files)
	repos.commit(root, "c1", false)
	_, first := getModel(t, s, id, "")
	if first.code != http.StatusOK || first.etag == "" {
		t.Fatalf("%d %q", first.code, first.etag)
	}
	ifNone := func(etag string) func(*http.Request) {
		return func(r *http.Request) { r.Header.Set("If-None-Match", etag) }
	}
	if _, again := getModel(t, s, id, "", ifNone(first.etag)); again.code != http.StatusNotModified {
		t.Fatalf("unchanged: %d", again.code)
	}
	// An edit in the folder, as from another editor, shows up.
	files[".markitect/model/members/member.md"] = "---\nkind: std/Concept\n---\n# Member\n\nSomeone with a card.\n"
	v, changed := getModel(t, s, id, "", ifNone(first.etag))
	if changed.code != http.StatusOK || changed.etag == first.etag || !strings.Contains(v.definition("members/member").Text, "with a card") {
		t.Fatalf("after an edit: %d %q", changed.code, changed.etag)
	}
	// So does a new commit, which moves the base.
	repos.commit(root, "c2", false)
	v, committed := getModel(t, s, id, "", ifNone(changed.etag))
	if committed.code != http.StatusOK || v.Base.Commit != "c2" || len(v.Diff.Definitions) != 0 {
		t.Fatalf("after a commit: %d %+v %+v", committed.code, v.Base, v.Diff.Definitions)
	}
}

func TestTheModelViewMeasuresFromTheLastLandingUnlessAskedForTheLastCommit(t *testing.T) {
	files := library(t)
	s, repos, root, id := modelServer(t, files)
	repos.commit(root, "landed", true)
	files[".markitect/model/members/member.md"] = "---\nkind: std/Concept\n---\n# Member\n\nSomeone with a card.\n"
	repos.commit(root, "later", false)

	v, _ := getModel(t, s, id, "")
	if v.Base.Kind != baseLanding || v.Base.Commit != "landed" || v.Bases.Commit != "later" || len(v.Diff.Definitions) != 1 {
		t.Fatalf("default: %+v %+v %+v", v.Base, v.Bases, v.Diff.Definitions)
	}
	v, _ = getModel(t, s, id, "?base=commit")
	if v.Base.Kind != baseCommit || v.Base.Commit != "later" || len(v.Diff.Definitions) != 0 {
		t.Fatalf("commit: %+v %+v", v.Base, v.Diff.Definitions)
	}
	if _, res := getModel(t, s, id, "?base=yesterday"); res.code != http.StatusBadRequest {
		t.Fatalf("an unknown base: %d", res.code)
	}
}

func TestWithNothingCommittedTheWholeModelIsTheChange(t *testing.T) {
	s, _, _, id := modelServer(t, library(t))
	v, res := getModel(t, s, id, "")
	if res.code != http.StatusOK || v.Base != nil || v.Bases.Commit != "" {
		t.Fatalf("%d %+v", res.code, v.Base)
	}
	if len(v.Diff.Definitions) != 5 || v.Diff.Definitions[0].Change != "added" || len(v.Diff.Packages) != 2 || len(v.Diff.Edges) != 1 {
		t.Fatalf("diff = %+v", v.Diff)
	}
	if v.Impact == nil || !v.Impact.Full || !strings.Contains(v.ImpactNote, "Nothing is committed") {
		t.Fatalf("impact = %+v %q", v.Impact, v.ImpactNote)
	}
}

func TestTheModelViewMarksCyclesAndPlacesErrors(t *testing.T) {
	files := library(t)
	files[".markitect/model/members/member.md"] = "---\nkind: std/Concept\n---\n# Member\n\nSomeone who may hold a [[loans/loan]].\n"
	files[".markitect/model/members/broken.md"] = "---\nkind: std/Concept\n---\n# Broken\n\nSee [[nowhere]].\n"
	s, _, _, id := modelServer(t, files)
	v, res := getModel(t, s, id, "")
	if res.code != http.StatusOK {
		t.Fatalf("%d %s", res.code, res.body)
	}
	if len(v.Cycles) != 1 || strings.Join(v.Cycles[0], ",") != "loans,members" {
		t.Fatalf("cycles = %v", v.Cycles)
	}
	for _, e := range v.Edges {
		if !e.Cycle {
			t.Errorf("edge %s → %s is not marked", e.From, e.To)
		}
	}
	var placed []string
	for _, d := range v.Errors {
		placed = append(placed, d.Code+" "+d.Package+" "+d.Definition)
	}
	if !slices.Contains(placed, "link.unresolved members members/broken") {
		t.Errorf("errors = %v", placed)
	}
	if v.Impact != nil || !strings.Contains(v.ImpactNote, "compiles") {
		t.Errorf("impact over compile errors: %+v %q", v.Impact, v.ImpactNote)
	}
}

func TestAProjectWithoutMarkitectHasNoModelView(t *testing.T) {
	s, _, _, id := modelServer(t, map[string]string{"README.md": "# x\n"})
	v, res := getModel(t, s, id, "")
	if res.code != http.StatusOK || v.Initialized {
		t.Fatalf("%d %s", res.code, res.body)
	}
}
