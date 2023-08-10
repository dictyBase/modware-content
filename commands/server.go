package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/dictyBase/apihelpers/aphgrpc"
	pb "github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/message/nats"
	"github.com/dictyBase/modware-content/server"
	"github.com/go-chi/cors"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	dat "gopkg.in/mgutz/dat.v2/dat"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
)

const ExitError = 2

func RunServer(c *cli.Context) error {
	dat.EnableInterpolation = true
	dbh, err := getPgWrapper(c)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("Unable to create database connection %s", err.Error()),
			ExitError,
		)
	}
	nrs, err := nats.NewRequest(
		c.String("nats-host"),
		c.String("nats-port"),
	)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("cannot connect to messaging server %s", err.Error()),
			ExitError,
		)
	}
	grpcS := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(getLogger(c)),
		),
	)
	pb.RegisterContentServiceServer(
		grpcS,
		server.NewContentService(
			dbh,
			nrs,
			aphgrpc.BaseURLOption(setAPIHost(c)),
		))
	reflection.Register(grpcS)

	// http requests muxer
	runtime.HTTPError = aphgrpc.CustomHTTPError
	httpMux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(aphgrpc.HandleCreateResponse),
	)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	endP := fmt.Sprintf(":%s", c.String("port"))
	err = pb.RegisterContentServiceHandlerFromEndpoint(
		context.Background(),
		httpMux,
		endP,
		opts,
	)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf(
				"unable to register http endpoint for content microservice %s",
				err,
			),
			ExitError,
		)
	}

	// create listener
	lis, err := net.Listen("tcp", endP)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to listen %s", err),
			ExitError,
		)
	}
	// create the cmux object that will multiplex ExitError protocols on same port
	cmlis := cmux.New(lis)
	// match gRPC requests, otherwise regular HTTP requests
	// see https://github.com/grpc/grpc-go/issues/ExitError636#issuecomment-472209287 for why we need to use MatchWithWriters()
	grpcL := cmlis.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldSendSettings(
			"content-type",
			"application/grpc",
		),
	)
	httpL := cmlis.Match(cmux.Any())

	// CORS setup
	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"OPTIONS",
			"PATCH",
		},
		OptionsPassthrough: false,
		AllowedHeaders:     []string{"*"},
	})
	httpS := &http.Server{Handler: cors.Handler(httpMux)}
	// collect on this channel the exits of each protocol's .Serve() call
	ech := make(chan error, ExitError)
	// start the listeners for each protocol
	go func() { ech <- grpcS.Serve(grpcL) }()
	go func() { ech <- httpS.Serve(httpL) }()
	log.Printf("starting multiplexed  server on %s", endP)
	var failed bool
	if err := cmlis.Serve(); err != nil {
		log.Printf("cmux server error: %v", err)
		failed = true
	}
	icount := 0
	for err := range ech {
		if err != nil {
			log.Printf("protocol serve error:%v", err)
			failed = true
		}
		icount++
		if cap(ech) == icount {
			close(ech)

			break
		}
	}
	if failed {
		return cli.NewExitError("error in running cmux server", ExitError)
	}

	return nil
}

func setAPIHost(c *cli.Context) string {
	if len(c.String("content-api-http-host")) > 0 {
		return c.String("content-api-http-host")
	}

	return fmt.Sprintf("http://localhost:%s", c.String("port"))
}

func getPgxDbHandler(cltx *cli.Context) (*sql.DB, error) {
	cStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cltx.String("dictycontent-user"),
		cltx.String("dictycontent-pass"),
		cltx.String("dictycontent-host"),
		cltx.String("dictycontent-port"),
		cltx.String("dictycontent-db"),
	)
	return sql.Open("pgx", cStr)
}

func getPgWrapper(c *cli.Context) (*runner.DB, error) {
	var dbh *runner.DB
	h, err := getPgxDbHandler(c)
	if err != nil {
		return dbh, err
	}
	return runner.NewDB(h, "postgres"), nil
}

func getLogger(c *cli.Context) *logrus.Entry {
	log := logrus.New()
	log.Out = os.Stderr
	switch c.GlobalString("log-format") {
	case "text":
		log.Formatter = &logrus.TextFormatter{
			TimestampFormat: "0ExitError/Jan/2006:15:04:05",
		}
	case "json":
		log.Formatter = &logrus.JSONFormatter{
			TimestampFormat: "0ExitError/Jan/2006:15:04:05",
		}
	}
	l := c.GlobalString("log-level")
	switch l {
	case "debug":
		log.Level = logrus.DebugLevel
	case "warn":
		log.Level = logrus.WarnLevel
	case "error":
		log.Level = logrus.ErrorLevel
	case "fatal":
		log.Level = logrus.FatalLevel
	case "panic":
		log.Level = logrus.PanicLevel
	}
	return logrus.NewEntry(log)
}
