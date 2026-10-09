package projectrun

import (
	"fmt"
	"sort"
	"strings"
)

func newLocalObligations(managerID string, questions, risks []string) ([]Obligation, error) {
	var records []Obligation
	for _, item := range []struct {
		kind   string
		values []string
	}{{"question", questions}, {"risk", risks}} {
		seen := map[string]bool{}
		for _, value := range item.values {
			if strings.TrimSpace(value) == "" || seen[value] {
				return nil, fmt.Errorf("empty or duplicate %s", item.kind)
			}
			seen[value] = true
			id, err := newID()
			if err != nil {
				return nil, err
			}
			records = append(records, Obligation{ID: id, Kind: item.kind, Text: value, OriginManager: managerID, Chain: []string{managerID}})
		}
	}
	return sortObligations(records), nil
}

func newLocalResponseObligations(managerID string, carried []Obligation, questions, risks []string) ([]Obligation, error) {
	carriedByKindAndText := map[string]bool{}
	for _, record := range carried {
		carriedByKindAndText[record.Kind+"\x00"+record.Text] = true
	}
	var newQuestions, newRisks []string
	for _, item := range []struct {
		kind   string
		values []string
		output *[]string
	}{{"question", questions, &newQuestions}, {"risk", risks, &newRisks}} {
		seen := map[string]bool{}
		for _, value := range item.values {
			if strings.TrimSpace(value) == "" || seen[value] {
				return nil, fmt.Errorf("empty or duplicate %s in integration response", item.kind)
			}
			seen[value] = true
			if !carriedByKindAndText[item.kind+"\x00"+value] {
				*item.output = append(*item.output, value)
			}
		}
	}
	return newLocalObligations(managerID, newQuestions, newRisks)
}

func managerObligationRecords(parent ManagerTask, tasks []ManagerTask, children []string) ([]Obligation, error) {
	records, err := validatedTaskObligations(parent, tasks)
	if err != nil {
		return nil, err
	}
	for _, childID := range children {
		child := findTask(tasks, childID)
		if child == nil || child.ParentTask != parent.ManagerID {
			return nil, fmt.Errorf("obligation source %s is not an active direct child", childID)
		}
		childRecords, err := validatedTaskObligations(*child, tasks)
		if err != nil {
			return nil, err
		}
		records = append(records, childRecords...)
	}
	byID := map[string]Obligation{}
	for _, record := range records {
		if prior, exists := byID[record.ID]; exists {
			if prior.Kind != record.Kind || prior.Text != record.Text || prior.OriginManager != record.OriginManager {
				return nil, fmt.Errorf("obligation ID %q has conflicting immutable provenance", record.ID)
			}
			switch {
			case stringSlicePrefix(prior.Chain, record.Chain):
				byID[record.ID] = cloneObligation(record)
			case stringSlicePrefix(record.Chain, prior.Chain):
				// Keep the already stored longer forwarding chain.
			default:
				return nil, fmt.Errorf("obligation ID %q has divergent forwarding chains", record.ID)
			}
			continue
		}
		byID[record.ID] = cloneObligation(record)
	}
	byText := map[string]string{}
	sortedRecords := make([]Obligation, 0, len(byID))
	for _, record := range byID {
		sortedRecords = append(sortedRecords, record)
	}
	sortedRecords = sortObligations(sortedRecords)
	for _, record := range sortedRecords {
		key := record.Kind + "\x00" + record.Text
		if prior, exists := byText[key]; exists && prior != record.ID {
			return nil, fmt.Errorf("%s %q is ambiguous between provenance records %s and %s", record.Kind, record.Text, prior, record.ID)
		}
		byText[key] = record.ID
	}
	return sortedRecords, nil
}

func validatedTaskObligations(task ManagerTask, tasks []ManagerTask) ([]Obligation, error) {
	if len(task.Questions)+len(task.Risks) == 0 {
		if len(task.Obligations) != 0 {
			return nil, fmt.Errorf("Manager %s has stale obligation provenance without outstanding questions or risks", task.ManagerID)
		}
		return nil, nil
	}
	if len(task.Obligations) == 0 {
		return nil, fmt.Errorf("Manager %s has legacy questions or risks without provenance", task.ManagerID)
	}
	questions, risks := map[string]int{}, map[string]int{}
	for _, value := range task.Questions {
		questions[value]++
	}
	for _, value := range task.Risks {
		risks[value]++
	}
	seenIDs := map[string]bool{}
	records := cloneObligations(task.Obligations)
	for _, record := range records {
		if record.ID == "" || seenIDs[record.ID] || record.OriginManager == "" || strings.TrimSpace(record.Text) == "" {
			return nil, fmt.Errorf("Manager %s has empty or duplicate obligation provenance", task.ManagerID)
		}
		seenIDs[record.ID] = true
		if record.Kind == "question" {
			questions[record.Text]--
		} else if record.Kind == "risk" {
			risks[record.Text]--
		} else {
			return nil, fmt.Errorf("Manager %s has unknown obligation kind %q", task.ManagerID, record.Kind)
		}
		if err := validateObligationChain(record, task.ManagerID, tasks); err != nil {
			return nil, err
		}
	}
	for _, count := range questions {
		if count != 0 {
			return nil, fmt.Errorf("Manager %s question text does not match its provenance", task.ManagerID)
		}
	}
	for _, count := range risks {
		if count != 0 {
			return nil, fmt.Errorf("Manager %s risk text does not match its provenance", task.ManagerID)
		}
	}
	return sortObligations(records), nil
}

func validateObligationChain(record Obligation, owner string, tasks []ManagerTask) error {
	if len(record.Chain) == 0 || record.Chain[0] != record.OriginManager || record.Chain[len(record.Chain)-1] != owner {
		return fmt.Errorf("obligation %s has invalid origin or terminal forwarding Manager", record.ID)
	}
	seen := map[string]bool{}
	for i, managerID := range record.Chain {
		if seen[managerID] {
			return fmt.Errorf("obligation %s repeats Manager %s in its forwarding chain", record.ID, managerID)
		}
		seen[managerID] = true
		manager := findTaskByManager(tasks, managerID)
		if manager == nil {
			return fmt.Errorf("obligation %s references inactive Manager %s", record.ID, managerID)
		}
		if i > 0 {
			previous := findTaskByManager(tasks, record.Chain[i-1])
			if previous == nil || previous.ParentTask != manager.ManagerID {
				return fmt.Errorf("obligation %s forwarding chain is not a Manager parent chain", record.ID)
			}
		}
	}
	return nil
}

func resolveObligationRecords(outstanding []Obligation, resolvedQuestions, resolvedRisks []string) ([]Obligation, []Obligation, error) {
	resolved := map[string]bool{}
	var selected []Obligation
	for _, request := range []struct {
		kind   string
		values []string
	}{{"question", resolvedQuestions}, {"risk", resolvedRisks}} {
		seen := map[string]bool{}
		for _, text := range request.values {
			if seen[text] {
				return nil, nil, fmt.Errorf("resolution duplicates %q", text)
			}
			seen[text] = true
			var match *Obligation
			for i := range outstanding {
				if outstanding[i].Kind == request.kind && outstanding[i].Text == text {
					if match != nil {
						return nil, nil, fmt.Errorf("resolution for %s %q is ambiguous", request.kind, text)
					}
					match = &outstanding[i]
				}
			}
			if match == nil {
				return nil, nil, fmt.Errorf("resolution invents or repeats %s %q", request.kind, text)
			}
			resolved[match.ID] = true
			selected = append(selected, cloneObligation(*match))
		}
	}
	var remaining []Obligation
	for _, record := range outstanding {
		if !resolved[record.ID] {
			remaining = append(remaining, cloneObligation(record))
		}
	}
	return sortObligations(selected), sortObligations(remaining), nil
}

func obligationTexts(records []Obligation) ([]string, []string) {
	var questions, risks []string
	for _, record := range records {
		switch record.Kind {
		case "question":
			questions = append(questions, record.Text)
		case "risk":
			risks = append(risks, record.Text)
		}
	}
	return uniqueSorted(questions), uniqueSorted(risks)
}

func appendForwardedObligations(records []Obligation, managerID string) []Obligation {
	forwarded := cloneObligations(records)
	for i := range forwarded {
		if len(forwarded[i].Chain) == 0 || forwarded[i].Chain[len(forwarded[i].Chain)-1] != managerID {
			forwarded[i].Chain = append(forwarded[i].Chain, managerID)
		}
	}
	return sortObligations(forwarded)
}

func applyObligationResolutions(report *RunReport, resolved []Obligation) error {
	if len(resolved) == 0 {
		return nil
	}
	resolvedIDs := map[string]Obligation{}
	for _, record := range resolved {
		if prior, exists := resolvedIDs[record.ID]; exists && (prior.Kind != record.Kind || prior.Text != record.Text || prior.OriginManager != record.OriginManager || !equalStrings(prior.Chain, record.Chain)) {
			return fmt.Errorf("resolved obligation ID %q has conflicting provenance", record.ID)
		}
		terminalManager := ""
		if len(record.Chain) > 0 {
			terminalManager = record.Chain[len(record.Chain)-1]
		}
		if err := validateObligationChain(record, terminalManager, report.Tasks); err != nil {
			return err
		}
		for i, managerID := range record.Chain {
			task := findTaskByManager(report.Tasks, managerID)
			if task == nil {
				return fmt.Errorf("resolved obligation %s has missing forwarding Manager %s", record.ID, managerID)
			}
			var found *Obligation
			for j := range task.Obligations {
				if task.Obligations[j].ID == record.ID {
					if found != nil {
						return fmt.Errorf("Manager %s has duplicate obligation ID %s", managerID, record.ID)
					}
					found = &task.Obligations[j]
				}
			}
			if found == nil || found.Kind != record.Kind || found.Text != record.Text || found.OriginManager != record.OriginManager || !equalStrings(found.Chain, record.Chain[:i+1]) {
				return fmt.Errorf("resolved obligation %s lacks matching provenance at Manager %s", record.ID, managerID)
			}
		}
		for i := range report.Tasks {
			for _, existing := range report.Tasks[i].Obligations {
				if existing.ID == record.ID && !containsString(record.Chain, report.Tasks[i].ManagerID) {
					return fmt.Errorf("obligation ID %s collides outside its forwarding chain at Manager %s", record.ID, report.Tasks[i].ManagerID)
				}
			}
		}
		resolvedIDs[record.ID] = record
	}
	for i := range report.Tasks {
		task := &report.Tasks[i]
		var kept []Obligation
		changed := false
		for _, record := range task.Obligations {
			if _, ok := resolvedIDs[record.ID]; ok {
				changed = true
				continue
			}
			kept = append(kept, cloneObligation(record))
		}
		if !changed {
			continue
		}
		task.Obligations = sortObligations(kept)
		task.Questions, task.Risks = obligationTexts(task.Obligations)
		if len(task.Obligations) == 0 && task.ReportStatus == "partial" {
			task.ReportStatus = "complete"
		}
	}
	for i := range report.Escalations {
		escalation := &report.Escalations[i]
		if len(escalation.ObligationIDs) == 0 || escalation.Status == "resolved" {
			continue
		}
		from := findTaskByManager(report.Tasks, escalation.FromManager)
		active := map[string]bool{}
		if from != nil {
			for _, record := range from.Obligations {
				active[record.ID] = true
			}
		}
		allResolved := true
		for _, id := range escalation.ObligationIDs {
			if active[id] {
				allResolved = false
				break
			}
		}
		if allResolved {
			escalation.Status = "resolved"
		}
	}
	return nil
}

func markObligationEscalationsEscalated(report *RunReport, recipient string, obligations []Obligation) {
	active := map[string]bool{}
	for _, record := range obligations {
		active[record.ID] = true
	}
	for i := range report.Escalations {
		escalation := &report.Escalations[i]
		if escalation.Status != "open" || escalation.ToManager != recipient {
			continue
		}
		for _, id := range escalation.ObligationIDs {
			if active[id] {
				escalation.Status = "escalated"
				break
			}
		}
	}
}

func validateReportClosure(report RunReport) error {
	for _, task := range report.Tasks {
		if (task.ReportStatus != "complete" && task.ReportStatus != "no-op") || len(task.Questions) != 0 || len(task.Risks) != 0 || len(task.Obligations) != 0 {
			return fmt.Errorf("manager %s has unresolved report status or obligations", task.ManagerID)
		}
	}
	for _, escalation := range report.Escalations {
		if escalation.Status != "resolved" {
			return fmt.Errorf("unresolved escalation %s from %s", escalation.ID, escalation.FromManager)
		}
	}
	return nil
}

func sortObligations(records []Obligation) []Obligation {
	cloned := cloneObligations(records)
	sort.Slice(cloned, func(i, j int) bool {
		if cloned[i].Kind != cloned[j].Kind {
			return cloned[i].Kind < cloned[j].Kind
		}
		if cloned[i].Text != cloned[j].Text {
			return cloned[i].Text < cloned[j].Text
		}
		return cloned[i].ID < cloned[j].ID
	})
	return cloned
}

func cloneObligations(records []Obligation) []Obligation {
	cloned := make([]Obligation, len(records))
	for i, record := range records {
		cloned[i] = cloneObligation(record)
	}
	return cloned
}

func cloneObligation(record Obligation) Obligation {
	record.Chain = append([]string(nil), record.Chain...)
	return record
}

func stringSlicePrefix(prefix, value []string) bool {
	if len(prefix) > len(value) {
		return false
	}
	for i := range prefix {
		if prefix[i] != value[i] {
			return false
		}
	}
	return true
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func findTaskByManager(tasks []ManagerTask, managerID string) *ManagerTask {
	for i := range tasks {
		if tasks[i].ManagerID == managerID {
			return &tasks[i]
		}
	}
	return nil
}
