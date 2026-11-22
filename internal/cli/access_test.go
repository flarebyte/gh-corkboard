package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/testrepo"
)

func TestAccessCLI(t *testing.T) {
	t.Chdir(testrepo.New(t))
	for _, tc := range []struct {
		args, stdout, stderr string
		code                 int
	}{
		{"human:owner init", "Initialized .gh-corkboard. Added the root ignore rule.\n", "", 0},
		{"human:owner access principal add agent:test", "Registered agent:test (member).\n", "", 0},
		{"human:owner access principal add human:reviewer", "Registered human:reviewer (member).\n", "", 0},
		{"human:owner access principal set human:reviewer --role admin", "Changed human:reviewer to admin.\n", "", 0},
		{"human:owner access topic create specs", "Created topic:specs.\n", "", 0},
		{"human:owner access grant topic:specs --to agent:test --permission status", "Granted status on topic:specs to agent:test.\n", "", 0},
		{"human:owner access list --scope topic:specs", "topic:specs  agent:test  status\n", "", 0},
		{"agent:test access whoami --json", "{\"principal\":\"agent:test\",\"role\":\"member\",\"scopes\":[{\"scope\":\"agent:test\",\"permissions\":[\"read\",\"write\"]},{\"scope\":\"topic:specs\",\"permissions\":[\"read\",\"status\"]}]}\n", "", 0},
		{"agent:test access topic delete specs", "", "requires a board administrator", 2},
		{"human:owner access revoke topic:specs --from agent:test --permission status", "Revoked status on topic:specs from agent:test.\n", "", 0},
		{"human:owner access topic delete specs", "Removed the empty topic:specs folder and its grants.\n", "", 0},
		{"human:owner access principal list --json", "{\"principals\":[{\"id\":\"agent:test\",\"role\":\"member\"},{\"id\":\"human:owner\",\"role\":\"admin\"},{\"id\":\"human:reviewer\",\"role\":\"admin\"}]}\n", "", 0},
	} {
		var out, errOut bytes.Buffer
		args := append([]string{"--as"}, strings.Fields(tc.args)...)
		code := Run(args, nil, &out, &errOut)
		if code != tc.code || out.String() != tc.stdout || !strings.Contains(errOut.String(), tc.stderr) || (tc.stderr == "" && errOut.Len() > 0) {
			t.Fatalf("%s: exit %d, stdout %q, stderr %q", tc.args, code, &out, &errOut)
		}
	}
}
