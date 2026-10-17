package agentd

import (
	"os"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// census answers with the processes in the container other than agentd and
// its ancestors, such as the container's init. A platform without a process
// table answers with a failure, never with an empty list that would look
// quiescent.
func (s *server) census() error {
	processes, err := listProcesses(os.Getpid())
	if err != nil {
		return s.sendControl(frame.ControlMessage{Type: frame.TypeFailure, Failure: &frame.Failure{Message: "census: " + err.Error()}})
	}
	return s.sendControl(frame.ControlMessage{Type: frame.TypeCensus, Census: &frame.Census{Processes: processes}})
}

func sortProcesses(p []frame.Process) {
	sort.Slice(p, func(i, j int) bool { return p[i].PID < p[j].PID })
}
