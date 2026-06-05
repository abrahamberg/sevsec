package runner

import (
	"context"
	"os"
	"os/exec"
)

type Runner struct{}

func New() *Runner {
	return &Runner{}
}

func (r *Runner) Run(ctx context.Context, command []string, extraEnv map[string]string) error {
	if len(command) == 0 {
		return nil
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = mergeEnv(os.Environ(), extraEnv)

	return cmd.Run()
}

func mergeEnv(base []string, extra map[string]string) []string {
	env := append([]string{}, base...)

	for key, value := range extra {
		env = append(env, key+"="+value)
	}

	return env
}
