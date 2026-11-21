package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type inputReader func([]byte) (int, error)

func (r inputReader) Read(p []byte) (int, error) { return r(p) }

func TestCreateReadsInputBeforeTakingBoardLock(t *testing.T) {
	cli, _ := updateFixture(t)
	source := cli.create(updateAdmin, "copy me", updateScope, "source", "md")
	upstream := make(chan int, 1)
	var content bytes.Buffer
	started := false
	input := inputReader(func(p []byte) (int, error) {
		if !started {
			started = true
			go func() {
				upstream <- Run([]string{"--as", updateAdmin, "doc", "get", source.ID}, nil, &content, io.Discard)
			}()
			select {
			case status := <-upstream:
				if status != 0 {
					return 0, errors.New("upstream read failed")
				}
			case <-time.After(5 * time.Second):
				return 0, errors.New("upstream read blocked by document creation")
			}
		}
		return content.Read(p)
	})
	var out, stderr bytes.Buffer
	args := []string{"--as", updateAdmin, "doc", "create", "--in", updateScope, "--name", "copy", "--format", "md"}
	if status := Run(args, input, &out, &stderr); status != 0 {
		t.Fatalf("create returned %d: %s", status, stderr.String())
	}
	copied := cli.findOne(updateAdmin, "--in", updateScope, "--query", "copy")
	if got := cli.readRevision(updateAdmin, copied).Content; got != "copy me\n" {
		t.Fatalf("copied content: %q", got)
	}
}

func TestCreateRejectsOversizeInput(t *testing.T) {
	cli, _ := updateFixture(t)
	_, stderr := cli.run(2, updateAdmin, strings.Repeat("a", maxCreateBytes+1),
		"doc", "create", "--in", updateScope, "--name", "too-large", "--format", "md")
	if !strings.Contains(stderr, "input exceeds 32 MiB") {
		t.Fatalf("unexpected error: %s", stderr)
	}
	cli.run(2, updateAdmin, "", "doc", "locate", "--in", updateScope, "--name", "too-large")
}
