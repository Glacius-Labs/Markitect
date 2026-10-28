package remoteserver

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// isolated reports whether checks run as throwaway users: the remote runs as
// root in its container and nowhere else.
func isolated() bool { return os.Geteuid() == 0 }

// prepare puts the check into its own process group and, when isolated, runs
// it as the throwaway user with no supplementary groups.
func prepare(cmd *exec.Cmd, uid int, isolate bool) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if isolate {
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(uid), Groups: []uint32{}}
	}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}

// started makes a check the first victim of the container's memory limit,
// so a greedy check cannot take the remote down with it.
func started(cmd *exec.Cmd) {
	if isolated() {
		_ = os.WriteFile(fmt.Sprintf("/proc/%d/oom_score_adj", cmd.Process.Pid), []byte("1000"), 0)
	}
}

// finished ends whatever the check left running in its process group.
func finished(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func chownTree(dir string, uid int) error {
	return filepath.WalkDir(dir, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Lchown(p, uid, uid)
	})
}

// sweep kills every process of the check user, also those that left their
// process group, and fails if any survives.
func sweep(uid int, isolate bool) error {
	if !isolate {
		return nil
	}
	for range 100 {
		pids := processesOf(uid)
		if len(pids) == 0 {
			return nil
		}
		for _, pid := range pids {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("processes of check user %d survived being killed", uid)
}

func processesOf(uid int) []int {
	entries, _ := os.ReadDir("/proc")
	var pids []int
	want := strconv.Itoa(uid)
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		status, err := os.ReadFile(filepath.Join("/proc", e.Name(), "status"))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(status), "\n") {
			if fields := strings.Fields(line); len(fields) >= 2 && fields[0] == "Uid:" {
				// real, effective, saved and file-system uid
				for _, f := range fields[1:] {
					if f == want {
						pids = append(pids, pid)
						break
					}
				}
				break
			}
		}
	}
	return pids
}
