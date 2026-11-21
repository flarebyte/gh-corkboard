package board

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"unicode/utf8"
)

var snapshotFolders = []string{"agents", "index", "schemas", "skills", "topics"}

func (a *Access) Snapshot(out io.Writer, expected *string) error {
	if a.policy.role(a.actor) != "admin" {
		return errors.New("snapshot create requires a board administrator")
	}
	if expected != nil {
		if err := ValidatePlanID(*expected); err != nil {
			return err
		}
		plan, err := a.PreviewPrune()
		if err != nil {
			return err
		}
		if plan.ID != *expected {
			return ErrPrunePlanMismatch
		}
	}

	archive := zip.NewWriter(out)
	if _, err := archive.Create(boardName + "/"); err != nil {
		return err
	}
	for _, name := range snapshotFolders {
		if err := archiveDirectory(archive, a.root, name, path.Join(boardName, name)); err != nil {
			return fmt.Errorf("snapshot: %w", err)
		}
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("finish snapshot ZIP: %w", err)
	}
	return nil
}

func archiveDirectory(archive *zip.Writer, parent *os.Root, name, zipPath string) error {
	root, err := openDirectory(parent, name)
	if err != nil {
		return err
	}
	defer root.Close()
	if _, err := archive.Create(zipPath + "/"); err != nil {
		return err
	}
	entries, err := readDirectory(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if isTemporary(name) || (zipPath == boardName+"/index" && name == "board.lock") {
			continue
		}
		if !safeArchiveName(name) {
			return fmt.Errorf("unsafe archive name %q", path.Join(zipPath, name))
		}
		archivePath := path.Join(zipPath, name)
		if entry.IsDir() {
			err = archiveDirectory(archive, root, name, archivePath)
		} else {
			err = archiveFile(archive, root, name, archivePath)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func safeArchiveName(name string) bool {
	stem, _, _ := strings.Cut(strings.ToLower(name), ".")
	stem = strings.TrimRight(stem, " ")
	if reserved.MatchString(stem) || stem == "conin$" || stem == "conout$" {
		return false
	}
	return utf8.ValidString(name) && displayText(name, false) && name != "" && name != "." && name != ".." &&
		!strings.ContainsAny(name, `\/:<>"|?*`) && !strings.HasSuffix(name, " ") && !strings.HasSuffix(name, ".")
}

func archiveFile(archive *zip.Writer, root *os.Root, name, zipPath string) error {
	file, err := openRegular(root, name, os.O_RDONLY)
	if err != nil {
		return fmt.Errorf("archive %q: %w", zipPath, err)
	}
	defer file.Close()
	dst, err := archive.Create(zipPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, file)
	return err
}
