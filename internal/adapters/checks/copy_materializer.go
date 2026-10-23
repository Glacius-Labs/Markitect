package checks

import (
	"context"
	"fmt"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// The real Run copy creates exact scratch through its already-owned service.
// This adapter projects the lifecycle into Runner's narrow consumer contract.
type copyMaterializer struct {
	source run.Copy
	mu     sync.Mutex
	copies map[CopyID]run.DisposableCopy
}

func (m *copyMaterializer) Materialize(ctx context.Context, spec CopySpec) (CopyObservation, error) {
	copy, err := m.source.Materialize(ctx, spec.Tree)
	o := CopyObservation{ID: spec.ID}
	if copy != nil {
		m.mu.Lock()
		m.copies[spec.ID] = copy
		m.mu.Unlock()
		location := copy.Location()
		o.Volume = location.Volume
		o.Subpath = location.Subpath
		o.Tree = copy.Tree()
		if !location.Valid() || location.Directory != "" {
			return o, fmt.Errorf("container checks require a verified Linux volume copy")
		}
	}
	return o, err
}
func (m *copyMaterializer) Close(ctx context.Context, id CopyID) (CopyObservation, error) {
	m.mu.Lock()
	copy := m.copies[id]
	m.mu.Unlock()
	o := CopyObservation{ID: id}
	if copy == nil {
		return o, fmt.Errorf("disposable check handle unavailable")
	}
	l := copy.Location()
	o.Volume = l.Volume
	o.Subpath = l.Subpath
	o.Tree = copy.Tree()
	err := copy.Close(ctx)
	o.Closed = err == nil
	if err != nil {
		o.Reason = err.Error()
	}
	return o, err
}
