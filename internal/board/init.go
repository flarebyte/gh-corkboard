package board

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const boardName = ".gh-corkboard"
const ignoreRule = "/.gh-corkboard/"

type InitResult struct {
	Initialized     bool     `json:"initialized"`
	IgnoreRuleAdded bool     `json:"ignoreRuleAdded"`
	TrackedPaths    []string `json:"trackedPaths,omitempty"`
}

func Init(dir, actor string) (InitResult, error) {
	return initBoard(dir, actor, publishBoard)
}

func initBoard(dir, actor string, publish func(*os.Root, string) error) (InitResult, error) {
	var result InitResult
	kind, _, valid := principalParts(actor)
	if !valid || kind != "human" {
		return result, errors.New("init requires --as human:<id>")
	}
	worktree, gitDir, err := workspace(dir)
	if err != nil {
		return result, err
	}

	bookkeeping, err := os.OpenRoot(gitDir)
	if err != nil {
		return result, err
	}
	defer bookkeeping.Close()
	lock, err := openLock(bookkeeping, "corkboard-init.lock")
	if err != nil {
		return result, err
	}
	defer lock.Close()

	root, err := os.OpenRoot(worktree)
	if err != nil {
		return result, err
	}
	defer root.Close()

	tracked, err := git(worktree, "ls-files", "-z", "--", boardName)
	if err != nil {
		return result, err
	}
	if tracked != "" {
		result.TrackedPaths = strings.Split(strings.TrimSuffix(tracked, "\x00"), "\x00")
	}

	existing, err := openDirectory(root, boardName)
	if err == nil {
		defer existing.Close()
		result.IgnoreRuleAdded, err = reinitialize(existing, root, worktree, actor)
		return result, err
	}
	// A disappearing or replaced board is an error unless the canonical entry is absent.
	if _, statErr := root.Lstat(boardName); !os.IsNotExist(statErr) {
		return result, err
	}

	stage := ".corkboard-init-" + rand.Text()
	if err := root.Mkdir(stage, 0700); err != nil {
		return result, err
	}
	defer root.RemoveAll(stage)
	if err := prepareBoard(root, stage, actor); err != nil {
		return result, err
	}
	result.IgnoreRuleAdded, err = ensureIgnored(root, worktree)
	if err != nil {
		return result, err
	}
	if err := publish(root, stage); err != nil {
		return result, err
	}
	result.Initialized = true
	return result, nil
}

func reinitialize(board, root *os.Root, worktree, actor string) (bool, error) {
	index, err := openDirectory(board, "index")
	if err != nil {
		return false, err
	}
	defer index.Close()
	lock, err := openRegular(index, "board.lock", os.O_RDWR)
	if err != nil {
		return false, err
	}
	defer lock.Close()
	if err := lockExclusive(lock); err != nil {
		return false, err
	}
	policy, err := loadAccess(index)
	if err != nil {
		return false, err
	}
	if policy.role(actor) != "admin" {
		return false, errors.New("init requires an existing board administrator")
	}
	return ensureIgnored(root, worktree)
}

func prepareBoard(root *os.Root, stage, actor string) error {
	board, err := openDirectory(root, stage)
	if err != nil {
		return err
	}
	defer board.Close()
	for _, dir := range []string{"index", "schemas/json", "schemas/csv", "schemas/md", "agents", "skills", "topics"} {
		if err := board.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	index, err := openDirectory(board, "index")
	if err != nil {
		return err
	}
	defer index.Close()
	policy := accessPolicy{Version: 1, Principals: []principal{{ID: actor, Role: "admin"}}, Topics: []string{}, Grants: []grant{}}
	if err := writeMetadata(index, "access", policy); err != nil {
		return err
	}
	if err := writeMetadata(index, "documents", catalog{Version: 1, Documents: []document{}}); err != nil {
		return err
	}
	lock, err := index.OpenFile("board.lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if err := lock.Close(); err != nil {
		return err
	}
	loaded, err := loadAccess(index)
	if err != nil {
		return err
	}
	_, err = loadCatalog(index, loaded)
	return err
}

func writeMetadata(root *os.Root, name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return replaceFile(root, name+".json", append(data, '\n'), 0600)
}

func ensureIgnored(root *os.Root, worktree string) (bool, error) {
	data, mode, err := readIgnore(root)
	if err != nil {
		return false, err
	}

	trimmed := bytes.TrimRight(data, "\r\n")
	added := !bytes.Equal(trimmed, []byte(ignoreRule)) && !bytes.HasSuffix(trimmed, []byte("\n"+ignoreRule))
	if added {
		if len(data) > 0 && data[len(data)-1] != '\n' {
			data = append(data, '\n')
		}
		data = append(data, ignoreRule+"\n"...)
		if err := replaceFile(root, ".gitignore", data, mode); err != nil {
			return false, err
		}
	}

	cmd := exec.Command("git", "check-ignore", "--no-index", "--quiet", "--", boardName+"/")
	cmd.Dir = worktree
	if err := cmd.Run(); err != nil {
		return added, fmt.Errorf("verify root corkboard ignore rule: %w", err)
	}
	return added, nil
}

func readIgnore(root *os.Root) ([]byte, os.FileMode, error) {
	file, err := openRegular(root, ".gitignore", os.O_RDONLY)
	if os.IsNotExist(err) {
		return nil, 0644, nil
	}
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	data, err := io.ReadAll(file)
	return data, info.Mode().Perm(), err
}
