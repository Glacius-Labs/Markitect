package remoteserver

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// StartFrom moves a working Manager's branch, still at the base, to a state
// its parent's branch held in this Run, before the Manager starts (concept
// 2.0 §6.2: a parent with own items submits first, and its children start
// from its approved commit). Only the host calls it; no agent can. The
// decision log records it as user host, kind start.
func StartFrom(l Layout, git, branch, commit string) error {
	plan, err := LoadPlan(l)
	if err != nil {
		return err
	}
	var m runremote.Manager
	for _, c := range plan.Managers {
		if c.Working && c.ID != model.RootPackage && runremote.Branch(c.ID) == branch {
			m = c
		}
	}
	if m.ID == "" {
		return fmt.Errorf("%s is no branch of a working Manager below Root", branch)
	}
	repo := gitRepo{git: git, dir: l.Repo(), env: append(gitConfigEnv(), "PATH="+os.Getenv("PATH"), "HOME="+l.Root)}
	tips, err := repo.Tips(branch)
	if err != nil {
		return err
	}
	if len(tips) != 1 || tips[0] != plan.Base {
		return fmt.Errorf("%s has moved from the base; a Manager starts from its parent's state only once, before it works", branch)
	}
	parent, err := repo.Tips(runremote.Branch(model.ParentOf(m.ID)))
	if err != nil {
		return err
	}
	if !slices.Contains(parent, commit) {
		return fmt.Errorf("%s is no state of the parent's branch in this Run", commit)
	}
	// The old value makes the update fail if the branch moved meanwhile.
	if _, err := repo.run("update-ref", "-m", "markitect: start from the parent's state", "refs/heads/"+branch, commit, plan.Base); err != nil {
		return err
	}
	return appendLog(l, Entry{Time: time.Now().UTC().Format(time.RFC3339Nano), User: "host",
		Updates:  []runremote.Update{{Old: plan.Base, New: commit, Ref: "refs/heads/" + branch}},
		Decision: runremote.Decision{Accepted: true, Kind: "start", Manager: runremote.Name(m.ID)}})
}
