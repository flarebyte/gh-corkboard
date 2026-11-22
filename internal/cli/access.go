package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/spf13/cobra"
)

type accessResult struct {
	data any
	text string
}

type accessAction func(*board.Access, []string) (accessResult, error)

func accessCommand(actor *string, jsonOutput *bool) *cobra.Command {
	root := &cobra.Command{Use: "access", Short: "Manage principals, topics, and grants"}
	command := func(use, summary string, nargs int, edit bool, action accessAction) *cobra.Command {
		permission := "Requires a registered principal."
		if edit {
			permission = "Requires a board administrator."
		}
		return &cobra.Command{
			Use: use, Short: summary,
			Long: summary + ". " + permission + " Reads no stdin and writes results to stdout.",
			Args: cobra.ExactArgs(nargs),
			RunE: func(cmd *cobra.Command, args []string) error {
				var result accessResult
				err := board.WithAccess("", *actor, edit, func(a *board.Access) error {
					var err error
					result, err = action(a, args)
					return err
				})
				if err != nil {
					return err
				}
				if *jsonOutput {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(result.data)
				}
				_, err = io.WriteString(cmd.OutOrStdout(), result.text)
				return err
			},
		}
	}
	root.AddCommand(command("whoami", "Show your role and effective permissions", 0, false, func(a *board.Access, _ []string) (accessResult, error) {
		identity := a.Whoami()
		rows := make([][]string, 0, len(identity.Scopes))
		for _, scope := range identity.Scopes {
			rows = append(rows, []string{scope.Scope, strings.Join(scope.Permissions, ", ")})
		}
		text := fmt.Sprintf("%s (%s)\n", identity.Principal, identity.Role) + accessTable(rows)
		return accessResult{identity, text}, nil
	}))

	principals := &cobra.Command{Use: "principal", Short: "Manage registered principals"}
	root.AddCommand(principals)
	var addRole, setRole string
	add := command("add <principal>", "Register a principal", 1, true, func(a *board.Access, args []string) (accessResult, error) {
		id := args[0]
		err := a.AddPrincipal(id, addRole)
		return accessResult{map[string]string{"principal": id, "role": addRole}, fmt.Sprintf("Registered %s (%s).\n", id, addRole)}, err
	})
	add.Flags().StringVar(&addRole, "role", "member", "Assign member or admin")
	set := command("set <principal>", "Change a registered principal's role", 1, true, func(a *board.Access, args []string) (accessResult, error) {
		id := args[0]
		changed, err := a.SetRole(id, setRole)
		text := fmt.Sprintf("Changed %s to %s.\n", id, setRole)
		if !changed {
			text = fmt.Sprintf("%s is already %s.\n", id, setRole)
		}
		return accessResult{map[string]any{"principal": id, "role": setRole, "changed": changed}, text}, err
	})
	set.Flags().StringVar(&setRole, "role", "", "Assign member or admin")
	_ = set.MarkFlagRequired("role")
	list := command("list", "List registered principals and their roles", 0, false, func(a *board.Access, _ []string) (accessResult, error) {
		entries, err := a.Principals()
		rows := make([][]string, 0, len(entries))
		for _, entry := range entries {
			rows = append(rows, []string{entry.ID, entry.Role})
		}
		return accessResult{map[string]any{"principals": entries}, accessTable(rows)}, err
	})
	list.Long = "List registered principals and their roles. Requires a board administrator. Reads no stdin and writes results to stdout."
	principals.AddCommand(add, set, list)

	topics := &cobra.Command{Use: "topic", Short: "Manage shared topics"}
	topics.AddCommand(
		command("create <topic-id>", "Create a shared topic", 1, true, func(a *board.Access, args []string) (accessResult, error) {
			scope := "topic:" + args[0]
			return accessResult{map[string]any{"scope": scope, "created": true}, fmt.Sprintf("Created %s.\n", scope)}, a.CreateTopic(args[0])
		}),
		command("delete <topic-id>", "Remove an empty topic and its grants", 1, true, func(a *board.Access, args []string) (accessResult, error) {
			scope := "topic:" + args[0]
			return accessResult{map[string]any{"scope": scope, "deleted": true}, fmt.Sprintf("Removed the empty %s folder and its grants.\n", scope)}, a.DeleteTopic(args[0])
		}),
	)
	root.AddCommand(topics)

	var selected string
	grants := command("list", "List visible explicit grants", 0, false, func(a *board.Access, _ []string) (accessResult, error) {
		entries, err := a.Grants(selected)
		rows := make([][]string, 0, len(entries))
		for _, entry := range entries {
			rows = append(rows, []string{entry.Scope, entry.Principal, strings.Join(entry.Permissions, ", ")})
		}
		return accessResult{map[string]any{"grants": entries}, accessTable(rows)}, err
	})
	grants.Long = "List explicit grants. Administrators may list all grants. Owners of agent and skill scopes must select their scope. Reads no stdin and writes results to stdout."
	grants.Flags().StringVar(&selected, "scope", "", "Select a topic, agent, or skill scope")
	root.AddCommand(grants)

	for _, change := range []struct {
		verb, subject, summary string
		add                    bool
	}{
		{"grant", "to", "Add an explicit scope permission", true},
		{"revoke", "from", "Remove an explicit scope permission", false},
	} {
		var subject, permission string
		leaf := command(change.verb+" <scope>", change.summary, 1, true, func(a *board.Access, args []string) (accessResult, error) {
			scope := args[0]
			changed, err := a.ChangeGrant(scope, subject, permission, change.add)
			text := fmt.Sprintf("Revoked %s on %s from %s.\n", permission, scope, subject)
			if change.add {
				text = fmt.Sprintf("Granted %s on %s to %s.\n", permission, scope, subject)
			}
			if !changed {
				text = fmt.Sprintf("Grant for %s on %s unchanged.\n", subject, scope)
			}
			return accessResult{map[string]any{"scope": scope, "principal": subject, "permission": permission, "changed": changed}, text}, err
		})
		leaf.Flags().StringVar(&subject, change.subject, "", "Select a registered principal")
		leaf.Flags().StringVar(&permission, "permission", "", "Select read, write, delete, or status")
		_ = leaf.MarkFlagRequired(change.subject)
		_ = leaf.MarkFlagRequired("permission")
		root.AddCommand(leaf)
	}

	for _, group := range []*cobra.Command{root, principals, topics} {
		for _, leaf := range group.Commands() {
			if leaf.HasSubCommands() {
				continue
			}
			leaf.Example = accessExample(leaf)
		}
	}
	return root
}

func accessExample(cmd *cobra.Command) string {
	examples := map[string]string{
		"access whoami":         "whoami",
		"access principal add":  "principal add agent:test",
		"access principal set":  "principal set human:reviewer --role admin",
		"access principal list": "principal list",
		"access topic create":   "topic create scratch",
		"access topic delete":   "topic delete scratch",
		"access list":           "list --scope topic:specs",
		"access grant":          "grant topic:specs --to agent:test --permission read",
		"access revoke":         "revoke topic:specs --from agent:test --permission read",
	}
	return "  gh corkboard --as human:owner access " + examples[cmd.CommandPath()]
}

func accessTable(rows [][]string) string {
	var out strings.Builder
	table := tabwriter.NewWriter(&out, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		fmt.Fprintln(table, strings.Join(row, "\t"))
	}
	_ = table.Flush()
	return out.String()
}
