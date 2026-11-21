package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/cli"
)

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"--help"}, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "gh corkboard [flags]") {
		t.Errorf("stdout = %q, want corkboard usage", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}
