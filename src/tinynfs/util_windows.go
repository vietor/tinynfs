package tinynfs

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                = syscall.MustLoadDLL("kernel32.dll")
	procLockFileEx          = kernel32.MustFindProc("LockFileEx")
	procUnlockFileEx        = kernel32.MustFindProc("UnlockFileEx")
	procGetDiskFreeSpaceExW = kernel32.MustFindProc("GetDiskFreeSpaceExW")
)

const (
	// see https://msdn.microsoft.com/en-us/library/windows/desktop/aa365203(v=vs.85).aspx
	flagLockExclusive       = 2
	flagLockFailImmediately = 1
	// see https://msdn.microsoft.com/en-us/library/windows/desktop/ms681382(v=vs.85).aspx
	errLockViolation syscall.Errno = 0x21
)

func lockFileEx(h syscall.Handle, flags, reserved, locklow, lockhigh uint32, ol *syscall.Overlapped) error {
	r, _, err := procLockFileEx.Call(uintptr(h), uintptr(flags), uintptr(reserved), uintptr(locklow), uintptr(lockhigh), uintptr(unsafe.Pointer(ol)))
	if r == 0 {
		return err
	}
	return nil
}

func unlockFileEx(h syscall.Handle, reserved, locklow, lockhigh uint32, ol *syscall.Overlapped) error {
	r, _, err := procUnlockFileEx.Call(uintptr(h), uintptr(reserved), uintptr(locklow), uintptr(lockhigh), uintptr(unsafe.Pointer(ol)), 0)
	if r == 0 {
		return err
	}
	return nil
}

func SysFlock(fd int) error {
	var flag uint32 = flagLockFailImmediately | flagLockExclusive
	// Attempt to obtain an exclusive lock.
	err := lockFileEx(syscall.Handle(fd), flag, 0, 1, 0, &syscall.Overlapped{})
	if err == nil {
		return nil
	} else if err != errLockViolation {
		return err
	}
	return nil
}

func SysUnflock(fd int) error {
	return unlockFileEx(syscall.Handle(fd), 0, 1, 0, &syscall.Overlapped{})
}

func GetDiskStat(path string) (*DiskStat, error) {
	lpFreeBytesAvailable := int64(0)
	lpTotalNumberOfBytes := int64(0)
	lpTotalNumberOfFreeBytes := int64(0)
	r, _, err := procGetDiskFreeSpaceExW.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		uintptr(unsafe.Pointer(&lpFreeBytesAvailable)),
		uintptr(unsafe.Pointer(&lpTotalNumberOfBytes)),
		uintptr(unsafe.Pointer(&lpTotalNumberOfFreeBytes)))
	if r == 0 {
		return nil, err
	}
	info := &DiskStat{
		Size: uint64(lpTotalNumberOfBytes),
		Free: uint64(lpTotalNumberOfFreeBytes),
	}
	info.Used = info.Size - info.Free
	return info, nil
}
