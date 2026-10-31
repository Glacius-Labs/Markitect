package desktop

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func save(s *server, id, def, text, base string) (int, map[string]any) {
	body, _ := json.Marshal(editRequest{Text: text, Base: base})
	w := call(s, "PUT", "/api/projects/"+id+"/definitions/"+def, string(body))
	var v map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &v)
	return w.Code, v
}

func TestAHandEditSavesIntoTheWorkingTree(t *testing.T) {
	files := library(t)
	s, _, _, id := modelServer(t, files)
	v, _ := getModel(t, s, id, "")
	member := v.definition("members/member")

	text := "---\nkind: std/Concept\n---\n# Member\n\nSomeone with a library card.\n"
	if code, body := save(s, id, "members/member", text, member.Digest); code != http.StatusOK || body["digest"] != digest([]byte(text)) {
		t.Fatalf("save: %d %v", code, body)
	}
	if files[".markitect/model/members/member.md"] != text {
		t.Fatalf("file = %q", files[".markitect/model/members/member.md"])
	}
	// The views show it, and check runs on it again.
	v, _ = getModel(t, s, id, "")
	if !strings.Contains(v.definition("members/member").Text, "library card") {
		t.Error("the views do not show the edit")
	}

	// A new Definition is written where none is.
	fresh := "---\nkind: std/Feature\n---\n# Renew a loan\n"
	if code, body := save(s, id, "loans/renew", fresh, ""); code != http.StatusOK {
		t.Fatalf("create: %d %v", code, body)
	}
	if files[".markitect/model/loans/renew.md"] != fresh {
		t.Error("the new Definition is not written")
	}
}

func TestAHandEditOverwritesNothingItDidNotSee(t *testing.T) {
	files := library(t)
	s, _, _, id := modelServer(t, files)
	v, _ := getModel(t, s, id, "")
	opened := v.definition("members/member").Digest

	// Another editor saves first.
	theirs := "---\nkind: std/Concept\n---\n# Member\n\nChanged elsewhere.\n"
	files[".markitect/model/members/member.md"] = theirs
	code, body := save(s, id, "members/member", "mine\n", opened)
	if code != http.StatusConflict || body["text"] != theirs || body["digest"] != digest([]byte(theirs)) {
		t.Fatalf("stale edit: %d %v", code, body)
	}
	if files[".markitect/model/members/member.md"] != theirs {
		t.Fatal("the stale edit overwrote the file")
	}
	// Knowing their text, the owner may keep theirs or replace it.
	if code, _ := save(s, id, "members/member", "mine\n", body["digest"].(string)); code != http.StatusOK {
		t.Fatalf("after seeing theirs: %d", code)
	}

	if code, body := save(s, id, "members/member", "again\n", ""); code != http.StatusConflict || !strings.Contains(body["error"].(string), "exists already") {
		t.Errorf("creating what exists: %d %v", code, body)
	}
	delete(files, ".markitect/model/loans/loan.md")
	if code, body := save(s, id, "loans/loan", "back\n", "abc"); code != http.StatusConflict || body["digest"] != "" || !strings.Contains(body["error"].(string), "removed") {
		t.Errorf("editing what is gone: %d %v", code, body)
	}
}

func TestAHandEditKeepsTheFilesLineEnds(t *testing.T) {
	files := library(t)
	files[".markitect/model/policy.md"] = "---\r\nkind: std/Rule\r\n---\r\n# Loan limit\r\n"
	s, _, _, id := modelServer(t, files)
	v, _ := getModel(t, s, id, "")
	policy := v.definition("policy")
	if strings.Contains(policy.Text, "\r") || policy.Digest != digest([]byte(policy.Text)) {
		t.Fatalf("the document's text %q, digest %s", policy.Text, policy.Digest)
	}
	if code, body := save(s, id, "policy", "---\nkind: std/Rule\n---\n# Loan limit\n\nThree loans.\n", policy.Digest); code != http.StatusOK {
		t.Fatalf("save: %d %v", code, body)
	}
	if got := files[".markitect/model/policy.md"]; got != "---\r\nkind: std/Rule\r\n---\r\n# Loan limit\r\n\r\nThree loans.\r\n" {
		t.Errorf("file = %q", got)
	}
}

func TestAHandEditWritesOnlyDefinitionsOfTheModel(t *testing.T) {
	s, repos, root, id := modelServer(t, library(t))
	for _, def := range []string{"..%2Fx", ".hidden", "a%20b", "CON", "members/", "members//x", "x.md", "-x", "%C3%A4"} {
		// The mux answers a path it would clean with a redirect.
		if code, _ := save(s, id, def, "x\n", ""); code != http.StatusBadRequest && code != http.StatusNotFound && code != http.StatusTemporaryRedirect {
			t.Errorf("%s: %d", def, code)
		}
	}
	if n := len(repos.repos[root].files); n != len(library(t)) {
		t.Errorf("%d files after the refused writes", n)
	}
	w := call(s, "PUT", "/api/projects/"+id+"/definitions/x", `{"text":"x","base":"","path":"/etc/x"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("an unknown field: %d", w.Code)
	}
	w = call(s, "PUT", "/api/projects/"+id+"/definitions/x", `{"text":"x"}`, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") })
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("a form's write: %d", w.Code)
	}
	if code, _ := save(s, id, "x", strings.Repeat("x", maxDefinition), ""); code != http.StatusRequestEntityTooLarge {
		t.Errorf("an oversized text: %d", code)
	}
}
