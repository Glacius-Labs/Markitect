package run

import (
	"context"
	"fmt"
)

type childCompletion struct {
	path string
	err  error
}

// children fixes the parent's Go stand and delivers tasks to its already warm
// children concurrently. Fixed results rejoin in observed completion order;
// a returned correction joins that same queue while siblings keep working.
func (s *assignmentScheduler) children(ctx context.Context, parent string, tasks []assignmentAction) error {
	var stand string
	err := s.flow.serial(ctx, parent, func(ctx context.Context) error {
		var err error
		stand, err = s.flow.Workspace(parent).Commit(ctx, "Parent Go for concurrent child tasks")
		return err
	})
	if err != nil {
		return err
	}
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	completed := make(chan childCompletion, len(tasks))
	pending := 0
	launch := func(path string, work func() error) {
		pending++
		go s.worker(path, work, completed)
	}
	for _, task := range tasks {
		launch(task.child, func() error {
			_, err := s.work(childCtx, task.child, task.task, stand)
			return err
		})
	}
	abort := func(cause error) error {
		cancel()
		return s.joinCancelled(completed, pending, cause)
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
