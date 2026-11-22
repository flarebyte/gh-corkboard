package cli

import (
	"archive/zip"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/board"
)

func (c docCLI) prunePlan() board.PrunePlan {
	c.t.Helper()
	out, _ := c.run(0, updateAdmin, "", "index", "prune", "--json")
	var plan board.PrunePlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil || plan.ID == "" {
		c.t.Fatalf("plan: %q, %v", out, err)
	}
	return plan
}

func TestRetireWorkflow(t *testing.T) {
	cli, root := updateFixture(t)
	candidate := cli.create(updateAdmin, "retired notes", updateScope, "retired", "md", "--status", "archived")
	changed := cli.create(updateAdmin, "edited notes", updateScope, "changed", "md", "--status", "archived")
	missing := cli.create(updateAdmin, "lost notes", updateScope, "missing", "md", "--status", "archived")
	scope := filepath.Join(root, ".gh-corkboard", "topics", "specs")
	if err := os.WriteFile(filepath.Join(scope, changed.ID+"--changed.md"), []byte("external edit"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(scope, missing.ID+"--missing.md")); err != nil {
		t.Fatal(err)
	}
	cli.run(2, updateWriter, "", "index", "prune")
	cli.run(2, updateWriter, "", "snapshot", "create")
	cli.run(2, updateAdmin, "", "index", "prune", "--apply")

	plan := cli.prunePlan()
	if len(plan.Candidates) != 1 || plan.Candidates[0].ID != candidate.ID || len(plan.Skipped) != 2 {
		t.Fatalf("preview: %+v", plan)
	}
	reasons := map[string]string{}
	for _, item := range plan.Skipped {
		reasons[item.ID] = item.Reason
	}
	if reasons[changed.ID] != "changed" || reasons[missing.ID] != "missing" || cli.prunePlan().ID != plan.ID {
		t.Fatalf("unstable preview or missing skips: %+v", plan)
	}

	archive, _ := cli.run(0, updateAdmin, "", "snapshot", "create", "--expect", plan.ID)
	archiveName := filepath.Join(root, "before-prune.zip")
	if err := os.WriteFile(archiveName, []byte(archive), 0600); err != nil {
		t.Fatal(err)
	}
	out, _ := cli.run(0, updateAdmin, "", "index", "prune", "--apply", "--expect", plan.ID, "--json")
	var result board.PruneResult
	if err := json.Unmarshal([]byte(out), &result); err != nil || result.PlanID != plan.ID ||
		!slices.Equal(result.Deleted, []string{candidate.ID}) || len(result.Remaining) != 0 {
		t.Fatalf("apply: %q, %v", out, err)
	}
	cli.run(2, updateAdmin, "", "doc", "get", candidate.ID)
	if _, err := os.Stat(filepath.Join(scope, candidate.ID+"--retired.md")); !os.IsNotExist(err) {
		t.Fatalf("candidate file still exists: %v", err)
	}
	zipFile, err := zip.OpenReader(archiveName)
	if err != nil {
		t.Fatal(err)
	}
	defer zipFile.Close()
	raw, err := fs.ReadFile(zipFile, ".gh-corkboard/topics/specs/"+candidate.ID+"--retired.md")
	if err != nil || string(raw) != "retired notes\n" {
		t.Fatalf("archived candidate: %q, %v", raw, err)
	}
}

func TestStalePrunePlanRejectsApplyAndSnapshot(t *testing.T) {
	cli, root := updateFixture(t)
	doc := cli.create(updateAdmin, "retired notes", updateScope, "retired", "md", "--status", "archived")
	other := cli.create(updateAdmin, "keep other candidate", updateScope, "other", "md", "--status", "archived")
	plan := cli.prunePlan()
	file := filepath.Join(root, ".gh-corkboard", "topics", "specs", doc.ID+"--retired.md")
	if err := os.WriteFile(file, []byte("edited"), 0600); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"snapshot", "create", "--expect", plan.ID, "--json"},
		{"index", "prune", "--apply", "--expect", plan.ID, "--json"},
	} {
		_, diagnostic := cli.run(2, updateAdmin, "", args...)
		if !strings.Contains(diagnostic, `"code":"prune_plan_mismatch"`) {
			t.Fatalf("stale plan: %q", diagnostic)
		}
	}
	if got := cli.readRevision(updateAdmin, doc.ID); got.Content != "edited" {
		t.Fatalf("stale plan altered document: %+v", got)
	}
	if got := cli.readRevision(updateAdmin, other.ID); got.Content != "keep other candidate\n" {
		t.Fatalf("stale plan pruned another candidate: %+v", got)
	}
}
