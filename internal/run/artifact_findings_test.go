package run

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func artifactFixture(t *testing.T) (*fixture, *[]string) {
	t.Helper()
	f := newFixture(t)
	var mu sync.Mutex
	var asked []string
	f.cfg.ArtifactFindingsAdvisory = true
	f.cfg.ResolveDefinition = func(tree snapshot.Tree, id string) bool {
		mu.Lock()
		defer mu.Unlock()
		asked = append(asked, id)
		return id == "library/lending-rules" && tree.Digest().String() != ""
	}
	return f, &asked
}

func subjectFinding(owner, subject, definition string, blocking bool) fnd {
	f := finding(owner, subject+" obligation", blocking)
	if subject != "" {
		f["subject"] = subject
	}
	if definition != "" {
		f["definition"] = definition
	}
	return f
}

func TestAssessSchemaStaysUnchangedWithoutArtifactFindings(t *testing.T) {
	f := newFixture(t)
	f.agents.script = func(c *call) error {
		if c.turn.Kind == VerifyTurn {
			if len(c.turn.Tools) != 1 || string(c.turn.Tools[0].InputSchema) != assessSchema {
				t.Errorf("assess schema changed: %+v", c.turn.Tools)
			}
			if strings.Contains(assessSchema, "subject") || strings.Contains(assessSchema, "definition") {
				t.Error("the Apply schema names the new fields")
			}
			// The fields do not exist: they are refused as any unknown field.
			for _, fd := range []fnd{subjectFinding("library", "artifact", "library/lending-rules", false), {"owner": "library", "obligation": "o", "observed": "b", "blocking": false, "definition": "d"}} {
				a := c.tool(toolAssess, map[string]any{"summary": "s", "satisfied": true, "findings": []fnd{fd}})
				if !a.Refused || !strings.Contains(a.Text, "unknown field") {
					t.Errorf("fields accepted without the option: %+v", a)
				}
			}
		}
		return c.defaultScript()
	}
	if out := f.run(); out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
}

func TestAssessSchemaNamesSubjectAndDefinitionWithArtifactFindings(t *testing.T) {
	f, _ := artifactFixture(t)
	f.agents.script = func(c *call) error {
		if c.turn.Kind == VerifyTurn {
			schema := string(c.turn.Tools[0].InputSchema)
			contains(t, "assess schema", schema, `"required": ["owner", "subject", "obligation", "observed", "blocking"]`, `"enum": ["artifact", "model"]`, `"definition"`, "an artifact does not conform to the model", "the model itself is defective")
		}
		return c.defaultScript()
	}
	if out := f.run(); out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
}

func TestArtifactFindingsOptionRequiresResolveDefinition(t *testing.T) {
	f := newFixture(t)
	f.cfg.ArtifactFindingsAdvisory = true
	if _, err := Reconcile(t.Context(), f.cfg); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "ResolveDefinition") {
		t.Fatalf("error %v", err)
	}
}

func TestAssessRefusesFindingsWithoutAValidSubjectOrDefinition(t *testing.T) {
	f, asked := artifactFixture(t)
	f.agents.script = func(c *call) error {
		if c.is(1, "library", VerifyTurn) {
			bad := []struct {
				name string
				fnd  fnd
				want string
			}{
				{"no subject", subjectFinding("library", "", "", false), "subject must be"},
				{"unknown subject", subjectFinding("library", "realization", "", false), "subject must be"},
				{"artifact without definition", subjectFinding("library", "artifact", "", false), "definition is required"},
				{"artifact with blank definition", subjectFinding("library", "artifact", "  ", false), "definition is required"},
				{"unknown definition", subjectFinding("library", "artifact", "library/other", false), "not a Definition of the candidate model"},
			}
			for _, b := range bad {
				a := c.tool(toolAssess, map[string]any{"summary": "s", "satisfied": true, "findings": []fnd{b.fnd}})
				if !a.Refused || !strings.Contains(a.Text, b.want) {
					t.Errorf("%s: %+v", b.name, a)
				}
			}
			// A refused report is repeatable.
			if a := c.tool(toolAssess, map[string]any{"summary": "s", "satisfied": true, "findings": []fnd{subjectFinding("library", "artifact", "library/lending-rules", false)}}); a.Refused {
				t.Errorf("valid report refused: %s", a.Text)
			}
			return nil
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified {
		t.Fatalf("%s %s", out.Status, out.Reason)
	}
	if !slices.Contains(*asked, "library/lending-rules") || !slices.Contains(*asked, "library/other") {
		t.Errorf("definitions resolved: %v", *asked)
	}
}

func TestArtifactFindingsAreStoredAdvisoryAndModelFindingsKeepTheirFlag(t *testing.T) {
	f, _ := artifactFixture(t)
	f.agents.script = func(c *call) error {
		if c.is(1, "library", VerifyTurn) {
			// The reviewer calls the artifact finding blocking; it cannot block, so
			// only the model finding keeps the report unsatisfied.
			if a := c.assess(true, []fnd{subjectFinding("library", "artifact", "library/lending-rules", true)}); a.Refused {
				t.Errorf("artifact finding refused: %s", a.Text)
			}
			if a := c.assess(false, []fnd{subjectFinding("library", "artifact", "library/lending-rules", true)}); !a.Refused || !strings.Contains(a.Text, "satisfied is false") {
				t.Errorf("an artifact finding counted as blocking: %+v", a)
			}
			if a := c.assess(false, []fnd{
				subjectFinding("library", "artifact", "library/lending-rules", true),
				subjectFinding("library", "model", "", true),
			}); a.Refused {
				t.Errorf("model finding refused: %s", a.Text)
			}
			return nil
		}
		return c.defaultScript()
	}
	out := f.run()
	if out.Status != Verified || len(out.Rounds) != 2 || out.Rounds[0].Green {
		t.Fatalf("%s %s rounds %+v", out.Status, out.Reason, out.Rounds)
	}
	var artifact, model *Finding
	for i, fd := range out.Findings {
		switch fd.Subject {
		case "artifact":
			artifact = &out.Findings[i]
		case "model":
			model = &out.Findings[i]
		}
	}
	if artifact == nil || model == nil || len(out.Findings) != 2 {
		t.Fatalf("findings %+v", out.Findings)
	}
	if artifact.Blocking || artifact.Definition != "library/lending-rules" || artifact.RaisedBy != "library" {
		t.Errorf("artifact finding %+v", *artifact)
	}
	if !model.Blocking || model.Definition != "" {
		t.Errorf("model finding %+v", *model)
	}
	for _, r := range f.ev.kinds("finding.raised") {
		fd, ok := r.Data.(Finding)
		if !ok || (fd.Subject == "artifact") == fd.Blocking {
			t.Errorf("recorded finding %+v", r.Data)
		}
	}
}

func TestAssessSchemasDifferOnlyInTheFindingSubject(t *testing.T) {
	findingItems := func(name, schema string) (items struct {
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}) {
		var parsed struct {
			Properties struct {
				Findings struct {
					Items struct {
						Required   []string                   `json:"required"`
						Properties map[string]json.RawMessage `json:"properties"`
					} `json:"items"`
				} `json:"findings"`
			} `json:"properties"`
		}
		if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
			t.Fatalf("%s is not JSON: %v", name, err)
		}
		return parsed.Properties.Findings.Items
	}
	on, off := findingItems("assessSchemaSubjects", assessSchemaSubjects), findingItems("assessSchema", assessSchema)
	var subject struct {
		Enum []string `json:"enum"`
	}
	if err := json.Unmarshal(on.Properties["subject"], &subject); err != nil || !slices.Equal(subject.Enum, []string{"artifact", "model"}) {
		t.Errorf("subject property %s: %v", on.Properties["subject"], err)
	}
	if _, ok := on.Properties["definition"]; !ok || !slices.Contains(on.Required, "subject") {
		t.Errorf("option-on findings: properties %v, required %v", on.Properties, on.Required)
	}
	if _, ok := off.Properties["subject"]; ok || slices.Contains(off.Required, "subject") {
		t.Errorf("option-off findings have a subject: properties %v, required %v", off.Properties, off.Required)
	}
	if _, ok := off.Properties["definition"]; ok {
		t.Error("option-off findings have a definition")
	}
}
