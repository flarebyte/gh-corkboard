package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/spf13/cobra"
)

func docSearchCommand(actor *string, jsonOutput *bool) *cobra.Command {
	var o board.SearchOptions
	var format string
	cmd := &cobra.Command{
		Use: "search", Short: "Find documents in accessible scopes",
		Long:    "Filter indexed documents by metadata. Requires a registered principal. Results include readable scopes and are written to stdout.",
		Example: "  gh corkboard --as agent:specs doc search --milestone 'v1 launch/*' --status active",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			for _, flag := range []string{"in", "assignee", "milestone", "concept"} {
				if cmd.Flags().Changed(flag) && cmd.Flags().Lookup(flag).Value.String() == "" {
					return fmt.Errorf("--%s requires a value", flag)
				}
			}
			if cmd.Flags().Changed("format") {
				if cmd.Flags().Changed("json") {
					return fmt.Errorf("use one output selector: --json or --format")
				}
				if format != "table" && format != "csv" && format != "json" {
					return fmt.Errorf("--format must be table, csv, or json")
				}
			}
			if *jsonOutput {
				format = "json"
			}
			*jsonOutput = format == "json"
			var docs []board.DocumentMetadata
			err := board.WithDocuments("", *actor, false, func(d *board.Documents) error {
				var err error
				docs, err = d.Search(o)
				return err
			})
			if err != nil {
				return err
			}
			return writeSearch(cmd.OutOrStdout(), format, o, docs)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&o.Scope, "in", "", "Filter by scope")
	flags.StringArrayVar(&o.Statuses, "status", nil, "Filter by lifecycle status, repeat to match any")
	flags.StringVar(&o.Assignee, "assignee", "", "Filter by assigned principal")
	flags.StringVar(&o.Milestone, "milestone", "", "Filter by milestone, with optional trailing *")
	flags.StringVar(&o.Concept, "concept", "", "Filter by concept, with optional trailing *")
	flags.StringVar(&o.Query, "query", "", "Match a substring in name, description, milestone, or concept")
	flags.StringVar(&format, "format", "", "Select table, csv, or json output")
	return cmd
}

func writeSearch(out io.Writer, format string, o board.SearchOptions, docs []board.DocumentMetadata) error {
	switch format {
	case "json":
		return json.NewEncoder(out).Encode(map[string]any{"documents": docs})
	case "csv", "table":
		rows := compactDocuments(o, docs)
		if format == "csv" {
			return csv.NewWriter(out).WriteAll(rows)
		}
		for i, heading := range rows[0] {
			rows[0][i] = strings.ToUpper(heading)
		}
		_, err := io.WriteString(out, accessTable(rows))
		return err
	default:
		_, err := io.WriteString(out, describeDocuments(docs))
		return err
	}
}

func compactDocuments(o board.SearchOptions, docs []board.DocumentMetadata) [][]string {
	statusFixed := len(o.Statuses) > 0
	for _, status := range o.Statuses {
		if status != o.Statuses[0] {
			statusFixed = false
			break
		}
	}
	omit := []bool{statusFixed, o.Assignee != "", exactLabel(o.Milestone), exactLabel(o.Concept)}
	optional := []string{"status", "assignee", "milestone", "concept"}
	header := []string{"id", "name", "scope", "description"}
	for i, name := range optional {
		if !omit[i] {
			header = append(header, name)
		}
	}
	rows := make([][]string, 1, len(docs)+1)
	rows[0] = header
	for _, doc := range docs {
		description := []rune(doc.Description)
		short := doc.Description
		if len(description) > 60 {
			short = string(description[:57]) + "..."
		}
		row := []string{doc.ID, doc.Name, doc.Scope, short}
		for i, value := range []string{doc.Status, doc.Assignee, doc.Milestone, doc.Concept} {
			if !omit[i] {
				row = append(row, value)
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func exactLabel(filter string) bool {
	label := strings.TrimSpace(filter)
	return label != "" && !strings.HasSuffix(label, "*")
}

func describeDocuments(docs []board.DocumentMetadata) string {
	if len(docs) == 0 {
		return "No documents matched.\n"
	}
	var out strings.Builder
	for _, doc := range docs {
		fmt.Fprintf(&out, "%s  %s  %s / %s (%s)\n", doc.ID, doc.Status, doc.Scope, doc.Name, doc.Format)
		if doc.Description != "" {
			fmt.Fprintln(&out, "  "+doc.Description)
		}
		if doc.Milestone != "" {
			fmt.Fprintln(&out, "  Milestone: "+doc.Milestone)
		}
		if doc.Concept != "" {
			fmt.Fprintln(&out, "  Concept: "+doc.Concept)
		}
		if doc.Assignee != "" {
			fmt.Fprintln(&out, "  Assignee: "+doc.Assignee)
		}
		for _, author := range slices.Sorted(maps.Keys(doc.Reviews)) {
			review := doc.Reviews[author]
			comment := strings.ReplaceAll(strings.ReplaceAll(review.Comment, "\t", `\t`), "\n", "\n    ")
			fmt.Fprintf(&out, "  %s [%s] %s\n", author, review.Status, comment)
			if review.Locator != "" {
				fmt.Fprintln(&out, "    Review document: "+review.Locator)
			}
		}
	}
	return out.String()
}
