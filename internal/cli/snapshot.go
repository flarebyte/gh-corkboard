package cli

import (
	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/spf13/cobra"
)

func snapshotCommand(actor *string) *cobra.Command {
	group := &cobra.Command{Use: "snapshot", Short: "Create a ZIP archive of the board"}
	var expected string
	create := &cobra.Command{
		Use: "create", Short: "Stream the local board as a ZIP",
		Long:    "Write a ZIP archive to stdout. Requires a board administrator. Use --expect to verify a reviewed prune plan before emitting ZIP bytes.",
		Example: "  gh corkboard --as human:owner snapshot create > corkboard.zip",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var plan *string
			if cmd.Flags().Changed("expect") {
				if err := board.ValidatePlanID(expected); err != nil {
					return err
				}
				plan = &expected
			}
			return board.WithAccess("", *actor, false, func(a *board.Access) error {
				return a.Snapshot(cmd.OutOrStdout(), plan)
			})
		},
	}
	create.Flags().StringVar(&expected, "expect", "", "Verify the reviewed prune plan ID")
	group.AddCommand(create)
	return group
}
