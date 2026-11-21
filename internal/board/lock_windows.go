package board

import (
	"os"

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
	return renameChildExclusive(root, stage, boardName, windows.FILE_DIRECTORY_FILE)
}

func removeEmptyDirectory(root *os.Root, name string) error {
	parent, err := root.Open(".")
	if err != nil {
		return err
	}
	defer parent.Close()
	child, err := openChildForDelete(windows.Handle(parent.Fd()), name, windows.FILE_DIRECTORY_FILE)
	if err != nil {
		return err
	}

	// The kernel enforces directory emptiness on the opened handle.
	var status windows.IO_STATUS_BLOCK
	deleteOnClose := byte(1)
	err = windows.NtSetInformationFile(child, &status, &deleteOnClose, 1, windows.FileDispositionInformation)
	closeErr := windows.CloseHandle(child)
	if err != nil {
		return windowsError(err)
	}
	return closeErr
}

func removeDocumentFile(root *os.Root, name string) error {
	return root.Remove(name)
}
