package board

import (
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Use relative NT handles so both names stay in the anchored scope and a
// target that appears after planning cannot be replaced.
func renameEntryExclusive(root *os.Root, source, target string) error {
	return renameChildExclusive(root, source, target, windows.FILE_NON_DIRECTORY_FILE)
}

func renameChildExclusive(root *os.Root, source, target string, kind uint32) error {
	if !validWindowsChild(target) {
		return windows.ERROR_INVALID_NAME
	}
	parent, err := root.Open(".")
	if err != nil {
		return err
	}
	defer parent.Close()
	directory := windows.Handle(parent.Fd())
	file, err := openChildForDelete(directory, source, kind)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(file)

	var info struct {
		ReplaceIfExists uint8
		RootDirectory   windows.Handle
		FileNameLength  uint32
		FileName        [256]uint16
	}
	utf16, err := windows.UTF16FromString(target)
	if err != nil {
		return err
	}
	if len(utf16) > len(info.FileName) {
		return windows.ERROR_FILENAME_EXCED_RANGE
	}
	info.RootDirectory = directory
	info.FileNameLength = uint32(2 * (len(utf16) - 1))
	copy(info.FileName[:], utf16)
	var status windows.IO_STATUS_BLOCK
	return windowsError(windows.NtSetInformationFile(file, &status, (*byte)(unsafe.Pointer(&info)),
		uint32(unsafe.Offsetof(info.FileName))+info.FileNameLength, windows.FileRenameInformation))
}

func validWindowsChild(name string) bool {
	return name != "." && filepath.IsLocal(name) && filepath.Base(name) == name
}

func openChildForDelete(directory windows.Handle, child string, kind uint32) (windows.Handle, error) {
	if !validWindowsChild(child) {
		return 0, windows.ERROR_INVALID_NAME
	}
	name, err := windows.NewNTUnicodeString(child)
	if err != nil {
		return 0, err
	}
	attrs := windows.OBJECT_ATTRIBUTES{RootDirectory: directory, ObjectName: name, Attributes: windows.OBJ_DONT_REPARSE}
	attrs.Length = uint32(unsafe.Sizeof(attrs))
	var file windows.Handle
	var status windows.IO_STATUS_BLOCK
	err = windows.NtCreateFile(&file, windows.DELETE|windows.SYNCHRONIZE, &attrs, &status, nil, 0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, windows.FILE_OPEN,
		kind|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	return file, windowsError(err)
}

func windowsError(err error) error {
	if status, ok := err.(windows.NTStatus); ok {
		return status.Errno()
	}
	return err
}
