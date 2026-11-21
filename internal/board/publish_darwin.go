package board

import "golang.org/x/sys/unix"

func renameBoard(fd int, stage string) error {
	return unix.RenameatxNp(fd, stage, fd, boardName, unix.RENAME_EXCL)
}
