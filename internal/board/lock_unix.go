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
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	return renameBoard(int(dir.Fd()), stage)
}

func removeEmptyDirectory(root *os.Root, name string) error {
	parent, err := root.Open(".")
	if err != nil {
		return err
	}
	defer parent.Close()
	return unix.Unlinkat(int(parent.Fd()), name, unix.AT_REMOVEDIR)
}
