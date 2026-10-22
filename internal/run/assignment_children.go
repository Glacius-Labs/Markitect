package run

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type childCompletion struct {
	path string
	err  error
}

// children prepares fixed foundations under the parent's queue, then starts
// every child independently. Fixed results rejoin in observed completion order;
// a returned correction joins that same queue while siblings keep working.
func (s *assignmentScheduler) children(ctx context.Context, parent string, tasks []assignmentAction) error {
	requests := make(map[string]AssignmentRequest, len(tasks))
	err := s.flow.serial(ctx, parent, func(ctx context.Context) error {
		for _, task := range tasks {
			request, err := s.request(ctx, task.child)
			if err != nil {
				return err
			}
			request, err = s.flow.prepareChild(ctx, parent, task.child, request)
			if err != nil {
				return err
			}
			requests[task.child] = request
		}
		return nil
	})
	if err != nil {
		s.flow.mu.Lock()
		for path := range requests {
			s.flow.team[path].starting = false
		}
		s.flow.mu.Unlock()
		return err
	}
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	completed := make(chan childCompletion, len(tasks))
	pending := 0
	launch := func(path string, work func() error) {
		pending++
		go func() {
			err := work()
			// Close may already have returned an uncertain outcome while an
			// uncooperative Open/Message was still running. A late start must
			// also receive a stop attempt, with its copy retained in Flow.
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				err = errors.Join(err, s.closePaths([]string{path}))
			}
			completed <- childCompletion{path, err}
		}()
	}
	for _, task := range tasks {
		launch(task.child, func() error {
			if err := s.flow.openReserved(childCtx, parent, task.child, requests[task.child]); err != nil {
				return err
			}
			_, err := s.work(childCtx, task.child, task.task)
			return err
		})
	}
	abort := func(cause error) error {
		cancel()
		joinCtx, stop := context.WithTimeout(context.Background(), min(s.options.Timeout, time.Minute))
		defer stop()
		for pending > 0 {
			select {
			case result := <-completed:
				pending--
				if result.err != nil {
					cause = errors.Join(cause, fmt.Errorf("child %q: %w", result.path, result.err))
				}
			case <-joinCtx.Done():
				return errors.Join(cause, fmt.Errorf("%d child outcomes remain uncertain after cancellation: %w", pending, joinCtx.Err()))
			}
		}
		return cause
	}
	for unresolved := len(tasks); unresolved > 0; {
		select {
		case result := <-completed:
			pending--
			if result.err != nil {
				return abort(fmt.Errorf("child %q: %w", result.path, result.err))
			}
			correction, err := s.reviewResult(childCtx, parent, result.path)
			if err != nil {
				return abort(err)
			}
			if correction == "" {
				unresolved--
			} else {
				launch(result.path, func() error { return s.correct(childCtx, parent, result.path, correction) })
			}
		case <-ctx.Done():
			return abort(ctx.Err())
		}
	}
	return nil
}
