package board

import (
	"crypto/rand"
	"fmt"
	"io/fs"
	"os"
)

func openRegular(root *os.Root, name string, flags int) (*os.File, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%q is not a regular file", name)
	}
	file, err := root.OpenFile(name, flags|safeOpenFlags, 0)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) {
		file.Close()
		return nil, fmt.Errorf("%q changed while opening", name)
	}
	return file, nil
}

// openDirectory anchors subsequent operations to the directory we inspected.
func openDirectory(root *os.Root, name string) (*os.Root, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", name)
	}
	child, err := root.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	after, err := child.Stat(".")
	if err != nil || !os.SameFile(before, after) {
		child.Close()
		return nil, fmt.Errorf("%q changed while opening", name)
	}
	return child, nil
}

func replaceFile(root *os.Root, name string, data []byte, mode fs.FileMode) error {
	return stageFile(root, data, mode, func(temp string) error { return root.Rename(temp, name) })
}

func stageFile(root *os.Root, data []byte, mode fs.FileMode, publish func(string) error) error {
	temp := ".corkboard-tmp-" + rand.Text()
	file, err := root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer root.Remove(temp)
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return publish(temp)
}

func openLock(root *os.Root, name string) (*os.File, error) {
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if os.IsExist(err) {
		file, err = openRegular(root, name, os.O_RDWR)
	}
	if err != nil {
		return nil, err
	}
	if err := lockExclusive(file); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

// createFile publishes complete bytes while refusing to overwrite an existing name.
func createFile(root *os.Root, name string, data []byte) error {
	return stageFile(root, data, 0600, func(temp string) error { return root.Link(temp, name) })
}

func readDirectory(root *os.Root) ([]fs.DirEntry, error) {
	return fs.ReadDir(root.FS(), ".")
}
