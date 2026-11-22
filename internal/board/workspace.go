package board

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func git(dir string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		if failure, ok := err.(*exec.ExitError); ok && len(failure.Stderr) != 0 {
			return "", fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(string(failure.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return strings.TrimSuffix(strings.TrimSuffix(string(output), "\n"), "\r"), nil
}

func workspace(dir string) (string, string, error) {
	root, err := git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", fmt.Errorf("a Git worktree is required: %w", err)
	}
	gitDir, err := git(root, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", "", err
	}
	return filepath.FromSlash(root), filepath.FromSlash(gitDir), nil
}
