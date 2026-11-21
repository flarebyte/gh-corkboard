package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/spf13/cobra"
)

func docUpdateCommand(actor *string, jsonOutput *bool) *cobra.Command {
	var o board.UpdateOptions
	var readStdin bool
	fields := []struct {
		flag, help  string
		destination **string
	}{
		{"description", "Replace the description", &o.Description},
		{"milestone", "Set the milestone, or pass an empty value to clear it", &o.Milestone},
		{"concept", "Set the concept, or pass an empty value to clear it", &o.Concept},
		{"assignee", "Set the assignee, or pass an empty value to clear it", &o.Assignee},
		{"status", "Set the lifecycle status", &o.Status},
		{"expect", "Require the revision returned by doc get --with-revision", &o.Expect},
		{"review-comment", "Set your own review text", &o.ReviewComment},
		{"review-status", "Set your review outcome to pass, warning, or fail", &o.ReviewStatus},
		{"review-locator", "Link your review to a readable document UUID", &o.ReviewLocator},
	}
	cmd := &cobra.Command{
		Use: "update <doc-id>", Short: "Replace content or update metadata",
		Long:    "Update an existing document. Changing content, description, milestone, concept, or assignee requires scope write permission. Changing lifecycle status or your review requires scope status permission. Reads replacement bytes with --stdin and writes a summary to stdout.",
		Example: "  gh corkboard --as agent:test doc update 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb --status approved",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, field := range fields {
				if cmd.Flags().Changed(field.flag) {
					value, err := cmd.Flags().GetString(field.flag)
					if err != nil {
						return err
					}
					*field.destination = &value
				}
			}
			// Read piped content before acquiring an exclusive lock so another Corkboard
			// command can supply stdin from the same board.
			if readStdin {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				o.Content = &data
			}
			var result board.UpdateResult
			err := board.WithDocuments("", *actor, true, func(d *board.Documents) error {
				var err error
				result, err = d.Update(args[0], o)
				return err
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if *jsonOutput {
				return json.NewEncoder(out).Encode(result)
			}
			return writeUpdateSummary(out, *actor, o, result)
		},
	}
	for _, field := range fields {
		cmd.Flags().String(field.flag, "", field.help)
	}
	cmd.Flags().BoolVar(&readStdin, "stdin", false, "Read replacement document bytes from stdin")
	cmd.Flags().BoolVar(&o.PreserveRowOrder, "preserve-row-order", false, "Keep CSV rows in input order")
	cmd.Flags().BoolVar(&o.ClearReview, "clear-review-comment", false, "Remove your own review")
	return cmd
}

func writeUpdateSummary(out io.Writer, actor string, o board.UpdateOptions, result board.UpdateResult) error {
	var text strings.Builder
	if o.Content != nil || o.Description != nil || o.Assignee != nil {
		fmt.Fprintf(&text, "Updated document %s.", result.ID)
	} else {
		fmt.Fprintf(&text, "Updated %s.", result.ID)
	}
	if o.Status != nil {
		fmt.Fprintf(&text, " Status: %s.", result.Status)
	}
	if o.ReviewComment != nil {
		fmt.Fprintf(&text, " Review by %s: %s.", actor, result.Reviews[actor].Status)
	}
	if o.ClearReview {
		fmt.Fprintf(&text, " Review by %s cleared.", actor)
	}
	text.WriteByte('\n')
	if o.Milestone != nil {
		if result.Milestone == "" {
			text.WriteString("Milestone cleared.\n")
		} else {
			fmt.Fprintln(&text, "Milestone: "+result.Milestone)
		}
	}
	if o.Concept != nil {
		if result.Concept == "" {
			text.WriteString("Concept cleared.\n")
		} else {
			fmt.Fprintln(&text, "Concept: "+result.Concept)
		}
	}
	if result.FormattingChanged != nil && *result.FormattingChanged {
		text.WriteString("Formatting changed the supplied bytes.\n")
	}
	_, err := io.WriteString(out, text.String())
	return err
}
