package board

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/testrepo"
)

func TestInitFromNestedDirectory(t *testing.T) {
	root := testrepo.New(t)
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	prefix := "*.local\n!/.gh-corkboard/"
	ignore := filepath.Join(root, ".gitignore")
	if err := os.WriteFile(ignore, []byte(prefix), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := Init(nested, "human:owner")
	if err != nil || !result.Initialized || !result.IgnoreRuleAdded {
		t.Fatalf("init: %+v, %v", result, err)
	}
	for _, name := range []string{"index", "schemas/json", "schemas/csv", "schemas/md", "agents", "skills", "topics"} {
		if info, err := os.Stat(filepath.Join(root, boardName, name)); err != nil || !info.IsDir() {
			t.Fatalf("missing board directory %s: %v", name, err)
		}
	}
	data, err := os.ReadFile(ignore)
	if err != nil || string(data) != prefix+"\n"+ignoreRule+"\n" {
		t.Fatalf("ignore contents: %q, %v", data, err)
	}
	testrepo.Git(t, root, "check-ignore", "--no-index", "--quiet", "--", boardName+"/index/access.json")
	note := filepath.Join(root, boardName, "agents", "note")
	if err := os.WriteFile(note, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ignore, []byte(prefix), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, "human:another"); err == nil {
		t.Fatal("unauthorized rerun succeeded")
	}
	result, err = Init(root, "human:owner")
	if err != nil || result.Initialized || !result.IgnoreRuleAdded {
		t.Fatalf("rerun: %+v, %v", result, err)
	}
	content, err := os.ReadFile(note)
	if err != nil || string(content) != "preserve" {
		t.Fatalf("note after rerun: %q, %v", content, err)
	}
}

func TestInitRejectsUnsafeOrDamagedBoard(t *testing.T) {
	for _, actor := range []string{"", "agent:owner", "human:CON", "human:nul"} {
		t.Run(actor, func(t *testing.T) {
			if _, err := Init(testrepo.New(t), actor); err == nil {
				t.Fatalf("accepted %q", actor)
			}
		})
	}
	root := testrepo.New(t)
	if _, err := Init(root, "human:owner"); err != nil {
		t.Fatal(err)
	}
	access := filepath.Join(root, boardName, "index", "access.json")
	damaged := []byte(`{"version":1,"version":1}`)
	if err := os.WriteFile(access, damaged, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, "human:owner"); err == nil {
		t.Fatal("damaged board accepted")
	}
	data, err := os.ReadFile(access)
	if err != nil || string(data) != string(damaged) {
		t.Fatalf("policy changed: %q, %v", data, err)
	}
}

func TestConcurrentInit(t *testing.T) {
	root := testrepo.New(t)
	results := make(chan error, 2)
	for _, actor := range []string{"human:first", "human:second"} {
		go func(actor string) {
			_, err := Init(root, actor)
			results <- err
		}(actor)
	}
	first, second := <-results, <-results
	if (first == nil) == (second == nil) {
		t.Fatalf("wanted one winner: %v, %v", first, second)
	}
}

func TestInitRetryAroundPublication(t *testing.T) {
	for _, published := range []bool{false, true} {
		name := "before"
		if published {
			name = "after"
		}
		t.Run(name, func(t *testing.T) {
			root := testrepo.New(t)
			interruption := errors.New("interrupted")
			_, err := initBoard(root, "human:owner", func(root *os.Root, stage string) error {
				if published {
					if err := publishBoard(root, stage); err != nil {
						return err
					}
				}
				return interruption
			})
			if !errors.Is(err, interruption) {
				t.Fatalf("init: %v", err)
			}
			if !published {
				// A subsequent invocation tolerates orphan staging left by a terminated process.
				if err := os.Mkdir(filepath.Join(root, ".corkboard-init-stale"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			result, err := Init(root, "human:owner")
			if err != nil || result.Initialized == published {
				t.Fatalf("retry: %+v, %v", result, err)
			}
		})
	}
}

func TestLinkedWorktreeIsolation(t *testing.T) {
	source, _, err := workspace("")
	if err != nil {
		t.Skip("requires a Git source checkout with a HEAD")
	}
	root := testrepo.New(t)
	testrepo.Git(t, root, "fetch", "-q", source, "HEAD")
	linked := filepath.Join(t.TempDir(), "linked")
	testrepo.Git(t, root, "worktree", "add", "--quiet", "--detach", linked, "FETCH_HEAD")
	if _, err := Init(linked, "human:linked"); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, "human:main"); err != nil {
		t.Fatal(err)
	}
	for dir, expected := range map[string]string{linked: "human:linked", root: "human:main"} {
		data, err := os.ReadFile(filepath.Join(dir, boardName, "index", "access.json"))
		if err != nil || !strings.Contains(string(data), expected) {
			t.Fatalf("%s: %s, %v", dir, data, err)
		}
	}
}
