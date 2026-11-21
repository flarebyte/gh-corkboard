package board

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

const safeOpenFlags = 0

func lockExclusive(file *os.File) error {
	var position windows.Overlapped
	return windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &position)
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
