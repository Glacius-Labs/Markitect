package apply

import (
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// AssignmentWorkspaceIDs reserves every possible workspace of one invocation
// before the Git container starts. A review consumes at least one Message, so
// the invocation's Message allowance bounds the number of Verify copies.
func AssignmentWorkspaceIDs(prepared CapturedSettingsAttempt, id string, maxMessages int) ([]run.WorkspaceID, error) {
	if !realizationID.MatchString(id) || maxMessages < 1 || maxMessages > 128 {
		return nil, errors.New("invalid assignment workspace reservation")
	}
	topology, err := organization.BuildTopology(prepared.Structural().Model().Checked())
	if err != nil {
		return nil, err
	}
	managers := topology.Managers()
	if len(managers) == 0 {
		return nil, errors.New("assignment topology has no Root")
	}
	ids := make([]run.WorkspaceID, 0, len(managers)+maxMessages+1)
	ids = append(ids, run.WorkspaceID(id+"-work-0"), run.WorkspaceID(id+"-assess"))
	for i := 1; i < len(managers); i++ {
		ids = append(ids, run.WorkspaceID(fmt.Sprintf("%s-work-%d", id, i)))
	}
	for i := 1; i <= maxMessages; i++ {
		ids = append(ids, run.WorkspaceID(fmt.Sprintf("%s-verify-%d", id, i)))
	}
	return ids, nil
}
