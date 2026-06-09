package command

import (
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

func newServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Server commands",
	}

	cmd.AddCommand(newServerHealthCommand())

	return cmd
}

func newServerHealthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check server health",
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := http.NewRequestWithContext(
				cmd.Context(),
				http.MethodGet,
				"http://localhost:8080/healthz",
				nil,
			)
			if err != nil {
				return err
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			fmt.Println(string(body))
			return nil
		},
	}
}
