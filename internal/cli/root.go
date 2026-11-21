package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/flarebyte/gh-corkboard/internal/docformat"
	"github.com/spf13/cobra"
)

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	root := &cobra.Command{
		Use:           "corkboard",
		Annotations:   map[string]string{cobra.CommandDisplayNameAnnotation: "gh corkboard"},
		Short:         "Manage a local project corkboard for agent handoffs and human review.",
		Example:       "  gh corkboard --as human:owner init\n  gh corkboard help init",
		SilenceErrors: true, SilenceUsage: true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		Version:           "0.1.0",
		RunE:              func(cmd *cobra.Command, args []string) error { return cmd.Help() },
		Args:              cobra.NoArgs,
	}
	root.SetVersionTemplate("gh corkboard {{.Version}}\n")
	root.SetArgs(append([]string{}, args...))
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)

	var actor string
	var jsonOutput bool
	root.PersistentFlags().StringVar(&actor, "as", "", "Select the caller for a board command")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Emit JSON for supported results")
	root.AddCommand(initCommand(&actor, &jsonOutput), accessCommand(&actor, &jsonOutput), docCommand(&actor, &jsonOutput), schemaCommand(&actor, &jsonOutput), indexCommand(&actor, &jsonOutput))
	var helpTarget *cobra.Command
	root.SetHelpCommand(&cobra.Command{
		Use:   "help [<command> ...]",
		Short: "Show root or command-specific help",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, remaining, err := root.Find(args)
			helpTarget = target
			if err != nil {
				return err
			}
			if len(remaining) != 0 {
				return fmt.Errorf("unknown command %q under %q", remaining[0], target.Name())
			}
			target.InitDefaultHelpFlag()
			target.InitDefaultVersionFlag()
			return target.Help()
		},
	})

	if cmd, err := root.ExecuteC(); err != nil {
		if errors.Is(err, errFindings) {
			return 1
		}
		var input *docformat.InputError
		if errors.As(err, &input) {
			if reportErr := writeInputDiagnostic(stdout, jsonOutput, input); reportErr == nil {
				return 1
			} else {
				err = reportErr
			}
		}
		if jsonOutput {
			code, message := "command_error", err.Error()
			if errors.Is(err, board.ErrRevisionMismatch) {
				code = "revision_mismatch"
				message = "Document changed since it was read. Get the current content and revision before retrying."
			}
			if errors.Is(err, docformat.ErrPatchContext) {
				code = "patch_context_mismatch"
				message = "Patch hunk does not match the current content and position. Get the current content and revision before retrying."
			}
			_ = json.NewEncoder(stderr).Encode(map[string]string{"code": code, "message": message})
		} else {
			if helpTarget != nil {
				cmd = helpTarget
			}
			path := cmd.CommandPath()
			suffix := strings.TrimSpace(strings.TrimPrefix(path, root.CommandPath()))
			help := "gh corkboard help"
			if suffix != "" && suffix != "help" {
				help += " " + suffix
			}
			fmt.Fprintf(stderr, "error: %s\nRun %s for usage.\n", err, help)
		}
		return 2
	}
	return 0
}

func writeInputDiagnostic(out io.Writer, asJSON bool, input *docformat.InputError) error {
	message := singleLine(input.Error())
	if errors.Is(input, io.EOF) || errors.Is(input, io.ErrUnexpectedEOF) {
		message = "unexpected end of input"
	}
	message = strings.TrimSuffix(message, ".") + "."
	if asJSON {
		first := strings.ToUpper(message[:1])
		return json.NewEncoder(out).Encode(map[string]any{"diagnostics": []map[string]string{
			{"code": "invalid_" + input.Format, "message": first + message[1:]},
		}})
	}
	format := strings.ToUpper(input.Format)
	if format == "SCHEMA" {
		format = "schema"
	}
	if format == "MD" {
		format = "Markdown"
	}
	_, err := fmt.Fprintf(out, "Invalid %s: %s\n", format, message)
	return err
}
