package state

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// FactoryFigures are the figures of how the factory runs, computed from the
// results of control samples, the history and the reports. They read no
// clock, so the same files give the same figures anywhere.
type FactoryFigures struct {
	// Samples is the hit rate of each control sample, by name; Planted and
	// Hit sum them.
	Samples      []SampleRate
	Planted, Hit int
	// Reverdicts are the verdicts on a document whose inputs an earlier
	// verdict on it had too; Contradictions those of them whose verdict
	// differs from that earlier one.
	Reverdicts     int
	Contradictions []Contradiction
	// Reports, Open count the reports and the open ones; Days the reports
	// raised and closed each day a report was raised or closed, and how many
	// were open at its end.
	Reports, Open int
	Days          []ReportDay
	// Areas are the reports per area: the top folder of a place below the
	// spec folder, "(spec)" for a place at its top, or "artifacts".
	Areas []Area
	// Recurring are the reports that recur: duplicates of one report, and
	// several reports at one place.
	Recurring []Recurring
	// Reference is the project's own reference for each figure.
	Reference map[string]string
}

// The figures of the factory, in the order figures shows them.
var FactorySignals = []string{"hit-rate", "contradictions", "open-reports", "findings-per-area", "recurring-reports", "stopped-jobs"}

// StoppedJobs is why figures does not count the jobs that stopped without
// progress.
const StoppedJobs = "not computed: no report says yet that a job stopped without progress"

// SampleRate is the hit rate of one control sample: the planted documents
// that got fail, out of all planted, and those that did not.
type SampleRate struct {
	Name, Day    string
	Planted, Hit int
	Missed       []string
}

// Contradiction is a verdict that differs from the earlier verdict on the
// same document at the same inputs.
type Contradiction struct {
	Doc, Was, Now string
	Time          time.Time
}

// ReportDay is one day on which reports were raised or closed.
type ReportDay struct {
	Day                  string
	Raised, Closed, Open int
}

// Area is the number of reports, and of open ones, at one area.
type Area struct {
	Name          string
	Reports, Open int
}

// Recurring is a report with its duplicates, or a place with several
// reports.
type Recurring struct {
	// Kind is "duplicates" or "place".
	Kind    string
	Place   string
	Reports []string
	Detail  string
}

// ComputeFactoryFigures computes the figures of the factory from the files
// of the factory and the history of the project p.
func ComputeFactoryFigures(p *Project, f *Factory, events []Event) *FactoryFigures {
	ff := &FactoryFigures{Reference: map[string]string{}}
	ff.hitRate(f.Samples)
	ff.spotChecks(events)
	ff.reports(p, f.Reports)
	ff.Reference["stopped-jobs"] = StoppedJobs
	return ff
}

// hitRate counts, per sample and over all, the planted documents that got
// fail.
func (ff *FactoryFigures) hitRate(samples []*Sample) {
	for _, s := range samples {
		r := SampleRate{Name: s.Name, Day: s.Day, Planted: len(s.Planted), Missed: []string{}}
		for _, doc := range s.Planted {
			if s.Verdicts[doc] == VerdictFail {
				r.Hit++
			} else {
				r.Missed = append(r.Missed, doc)
			}
		}
		ff.Samples = append(ff.Samples, r)
		ff.Planted += r.Planted
		ff.Hit += r.Hit
	}
	sort.Slice(ff.Samples, func(i, j int) bool { return ff.Samples[i].Name < ff.Samples[j].Name })
	ff.Reference["hit-rate"] = fmt.Sprintf("%s over %s", rate(ff.Hit, ff.Planted, "planted document"), count(len(samples), "sample"))
}

// spotChecks compares each verdict that names its inputs with the latest
// earlier verdict on the same document at the same inputs. Verdicts written
// before the history recorded inputs do not count.
func (ff *FactoryFigures) spotChecks(events []Event) {
	last := map[string]string{}
	with := 0
	for _, e := range events {
		if e.Type != EventVerdict || e.Inputs == "" {
			continue
		}
		with++
		key := e.Doc + "\x00" + e.Inputs
		if was, ok := last[key]; ok {
			ff.Reverdicts++
			if was != e.Verdict {
				ff.Contradictions = append(ff.Contradictions, Contradiction{Doc: e.Doc, Was: was, Now: e.Verdict, Time: e.Time})
			}
		}
		last[key] = e.Verdict
	}
	ff.Reference["contradictions"] = fmt.Sprintf("%s at unchanged inputs, of %s that name their inputs",
		count(ff.Reverdicts, "re-verdict"), count(with, "verdict"))
}

// reports counts the reports by day, by area and where they recur.
func (ff *FactoryFigures) reports(p *Project, reports []*Report) {
	days := map[string]*ReportDay{}
	at := func(d string) *ReportDay {
		if days[d] == nil {
			days[d] = &ReportDay{Day: d}
		}
		return days[d]
	}
	areas := map[string]*Area{}
	places := map[string][]string{}
	duplicates := map[string][]string{}
	for _, r := range reports {
		ff.Reports++
		if r.Open() {
			ff.Open++
		}
		if r.Raised != "" {
			at(r.Raised).Raised++
		}
		if r.ClosedOn != "" {
			at(r.ClosedOn).Closed++
		}
		if r.Place != "" {
			name := areaOf(p, r.Place)
			if areas[name] == nil {
				areas[name] = &Area{Name: name}
			}
			areas[name].Reports++
			if r.Open() {
				areas[name].Open++
			}
			places[r.Place] = append(places[r.Place], r.Name)
		}
		if r.Outcome == ClosedDuplicate {
			duplicates[r.DuplicateOf] = append(duplicates[r.DuplicateOf], r.Name)
		}
	}
	var order []string
	for d := range days {
		order = append(order, d)
	}
	sort.Strings(order)
	open := 0
	var counts []int
	for _, d := range order {
		day := days[d]
		open += day.Raised - day.Closed
		day.Open = open
		counts = append(counts, open)
		ff.Days = append(ff.Days, *day)
	}
	ff.Reference["open-reports"] = fmt.Sprintf("%s, %d open; open at the end of a day with reports raised or closed: %s",
		count(ff.Reports, "report"), ff.Open, spreadOf(counts))
	var perArea []int
	for _, a := range areas {
		ff.Areas = append(ff.Areas, *a)
		perArea = append(perArea, a.Reports)
	}
	sort.Slice(ff.Areas, func(i, j int) bool {
		if ff.Areas[i].Reports != ff.Areas[j].Reports {
			return ff.Areas[i].Reports > ff.Areas[j].Reports
		}
		return ff.Areas[i].Name < ff.Areas[j].Name
	})
	ff.Reference["findings-per-area"] = fmt.Sprintf("%s; reports per area: median %s", count(len(areas), "area"), median(perArea))
	for original, dups := range duplicates {
		sort.Strings(dups)
		place := ""
		for _, r := range reports {
			if r.Name == original {
				place = r.Place
			}
		}
		ff.Recurring = append(ff.Recurring, Recurring{Kind: "duplicates", Place: place, Reports: append([]string{original}, dups...),
			Detail: fmt.Sprintf("%s recurred %s: %s", original, count(len(dups), "time"), strings.Join(dups, ", "))})
	}
	for place, names := range places {
		if len(names) < 2 {
			continue
		}
		sort.Strings(names)
		ff.Recurring = append(ff.Recurring, Recurring{Kind: "place", Place: place, Reports: names,
			Detail: fmt.Sprintf("%s: %s: %s", place, count(len(names), "report"), strings.Join(names, ", "))})
	}
	sort.Slice(ff.Recurring, func(i, j int) bool {
		a, b := ff.Recurring[i], ff.Recurring[j]
		if len(a.Reports) != len(b.Reports) {
			return len(a.Reports) > len(b.Reports)
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return strings.Join(a.Reports, "\n") < strings.Join(b.Reports, "\n")
	})
	ff.Reference["recurring-reports"] = fmt.Sprintf("%s at %s", count(ff.Reports, "report"), count(len(places), "place"))
}

// areaOf is the area of a place: the top folder of a place below the spec
// folder, "(spec)" for a place at the spec folder's top, or "artifacts".
func areaOf(p *Project, place string) string {
	rel, ok := p.specFile(place)
	switch {
	case place == p.SpecFolder:
		return "(spec)"
	case !ok:
		return "artifacts"
	case !strings.Contains(rel, "/") && !isFolder(p.FS, place):
		return "(spec)"
	}
	return strings.SplitN(rel, "/", 2)[0]
}

// rate is "hit of all noun", with the percentage.
func rate(hit, all int, noun string) string {
	if all == 0 {
		return "no " + noun + "s"
	}
	return fmt.Sprintf("%d of %s (%d%%)", hit, count(all, noun), hit*100/all)
}

// spreadOf is the median and range of a list of numbers.
func spreadOf(values []int) string {
	if len(values) == 0 {
		return "none"
	}
	lo, hi := values[0], values[0]
	for _, v := range values {
		lo, hi = min(lo, v), max(hi, v)
	}
	return fmt.Sprintf("median %s, range %d–%d", median(values), lo, hi)
}
