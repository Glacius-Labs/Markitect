//go:build windows

package codexappserver

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

var metadataNTDLL = syscall.NewLazyDLL("ntdll.dll")
var metadataNtCreateFile = metadataNTDLL.NewProc("NtCreateFile")
var metadataStatusError = metadataNTDLL.NewProc("RtlNtStatusToDosError")

type metadataUnicodeString struct {
	Length, MaximumLength uint16
	Buffer                *uint16
}
type metadataObjectAttributes struct {
	Length                                       uint32
	RootDirectory                                syscall.Handle
	ObjectName                                   *metadataUnicodeString
	Attributes                                   uint32
	SecurityDescriptor, SecurityQualityOfService uintptr
}
type metadataIOStatus struct{ Status, Information uintptr }

func openMetadataVolume(name string) (*os.File, error) {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, &os.PathError{Op: "metadata-volume", Path: name, Err: err}
	}
	return os.NewFile(uintptr(h), name), nil
}

func openMetadataChild(parent *os.File, name string, directory bool) (*os.File, error) {
	if name == "." && directory {
		process, err := syscall.GetCurrentProcess()
		if err != nil {
			return nil, err
		}
		var duplicate syscall.Handle
		err = syscall.DuplicateHandle(process, syscall.Handle(parent.Fd()), process, &duplicate, 0, false, syscall.DUPLICATE_SAME_ACCESS)
		runtime.KeepAlive(parent)
		if err != nil {
			return nil, err
		}
		return os.NewFile(uintptr(duplicate), parent.Name()), nil
	}
	text, err := syscall.UTF16FromString(name)
	if err != nil {
		return nil, err
	}
	if len(text)*2 > 65535 {
		return nil, invalid("native metadata component too long")
	}
	unicode := metadataUnicodeString{Length: uint16((len(text) - 1) * 2), MaximumLength: uint16(len(text) * 2), Buffer: &text[0]}
	attributes := metadataObjectAttributes{RootDirectory: syscall.Handle(parent.Fd()), ObjectName: &unicode, Attributes: 0x1000} // OBJ_DONT_REPARSE
	attributes.Length = uint32(unsafe.Sizeof(attributes))
	var handle syscall.Handle
	var status metadataIOStatus
	options := uintptr(0x20 | 0x200000) // FILE_SYNCHRONOUS_IO_NONALERT | FILE_OPEN_REPARSE_POINT
	if directory {
		options |= 0x1
	} // FILE_DIRECTORY_FILE
	result, _, _ := metadataNtCreateFile.Call(
		uintptr(unsafe.Pointer(&handle)), syscall.GENERIC_READ|0x100000,
		uintptr(unsafe.Pointer(&attributes)), uintptr(unsafe.Pointer(&status)), 0, 0,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		1, options, 0, 0, // FILE_OPEN, no create or write access
	)
	runtime.KeepAlive(parent)
	runtime.KeepAlive(text)
	if result != 0 {
		if handle != 0 && handle != syscall.InvalidHandle {
			_ = syscall.CloseHandle(handle)
		}
		if uint32(result) == 0xc000050b || uint32(result) == 0x8000002d || uint32(result) == 0xc0000279 {
			return nil, invalid("refusing native reparse point %q", name)
		}
		code, _, _ := metadataStatusError.Call(result)
		return nil, &os.PathError{Op: "metadata-open-relative", Path: name, Err: syscall.Errno(code)}
	}
	return os.NewFile(uintptr(handle), name), nil
}
