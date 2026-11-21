package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/spf13/cobra"
)

func docPatchCommand(actor *string, jsonOutput *bool) *cobra.Command {
	var expected string
	var preserveRowOrder bool
	cmd := &cobra.Command{
		Use: "patch <doc-id> --expect <revision>", Short: "Apply a patch against an exact base revision",
		Long:    "Patch an existing document with scope write permission. Reads a single unified diff for Markdown or CSV, or a JSON merge patch for JSON, from stdin. Get the base revision with doc get --with-revision.",
		Example: "  gh corkboard --as agent:specs doc patch 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb --expect \"$revision\" < changes.diff",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if expected == "" {
				return errors.New("--expect is required. Get the base revision with doc get --with-revision")
			}
			if err := board.ValidateRevision(expected); err != nil {
				return err
			}
			patch, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
			var result board.UpdateResult
			err = board.WithDocuments("", *actor, true, func(d *board.Documents) error {
				var err error
				result, err = d.Patch(args[0], expected, patch, preserveRowOrder)
				return err
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if *jsonOutput {
				return json.NewEncoder(out).Encode(result)
			}
			if _, err := fmt.Fprintf(out, "Patched document %s.\n", result.ID); err != nil {
				return err
			}
			if result.FormattingChanged != nil && *result.FormattingChanged {
				_, err = io.WriteString(out, "Formatting changed the patched content.\n")
			}
			return err
		},
	}
	cmd.Flags().StringVar(&expected, "expect", "", "Require the revision from doc get --with-revision")
	cmd.Flags().BoolVar(&preserveRowOrder, "preserve-row-order", false, "Keep patched CSV rows in patch order")
	return cmd
}
