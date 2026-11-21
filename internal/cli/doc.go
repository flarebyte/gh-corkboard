package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/spf13/cobra"
)

func docCommand(actor *string, jsonOutput *bool) *cobra.Command {
	group := &cobra.Command{Use: "doc", Short: "Create, read, find, edit, delete, and validate documents"}
	group.AddCommand(createCommand(actor, jsonOutput), getCommand(actor), locateCommand(actor, jsonOutput),
		docSearchCommand(actor, jsonOutput), docValidateCommand(actor, jsonOutput), docUpdateCommand(actor, jsonOutput), docPatchCommand(actor, jsonOutput), deleteCommand(actor, jsonOutput))
	return group
}

func createCommand(actor *string, jsonOutput *bool) *cobra.Command {
	var o board.CreateOptions
	var status string
	cmd := &cobra.Command{
		Use: "create", Short: "Create an indexed document",
		Long:    "Create an indexed document from stdin. Requires scope write permission and status permission when --status is supplied.",
		Example: "  gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md < pagination.md",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("schema") && o.SchemaID == "" {
				return fmt.Errorf("--schema must be a valid slug")
			}
			if cmd.Flags().Changed("status") {
				o.Status = &status
			}
			var result board.DocumentMetadata
			err := board.WithDocuments("", *actor, true, func(d *board.Documents) error {
				var err error
				result, err = d.Create(o, cmd.InOrStdin())
				return err
			})
			if err != nil {
				return err
			}
			if *jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created %s (%s / %s, %s, %s).\n", result.ID, result.Scope, result.Name, result.Format, result.Status)
			return err
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&o.Scope, "in", "", "Select the destination scope")
	flags.StringVar(&o.Name, "name", "", "Set the logical document name")
	flags.StringVar(&o.Format, "format", "", "Select md, json, or csv")
	flags.StringVar(&o.SchemaID, "schema", "", "Bind a schema by ID")
	flags.StringVar(&o.Description, "description", "", "Describe the document")
	flags.StringVar(&o.Milestone, "milestone", "", "Set the milestone label")
	flags.StringVar(&o.Concept, "concept", "", "Set the concept label")
	flags.StringVar(&o.Assignee, "assignee", "", "Set the assigned principal")
	flags.StringVar(&status, "status", "draft", "Set the lifecycle status")
	flags.BoolVar(&o.PreserveRowOrder, "preserve-row-order", false, "Keep CSV rows in input order")
	for _, flag := range []string{"in", "name", "format"} {
		_ = cmd.MarkFlagRequired(flag)
	}
	return cmd
}

func getCommand(actor *string) *cobra.Command {
	var withRevision bool
	cmd := &cobra.Command{
		Use: "get <doc-id>", Short: "Write the stored bytes to stdout",
		Long:    "Write exactly the stored bytes to stdout, or return content and its revision together as JSON. Requires scope read permission.",
		Example: "  gh corkboard --as agent:specs doc get 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb --with-revision",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			return board.WithDocuments("", *actor, false, func(d *board.Documents) error {
				content, err := d.Get(args[0], out, withRevision)
				if err != nil || !withRevision {
					return err
				}
				return json.NewEncoder(out).Encode(content)
			})
		},
	}
	cmd.Flags().BoolVar(&withRevision, "with-revision", false, "Return JSON with content and its revision")
	return cmd
}

func locateCommand(actor *string, jsonOutput *bool) *cobra.Command {
	var scope, name, milestone string
	cmd := &cobra.Command{
		Use: "locate", Short: "Resolve the UUID of a named document",
		Long:    "Resolve a name within a scope and print its UUID. Requires scope read permission. Select a milestone to resolve ambiguous names.",
		Example: "  gh corkboard --as agent:specs doc locate --in topic:specs --name pagination --milestone v1/Specs",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var selected *string
			if cmd.Flags().Changed("milestone") {
				selected = &milestone
			}
			var id string
			err := board.WithDocuments("", *actor, false, func(d *board.Documents) error {
				var err error
				id, err = d.Locate(scope, name, selected)
				return err
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if *jsonOutput {
				return json.NewEncoder(out).Encode(map[string]string{"id": id})
			}
			_, err = fmt.Fprintln(out, id)
			return err
		},
	}
	cmd.Flags().StringVar(&scope, "in", "", "Select the document scope")
	cmd.Flags().StringVar(&name, "name", "", "Match the exact logical name")
	cmd.Flags().StringVar(&milestone, "milestone", "", "Match a milestone exactly, or pass an empty value for unset")
	_ = cmd.MarkFlagRequired("in")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func deleteCommand(actor *string, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use: "delete <doc-id>", Short: "Delete one indexed document",
		Long:    "Delete the document and its index entry while retaining the scope folder. Requires topic delete permission or a board administrator.",
		Example: "  gh corkboard --as human:owner doc delete 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			err := board.WithDocuments("", *actor, true, func(d *board.Documents) error { return d.Delete(id) })
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if *jsonOutput {
				return json.NewEncoder(out).Encode(map[string]any{"id": id, "deleted": true})
			}
			_, err = io.WriteString(out, "Deleted document "+id+".\n")
			return err
		},
	}
}
