package agentd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// listProcesses reads the container's process table from /proc. The
// container has its own PID namespace, so the table holds exactly the
// session's processes. self and its ancestors are left out.
func listProcesses(self int) ([]frame.Process, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	all := map[int]frame.Process{}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || !e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("/proc", e.Name(), "stat"))
		if err != nil {
			continue // the process ended while the table was read
		}
		p, err := parseStat(string(raw))
		if err != nil {
			return nil, fmt.Errorf("process %d: %w", pid, err)
		}
		all[p.PID] = p
	}
	skip := map[int]bool{}
	for pid := self; pid > 0 && !skip[pid]; {
		skip[pid] = true
		p, ok := all[pid]
		if !ok {
			break
		}
		pid = p.PPID
	}
	var out []frame.Process
	for pid, p := range all {
		if !skip[pid] {
			out = append(out, p)
		}
	}
	sortProcesses(out)
	return out, nil
}

// parseStat reads pid, command, state, parent, process group and, when the
// line has it, the thread count from one /proc/<pid>/stat line. The command
// may itself contain spaces and parentheses, so it ends at the last closing
// parenthesis.
func parseStat(line string) (frame.Process, error) {
	lp, rp := strings.IndexByte(line, '('), strings.LastIndexByte(line, ')')
	if lp < 0 || rp < lp {
		return frame.Process{}, fmt.Errorf("malformed stat %q", line)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(line[:lp]))
	if err != nil {
		return frame.Process{}, fmt.Errorf("malformed pid in %q", line)
	}
	rest := strings.Fields(line[rp+1:])
	if len(rest) < 3 {
		return frame.Process{}, fmt.Errorf("malformed stat %q", line)
	}
	ppid, err1 := strconv.Atoi(rest[1])
	pgid, err2 := strconv.Atoi(rest[2])
	if err1 != nil || err2 != nil {
		return frame.Process{}, fmt.Errorf("malformed stat %q", line)
	}
	p := frame.Process{PID: pid, PPID: ppid, PGID: pgid, State: rest[0], Command: line[lp+1 : rp]}
	// num_threads is field 20 of the line, the 18th after the command.
	if len(rest) > 17 {
		if n, err := strconv.Atoi(rest[17]); err == nil {
			p.Threads = n
		}
	}
	return p, nil
}
