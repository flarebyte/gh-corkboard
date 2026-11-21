package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/spf13/cobra"
)

func indexCommand(actor *string, jsonOutput *bool) *cobra.Command {
	group := &cobra.Command{Use: "index", Short: "Audit and reconcile the catalog as an admin"}
	group.AddCommand(&cobra.Command{
		Use: "audit", Short: "Report differences between indexed records and stored files",
		Long:    "Report catalog findings without changing data. Requires a board administrator. Writes findings to stdout and exits 1 if drift is found.",
		Example: "  gh corkboard --as human:owner index audit",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var findings []board.IndexFinding
			err := board.WithAccess("", *actor, false, func(a *board.Access) error {
				var err error
				findings, err = a.AuditIndex()
				return err
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if *jsonOutput {
				err = json.NewEncoder(out).Encode(map[string]any{"findings": findings})
			} else if len(findings) == 0 {
				_, err = io.WriteString(out, "Index is in sync.\n")
			} else {
				_, err = io.WriteString(out, describeIndexFindings(findings))
			}
			if err != nil {
				return err
			}
			if len(findings) > 0 {
				return errFindings
			}
			return nil
		},
	}, &cobra.Command{
		Use: "resync", Short: "Reconcile the index with managed document files",
		Long:    "Adopt unindexed files as drafts, remove missing entries, and update digests. Requires a board administrator. Writes a reconciliation summary to stdout.",
		Example: "  gh corkboard --as human:owner index resync",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var result board.ResyncResult
			err := board.WithAccess("", *actor, true, func(a *board.Access) error {
				var err error
				result, err = a.ResyncIndex()
				return err
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if *jsonOutput {
				return json.NewEncoder(out).Encode(result)
			}
			_, err = io.WriteString(out, describeResync(result))
			return err
		},
	})
	return group
}

func describeIndexFindings(findings []board.IndexFinding) string {
	var text strings.Builder
	for _, finding := range findings {
		fmt.Fprintln(&text, singleLine(finding.Description()))
	}
	return text.String()
}

func describeResync(result board.ResyncResult) string {
	var text strings.Builder
	if result.Rebuilt {
		text.WriteString("Rebuilt the missing index.\n")
	}
	groups := []struct {
		entries   []board.IndexChange
		one, many string
	}{
		{result.Added, "Indexed %d new document as draft.", "Indexed %d new documents as draft."},
		{result.Removed, "Removed %d missing index entry.", "Removed %d missing index entries."},
		{result.Updated, "Updated the digest for %d document.", "Updated digests for %d documents."},
		{result.ResetToDraft, "Reset %d archived document to draft.", "Reset %d archived documents to draft."},
	}
	for _, group := range groups {
		count := len(group.entries)
		if count == 0 {
			continue
		}
		format := group.one
		if count != 1 {
			format = group.many
		}
		fmt.Fprintf(&text, format+"\n", count)
		for _, change := range group.entries {
			fmt.Fprintf(&text, "%s  %s\n", change.ID, change.Path)
		}
	}
	if text.Len() == 0 {
		text.WriteString("No index changes.\n")
	}
	text.WriteString(describeIndexFindings(result.Findings))
	return text.String()
}
