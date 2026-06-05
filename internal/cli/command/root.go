package command

import "github.com/spf13/cobra"

func Execute() error {
	rootCmd := &cobra.Command{
		Use:   "devsec",
		Short: "Developer secret runner",
	}

	rootCmd.AddCommand(newRunCommand())

	return rootCmd.Execute()
}
