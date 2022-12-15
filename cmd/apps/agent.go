package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sunweiwe/kuber/pkg/agent"
	"github.com/urfave/cli"
)

func NewAgentCmd() cli.Command {
	options := agent.NewDefaultOptions()

	cmd := cli.Command{
		Name:  "controller",
		Usage: "run agent",
		Action: func(ctx *cli.Context) error {
			_context, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return agent.Run(_context, options)
		},
	}
	return cmd
}
