package projectcli

import "github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"

// overview remains an alias for the application result so existing CLI
// consumers can decode the public JSON shape without duplicating it here.
type overview = projectapp.Overview

func projectOverview(e env) (overview, error) {
	return e.ops.Overview(projectapp.OverviewOperation{Root: e.root, SourceRoot: e.sourceRoot})
}
