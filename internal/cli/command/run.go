package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <project> -- <command>",
		Short: "Run a command with project secrets",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("expected project and command")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			project := args[0]
			command := args[1:]

			fmt.Println("project:", project)
			fmt.Println("command:", command)

			return nil
		},
	}

	return cmd
}
