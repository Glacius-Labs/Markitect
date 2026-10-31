package desktop

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// A hand edit writes one Definition's file into the working tree, the second
// way to change the model beside the chat. The editor names the text it
// started from, so a file another editor changed meanwhile is never
// overwritten unseen.

// maxDefinition bounds the text of one hand edit.
const maxDefinition = 1 << 20

// editRequest is a hand edit of one Definition.
type editRequest struct {
	Text string `json:"text"`
	// Base is the digest of the text the editor started from, as the Model
	// area's document gives it; empty for a new Definition.
	Base string `json:"base"`
}

// conflict answers an edit whose base is no longer the file's text, with
// the text as it is now.
type conflict struct {
	Error  string `json:"error"`
	Text   string `json:"text"`
	Digest string `json:"digest"` // empty when the file is gone
}

// definitionName is a name of the model language: a letter or digit, then
// letters, digits, - and _.
var definitionName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// definitionID reports whether id names a Definition the editor may write:
// names of the model language that also mean the same on every platform.
func definitionID(id string) bool {
	if id == "" {
		return false
	}
	for _, part := range strings.Split(id, "/") {
		if !definitionName.MatchString(part) || !plainName(part) {
			return false
		}
	}
	return true
}

// digest marks a Definition's text, with its line ends as the compiler reads
// them.
func digest(data []byte) string {
	sum := sha256.Sum256(project.Normalize(data))
	return hex.EncodeToString(sum[:])
}

// saveDefinition answers PUT /api/projects/{id}/definitions/{path...}.
func (s *server) saveDefinition(w http.ResponseWriter, r *http.Request) {
	p, ok := s.find(w, r)
	if !ok {
		return
	}
	id := r.PathValue("path")
	if !definitionID(id) {
		fail(w, http.StatusBadRequest, fmt.Errorf("%q is no Definition's ID: names start with a letter or digit and hold only letters, digits, - and _", id))
		return
	}
	var req editRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDefinition))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	ws, err := s.repos.Open(r.Context(), p.Path)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	file := project.ModelDir + "/" + id + ".md"
	current, err := ws.Read(file)
	exists := err == nil
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	var refused string
	switch {
	case exists && req.Base == "":
		refused = fmt.Sprintf("%s exists already; open it in the Reader to change it", id)
	case !exists && req.Base != "":
		refused = fmt.Sprintf("%s was removed since you opened it", id)
	case exists && digest(current) != req.Base:
		refused = fmt.Sprintf("%s changed since you opened it", id)
	}
	if refused != "" {
		c := conflict{Error: refused}
		if exists {
			c.Text, c.Digest = string(project.Normalize(current)), digest(current)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(c)
		return
	}
	// The file keeps its line ends.
	data := project.Normalize([]byte(req.Text))
	if exists && bytes.Contains(current, []byte("\r\n")) {
		data = bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
	}
	if err := ws.Write(file, data); err != nil {
		fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	reply(w, map[string]string{"digest": digest(data)})
}
