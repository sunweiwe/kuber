package apps

import (
	"fmt"

	"github.com/sunweiwe/kuber/pkg/version"
	"github.com/urfave/cli"
)

func NewVersionCmd() cli.Command {
	cmd := cli.Command{
		Name:  "controller",
		Usage: "run agent",
		Action: func(ctx *cli.Context) error {
			fmt.Println(version.Get())
			return nil
		},
	}
	return cmd
}
