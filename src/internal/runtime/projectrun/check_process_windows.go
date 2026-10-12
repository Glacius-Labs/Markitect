//go:build windows

package projectrun

import (
	"errors"
	"os/exec"
	"syscall"
	"unsafe"
)

const (
	jobObjectExtendedLimitInformationClass = 9
	jobObjectLimitKillOnJobClose           = 0x2000
	processSetQuota                        = 0x0100
	processTerminate                       = 0x0001
)

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW         = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject  = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject       = kernel32.NewProc("TerminateJobObject")
)

// jobObjectExtendedLimitInformation is JOBOBJECT_EXTENDED_LIMIT_INFORMATION
// with its basic limits inlined; the layout matches 64-bit Windows.
type jobObjectExtendedLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
	IOCounters              [6]uint64
	ProcessMemoryLimit      uintptr
	JobMemoryLimit          uintptr
	PeakProcessMemoryUsed   uintptr
	PeakJobMemoryUsed       uintptr
}

// runCheckProcess runs a check inside a kill-on-close job object. Its timeout
// and its completion stop every process in the job, so a descendant can neither
// outlive the check nor hold its output pipes open past the check timeout. The
// check joins the job right after it starts, as the agentexec runner does; a
// descendant started before that is bounded only by checkWaitDelay.
func runCheckProcess(cmd *exec.Cmd) error {
	job, err := newCheckJob()
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(job)
	cmd.Cancel = func() error {
		err := cmd.Process.Kill()
		_, _, _ = procTerminateJobObject.Call(uintptr(job), 1)
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := assignCheckJob(job, cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	return cmd.Wait()
}

func newCheckJob() (syscall.Handle, error) {
	handle, _, _ := procCreateJobObjectW.Call(0, 0)
	if handle == 0 {
		return 0, errors.New("check process job could not be created")
	}
	job := syscall.Handle(handle)
	info := jobObjectExtendedLimitInformation{LimitFlags: jobObjectLimitKillOnJobClose}
	result, _, _ := procSetInformationJobObject.Call(uintptr(job), jobObjectExtendedLimitInformationClass, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info))
	if result == 0 {
		_ = syscall.CloseHandle(job)
		return 0, errors.New("check process job limits could not be established")
	}
	return job, nil
}

func assignCheckJob(job syscall.Handle, pid int) error {
	process, err := syscall.OpenProcess(processSetQuota|processTerminate, false, uint32(pid))
	if err != nil {
		return errors.New("check process could not be assigned to its job")
	}
	defer syscall.CloseHandle(process)
	result, _, _ := procAssignProcessToJobObject.Call(uintptr(job), uintptr(process))
	if result == 0 {
		return errors.New("check process could not be assigned to its job")
	}
	return nil
}
