package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sunweiwe/kuber/pkg/controller"
	"github.com/urfave/cli"
)

func NewControllerCmd() cli.Command {
	options := controller.NewDefaultOptions()

	cmd := cli.Command{
		Name:  "controller",
		Usage: "run controller",
		Action: func(ctx *cli.Context) error {
			_context, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return controller.Run(_context, options)
		},
	}
	return cmd
}
