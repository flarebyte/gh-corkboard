package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/spf13/cobra"
)

func schemaCommand(actor *string, jsonOutput *bool) *cobra.Command {
	group := &cobra.Command{Use: "schema", Short: "Manage document schemas"}
	for _, verb := range []string{"create", "update", "delete"} {
		group.AddCommand(schemaMutation(actor, jsonOutput, verb))
	}
	group.AddCommand(&cobra.Command{
		Use: "get <format> <schema-id>", Short: "Write the stored schema bytes",
		Long:    "Write the stored schema bytes to stdout. Available to every registered principal.",
		Example: "  gh corkboard --as agent:specs schema get json spec",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return board.WithAccess("", *actor, false, func(a *board.Access) error {
				return a.GetSchema(schemaKey(args), cmd.OutOrStdout())
			})
		},
	})
	var filter string
	list := &cobra.Command{
		Use: "list", Short: "List registered schemas by format and ID",
		Long:    "List schemas for a registered principal. Stdout contains the schema format and ID, ordered by format and ID.",
		Example: "  gh corkboard --as agent:specs schema list --format json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("format") && filter == "" {
				return fmt.Errorf("--format must be csv, json, or md")
			}
			var keys []board.SchemaKey
			err := board.WithAccess("", *actor, false, func(a *board.Access) error {
				var err error
				keys, err = a.ListSchemas(filter)
				return err
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if *jsonOutput {
				return json.NewEncoder(out).Encode(map[string]any{"schemas": keys})
			}
			rows := make([][]string, 0, len(keys))
			for _, key := range keys {
				rows = append(rows, []string{key.Format, key.ID})
			}
			_, err = io.WriteString(out, accessTable(rows))
			return err
		},
	}
	list.Flags().StringVar(&filter, "format", "", "Select schemas for csv, json, or md")
	group.AddCommand(list)
	return group
}

func schemaMutation(actor *string, jsonOutput *bool, verb string) *cobra.Command {
	type presentation struct{ summary, exampleSuffix, resultKey, message string }
	views := map[string]presentation{
		"create": {"Register a schema from stdin", " < spec.schema.json", "created", "Registered the %s schema %s.\n"},
		"update": {"Replace an existing schema from stdin", " < spec.schema.json", "updated", "Updated the %s schema %s.\n"},
		"delete": {"Delete an unused schema", "", "deleted", "Deleted the unused %s schema %s.\n"},
	}
	view := views[verb]
	return &cobra.Command{
		Use: verb + " <format> <schema-id>", Short: view.summary,
		Long:    view.summary + ". Requires a board administrator. Writes a mutation summary to stdout.",
		Example: "  gh corkboard --as human:owner schema " + verb + " json spec" + view.exampleSuffix,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := schemaKey(args)
			var data []byte
			if verb != "delete" {
				var err error
				data, err = io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
			}
			err := board.WithAccess("", *actor, true, func(a *board.Access) error {
				if verb == "delete" {
					return a.DeleteSchema(key)
				}
				return a.WriteSchema(key, data, verb == "create")
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if *jsonOutput {
				return json.NewEncoder(out).Encode(map[string]any{"format": key.Format, "schemaId": key.ID, view.resultKey: true})
			}
			_, err = fmt.Fprintf(out, view.message, key.Format, key.ID)
			return err
		},
	}
}

func schemaKey(args []string) board.SchemaKey {
	return board.SchemaKey{Format: args[0], ID: args[1]}
}
