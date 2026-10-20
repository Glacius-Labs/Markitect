package cockpit

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// Detail is every record up to the boundary that concerns one reference, with
// its complete data. The browser renders the fields it knows and shows the
// others as recorded.
type Detail struct {
	Ref     string      `json:"ref"`
	Records []RawRecord `json:"records"`
}

// RawRecord is one record with its data as recorded.
type RawRecord struct {
	Seq     int             `json:"seq"`
	At      time.Time       `json:"at,omitzero"`
	Kind    string          `json:"kind"`
	Pass    string          `json:"pass,omitempty"`
	Manager string          `json:"manager,omitempty"`
	Turn    string          `json:"turn,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Invalid string          `json:"invalid,omitempty"`
}

// maxManagerRecords bounds a Manager's detail to its latest records.
const maxManagerRecords = 400

// Select returns the records among the first n that concern ref:
// proposal:ID, finding:ID, question:ID, problem:ID, turn:ID, pass:ID,
// check:PASS/NAME, manager:PATH or seq:N. An unknown reference selects nothing.
func Select(records []Record, n int, ref string) Detail {
	if n < 0 || n > len(records) {
		n = len(records)
	}
	kind, id, _ := strings.Cut(ref, ":")
	d := Detail{Ref: ref, Records: []RawRecord{}}
	field := func(r Record, key string) string {
		var m map[string]json.RawMessage
		if json.Unmarshal(r.Data, &m) != nil {
			return ""
		}
		var s string
		_ = json.Unmarshal(m[key], &s)
		return s
	}
	var match func(Record) bool
	switch kind {
	case "proposal":
		match = func(r Record) bool {
			switch r.Kind {
			case "proposal.submitted", "proposal.checked", "proposal.invalidated", "proposal.withdrawn",
				"basis.accepted", "basis.returned", "basis.invalidated", "result.accepted", "result.returned", "result.reopened":
				return field(r, "proposal") == id
			}
			return false
		}
	case "finding":
		match = func(r Record) bool { return strings.HasPrefix(r.Kind, "finding.") && field(r, "id") == id }
	case "question":
		match = func(r Record) bool { return strings.HasPrefix(r.Kind, "question.") && field(r, "id") == id }
	case "problem":
		match = func(r Record) bool { return r.Kind == "problem.raised" && field(r, "id") == id }
	case "turn":
		match = func(r Record) bool { return r.Turn == id }
	case "pass":
		match = func(r Record) bool {
			switch r.Kind {
			case "pass.start", "pass.sealed", "gate", "candidate", "check", "registry.view":
				return r.Pass == id
			}
			return false
		}
	case "check":
		pass, name, _ := strings.Cut(id, "/")
		match = func(r Record) bool { return r.Kind == "check" && r.Pass == pass && field(r, "name") == name }
	case "manager":
		match = func(r Record) bool { return r.Manager == id && r.Kind != "manager.state_changed" }
	case "seq":
		seq, err := strconv.Atoi(id)
		match = func(r Record) bool { return err == nil && r.Seq == seq }
	default:
		return d
	}
	for _, r := range records[:n] {
		if match(r) {
			d.Records = append(d.Records, RawRecord{Seq: r.Seq, At: r.At, Kind: r.Kind, Pass: r.Pass, Manager: r.Manager, Turn: r.Turn, Data: r.Data, Invalid: r.Invalid})
		}
	}
	if kind == "manager" && len(d.Records) > maxManagerRecords {
		d.Records = d.Records[len(d.Records)-maxManagerRecords:]
	}
	return d
}
