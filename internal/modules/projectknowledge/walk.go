package projectknowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

type WalkRequest struct {
	Start         string `json:"start"`
	Reverse       bool   `json:"reverse,omitempty"`
	Bidirectional bool   `json:"bidirectional,omitempty"`
	MaxDepth      int    `json:"maxDepth"`
	MaxSteps      int    `json:"maxSteps"`
	MaxResults    int    `json:"maxResults"`
}
type Witness struct {
	Node string `json:"node"`
	Path []Edge `json:"path"`
}
type WalkResult struct {
	Binding      Binding   `json:"binding"`
	QueryVersion string    `json:"queryVersion"`
	QueryDigest  string    `json:"queryDigest"`
	Results      []Witness `json:"results"`
	Steps        int       `json:"steps"`
	Complete     bool      `json:"complete"`
	Reason       string    `json:"reason,omitempty"`
}

// Walk returns one deterministic shortest witness per reached node. Result
// truncation is explicit; this is not an enumeration of all possible paths.
// Reverse traversal follows incoming edges. Bidirectional traversal follows
// either direction and cannot be combined with Reverse. Witness Edge values
// always retain their original From/To orientation; for bidirectional paths,
// traversal direction is determined relative to the preceding witness node.
func (i *Index) Walk(q WalkRequest) (WalkResult, error) {
	if _, ok := i.byID[q.Start]; !ok {
		return WalkResult{}, ErrNotFound
	}
	if q.Reverse && q.Bidirectional {
		return WalkResult{}, errors.New("reverse and bidirectional traversal are mutually exclusive")
	}
	if q.MaxDepth <= 0 || q.MaxDepth > maxDepth || q.MaxSteps <= 0 || q.MaxSteps > maxSteps || q.MaxResults <= 0 || q.MaxResults > maxResults {
		return WalkResult{}, errors.New("walk limits are invalid or exceed hard caps")
	}
	qb, _ := json.Marshal(q)
	sum := sha256.Sum256(append([]byte(QueryVersion+"\x00"), qb...))
	res := WalkResult{Binding: i.binding, QueryVersion: QueryVersion, QueryDigest: "sha256:" + hex.EncodeToString(sum[:]), Results: []Witness{{Node: q.Start, Path: []Edge{}}}, Complete: true}
	type item struct {
		id    string
		path  []Edge
		depth int
	}
	queue := []item{{id: q.Start, path: []Edge{}, depth: 0}}
	visited := map[string]bool{q.Start: true}
	for head := 0; head < len(queue); head++ {
		cur := queue[head]
		type traversal struct {
			edgeID int
			next   string
		}
		var steps []traversal
		if q.Bidirectional {
			seen := make(map[int]bool, len(i.out[cur.id])+len(i.in[cur.id]))
			for _, ei := range i.out[cur.id] {
				seen[ei] = true
				steps = append(steps, traversal{edgeID: ei, next: i.edges[ei].To})
			}
			for _, ei := range i.in[cur.id] {
				if seen[ei] { // A self-loop has only one adjacent node.
					continue
				}
				steps = append(steps, traversal{edgeID: ei, next: i.edges[ei].From})
			}
			sort.Slice(steps, func(a, b int) bool {
				x, y := steps[a], steps[b]
				if x.next != y.next {
					return x.next < y.next
				}
				e, f := i.edges[x.edgeID], i.edges[y.edgeID]
				if e.Property != f.Property {
					return e.Property < f.Property
				}
				if e.From != f.From {
					return e.From < f.From
				}
				if e.To != f.To {
					return e.To < f.To
				}
				return x.edgeID < y.edgeID
			})
		} else {
			edgeIDs := i.out[cur.id]
			if q.Reverse {
				edgeIDs = i.in[cur.id]
			}
			for _, ei := range edgeIDs {
				next := i.edges[ei].To
				if q.Reverse {
					next = i.edges[ei].From
				}
				steps = append(steps, traversal{edgeID: ei, next: next})
			}
		}
		for _, step := range steps {
			if res.Steps == q.MaxSteps {
				res.Complete = false
				res.Reason = "step_limit"
				return res, nil
			}
			res.Steps++
			e := i.edges[step.edgeID]
			next := step.next
			if visited[next] {
				continue
			}
			if cur.depth >= q.MaxDepth {
				res.Complete = false
				res.Reason = "depth_limit"
				return res, nil
			}
			visited[next] = true
			path := append(append([]Edge(nil), cur.path...), e)
			queue = append(queue, item{id: next, path: path, depth: cur.depth + 1})
			if len(res.Results) == q.MaxResults {
				res.Complete = false
				res.Reason = "result_limit"
				return res, nil
			}
			res.Results = append(res.Results, Witness{Node: next, Path: path})
		}
	}
	return res, nil
}
