package harness

import (
	"strings"
	"testing"
)

func TestAChatCarriesItsRulesAndSkill(t *testing.T) {
	doc, err := Chat(ChatInput{Skill: ChatModeling})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Markitect chat", "Change only files below .markitect/", ChatStart} {
		if !strings.Contains(doc.Base, want) {
			t.Errorf("the modeling chat's rules lack %q", want)
		}
	}
	for _, want := range []string{"markitect-modeling/SKILL.md", "markitect-modeling/reference.md", "markitect-modeling/examples.md"} {
		if !strings.Contains(doc.Context, want) {
			t.Errorf("the modeling chat's skill lacks %q", want)
		}
	}
	if strings.Contains(doc.Context, "markitect-extract/SKILL.md") || doc.Task != "" {
		t.Errorf("the modeling chat carries more than it should: %q", doc.Task)
	}

	doc, err = Chat(ChatInput{Skill: ChatExtract, Earlier: "owner: Draw the model.\n\nagent: Which folders?"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Context, "markitect-extract/SKILL.md") || !strings.Contains(doc.Context, "markitect-modeling/SKILL.md") {
		t.Error("the Extract chat lacks a skill")
	}
	if !strings.Contains(doc.Task, "agent: Which folders?") || !strings.HasSuffix(doc.Task, "The owner's next message:") {
		t.Errorf("task %q", doc.Task)
	}
	if _, err := Chat(ChatInput{Skill: "poetry"}); err == nil {
		t.Error("an unknown skill rendered")
	}
}
