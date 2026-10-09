package projectcli

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/internal/host/projectonboarding"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

type visibleNotification struct {
	EventID          string                      `json:"eventId"`
	Revision         string                      `json:"revision"`
	CommitTime       string                      `json:"commitTime,omitempty"`
	DefinitionID     core.DefinitionIdentity     `json:"definitionId"`
	Change           string                      `json:"change"`
	Category         string                      `json:"category"`
	Severity         string                      `json:"severity"`
	AffectedManagers []string                    `json:"affectedManagers"`
	ResolutionStatus string                      `json:"resolutionStatus"`
	NextAction       string                      `json:"nextAction"`
	Resolution       *projectbriefing.Resolution `json:"resolution,omitempty"`
	commitUnix       int64
}

type managerEvent struct {
	projectbriefing.Event
	ResolutionStatus string                      `json:"resolutionStatus"`
	Resolution       *projectbriefing.Resolution `json:"resolution,omitempty"`
}

func runBriefing(opts options, out io.Writer) error {
	if opts.action == "briefings" {
		revision, err := gitHeadRevision(opts.repo)
		if err != nil {
			return err
		}
		project, err := projectwork.Load(opts.repo, revision)
		if err != nil {
			return err
		}
		if project.Config.WorkflowMode == "guided" {
			if _, err := projectbriefing.EnsureAcceptedHistory(opts.repo, revision); err != nil {
				return err
			}
		}
	}
	state, stateDigest, err := projectbriefing.Read(opts.repo)
	if err != nil {
		return err
	}
	if opts.action == "dismiss" {
		binding, err := projectbriefing.Dismiss(opts.repo, opts.event, opts.manager, opts.expect)
		if err != nil {
			return err
		}
		return writeJSON(out, struct {
			StateDigest string `json:"stateDigest"`
			Note        string `json:"note"`
		}{binding, "Dismissal affects visibility only; the event remains unresolved and available in manager briefings."})
	}
	if opts.action == "briefings" {
		if opts.manager == "" {
			notifications, hidden, err := visibleBriefings(opts.repo, state)
			if err != nil {
				return err
			}
			return writeJSON(out, struct {
				Notifications        []visibleNotification `json:"notifications"`
				HiddenDismissedCount int                   `json:"hiddenDismissedCount"`
				StateDigest          string                `json:"stateDigest"`
			}{notifications, hidden, stateDigest})
		}
		revision, err := gitHeadRevision(opts.repo)
		if err != nil {
			return fmt.Errorf("resolve the committed model revision for manager briefing: %w", err)
		}
		project, err := projectwork.Load(opts.repo, revision)
		if err != nil {
			return err
		}
		briefings, events, binding, err := projectbriefing.LoadForManager(opts.repo, project.Report.ModelDigest, opts.manager, project.Revision)
		if err != nil {
			return err
		}
		withStatus := make([]managerEvent, 0, len(events))
		for _, event := range events {
			resolution := projectbriefing.EventResolutionStatus(state, event.ID)
			withStatus = append(withStatus, managerEvent{Event: event, ResolutionStatus: resolution.Status, Resolution: resolution.Resolution})
		}
		return writeJSON(out, struct {
			Briefings     []projectbriefing.Briefing `json:"briefings"`
			Events        []managerEvent             `json:"events"`
			ContextDigest string                     `json:"contextDigest"`
			StateDigest   string                     `json:"stateDigest"`
		}{briefings, withStatus, binding, stateDigest})
	}
	bundle, err := projectbriefing.Generate(opts.repo, opts.since, opts.revision, projectbriefing.Provenance{
		DecisionReference: opts.provenance,
		Actor:             "cli-caller",
		Authority:         "explicit caller declaration; identity is not authenticated",
	})
	if err != nil {
		return err
	}
	if opts.write {
		stateDigest, err = projectbriefing.Write(opts.repo, bundle, opts.expect)
		if err != nil {
			return err
		}
	}
	return writeJSON(out, struct {
		Bundle      projectbriefing.Bundle `json:"bundle"`
		StateDigest string                 `json:"stateDigest"`
	}{bundle, stateDigest})
}

func visibleBriefings(root string, state projectbriefing.Store) ([]visibleNotification, int, error) {
	dismissed := make(map[string]map[string]bool)
	for _, item := range state.Dismissals {
		if dismissed[item.EventID] == nil {
			dismissed[item.EventID] = make(map[string]bool)
		}
		dismissed[item.EventID][item.ManagerID] = true
	}
	type eventRevision struct {
		event    projectbriefing.Event
		revision string
	}
	byID := make(map[string]eventRevision)
	for _, bundle := range state.Briefings {
		for _, event := range bundle.Events {
			if prior, ok := byID[event.ID]; ok && prior.revision != bundle.Revision {
				return nil, 0, fmt.Errorf("model event %s is attached to multiple revisions", event.ID)
			}
			byID[event.ID] = eventRevision{event: event, revision: bundle.Revision}
		}
	}
	notifications := make([]visibleNotification, 0, len(byID))
	hidden := 0
	commitTimes := make(map[string]int64)
	for _, item := range byID {
		event := item.event
		allDismissed := len(event.AffectedManagers) > 0
		for _, manager := range event.AffectedManagers {
			if !dismissed[event.ID][manager] {
				allDismissed = false
				break
			}
		}
		if allDismissed {
			hidden++
			continue
		}
		commitUnix, ok := commitTimes[item.revision]
		if !ok {
			value, err := gitCommitTime(root, item.revision)
			if err != nil {
				return nil, 0, fmt.Errorf("read accepted model commit time for %s: %w", item.revision, err)
			}
			commitUnix = value
			commitTimes[item.revision] = value
		}
		resolution := projectbriefing.EventResolutionStatus(state, event.ID)
		nextAction := "Review the declared model change and its affected responsibilities."
		if resolution.Status == "resolved" {
			nextAction = "Inspect the recorded full verification and Apply evidence."
		}
		notifications = append(notifications, visibleNotification{
			EventID: event.ID, Revision: item.revision, CommitTime: time.Unix(commitUnix, 0).UTC().Format(time.RFC3339),
			DefinitionID: event.DefinitionID, Change: event.Change, Category: event.Category, Severity: event.Severity,
			AffectedManagers: event.AffectedManagers, ResolutionStatus: resolution.Status, Resolution: resolution.Resolution,
			NextAction: nextAction, commitUnix: commitUnix,
		})
	}
	sort.Slice(notifications, func(i, j int) bool {
		left, right := severityOrder(notifications[i].Severity), severityOrder(notifications[j].Severity)
		if left != right {
			return left > right
		}
		if notifications[i].commitUnix != notifications[j].commitUnix {
			return notifications[i].commitUnix > notifications[j].commitUnix
		}
		return notifications[i].EventID < notifications[j].EventID
	})
	return notifications, hidden, nil
}

func severityOrder(value string) int {
	switch strings.ToLower(value) {
	case "critical":
		return 4
	case "error":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

func gitCommitTime(root, revision string) (int64, error) {
	output, err := source.GitOutput(root, "show", "-s", "--format=%ct", revision)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid Git commit timestamp %q", strings.TrimSpace(string(output)))
	}
	return value, nil
}

func gitHeadRevision(root string) (string, error) {
	identity, err := source.IdentifyGit(root)
	if err != nil {
		return "", err
	}
	output, err := source.GitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", err
	}
	revision := strings.TrimSpace(string(output))
	want := 40
	if identity.ObjectFormat == "sha256" {
		want = 64
	} else if identity.ObjectFormat != "sha1" {
		return "", fmt.Errorf("unsupported Git object format %q", identity.ObjectFormat)
	}
	if len(revision) != want || strings.ToLower(revision) != revision {
		return "", fmt.Errorf("Git returned an invalid full commit ID %q", revision)
	}
	if _, err := hex.DecodeString(revision); err != nil {
		return "", fmt.Errorf("Git returned a non-hexadecimal commit ID: %w", err)
	}
	return revision, nil
}

func runOnboarding(opts options, out io.Writer) error {
	var providers []projectonboarding.Provider
	switch opts.provider {
	case "codex":
		providers = []projectonboarding.Provider{projectonboarding.Codex}
	case "claude":
		providers = []projectonboarding.Provider{projectonboarding.Claude}
	case "both":
		providers = []projectonboarding.Provider{projectonboarding.Codex, projectonboarding.Claude}
	default:
		return fmt.Errorf("provider must be codex, claude or both")
	}
	project, err := projectwork.Load(opts.repo, "")
	if err != nil {
		return err
	}
	plan, err := projectonboarding.Preview(opts.repo, project.Report.ModelDigest, projectonboarding.Options{Providers: providers, DocumentationPath: opts.documentPath})
	if err != nil {
		return err
	}
	if opts.write {
		plan, err = projectonboarding.Apply(opts.repo, plan, opts.expect)
		if err != nil {
			return err
		}
	}
	return writeJSON(out, plan)
}
