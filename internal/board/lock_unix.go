//go:build darwin || linux

package board

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

const safeOpenFlags = unix.O_NOFOLLOW | unix.O_NONBLOCK

func lockExclusive(file *os.File) error {
	for {
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX)
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
