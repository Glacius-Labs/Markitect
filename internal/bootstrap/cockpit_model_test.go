package bootstrap

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cockpit"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func located(body, written string, target model.Identity, display *string) model.BodyLink {
	start := strings.LastIndex(body, written)
	return model.BodyLink{Target: target, Display: display, Coordinates: &model.LinkCoordinates{Body: &model.BodySpan{StartByte: start, EndByte: start + len(written)}}}
}

func render(segments []cockpit.BodySegment) string {
	var b strings.Builder
	for _, s := range segments {
		if s.Link != nil {
			b.WriteString("<" + s.Link.Package + "/" + s.Link.ID + ":" + s.Text + ">")
			continue
		}
		b.WriteString(s.Text)
	}
	return b.String()
}

// The cockpit links only the text the model check located: a [[...]] inside
// a Markdown link label is no link, so it must not take the next link's
// target.
func TestBodySegmentsFollowTheLocatedLinks(t *testing.T) {
	body := "[label [[api]]](url) and [[field]] or [[Vocab.rule|the rule]]."
	shown := "the rule"
	links := []model.BodyLink{
		located(body, "[[field]]", model.Identity{Package: "product", ID: "field"}, nil),
		located(body, "[[Vocab.rule|the rule]]", model.Identity{Package: "vocabulary", ID: "rule"}, &shown),
	}
	got := render(bodySegments(body, links))
	want := "[label [[api]]](url) and <product/field:field> or <vocabulary/rule:the rule>."
	if got != want {
		t.Fatalf("segments = %s, want %s", got, want)
	}
}

func TestBodySegmentsWithoutLocationsStayText(t *testing.T) {
	body := "See [[field]]."
	for name, links := range map[string][]model.BodyLink{
		"no coordinates": {{Target: model.Identity{Package: "p", ID: "field"}}},
		"out of order": {
			{Target: model.Identity{Package: "p", ID: "a"}, Coordinates: &model.LinkCoordinates{Body: &model.BodySpan{StartByte: 4, EndByte: 13}}},
			{Target: model.Identity{Package: "p", ID: "b"}, Coordinates: &model.LinkCoordinates{Body: &model.BodySpan{StartByte: 0, EndByte: 3}}},
		},
		"beyond the body": {{Target: model.Identity{Package: "p", ID: "a"}, Coordinates: &model.LinkCoordinates{Body: &model.BodySpan{StartByte: 4, EndByte: 99}}}},
	} {
		if got := bodySegments(body, links); len(got) != 1 || got[0].Link != nil || got[0].Text != body {
			t.Errorf("%s: segments = %+v", name, got)
		}
	}
}
