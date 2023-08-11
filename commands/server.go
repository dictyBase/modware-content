package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/dictyBase/apihelpers/aphgrpc"
	pb "github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/message/nats"
	"github.com/dictyBase/modware-content/server"
	"github.com/go-chi/cors"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	dat "gopkg.in/mgutz/dat.v2/dat"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
)

const ExitError = 2
const Timeout = 10

func startServers(
	grpcS *grpc.Server,
	httpMux *runtime.ServeMux,
	cmlis cmux.CMux,
	lis net.Listener,
	grpcL net.Listener,
	httpL net.Listener,
) error {
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
	httpS := &http.Server{
		Handler:           cors.Handler(httpMux),
		ReadHeaderTimeout: time.Duration(Timeout) * time.Second,
	}
	ech := make(chan error, ExitError)
	go func() { ech <- grpcS.Serve(grpcL) }()
	go func() { ech <- httpS.Serve(httpL) }()
	log.Printf("starting multiplexed server on %s", lis.Addr())
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
		return fmt.Errorf("error in running cmux server")
	}

	return nil
}

func createListener(c *cli.Context) (net.Listener, error) {
	endP := fmt.Sprintf(":%s", c.String("port"))
	lis, err := net.Listen("tcp", endP)
	if err != nil {
		return lis, fmt.Errorf("error creating Listener %s", err)
	}

	return lis, nil
}

func createCMux(lis net.Listener) (cmux.CMux, net.Listener, net.Listener) {
	cmlis := cmux.New(lis)
	grpcL := cmlis.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldSendSettings(
			"content-type",
			"application/grpc",
		),
	)
	httpL := cmlis.Match(cmux.Any())

	return cmlis, grpcL, httpL
}

func registerHTTPHandler(c *cli.Context, httpMux *runtime.ServeMux) error {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	endP := fmt.Sprintf(":%s", c.String("port"))
	err := pb.RegisterContentServiceHandlerFromEndpoint(
		context.Background(),
		httpMux,
		endP,
		opts,
	)

	if err != nil {
		return fmt.Errorf("error in registering http handler %s", err)
	}

	return nil
}

func createHTTPServeMux() *runtime.ServeMux {
	runtime.HTTPError = aphgrpc.CustomHTTPError
	httpMux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(aphgrpc.HandleCreateResponse),
	)

	return httpMux
}

func createGRPCServer(c *cli.Context) *grpc.Server {
	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(getLogger(c)),
		),
	)
}

func RunServer(clt *cli.Context) error {
	dat.EnableInterpolation = true
	dbh, err := getPgWrapper(clt)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("Unable to create database connection %s", err.Error()),
			ExitError,
		)
	}
	nrs, err := nats.NewRequest(
		clt.String("nats-host"),
		clt.String("nats-port"),
	)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("cannot connect to messaging server %s", err.Error()),
			ExitError,
		)
	}
	grpcS := createGRPCServer(clt)
	pb.RegisterContentServiceServer(
		grpcS,
		server.NewContentService(
			dbh,
			nrs,
			aphgrpc.BaseURLOption(setAPIHost(clt)),
		))
	reflection.Register(grpcS)
	httpMux := createHTTPServeMux()
	if err := registerHTTPHandler(clt, httpMux); err != nil {
		return cli.NewExitError(
			fmt.Sprintf(
				"unable to register http endpoint for content microservice %s",
				err,
			),
			ExitError,
		)
	}

	lis, err := createListener(clt)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to listen %s", err),
			ExitError,
		)
	}
	cmlis, grpcL, httpL := createCMux(lis)
	err = startServers(grpcS, httpMux, cmlis, lis, grpcL, httpL)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("error in starting servers %s", err),
			ExitError,
		)
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

	dbh, err := sql.Open("pgx", cStr)
	if err != nil {
		return &sql.DB{}, fmt.Errorf("error in opening database %s", err)
	}

	return dbh, nil
}

func getPgWrapper(c *cli.Context) (*runner.DB, error) {
	var dbh *runner.DB
	h, err := getPgxDbHandler(c)
	if err != nil {
		return dbh, err
	}

	return runner.NewDB(h, "postgres"), nil
}

func getLogger(cltx *cli.Context) *logrus.Entry {
	log := logrus.New()
	log.Out = os.Stderr
	switch cltx.GlobalString("log-format") {
	case "text":
		log.Formatter = &logrus.TextFormatter{
			TimestampFormat: "0ExitError/Jan/2006:15:04:05",
		}
	case "json":
		log.Formatter = &logrus.JSONFormatter{
			TimestampFormat: "0ExitError/Jan/2006:15:04:05",
		}
	}
	l := cltx.GlobalString("log-level")
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
