package cli

import (
	"encoding/json"
	"fmt"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/spf13/cobra"
)

func initCommand(actor *string, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:     "init",
		Short:   "Initialize the local board",
		Long:    "Initialize the board for the current Git worktree. First use requires --as human:<id>. Reruns require an existing board administrator. Reads no stdin and reports initialization on stdout.",
		Example: "  gh corkboard --as human:owner init",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := board.Init("", *actor)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if *jsonOutput {
				return json.NewEncoder(out).Encode(result)
			}
			state := "Initialized .gh-corkboard."
			if !result.Initialized {
				state = ".gh-corkboard is already initialized."
			}
			ignore := "The root ignore rule is already present."
			if result.IgnoreRuleAdded {
				ignore = "Added the root ignore rule."
			}
			if _, err := fmt.Fprintln(out, state, ignore); err != nil {
				return err
			}
			for _, tracked := range result.TrackedPaths {
				if _, err := fmt.Fprintf(out, "Tracked corkboard path: %q\n", tracked); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
