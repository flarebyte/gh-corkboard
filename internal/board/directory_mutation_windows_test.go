package board

import (
	"os"
	"testing"
)

func TestWindowsDirectoryRemovalLeavesRegularFile(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := root.WriteFile("regular", []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := removeEmptyDirectory(root, "regular"); err == nil {
		t.Fatal("directory removal accepted a regular file")
	}
	if content, err := root.ReadFile("regular"); err != nil || string(content) != "keep" {
		t.Fatalf("regular file changed: %q, %v", content, err)
	}

	if err := root.Mkdir("empty", 0700); err != nil {
		t.Fatal(err)
	}
	if err := removeEmptyDirectory(root, "empty"); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Lstat("empty"); !os.IsNotExist(err) {
		t.Fatalf("empty directory remains: %v", err)
	}
}
