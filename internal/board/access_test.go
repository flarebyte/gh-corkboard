package board

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/testrepo"
)

const owner = "human:owner"

func accessFixture(t *testing.T) string {
	t.Helper()
	dir := testrepo.New(t)
	if _, err := Init(dir, owner); err != nil {
		t.Fatal(err)
	}
	return dir
}

func editAccess(t *testing.T, dir string, action func(*Access) error) {
	t.Helper()
	if err := WithAccess(dir, owner, true, action); err != nil {
		t.Fatal(err)
	}
}

func TestAccessPermissionMatrix(t *testing.T) {
	dir := accessFixture(t)
	const agent = "agent:worker"
	const topic = "topic:specs"
	editAccess(t, dir, func(a *Access) error {
		for _, id := range []string{agent, "skill:review", "human:reviewer"} {
			if err := a.AddPrincipal(id, "member"); err != nil {
				return err
			}
		}
		return a.CreateTopic("specs")
	})
	for _, scope := range []string{"agents/worker", "skills/review", "topics/specs"} {
		if info, err := os.Stat(filepath.Join(dir, boardName, scope)); err != nil || !info.IsDir() {
			t.Fatalf("%s: %v", scope, err)
		}
	}
	editAccess(t, dir, func(a *Access) error {
		for _, step := range []struct {
			scope, subject, permission string
			add, changed               bool
		}{
			{agent, agent, "status", true, true},
			{topic, agent, "status", true, true},
			{topic, agent, "status", true, false},
			{topic, agent, "read", true, true},
			{topic, agent, "read", false, true},
			{topic, agent, "read", false, false},
			{"skill:review", agent, "read", true, true},
		} {
			changed, err := a.ChangeGrant(step.scope, step.subject, step.permission, step.add)
			if err != nil || changed != step.changed {
				return fmt.Errorf("%+v: changed=%v, %v", step, changed, err)
			}
		}
		return nil
	})
	if err := WithAccess(dir, agent, false, func(a *Access) error {
		want := []ScopePermissions{
			{agent, []string{"read", "write", "status"}},
			{"skill:review", []string{"read"}},
			{topic, []string{"read", "status"}},
		}
		if got := a.Whoami().Scopes; !reflect.DeepEqual(got, want) {
			t.Fatalf("permissions: %#v, want %#v", got, want)
		}
		grants, err := a.Grants(agent)
		if err != nil || len(grants) != 1 || !reflect.DeepEqual(grants[0].Permissions, []string{"status"}) {
			t.Fatalf("owner grants: %#v, %v", grants, err)
		}
		for _, scope := range []string{"", topic, "skill:review"} {
			if _, err := a.Grants(scope); err == nil {
				t.Fatalf("agent saw grants for %q", scope)
			}
		}
		if _, err := a.Principals(); err == nil {
			t.Fatal("agent listed principals")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := WithAccess(dir, agent, true, func(a *Access) error { t.Fatal("non-admin obtained mutation access"); return nil }); err == nil {
		t.Fatal("non-admin edit was accepted")
	}
	editAccess(t, dir, func(a *Access) error {
		changed, err := a.ChangeGrant(topic, agent, "status", false)
		if err != nil || !changed {
			return fmt.Errorf("revoke final grant: %v, %v", changed, err)
		}
		grants, err := a.Grants(topic)
		if err != nil || len(grants) != 0 || len(a.policy.permissions(agent, topic)) != 0 {
			t.Fatalf("revoke remaining grants: %v, %v", grants, err)
		}
		return nil
	})
}

func TestAccessRejectsInvalidPolicyChanges(t *testing.T) {
	dir := accessFixture(t)
	editAccess(t, dir, func(a *Access) error { return a.AddPrincipal("agent:worker", "member") })
	for _, act := range []func(*Access) error{
		func(a *Access) error { return a.AddPrincipal("skill:review", "admin") },
		func(a *Access) error { return a.AddPrincipal("human:reviewer", "") },
		func(a *Access) error { _, err := a.SetRole(owner, "member"); return err },
		func(a *Access) error { _, err := a.ChangeGrant("agent:worker", owner, "delete", true); return err },
	} {
		if err := WithAccess(dir, owner, true, act); err == nil {
			t.Fatal("invalid policy change accepted")
		}
	}
	editAccess(t, dir, func(a *Access) error {
		if _, err := a.SetRole("agent:worker", "member"); err != nil {
			return err
		}
		if a.Whoami().Role != "admin" {
			t.Fatal("administrator lost role")
		}
		return nil
	})
	if _, err := os.Stat(filepath.Join(dir, boardName, "skills", "review")); !os.IsNotExist(err) {
		t.Fatalf("invalid principal provisioned a folder: %v", err)
	}

	policyPath := filepath.Join(dir, boardName, "index", "access.json")
	if err := os.WriteFile(policyPath, []byte(`{"version":1,"version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WithAccess(dir, owner, false, func(a *Access) error { t.Fatal("damaged policy granted access"); return nil }); err == nil {
		t.Fatal("damaged policy accepted")
	}
}

func TestTopicDeletionChecksFolderAndIndex(t *testing.T) {
	dir := accessFixture(t)
	editAccess(t, dir, func(a *Access) error {
		if err := a.CreateTopic("scratch"); err != nil {
			return err
		}
		_, err := a.ChangeGrant("topic:scratch", owner, "read", true)
		return err
	})
	parent := filepath.Join(dir, boardName, "topics", "scratch")
	existing := filepath.Join(parent, "note")
	if err := os.WriteFile(existing, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	deleteTopic := func(a *Access) error { return a.DeleteTopic("scratch") }
	if err := WithAccess(dir, owner, true, deleteTopic); err == nil {
		t.Fatal("nonempty topic deleted")
	}
	if err := os.Remove(existing); err != nil {
		t.Fatal(err)
	}
	doc := document{ID: "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb", Scope: "topic:scratch", Name: "note", Format: "md",
		Status: "draft", Digest: strings.Repeat("0", 64), Reviews: map[string]review{}}
	doc.Path = doc.canonicalPath()
	editAccess(t, dir, func(a *Access) error {
		return writeMetadata(a.index, "documents", catalog{Version: 1, Documents: []document{doc}})
	})
	if err := WithAccess(dir, owner, true, deleteTopic); err == nil {
		t.Fatal("indexed topic deleted")
	}
	editAccess(t, dir, func(a *Access) error {
		return writeMetadata(a.index, "documents", catalog{Version: 1, Documents: []document{}})
	})
	editAccess(t, dir, deleteTopic)
	if _, err := os.Stat(parent); !os.IsNotExist(err) {
		t.Fatalf("topic folder remains: %v", err)
	}
	editAccess(t, dir, func(a *Access) error {
		if len(a.policy.Topics) != 0 || len(a.policy.Grants) != 0 {
			t.Fatalf("remaining policy: %+v", a.policy)
		}
		return nil
	})
}

func TestTopicRetryAfterFolderChange(t *testing.T) {
	dir := accessFixture(t)
	folder := filepath.Join(dir, boardName, "topics", "scratch")
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	create := func(a *Access) error { return a.CreateTopic("scratch") }
	draft := filepath.Join(folder, "draft.md")
	if err := os.WriteFile(draft, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WithAccess(dir, owner, true, create); err == nil {
		t.Fatal("adopted a nonempty orphan")
	}
	if err := os.Remove(draft); err != nil {
		t.Fatal(err)
	}

	editAccess(t, dir, create)
	if err := os.Remove(folder); err != nil {
		t.Fatal(err)
	}
	editAccess(t, dir, func(a *Access) error { return a.DeleteTopic("scratch") })
	editAccess(t, dir, create)
}
