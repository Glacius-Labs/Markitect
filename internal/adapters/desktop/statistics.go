package desktop

import (
	"context"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Statistics shows how the model develops: its size and what check and lint
// find in it, at each commit that changed Markitect's folder and in the
// working tree. The Runs' part comes with the Runs.

// ModelCommit is one commit that changed Markitect's folder.
type ModelCommit struct {
	ID      string
	When    time.Time
	Subject string
	// Landed: the commit adds a Run's Result page.
	Landed bool
}

// statsCommits bounds how far back Statistics reads the model.
const statsCommits = 60

// statPoint is the model at one commit, or in the working tree.
type statPoint struct {
	Commit      string    `json:"commit,omitempty"` // empty for the working tree
	When        time.Time `json:"when"`
	Subject     string    `json:"subject,omitempty"`
	Landed      bool      `json:"landed,omitempty"`
	Initialized bool      `json:"initialized"`
	Definitions int       `json:"definitions"`
	Packages    int       `json:"packages"`
	Warnings    int       `json:"warnings"`
	Errors      int       `json:"errors"`
}

// statsCache keeps each commit's numbers: a commit never changes.
type statsCache struct {
	mu     sync.Mutex
	points map[string]statPoint // by repository root and commit
}

// maxStats bounds the cached commits, across all projects; the cache starts
// over when full.
const maxStats = 5000

func count(p *project.Project) statPoint {
	pt := statPoint{Initialized: slices.Contains(p.Files, project.ManifestFile), Errors: len(p.Errors)}
	if p.Model != nil {
		pt.Definitions = len(p.Model.ModelDefinitions())
		pt.Packages = len(p.Model.Packages)
		pt.Warnings = len(analysis.Lint(p))
	}
	return pt
}

// at returns the numbers of one commit.
func (c *statsCache) at(ctx context.Context, repos Repos, root string, mc ModelCommit) (statPoint, error) {
	key := root + "\x00" + mc.ID
	c.mu.Lock()
	pt, ok := c.points[key]
	c.mu.Unlock()
	if !ok {
		snap, err := repos.Snapshot(ctx, root, mc.ID)
		if err != nil {
			return statPoint{}, err
		}
		p, err := project.Load(snap)
		if err != nil {
			return statPoint{}, err
		}
		// A read cut short by an ended request would stay wrong.
		if ctx.Err() != nil {
			return statPoint{}, ctx.Err()
		}
		pt = count(p)
		c.mu.Lock()
		if len(c.points) >= maxStats {
			clear(c.points)
		}
		c.points[key] = pt
		c.mu.Unlock()
	}
	pt.Commit, pt.When, pt.Subject, pt.Landed = mc.ID, mc.When, mc.Subject, mc.Landed
	return pt, nil
}

// statisticsRoute answers GET /api/projects/{id}/statistics: the model at
// each of the last commits that changed it, oldest first, and in the
// working tree.
func (s *server) statisticsRoute(w http.ResponseWriter, r *http.Request) {
	p, ok := s.find(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	ws, err := s.repos.Open(ctx, p.Path)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	commits, err := s.repos.History(ctx, ws.Root(), statsCommits)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	points := []statPoint{}
	for i := len(commits) - 1; i >= 0; i-- {
		pt, err := s.stats.at(ctx, s.repos, ws.Root(), commits[i])
		if err != nil {
			fail(w, http.StatusUnprocessableEntity, err)
			return
		}
		points = append(points, pt)
	}
	cur, err := project.Load(ws)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	now := count(cur)
	now.When = time.Now().UTC()
	points = append(points, now)
	reply(w, map[string]any{"model": points, "limit": statsCommits})
}
