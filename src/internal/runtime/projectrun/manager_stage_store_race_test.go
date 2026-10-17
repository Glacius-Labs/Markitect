package projectrun

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
)

func TestOrdinaryManagerAttemptLogConcurrentReservation(t *testing.T) {
	journal, task, invoker, runtime := ordinaryInterruptedJournalFixture(t)
	before := journal.snapshot()
	agent := runtime.Agents[task.ManagerID]
	config, err := agent.AgentConfig()
	if err != nil {
		t.Fatal(err)
	}
	request := agentexec.Request{Role: agentexec.RoleExecutor}
	reservation := before.RoleStartReservations[0]
	attemptID := nativeTaskAttemptID(task, "work")
	expectedID := ordinaryRunAttemptID(before, attemptID)
	const updates = 8
	start := make(chan struct{})
	errors := make(chan error, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		<-start
		for range updates {
			// The real reservation path updates both the reservation slice and
			// the persisted report's cost accounting, revision and timestamp.
			if err := journal.reserve(reservation); err != nil {
				errors <- err
				return
			}
		}
	}()
	go func() {
		defer workers.Done()
		<-start
		for range 256 {
			log, err := ordinaryManagerAttemptLog(journal, invoker, config, agent, task, "work", attemptID, request)
			if err != nil {
				errors <- err
				return
			}
			if log.Attempt.ID != expectedID || log.Attempt.StageKey != before.ID+"/"+expectedID {
				errors <- fmt.Errorf("concurrent persistence changed attempt identity: %+v", log.Attempt)
				return
			}
		}
	}()
	close(start)
	workers.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	after := journal.snapshot()
	if after.Revision != before.Revision+updates || len(after.RoleStartReservations) != 1 || len(after.Invocations) != 1 {
		t.Fatalf("log construction lost a reservation update or created a dispatch: %+v", after)
	}
}
