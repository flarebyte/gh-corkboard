//go:build darwin || linux

package board

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

const safeOpenFlags = unix.O_NOFOLLOW | unix.O_NONBLOCK

func lockExclusive(file *os.File) error {
	return lockFile(file, unix.LOCK_EX)
}

func lockShared(file *os.File) error {
	return lockFile(file, unix.LOCK_SH)
}

func lockFile(file *os.File, mode int) error {
	for {
		err := unix.Flock(int(file.Fd()), mode)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}

func publishBoard(root *os.Root, stage string) error {
	return renameEntryExclusive(root, stage, boardName)
}

func renameEntryExclusive(root *os.Root, source, target string) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	return renameExclusive(int(dir.Fd()), source, target)
}

func removeEmptyDirectory(root *os.Root, name string) error {
	return unlink(root, name, unix.AT_REMOVEDIR)
}

func removeDocumentFile(root *os.Root, name string) error {
	return unlink(root, name, 0)
}

func unlink(root *os.Root, name string, flags int) error {
	parent, err := root.Open(".")
	if err != nil {
		return err
	}
	defer parent.Close()
	return unix.Unlinkat(int(parent.Fd()), name, flags)
}
