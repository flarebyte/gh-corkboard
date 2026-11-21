package board

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Use relative NT handles so both names stay in the anchored scope and a
// target that appears after planning cannot be replaced.
func renameEntryExclusive(root *os.Root, source, target string) error {
	parent, err := root.Open(".")
	if err != nil {
		return err
	}
	defer parent.Close()
	directory := windows.Handle(parent.Fd())
	name, err := windows.NewNTUnicodeString(source)
	if err != nil {
		return err
	}
	attrs := windows.OBJECT_ATTRIBUTES{RootDirectory: directory, ObjectName: name, Attributes: windows.OBJ_DONT_REPARSE}
	attrs.Length = uint32(unsafe.Sizeof(attrs))
	var file windows.Handle
	var status windows.IO_STATUS_BLOCK
	err = windows.NtCreateFile(&file, windows.DELETE|windows.SYNCHRONIZE, &attrs, &status, nil, 0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, windows.FILE_OPEN,
		windows.FILE_NON_DIRECTORY_FILE|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
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
	return windows.NtSetInformationFile(file, &status, (*byte)(unsafe.Pointer(&info)),
		uint32(unsafe.Offsetof(info.FileName))+info.FileNameLength, windows.FileRenameInformation)
}
