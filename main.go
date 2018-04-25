package main

import (
	"os"

	"github.com/dictyBase/modware-content/commands"
	"github.com/dictyBase/modware-content/validate"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "modware-content"
	app.Usage = "cli for modware-content microservice"
	app.Version = "2.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "log-format",
			Usage: "format of the logging out, either of json or text.",
			Value: "json",
		},
		cli.StringFlag{
			Name:  "log-level",
			Usage: "log level for the application",
			Value: "error",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:   "start-server",
			Usage:  "starts the modware-content microservice with HTTP and grpc backends",
			Action: commands.RunServer,
			Before: validate.ValidateServerArgs,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "dictycontent-pass, pass",
					EnvVar: "DICTYCONTENT_PASS",
					Usage:  "dictycontent database password",
				},
				cli.StringFlag{
					Name:   "dictycontent-db, db",
					EnvVar: "DICTYCONTENT_DB",
					Usage:  "dictycontent database name",
				},
				cli.StringFlag{
					Name:   "dictycontent-user, u, user",
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
					Name:   "content-api-http-host",
					EnvVar: "CONTENT_API_HTTP_HOST",
					Usage:  "public hostname serving the http api, by default the default port will be appended to http://localhost",
				},
				cli.StringFlag{
					Name:   "nats-host",
					EnvVar: "NATS_SERVICE_HOST",
					Usage:  "nats messaging server host",
				},
				cli.StringFlag{
					Name:   "nats-port",
					EnvVar: "NATS_SERVICE_PORT",
					Usage:  "nats messaging server port",
				},
				cli.StringFlag{
					Name:  "port",
					Usage: "tcp port at which the servers will be available",
					Value: "9555",
				},
			},
		},
	}
	app.Run(os.Args)
}
