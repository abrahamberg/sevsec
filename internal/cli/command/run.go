package command

import (
	"fmt"

	"github.com/abrahamberg/devsec/internal/cli/client"
	"github.com/abrahamberg/devsec/internal/cli/runner"
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

			r := runner.New()

			api := client.New("http://localhost:8080")

			env, err := api.GetRuntimeEnv(cmd.Context(), project)
			if err != nil {
				return err
			}

			if err := r.Run(cmd.Context(), command, env); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
