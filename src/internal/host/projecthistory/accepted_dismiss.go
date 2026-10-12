package projecthistory

import (
	"errors"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectbriefing"
)

// DismissAcceptedHistory records a local visibility choice for an event on
// the accepted first-parent history. expectedDigest is the prospective store
// digest returned by ReadAcceptedHistory or a guided brief list. The accepted
// history and dismissal are persisted together only if the backing store still
// matches the digest observed while computing that history.
func DismissAcceptedHistory(root, targetRevision, eventID, managerID, expectedDigest string) (string, error) {
	if strings.TrimSpace(eventID) == "" || strings.TrimSpace(managerID) == "" {
		return "", ErrDismissal
	}
	active, next, receipt, persistedDigest, err := acceptedDismissalPreview(root, targetRevision)
	if err != nil {
		return "", err
	}
	return dismissAcceptedHistoryFrom(root, active, next, receipt, persistedDigest, eventID, managerID, expectedDigest)
}

func acceptedDismissalPreview(root, targetRevision string) (*line, projectbriefing.Store, projectbriefing.EnsureReceipt, string, error) {
	active, err := activeLine(root)
	if err != nil {
		return nil, projectbriefing.Store{}, projectbriefing.EnsureReceipt{}, "", err
	}
	stored, persistedDigest, err := readStore(root)
	if err != nil {
		return nil, projectbriefing.Store{}, projectbriefing.EnsureReceipt{}, "", err
	}
	next, receipt, err := acceptHistory(root, active, targetRevision, stored)
	if err != nil {
		return nil, projectbriefing.Store{}, projectbriefing.EnsureReceipt{}, "", err
	}
	receipt.StoreDigest = projectbriefing.StoreDigest(next)
	return active, next, receipt, persistedDigest, nil
}

func dismissAcceptedHistoryFrom(root string, active *line, next projectbriefing.Store, receipt projectbriefing.EnsureReceipt, persistedDigest, eventID, managerID, expectedDigest string) (string, error) {
	if strings.TrimSpace(eventID) == "" || strings.TrimSpace(managerID) == "" {
		return "", ErrDismissal
	}
	if receipt.StoreDigest != expectedDigest {
		return "", ErrStaleStore
	}
	accepted := active.accepted(next)
	found := false
	for _, bundle := range accepted.Briefings {
		for _, event := range bundle.Events {
			if event.ID == eventID && contains(event.AffectedManagers, managerID) {
				found = true
			}
		}
	}
	if !found {
		return "", errors.New("event is not available to the selected manager")
	}
	alreadyDismissed := false
	for _, dismissal := range accepted.Dismissals {
		if dismissal.EventID == eventID && dismissal.ManagerID == managerID {
			alreadyDismissed = true
			break
		}
	}
	if !alreadyDismissed {
		next.Dismissals = append(next.Dismissals, projectbriefing.Dismissal{EventID: eventID, ManagerID: managerID})
		sort.Slice(next.Dismissals, func(i, j int) bool {
			if next.Dismissals[i].EventID != next.Dismissals[j].EventID {
				return next.Dismissals[i].EventID < next.Dismissals[j].EventID
			}
			return next.Dismissals[i].ManagerID < next.Dismissals[j].ManagerID
		})
	}
	return update(root, persistedDigest, func(current *projectbriefing.Store) error {
		*current = next
		return nil
	})
}
