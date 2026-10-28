package remoteserver

import (
	"encoding/json"
	"errors"
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

// isolated reports whether checks run as users of their own: the remote runs
// as root in its container and nowhere else.
func isolated() bool { return os.Geteuid() == 0 }

// prepare puts the check into its own process group and, when isolated, runs
// it as its user with no supplementary groups.
func prepare(cmd *exec.Cmd, uid int, isolate bool) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if isolate {
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(uid), Groups: []uint32{}}
	}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}

// lockChecks makes check runs of the hook and of the host's landing check
// take turns, so no two check users live at once.
func lockChecks(l Layout) (func(), error) {
	f, err := os.OpenFile(filepath.Join(l.Root, "check.lock"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() { f.Close() }, nil
}

// finished ends whatever the check left running in its process group.
func finished(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

// protect makes the serving remote the last process the container's memory
// limit ends, and victim makes the hook, and every check it starts, the
// first. Both need CAP_SYS_RESOURCE, which also fixes the value as the
// lowest a check may set.
func protect() error { return os.WriteFile("/proc/self/oom_score_adj", []byte("-1000"), 0) }

func victim() error { return os.WriteFile("/proc/self/oom_score_adj", []byte("1000"), 0) }

func chownTree(dir string, uid int) error {
	return filepath.WalkDir(dir, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Lchown(p, uid, uid)
	})
}

// writableName lists the folders every user may write, found when the
// remote is seeded: the only places outside its copy where a check user can
// leave something.
const writableName = "writable.json"

// harden runs once at seeding in the container: it records the folders every
// user may write and takes write access for others from every file, so a
// check can change nothing it does not own.
func harden(l Layout) error {
	var dirs []string
	err := filepath.WalkDir("/", func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if p == "/proc" || p == "/sys" || p == l.Root {
			return fs.SkipDir
		}
		info, err := e.Info()
		if err != nil || info.Mode().Perm()&0o002 == 0 || info.Mode()&fs.ModeSymlink != 0 {
			return nil
		}
		switch {
		case e.IsDir():
			dirs = append(dirs, p)
		case info.Mode().IsRegular():
			return os.Chmod(p, info.Mode().Perm()&^0o002)
		}
		return nil
	})
	if err != nil {
		return err
	}
	b, err := json.Marshal(dirs)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(l.Root, writableName), b, 0o600)
}

// sweep removes everything a check user left: its processes, also those
// that left their process group, its entries in the folders every user may
// write, and its System V IPC objects. It fails if anything survives.
func sweep(l Layout, uid int, isolate bool) error {
	if !isolate {
		return nil
	}
	if err := killAll(uid); err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(l.Root, writableName))
	if err != nil {
		return err
	}
	var dirs []string
	if err := json.Unmarshal(b, &dirs); err != nil {
		return err
	}
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			return err
		}
		for _, e := range entries {
			p := filepath.Join(d, e.Name())
			if info, err := os.Lstat(p); err == nil && owner(info) == uid {
				if err := os.RemoveAll(p); err != nil {
					return fmt.Errorf("remove what check user %d left: %w", uid, err)
				}
			}
		}
	}
	return removeIPC(uid)
}

func killAll(uid int) error {
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

func owner(info fs.FileInfo) int {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return int(st.Uid)
	}
	return -1
}

// removeIPC removes the shared memory segments, semaphore sets and message
// queues of the check user.
func removeIPC(uid int) error {
	for _, kind := range []struct{ file, id, flag string }{{"shm", "shmid", "-m"}, {"sem", "semid", "-s"}, {"msg", "msqid", "-q"}} {
		b, err := os.ReadFile("/proc/sysvipc/" + kind.file)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		lines := strings.Split(strings.TrimSpace(string(b)), "\n")
		if len(lines) < 2 {
			continue
		}
		header := strings.Fields(lines[0])
		col := map[string]int{}
		for i, h := range header {
			col[h] = i
		}
		for _, line := range lines[1:] {
			f := strings.Fields(line)
			id, u := field(f, col, kind.id), field(f, col, "uid")
			if u != strconv.Itoa(uid) || id == "" {
				continue
			}
			// Its owner may remove it; the remote holds no CAP_IPC_OWNER.
			rm := exec.Command("ipcrm", kind.flag, id)
			rm.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(uid), Groups: []uint32{}}}
			if out, err := rm.CombinedOutput(); err != nil {
				return fmt.Errorf("remove IPC object %s of check user %d: %v: %s", id, uid, err, out)
			}
		}
	}
	return nil
}

func field(f []string, col map[string]int, name string) string {
	if i, ok := col[name]; ok && i < len(f) {
		return f[i]
	}
	return ""
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
