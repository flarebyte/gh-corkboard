package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/spf13/cobra"
)

var errFindings = errors.New("report contains findings")

func docValidateCommand(actor *string, jsonOutput *bool) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use: "validate <doc-id|--all>", Short: "Check a document or all readable documents",
		Long:    "Check native document syntax and any registered schema. Requires scope read permission. Reports valid, invalid, or schemaless results on stdout.",
		Example: "  gh corkboard --as agent:specs doc validate --all",
		Args: func(_ *cobra.Command, args []string) error {
			if (all && len(args) != 0) || (!all && len(args) != 1) {
				return fmt.Errorf("supply a document UUID or --all")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var results []board.ValidationResult
			id := ""
			if !all {
				id = args[0]
			}
			err := board.WithDocuments("", *actor, false, func(d *board.Documents) error {
				var err error
				results, err = d.Validate(id, all)
				return err
			})
			if err != nil {
				return err
			}
			if err := reportValidation(cmd.OutOrStdout(), *jsonOutput, results); err != nil {
				return err
			}
			for _, result := range results {
				if result.State == "invalid" {
					return errFindings
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Validate all readable indexed documents")
	return cmd
}

func reportValidation(out io.Writer, asJSON bool, results []board.ValidationResult) error {
	if asJSON {
		return json.NewEncoder(out).Encode(map[string]any{"results": results})
	}
	var report strings.Builder
	for _, result := range results {
		fmt.Fprintf(&report, "%s  %s\n", result.ID, result.State)
		for _, diagnostic := range result.Diagnostics {
			fmt.Fprint(&report, "  ", singleLine(diagnostic.Message))
			if diagnostic.Line != 0 {
				fmt.Fprintf(&report, " (line %d, column %d)", diagnostic.Line, diagnostic.Column)
			}
			report.WriteByte('\n')
		}
	}
	_, err := io.WriteString(out, report.String())
	return err
}

func singleLine(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Bidi_Control) || r == '\u2028' || r == '\u2029' {
			return ' '
		}
		return r
	}, text)
}
