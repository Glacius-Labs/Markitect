package state

import (
	"strings"
	"testing"
	"time"
)

func TestParseHistory(t *testing.T) {
	data := `{"time":"2026-11-20T09:00:02Z","type":"verdict","doc":"b.md","verdict":"ok","changed":[]}` + "\r\n" +
		`{"time":"2026-11-20T09:00:01Z","type":"move","from":"a.md","to":"c.md","rewritten":["d.md"]}` + "\n" +
		`{"time":"2026-11-20T09:00:02Z","type":"verdict","doc":"a.md","verdict":"fail","changed":[{"doc":"b.md","self":"` + hashA + `"}]}` + "\n" +
		`{"time":"2026-11-20T09:00:00Z","type":"report","doc":"x.md"}` + "\n" +
		"not json\n" +
		`{"type":"verdict","doc":"a.md","verdict":"ok"}` + "\n" +
		`{"time":"2026-11-20T09:00:03Z","type":"verdict","doc":"","verdict":"ok"}` + "\n" +
		`{"time":"2026-11-20T09:00:02Z","type":"verdict","doc":"b.md","verdict":"ok","changed":[]}` + "\n"
	events, problems := ParseHistory([]byte(data))
	var got []string
	for _, e := range events {
		got = append(got, e.Type+" "+e.Doc+e.From)
	}
	// In time order, the file's order among equal times; the unknown type
	// skipped; a line held twice read once.
	if strings.Join(got, ", ") != "move a.md, verdict b.md, verdict a.md" {
		t.Error(got)
	}
	if len(events) == 3 && (len(events[2].Changed) != 1 || events[2].Changed[0].Self != hashA || strings.Join(events[0].Rewritten, " ") != "d.md") {
		t.Errorf("%+v", events)
	}
	if len(problems) != 3 || problems[0].Line != 5 || problems[1].Line != 6 || problems[2].Line != 7 ||
		!strings.HasPrefix(problems[0].String(), ".markitect/history.jsonl:5: history.unreadable: ") {
		t.Errorf("%v", problems)
	}
}

func TestEventLines(t *testing.T) {
	at := time.Date(2026, 11, 20, 9, 0, 0, 5, time.FixedZone("CET", 3600))
	v := Event{Time: at, Type: EventVerdict, Doc: "a.md", Verdict: "ok"}
	if got := string(v.Line()); got != `{"time":"2026-11-20T08:00:00.000000005Z","type":"verdict","doc":"a.md","verdict":"ok","changed":[]}`+"\n" {
		t.Error(got)
	}
	m := Event{Time: at, Type: EventMove, From: "a.md", To: "b.md"}
	if got := string(m.Line()); got != `{"time":"2026-11-20T08:00:00.000000005Z","type":"move","from":"a.md","to":"b.md","rewritten":[]}`+"\n" {
		t.Error(got)
	}
}

func TestMoveRows(t *testing.T) {
	h := func(s string) string { return Hash([]byte(s)) }
	rows := []Row{
		{Doc: "types/kind.md", Verdict: "ok", Source: "mark", Inputs: Inputs{"self": h("kind v1")}},
		{Doc: "a.md", Verdict: "ok", Source: "mark", Inputs: Inputs{"self": h("a v1"), "type:types/kind.md": h("kind v1"), "link:b.md": h("b old")}},
		{Doc: "b.md", Verdict: "ok", Source: "mark", Inputs: Inputs{"self": h("b old"), "link:a.md": h("a v1")}},
		{Doc: "c.md", Verdict: "ok", Source: "mark", Inputs: Inputs{"self": h("c")}},
	}
	// The type moves; a's type line is rewritten; b had changed before.
	out := MoveRows(rows, "types/kind.md", "kinds/kind.md", map[string]Text{
		"types/kind.md": {[]byte("kind v1"), []byte("kind v2")},
		"a.md":          {[]byte("a v1"), []byte("a v2")},
		"b.md":          {[]byte("b changed"), []byte("b rewritten")},
	})
	want := []Row{
		{Doc: "kinds/kind.md", Verdict: "ok", Source: "move", Inputs: Inputs{"self": h("kind v2")}},
		{Doc: "a.md", Verdict: "ok", Source: "move", Inputs: Inputs{"self": h("a v2"), "type:kinds/kind.md": h("kind v2"), "link:b.md": h("b old")}},
		{Doc: "b.md", Verdict: "ok", Source: "move", Inputs: Inputs{"self": h("b old"), "link:a.md": h("a v2")}},
		{Doc: "c.md", Verdict: "ok", Source: "mark", Inputs: Inputs{"self": h("c")}},
	}
	for i := range want {
		if string(out[i].line()) != string(want[i].line()) {
			t.Errorf("got  %s\nwant %s", out[i].line(), want[i].line())
		}
	}
}
