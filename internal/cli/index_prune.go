package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/spf13/cobra"
)

func pruneCommand(actor *string, jsonOutput *bool) *cobra.Command {
	var apply bool
	var expected string
	cmd := &cobra.Command{
		Use: "prune", Short: "Preview or apply removal of archived documents",
		Long:    "Preview archived documents whose bytes match the index. An administrator can apply a reviewed plan with --apply --expect <plan-id>. Writes a plan or removal report to stdout.",
		Example: "  gh corkboard --as human:owner index prune --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if apply {
				if err := board.ValidatePlanID(expected); err != nil {
					return err
				}
				return applyPrune(cmd.OutOrStdout(), *actor, expected, *jsonOutput)
			}
			if cmd.Flags().Changed("expect") {
				return errors.New("--expect requires --apply on index prune")
			}
			return previewPrune(cmd.OutOrStdout(), *actor, *jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&apply, "apply", false, "Delete the documents in the reviewed plan")
	cmd.Flags().StringVar(&expected, "expect", "", "Require the reviewed prune plan ID")
	return cmd
}

func previewPrune(out io.Writer, actor string, asJSON bool) error {
	var plan board.PrunePlan
	err := board.WithAccess("", actor, false, func(a *board.Access) error {
		var err error
		plan, err = a.PreviewPrune()
		return err
	})
	if err != nil {
		return err
	}
	if asJSON {
		return json.NewEncoder(out).Encode(plan)
	}
	var text strings.Builder
	fmt.Fprintln(&text, "Plan "+plan.ID)
	if len(plan.Candidates) == 1 {
		fmt.Fprintln(&text, "1 archived document is eligible for deletion.")
	} else {
		fmt.Fprintf(&text, "%d archived documents are eligible for deletion.\n", len(plan.Candidates))
	}
	for _, candidate := range plan.Candidates {
		fmt.Fprintf(&text, "%s  %s\n", candidate.ID, candidate.Path)
	}
	for _, skipped := range plan.Skipped {
		fmt.Fprintf(&text, "Skipped %s  %s (%s).\n", skipped.ID, skipped.Path, skipped.Reason)
	}
	_, err = io.WriteString(out, text.String())
	return err
}

func applyPrune(out io.Writer, actor, expected string, asJSON bool) error {
	var result board.PruneResult
	err := board.WithAccess("", actor, true, func(a *board.Access) error {
		var err error
		result, err = a.ApplyPrune(expected)
		return err
	})
	if err != nil {
		return err
	}
	if asJSON {
		return json.NewEncoder(out).Encode(result)
	}
	var text strings.Builder
	if len(result.Deleted) == 1 {
		fmt.Fprintln(&text, "Deleted 1 document.")
	} else {
		fmt.Fprintf(&text, "Deleted %d documents.\n", len(result.Deleted))
	}
	for _, entry := range result.DeletedEntries {
		fmt.Fprintf(&text, "%s  %s\n", entry.ID, entry.Path)
	}
	_, err = io.WriteString(out, text.String())
	return err
}
