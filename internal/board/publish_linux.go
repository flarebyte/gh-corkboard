package board

import "golang.org/x/sys/unix"

func renameExclusive(fd int, source, target string) error {
	return unix.Renameat2(fd, source, fd, target, unix.RENAME_NOREPLACE)
}
