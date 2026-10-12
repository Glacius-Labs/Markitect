package projecthistory

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectbriefing"
)

func loadManagerView(root, revision, managerID string) (projectbriefing.ManagerView, error) {
	project, err := projectwork.Load(root, revision)
	if err != nil {
		return projectbriefing.ManagerView{}, fmt.Errorf("load manager briefing visibility at %s: %w", revision, err)
	}
	view, err := projectbriefing.NewManagerView(project.Report, managerID)
	if err != nil {
		return projectbriefing.ManagerView{}, fmt.Errorf("load manager briefing visibility at %s: %w", revision, err)
	}
	return view, nil
}

func emptyManagerView() projectbriefing.ManagerView {
	return projectbriefing.EmptyManagerView()
}
