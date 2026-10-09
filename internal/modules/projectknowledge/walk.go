package projectknowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

type WalkRequest struct {
	Start      string `json:"start"`
	Reverse    bool   `json:"reverse,omitempty"`
	MaxDepth   int    `json:"maxDepth"`
	MaxSteps   int    `json:"maxSteps"`
	MaxResults int    `json:"maxResults"`
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
// Reverse traversal follows incoming edges, while witness Edge values retain
// their original From/To orientation.
func (i *Index) Walk(q WalkRequest) (WalkResult, error) {
	if _, ok := i.byID[q.Start]; !ok {
		return WalkResult{}, ErrNotFound
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
		edgeIDs := i.out[cur.id]
		if q.Reverse {
			edgeIDs = i.in[cur.id]
		}
		for _, ei := range edgeIDs {
			if res.Steps == q.MaxSteps {
				res.Complete = false
				res.Reason = "step_limit"
				return res, nil
			}
			res.Steps++
			e := i.edges[ei]
			next := e.To
			if q.Reverse {
				next = e.From
			}
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
