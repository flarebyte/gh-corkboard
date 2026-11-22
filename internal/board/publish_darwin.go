package board

import "golang.org/x/sys/unix"

func renameExclusive(fd int, source, target string) error {
	return unix.RenameatxNp(fd, source, fd, target, unix.RENAME_EXCL)
}
