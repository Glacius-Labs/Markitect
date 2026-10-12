//go:build windows

package agentexec

import (
	"errors"
	"os/exec"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

const (
	jobObjectExtendedLimitInformationClass   = 9
	jobObjectBasicAccountingInformationClass = 1
	jobObjectLimitKillOnJobClose             = 0x2000
	processTerminate                         = 0x0001
	processSetQuota                          = 0x0100
)

var (
	kernel32                      = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW          = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject   = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject  = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject        = kernel32.NewProc("TerminateJobObject")
	procQueryInformationJobObject = kernel32.NewProc("QueryInformationJobObject")
	procOpenProcess               = kernel32.NewProc("OpenProcess")
	procCloseHandle               = kernel32.NewProc("CloseHandle")
)

type jobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	_                       uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	_                       uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type jobObjectIOCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type jobObjectExtendedLimitInformation struct {
	BasicLimitInformation jobObjectBasicLimitInformation
	IOInfo                jobObjectIOCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

type jobObjectBasicAccountingInformation struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

type windowsProcessTree struct {
	job syscall.Handle
}

func newProcessTreeGuard() (processTreeGuard, error) {
	handle, _, _ := procCreateJobObjectW.Call(0, 0)
	if handle == 0 {
		return nil, errors.New("runner process job could not be created")
	}
	g := &windowsProcessTree{job: syscall.Handle(handle)}
	info := jobObjectExtendedLimitInformation{}
	info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
	result, _, _ := procSetInformationJobObject.Call(
		uintptr(g.job),
		jobObjectExtendedLimitInformationClass,
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
	)
	runtime.KeepAlive(&info)
	if result == 0 {
		_ = g.closeHandle()
		return nil, errors.New("runner process job limits could not be established")
	}
	return g, nil
}

func (*windowsProcessTree) prepare(*exec.Cmd) error { return nil }

func (g *windowsProcessTree) attach(cmd *exec.Cmd) error {
	if cmd.Process == nil || cmd.Process.Pid <= 0 {
		return errors.New("runner process identity is unavailable")
	}
	process, _, _ := procOpenProcess.Call(processSetQuota|processTerminate, 0, uintptr(cmd.Process.Pid))
	if process == 0 {
		return errors.New("runner process could not be assigned to its job")
	}
	defer procCloseHandle.Call(process)
	result, _, _ := procAssignProcessToJobObject.Call(uintptr(g.job), process)
	if result == 0 {
		return errors.New("runner process could not be assigned to its job")
	}
	return nil
}

func (g *windowsProcessTree) terminate() error {
	if g.job == 0 {
		return nil
	}
	active, err := g.activeProcesses()
	if err != nil {
		_, _, _ = procTerminateJobObject.Call(uintptr(g.job), 1)
		return err
	}
	if active == 0 {
		return nil
	}
	result, _, _ := procTerminateJobObject.Call(uintptr(g.job), 1)
	if result == 0 {
		return errors.New("runner process job could not be stopped")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		active, err = g.activeProcesses()
		if err != nil {
			return err
		}
		if active == 0 {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return errors.New("runner process job did not stop within its bound")
}

func (g *windowsProcessTree) activeProcesses() (uint32, error) {
	info := jobObjectBasicAccountingInformation{}
	var returned uint32
	result, _, _ := procQueryInformationJobObject.Call(
		uintptr(g.job),
		jobObjectBasicAccountingInformationClass,
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
		uintptr(unsafe.Pointer(&returned)),
	)
	runtime.KeepAlive(&info)
	if result == 0 {
		return 0, errors.New("runner process job state could not be read")
	}
	return info.ActiveProcesses, nil
}

func (g *windowsProcessTree) closeHandle() error {
	if g.job == 0 {
		return nil
	}
	handle := g.job
	g.job = 0
	result, _, _ := procCloseHandle.Call(uintptr(handle))
	if result == 0 {
		return errors.New("runner process job handle could not be closed")
	}
	return nil
}

func (g *windowsProcessTree) close() error {
	terminateErr := g.terminate()
	closeErr := g.closeHandle()
	if terminateErr != nil || closeErr != nil {
		return errors.New("runner process job cleanup failed")
	}
	return nil
}
