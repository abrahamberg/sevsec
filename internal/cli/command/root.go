package command

import "github.com/spf13/cobra"

func Execute() error {
	rootCmd := &cobra.Command{
		Use:   "devsec",
		Short: "Developer secret runner",
	}

	rootCmd.AddCommand(newRunCommand())

	rootCmd.AddCommand(newRunCommand())
	rootCmd.AddCommand(newServerCommand())

	return rootCmd.Execute()
}
