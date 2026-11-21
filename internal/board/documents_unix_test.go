//go:build darwin || linux

package board

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestDocumentRejectsNonRegularPaths(t *testing.T) {
	root := accessFixture(t)
	editAccess(t, root, func(a *Access) error { return a.CreateTopic("specs") })
	var created DocumentMetadata
	err := WithDocuments(root, owner, true, func(d *Documents) error {
		var err error
		created, err = d.Create(CreateOptions{Scope: "topic:specs", Name: "notes", Format: "md"}, strings.NewReader("notes"))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	absolute := filepath.Join(root, boardName, "topics", "specs", created.ID+"--notes.md")
	if err := os.Remove(absolute); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "target.md")
	if err := os.WriteFile(outside, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		replace func() error
	}{
		{"symlink", func() error { return os.Symlink(outside, absolute) }},
		{"fifo", func() error { return unix.Mkfifo(absolute, 0600) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.replace(); err != nil {
				t.Fatal(err)
			}
			defer os.Remove(absolute)
			if err := WithDocuments(root, owner, true, func(d *Documents) error {
				if _, err := d.Get(created.ID, io.Discard, false); err == nil {
					t.Fatal("read accepted a non-regular file")
				}
				if err := d.Delete(created.ID); err == nil {
					t.Fatal("delete accepted a non-regular file")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(absolute); err != nil {
				t.Fatalf("non-regular entry changed: %v", err)
			}
		})
	}
	if value, err := os.ReadFile(outside); err != nil || string(value) != "untouched" {
		t.Fatalf("external file: %q, %v", value, err)
	}
}
