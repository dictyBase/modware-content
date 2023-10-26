package main

import (
	"log"
	"os"

	apiflag "github.com/dictyBase/aphgrpc"
	arangoflag "github.com/dictyBase/arangomanager/command/flag"
	"github.com/dictyBase/modware-content/internal/app/server"
	"github.com/dictyBase/modware-content/internal/app/validate"
	"github.com/dictyBase/modware-content/validate"
	"github.com/urfave/cli"
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
			Action: server.RunServer,
			Before: validate.ServerArgs,
			Flags:  getServerFlags(),
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalf("error in running command %s", err)
	}
}

func getServerFlags() []cli.Flag {
	flg := []cli.Flag{
		cli.StringFlag{
			Name:  "port",
			Usage: "tcp port at which the server will be available",
			Value: "9560",
		},
		cli.StringFlag{
			Name:  "content-collection",
			Usage: "arangodb collection for storing editor data",
			Value: "serialized_json",
		},
		cli.StringFlag{
			Name:   "arangodb-database, db",
			EnvVar: "ARANGODB_DATABASE",
			Usage:  "arangodb database name",
			Value:  "content",
		},
	}
	flg = append(flg, arangoflag.ArangoFlags()...)

	return append(flg, apiflag.NatsFlag()...)
}
