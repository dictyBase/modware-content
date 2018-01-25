package main

import (
	"fmt"
	"os"

	"github.com/dictyBase/modware-content/commands"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "modware-content"
	app.Usage = "starts the modware-content microservice with HTTP and grpc backends"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "dictycontent-pass",
			EnvVar: "DICTYCONTENT_PASS",
			Usage:  "dictycontent database password",
		},
		cli.StringFlag{
			Name:   "dictycontent-db",
			EnvVar: "DICTYCONTENT_DB",
			Usage:  "dictycontent database name",
		},
		cli.StringFlag{
			Name:   "dictycontent-user",
			EnvVar: "DICTYCONTENT_USER",
			Usage:  "dictycontent database user",
		},
		cli.StringFlag{
			Name:   "dictycontent-host",
			Value:  "dictycontent-backend",
			EnvVar: "DICTYCONTENT_BACKEND_SERVICE_HOST",
			Usage:  "dictycontent database host",
		},
		cli.StringFlag{
			Name:   "dictycontent-port",
			EnvVar: "DICTYCONTENT_BACKEND_SERVICE_PORT",
			Usage:  "dictycontent database port",
		},
		cli.StringFlag{
			Name:  "port",
			Usage: "tcp port at which the servers will be available",
			Value: "9597",
		},
	}
	app.Before = validateArgs
	app.Action = commands.RunServer
	app.Run(os.Args)
}

func validateArgs(c *cli.Context) error {
	for _, p := range []string{"dictycontent-pass", "dictycontent-db", "dictycontent-user"} {
		if len(c.String(p)) == 0 {
			return cli.NewExitError(
				fmt.Sprintf("argument %s is missing", p),
				2,
			)
		}
	}
	return nil
}
