package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/testrepo"
)

func TestHelpAndVersionOutsideGit(t *testing.T) {
	t.Chdir(t.TempDir())
	var defaultHelp string
	for _, args := range [][]string{nil, {"help"}, {"--help"}, {"-h"}} {
		var out, errOut bytes.Buffer
		if status := Run(args, nil, &out, &errOut); status != 0 || errOut.Len() != 0 || !strings.Contains(out.String(), "gh corkboard") {
			t.Fatalf("%v: %d, %q, %q", args, status, &out, &errOut)
		}
		if defaultHelp == "" {
			defaultHelp = out.String()
		}
		if out.String() != defaultHelp {
			t.Fatalf("different help for %v", args)
		}
	}
	for _, args := range [][]string{{"--version"}, {"help", "init"}, {"init", "-h"}} {
		var out, errOut bytes.Buffer
		if status := Run(args, nil, &out, &errOut); status != 0 || out.Len() == 0 || errOut.Len() != 0 {
			t.Fatalf("%v: %d, %q, %q", args, status, &out, &errOut)
		}
	}
	for _, args := range [][]string{{"help", "init", "open"}, {"--as", "human:owner", "init"}} {
		var out, errOut bytes.Buffer
		if status := Run(args, nil, &out, &errOut); status != 2 || out.Len() != 0 || errOut.Len() == 0 {
			t.Fatalf("%v: %d, %q, %q", args, status, &out, &errOut)
		}
	}
}

func TestInitJSON(t *testing.T) {
	t.Chdir(testrepo.New(t))
	var out, errOut bytes.Buffer
	status := Run([]string{"--as", "human:owner", "init", "--json"}, nil, &out, &errOut)
	if status != 0 || out.String() != "{\"initialized\":true,\"ignoreRuleAdded\":true}\n" || errOut.Len() != 0 {
		t.Fatalf("init: %d, %q, %q", status, &out, &errOut)
	}
}
