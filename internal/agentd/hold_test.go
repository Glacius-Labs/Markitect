package agentd

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// A zombie or dead entry can write nothing, unless other threads of a zombie
// leader's group still run; init never counts.
func TestLiveCountsTheThreadsOfAZombieLeader(t *testing.T) {
	got := live([]frame.Process{
		{PID: 1, State: "S"},
		{PID: 7, State: "S"},
		{PID: 8, State: "Z", Threads: 1},
		{PID: 9, State: "Z", Threads: 3},
		{PID: 10, State: "X"},
		{PID: 11, State: "D", Threads: 2},
	})
	var pids []int
	for _, p := range got {
		pids = append(pids, p.PID)
	}
	if len(pids) != 3 || pids[0] != 7 || pids[1] != 9 || pids[2] != 11 {
		t.Fatalf("live %v", pids)
	}
}
