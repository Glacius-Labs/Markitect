//go:build windows

package recordstore

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

var nativeDLL = syscall.NewLazyDLL("ntdll.dll")
var winDLL = syscall.NewLazyDLL("kernel32.dll")
var ntOpen = nativeDLL.NewProc("NtOpenFile")
var ntCreate = nativeDLL.NewProc("NtCreateFile")
var ntRename = nativeDLL.NewProc("NtSetInformationFile")
var ntVolume = nativeDLL.NewProc("NtQueryVolumeInformationFile")
var ntVersion = nativeDLL.NewProc("RtlGetVersion")
var ntDOS = nativeDLL.NewProc("RtlNtStatusToDosError")
var flushBuffers = winDLL.NewProc("FlushFileBuffers")
var lockFile = winDLL.NewProc("LockFileEx")
var unlockFile = winDLL.NewProc("UnlockFileEx")

const (
	shareAll        = 7
	synchronize     = 0x00100000
	deleteAccess    = 0x00010000
	readWriteAccess = 0xC0000000
	parentRights    = readWriteAccess | synchronize | 0x00000001 | 0x00000002 | 0x00000004 | 0x00000080
	writeThrough    = 0x00000002
	synchronousIO   = 0x00000020
	openReparse     = 0x00200000
	nonDirectory    = 0x00000040
	directoryFile   = 0x00000001
	statusCollision = 0xC0000035
)

type unicodeName struct {
	Length, MaximumLength uint16
	Buffer                *uint16
}
type objectAttributes struct {
	Length                                       uint32
	padding                                      uint32
	RootDirectory                                syscall.Handle
	ObjectName                                   *unicodeName
	Attributes                                   uint32
	padding2                                     uint32
	SecurityDescriptor, SecurityQualityOfService uintptr
}
type versionInfo struct {
	Size, Major, Minor, Build, Platform uint32
	ServicePack                         [128]uint16
}

func nativeError(status uint32) error {
	if status == statusCollision {
		return fmt.Errorf("NTSTATUS 0x%08x: %w", status, fs.ErrExist)
	}
	dos, _, _ := ntDOS.Call(uintptr(status))
	return fmt.Errorf("NTSTATUS 0x%08x: %w", status, syscall.Errno(dos))
}
func attributes(parent *os.File, leaf string) (objectAttributes, []uint16, error) {
	if leaf == "" || leaf == ".." {
		return objectAttributes{}, nil, errors.New("invalid native record leaf")
	}
	u, err := syscall.UTF16FromString(leaf)
	if err != nil || len(u) > 256 {
		return objectAttributes{}, nil, errors.New("native record leaf exceeds bound")
	}
	for _, c := range u[:len(u)-1] {
		if c == '/' || c == '\\' || c == ':' {
			return objectAttributes{}, nil, errors.New("native record leaf is not literal")
		}
	}
	name := &unicodeName{Length: uint16((len(u) - 1) * 2), MaximumLength: uint16(len(u) * 2), Buffer: &u[0]}
	// NT relative opens use an empty name to reopen the directory object itself.
	if leaf == "." {
		name.Length = 0
	}
	a := objectAttributes{Length: uint32(unsafe.Sizeof(objectAttributes{})), RootDirectory: syscall.Handle(parent.Fd()), ObjectName: name, Attributes: 0x40}
	return a, u, nil
}
func openNative(parent *os.File, leaf string, access, options uint32) (*os.File, error) {
	a, u, err := attributes(parent, leaf)
	if err != nil {
		return nil, err
	}
	var handle syscall.Handle
	var statusBlock [2]uintptr
	status, _, _ := ntOpen.Call(uintptr(unsafe.Pointer(&handle)), uintptr(access), uintptr(unsafe.Pointer(&a)), uintptr(unsafe.Pointer(&statusBlock)), shareAll, uintptr(options))
	runtime.KeepAlive(a.ObjectName)
	runtime.KeepAlive(u)
	if uint32(status) != 0 {
		return nil, nativeError(uint32(status))
	}
	return os.NewFile(uintptr(handle), parent.Name()+"/"+leaf), nil
}
func createNative(parent *os.File, leaf string, options, attrs, access uint32) (*os.File, error) {
	a, u, err := attributes(parent, leaf)
	if err != nil {
		return nil, err
	}
	var handle syscall.Handle
	var statusBlock [2]uintptr
	status, _, _ := ntCreate.Call(uintptr(unsafe.Pointer(&handle)), uintptr(access), uintptr(unsafe.Pointer(&a)), uintptr(unsafe.Pointer(&statusBlock)), 0, uintptr(attrs), shareAll, 2, uintptr(options|writeThrough|synchronousIO|openReparse), 0, 0)
	runtime.KeepAlive(a.ObjectName)
	runtime.KeepAlive(u)
	if uint32(status) != 0 {
		return nil, nativeError(uint32(status))
	}
	return os.NewFile(uintptr(handle), parent.Name()+"/"+leaf), nil
}
func platformIsReparse(info os.FileInfo) bool {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return !ok || info.Mode()&os.ModeSymlink != 0 || data.FileAttributes&0x400 != 0
}

// Numeric modes do not define a Windows DACL. Domain objects inherit/default
// their approved parent's descriptor; private agent logs have another owner.
func platformMode(os.FileInfo, os.FileMode) error { return nil }
func platformWritableParent(f *os.File) (*os.File, error) {
	opened, err := openNative(f, ".", parentRights, directoryFile|synchronousIO|openReparse|writeThrough)
	if err != nil {
		return nil, err
	}
	a, err := f.Stat()
	b, e := opened.Stat()
	if err != nil || e != nil || platformIsReparse(b) || !os.SameFile(a, b) {
		opened.Close()
		return nil, errors.New("native writer parent identity differs")
	}
	return opened, nil
}
func platformSyncDir(f *os.File) error {
	ok, _, err := flushBuffers.Call(f.Fd())
	if ok == 0 {
		if err == syscall.Errno(0) {
			err = syscall.EINVAL
		}
		return fmt.Errorf("required record FlushFileBuffers: %w", err)
	}
	return nil
}
func platformCheckWriter(parent *os.File) error {
	if runtime.GOARCH != "amd64" || unsafe.Sizeof(uintptr(0)) != 8 {
		return errors.New("record writes require measured Windows amd64 ABI")
	}
	for _, p := range []*syscall.LazyProc{ntOpen, ntCreate, ntRename, ntVolume, ntVersion, ntDOS, flushBuffers, lockFile, unlockFile} {
		if err := p.Find(); err != nil {
			return fmt.Errorf("record native API unavailable: %w", err)
		}
	}
	v := versionInfo{Size: uint32(unsafe.Sizeof(versionInfo{}))}
	status, _, _ := ntVersion.Call(uintptr(unsafe.Pointer(&v)))
	if uint32(status) != 0 {
		return nativeError(uint32(status))
	}
	if v.Major != 10 || v.Minor != 0 || v.Build != 26100 {
		return errors.New("record writes support only measured Windows API version 10.0.26100.0")
	}
	var iosb [2]uintptr
	attrs := make([]byte, 128)
	status, _, _ = ntVolume.Call(parent.Fd(), uintptr(unsafe.Pointer(&iosb)), uintptr(unsafe.Pointer(&attrs[0])), uintptr(len(attrs)), 5)
	if uint32(status) != 0 {
		return nativeError(uint32(status))
	}
	if iosb[1] < 12 || iosb[1] > uintptr(len(attrs)) {
		return errors.New("invalid filesystem attribute query length")
	}
	length := binary.LittleEndian.Uint32(attrs[8:12])
	if length != 8 || iosb[1] < 20 {
		return errors.New("record writes require local NTFS")
	}
	for i, c := range []uint16{'N', 'T', 'F', 'S'} {
		if binary.LittleEndian.Uint16(attrs[12+i*2:]) != c {
			return errors.New("record writes require local NTFS")
		}
	}
	var device [8]byte
	status, _, _ = ntVolume.Call(parent.Fd(), uintptr(unsafe.Pointer(&iosb)), uintptr(unsafe.Pointer(&device[0])), 8, 4)
	if uint32(status) != 0 {
		return nativeError(uint32(status))
	}
	deviceType := binary.LittleEndian.Uint32(device[:4])
	flags := binary.LittleEndian.Uint32(device[4:])
	if iosb[1] != 8 || (deviceType != 7 && deviceType != 8) || flags&0x10 != 0 {
		return errors.New("record writes require measured local NTFS device")
	}
	return platformSyncDir(parent)
}
func platformMkdir(_ *os.Root, parent *os.File, leaf string) error {
	f, err := createNative(parent, leaf, directoryFile, 0x10, parentRights)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = platformSyncDir(f); err == nil {
		err = platformSyncDir(parent)
	}
	if err != nil {
		return &PublicationError{leaf, Published, err}
	}
	return nil
}
func platformCreate(_ *os.Root, parent *os.File, leaf string) (*os.File, error) {
	return createNative(parent, leaf, nonDirectory, 0x80, readWriteAccess|deleteAccess|synchronize)
}
func platformPublish(_ *os.Root, parent, f *os.File, _, leaf string, replace bool) (bool, error) {
	u, err := syscall.UTF16FromString(leaf)
	if err != nil {
		return false, err
	}
	u = u[:len(u)-1]
	if len(u) == 0 || len(u) > 255 {
		return false, errors.New("invalid publication leaf")
	}
	data := make([]byte, 20+len(u)*2)
	if replace {
		binary.LittleEndian.PutUint32(data[:4], 1)
	}
	binary.LittleEndian.PutUint64(data[8:16], uint64(parent.Fd()))
	binary.LittleEndian.PutUint32(data[16:20], uint32(len(u)*2))
	for i, c := range u {
		binary.LittleEndian.PutUint16(data[20+i*2:], c)
	}
	var iosb [2]uintptr
	status, _, _ := ntRename.Call(f.Fd(), uintptr(unsafe.Pointer(&iosb)), uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), 65)
	runtime.KeepAlive(data)
	if uint32(status) != 0 {
		if !replace && uint32(status) == statusCollision {
			return false, nativeError(uint32(status))
		}
		return false, errors.Join(errAmbiguous, nativeError(uint32(status)))
	}
	if err := f.Sync(); err != nil {
		return true, err
	}
	if err := platformSyncDir(parent); err != nil {
		return true, err
	}
	return true, nil
}
func platformLock(f *os.File) (func() error, error) {
	var overlap syscall.Overlapped
	ok, _, err := lockFile.Call(f.Fd(), 3, 0, 1, 0, uintptr(unsafe.Pointer(&overlap)))
	if ok == 0 {
		return nil, fmt.Errorf("kernel record lease held or unavailable: %w", err)
	}
	return func() error {
		ok, _, err := unlockFile.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&overlap)))
		if ok == 0 {
			return err
		}
		return nil
	}, nil
}
