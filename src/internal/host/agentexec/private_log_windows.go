//go:build windows

package agentexec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	securityDescriptorRevision = 1
	seFileObject               = 1
	ownerSecurityInformation   = 0x00000001
	daclSecurityInformation    = 0x00000004
	seDACLProtected            = 0x1000
	accessAllowedACEType       = 0
	objectInheritACE           = 0x01
	containerInheritACE        = 0x02
	inheritedACE               = 0x10
	fileAllAccess              = 0x001F01FF
)

type privateACL struct {
	revision byte
	sbz1     byte
	aclSize  uint16
	aceCount uint16
	sbz2     uint16
}

type privateACEHeader struct {
	typ   byte
	flags byte
	size  uint16
}

var (
	privateAdvapi32                  = syscall.NewLazyDLL("advapi32.dll")
	procConvertStringSDDLToSD        = privateAdvapi32.NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
	procGetNamedSecurityInfo         = privateAdvapi32.NewProc("GetNamedSecurityInfoW")
	procGetSecurityDescriptorControl = privateAdvapi32.NewProc("GetSecurityDescriptorControl")
	procSetNamedSecurityInfo         = privateAdvapi32.NewProc("SetNamedSecurityInfoW")
)

func createPrivateLogDirectory(path string) error {
	sid, err := currentUserSID()
	if err != nil {
		return err
	}
	// Set the owner explicitly. Elevated Windows tokens can use an owner group
	// (for example BUILTIN\\Administrators) as their default object owner even
	// when the token's user SID is runneradmin. The DACL remains protected and
	// grants access only to that user SID.
	sddl, err := syscall.UTF16PtrFromString(fmt.Sprintf("O:%sD:P(A;OICI;FA;;;%s)", sid, sid))
	if err != nil {
		return err
	}
	var descriptor uintptr
	var descriptorBytes uint32
	converted, _, callErr := procConvertStringSDDLToSD.Call(
		uintptr(unsafe.Pointer(sddl)),
		securityDescriptorRevision,
		uintptr(unsafe.Pointer(&descriptor)),
		uintptr(unsafe.Pointer(&descriptorBytes)),
	)
	if converted == 0 || descriptor == 0 {
		return winCallError(callErr, "owner-only access descriptor could not be built")
	}
	defer syscall.LocalFree(syscall.Handle(descriptor))
	pathPtr, err := syscall.UTF16PtrFromString(filepath.Clean(path))
	if err != nil {
		return err
	}
	attributes := syscall.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(syscall.SecurityAttributes{})),
		SecurityDescriptor: descriptor,
		InheritHandle:      0,
	}
	if err := syscall.CreateDirectory(pathPtr, &attributes); err != nil {
		return fmt.Errorf("CreateDirectoryW: %w", err)
	}
	if err := verifyPrivateLogDirectory(path); err != nil {
		return fmt.Errorf("verify created private log directory: %w", err)
	}
	return nil
}

func verifyPrivateLogDirectory(path string) error {
	return verifyPrivateObjectACL(path, true)
}

func verifyPrivateLogFile(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("private log must be a regular non-reparse file")
	}
	if err := verifyPrivateObjectACL(path, false); err == nil {
		return nil
	} else if !errors.Is(err, errPrivateLogOwnerMismatch) {
		return err
	}
	// Files created beneath a protected directory inherit its DACL, but Windows
	// may assign the token's default owner group. First verify the complete
	// owner-only DACL while allowing only that owner mismatch, then set the file
	// owner to the current user and run the strict verification again.
	if err := verifyPrivateObjectACLWithOwner(path, false, false); err != nil {
		return err
	}
	ownerSID, err := currentUserSID()
	if err != nil {
		return err
	}
	if err := setPrivateObjectOwner(path, ownerSID); err != nil {
		return err
	}
	return verifyPrivateObjectACL(path, false)
}

var errPrivateLogOwnerMismatch = errors.New("private log owner differs from the current user")

func verifyPrivateObjectACL(path string, directory bool) error {
	return verifyPrivateObjectACLWithOwner(path, directory, true)
}

func verifyPrivateObjectACLWithOwner(path string, directory, requireCurrentUserOwner bool) error {
	expectedOwner, err := currentUserSID()
	if err != nil {
		return err
	}
	pathPtr, err := syscall.UTF16PtrFromString(filepath.Clean(path))
	if err != nil {
		return err
	}
	var owner *syscall.SID
	var dacl *privateACL
	var descriptor uintptr
	result, _, callErr := procGetNamedSecurityInfo.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		seFileObject,
		ownerSecurityInformation|daclSecurityInformation,
		uintptr(unsafe.Pointer(&owner)),
		0,
		uintptr(unsafe.Pointer(&dacl)),
		0,
		uintptr(unsafe.Pointer(&descriptor)),
	)
	if result != 0 || descriptor == 0 {
		if result != 0 {
			return fmt.Errorf("GetNamedSecurityInfoW: %w", syscall.Errno(uint32(result)))
		}
		return winCallError(callErr, "private log access list could not be read")
	}
	defer syscall.LocalFree(syscall.Handle(descriptor))
	if owner == nil || dacl == nil {
		return errors.New("private log access list is absent")
	}
	ownerSID, err := owner.String()
	if err != nil {
		return errors.New("private log owner could not be read")
	}
	if requireCurrentUserOwner && ownerSID != expectedOwner {
		return errPrivateLogOwnerMismatch
	}
	if directory {
		var control uint16
		var revision uint32
		ok, _, controlErr := procGetSecurityDescriptorControl.Call(
			descriptor,
			uintptr(unsafe.Pointer(&control)),
			uintptr(unsafe.Pointer(&revision)),
		)
		if ok == 0 || control&seDACLProtected == 0 {
			return winCallError(controlErr, "private log directory access list is not protected from inheritance")
		}
	}
	if dacl.aceCount != 1 || dacl.aclSize < uint16(unsafe.Sizeof(privateACL{}))+uint16(unsafe.Sizeof(privateACEHeader{}))+4 {
		return errors.New("private log access list is not owner-only")
	}
	aclBytes := unsafe.Slice((*byte)(unsafe.Pointer(dacl)), int(dacl.aclSize))
	aceOffset := int(unsafe.Sizeof(privateACL{}))
	if aceOffset+int(unsafe.Sizeof(privateACEHeader{})) > len(aclBytes) {
		return errors.New("private log access entry is malformed")
	}
	header := (*privateACEHeader)(unsafe.Pointer(&aclBytes[aceOffset]))
	if header.typ != accessAllowedACEType || int(header.size) < int(unsafe.Sizeof(privateACEHeader{}))+4 || aceOffset+int(header.size) > len(aclBytes) {
		return errors.New("private log access entry is malformed")
	}
	ace := aclBytes[aceOffset : aceOffset+int(header.size)]
	mask := *(*uint32)(unsafe.Pointer(&ace[unsafe.Sizeof(privateACEHeader{})]))
	if mask != fileAllAccess {
		return errors.New("private log access entry does not grant only owner full control")
	}
	flags := header.flags
	if directory {
		if flags != objectInheritACE|containerInheritACE {
			return errors.New("private log directory access entry has unexpected inheritance flags")
		}
	} else if flags & ^byte(objectInheritACE|containerInheritACE|inheritedACE) != 0 {
		return errors.New("private log file access entry has unexpected flags")
	}
	sidSize, err := privateACEOwnerSIDSize(ace)
	if err != nil {
		return err
	}
	aceSID := (*syscall.SID)(unsafe.Pointer(&ace[8]))
	if int(aceSID.Len()) != sidSize {
		return errors.New("private log access entry has a truncated owner identity")
	}
	aceOwnerSID, err := aceSID.String()
	if err != nil || aceOwnerSID != expectedOwner {
		return errors.New("private log access entry grants a different identity")
	}
	return nil
}

func privateACEOwnerSIDSize(ace []byte) (int, error) {
	// An allowed ACE needs its header and mask (8 bytes), followed by the
	// fixed 8-byte SID header before the subauthority count at offset 9 is safe
	// to read.
	if len(ace) < 16 {
		return 0, errors.New("private log access entry has a truncated owner identity")
	}
	sidSize := 8 + int(ace[9])*4
	if sidSize > len(ace)-8 {
		return 0, errors.New("private log access entry has a truncated owner identity")
	}
	return sidSize, nil
}

func setPrivateObjectOwner(path, sid string) error {
	owner, err := syscall.StringToSid(sid)
	if err != nil {
		return errors.New("current Windows user identity could not be represented as a SID")
	}
	pathPtr, err := syscall.UTF16PtrFromString(filepath.Clean(path))
	if err != nil {
		return err
	}
	result, _, _ := procSetNamedSecurityInfo.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		seFileObject,
		ownerSecurityInformation,
		uintptr(unsafe.Pointer(owner)),
		0,
		0,
		0,
	)
	if result != 0 {
		return fmt.Errorf("SetNamedSecurityInfoW: %w", syscall.Errno(uint32(result)))
	}
	return nil
}

func currentUserSID() (string, error) {
	token, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return "", err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil || user == nil || user.User.Sid == nil {
		return "", errors.New("current Windows user identity could not be resolved")
	}
	sid, err := user.User.Sid.String()
	if err != nil {
		return "", errors.New("current Windows user identity could not be resolved")
	}
	return sid, nil
}

func winCallError(callErr error, fallback string) error {
	if callErr != nil && callErr != syscall.Errno(0) {
		return fmt.Errorf("%s: %w", fallback, callErr)
	}
	return errors.New(fallback)
}
