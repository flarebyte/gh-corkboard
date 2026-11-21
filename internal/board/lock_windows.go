package board

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

const safeOpenFlags = 0

func lockExclusive(file *os.File) error {
	return lockFile(file, windows.LOCKFILE_EXCLUSIVE_LOCK)
}

func lockShared(file *os.File) error {
	return lockFile(file, 0)
}

func lockFile(file *os.File, flags uint32) error {
	var position windows.Overlapped
	return windows.LockFileEx(windows.Handle(file.Fd()), flags, 0, 1, 0, &position)
}

func publishBoard(root *os.Root, stage string) error {
	source, err := windows.UTF16PtrFromString(filepath.Join(root.Name(), stage))
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(filepath.Join(root.Name(), boardName))
	if err != nil {
		return err
	}
	return windows.MoveFile(source, target)
}

func removeEmptyDirectory(root *os.Root, name string) error {
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", name)
	}
	target, err := windows.UTF16PtrFromString(filepath.Join(root.Name(), name))
	if err != nil {
		return err
	}
	return windows.RemoveDirectory(target)
}

func removeDocumentFile(root *os.Root, name string) error {
	return root.Remove(name)
}
