package board

import "golang.org/x/sys/unix"

func renameBoard(fd int, stage string) error {
	return unix.Renameat2(fd, stage, fd, boardName, unix.RENAME_NOREPLACE)
}
