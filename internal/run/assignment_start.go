package run

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// openAll creates independent copies and homes, then starts every Manager
// container together. No Message is sent here, including to Root.
func (s *assignmentScheduler) openAll(ctx context.Context, team []Manager) error {
	forks, ok := s.flow.workspaces.(AssignmentCommitForks)
	if !ok {
		return errors.New("assignment tree needs fixed-commit workspace forks")
	}
	requests := make(map[string]AssignmentRequest, len(team))
	root, err := s.request(ctx, ".")
	if err != nil {
		return err
	}
	root.Prepared = s.options.PreparedRoot
	if root.Prepared == nil {
		root.Prepared, err = s.flow.workspaces.PrepareAssignment(ctx, WorkspaceSpec{ID: root.ID, Tree: s.options.Tree})
	}
	s.prepared(".", root)
	if err != nil {
		return err
	}
	if root.Prepared == nil || root.Prepared.ID() != root.ID || root.Prepared.Volume() == "" || root.Prepared.StartingCommit() == "" {
		return errors.New("initial Root copy does not match the assignment")
	}
	requests["."] = root
	for _, manager := range team[1:] {
		request, err := s.request(ctx, manager.Path)
		if err != nil {
			return err
		}
		request.Prepared, err = forks.ForkAssignmentCommit(ctx, root.Prepared, root.Prepared.StartingCommit(), request.ID)
		s.prepared(manager.Path, request)
		if err != nil {
			return err
		}
		requests[manager.Path] = request
	}
	startCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	completed := make(chan childCompletion, len(team))
	for _, manager := range team {
		s.flow.mu.Lock()
		s.flow.team[manager.Path].starting = true
		s.flow.mu.Unlock()
		go s.worker(manager.Path, func() error {
			return s.flow.openReserved(startCtx, manager.Parent, manager.Path, requests[manager.Path])
		}, completed)
	}
	for pending := len(team); pending > 0; {
		select {
		case result := <-completed:
			pending--
			if result.err != nil {
				cancel()
				return s.joinCancelled(completed, pending, fmt.Errorf("start Manager %q: %w", result.path, result.err))
			}
		case <-ctx.Done():
			cancel()
			return s.joinCancelled(completed, pending, ctx.Err())
		}
	}
	return nil
}

func (s *assignmentScheduler) prepared(path string, request AssignmentRequest) {
	s.flow.mu.Lock()
	defer s.flow.mu.Unlock()
	m := s.flow.team[path]
	m.transcript = request.Transcript
	if request.Prepared != nil {
		m.partial = &Assignment{work: request.Prepared}
		m.base = request.Prepared.StartingCommit()
	}
}

func (s *assignmentScheduler) worker(path string, work func() error, completed chan<- childCompletion) {
	err := work()
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		err = errors.Join(err, s.closePaths([]string{path}))
	}
	completed <- childCompletion{path, err}
}

func (s *assignmentScheduler) joinCancelled(completed <-chan childCompletion, pending int, cause error) error {
	ctx, stop := context.WithTimeout(context.Background(), min(s.options.Timeout, time.Minute))
	defer stop()
	for pending > 0 {
		select {
		case result := <-completed:
			pending--
			if result.err != nil {
				cause = errors.Join(cause, fmt.Errorf("Manager %q: %w", result.path, result.err))
			}
		case <-ctx.Done():
			return errors.Join(cause, fmt.Errorf("%d Manager outcomes remain uncertain after cancellation: %w", pending, ctx.Err()))
		}
	}
	return cause
}
