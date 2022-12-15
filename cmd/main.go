package main

import (
	"log"
	"os"

	"github.com/sunweiwe/kuber/cmd/apps"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "kuber"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "log",
			Value: "",
			Usage: "set the log file to write kuber logs to (default is '/dev/stderr')",
		},
	}

	app.Commands = []cli.Command{
		apps.NewControllerCmd(),
		apps.NewAgentCmd(),
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
