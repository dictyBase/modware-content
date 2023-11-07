package server

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/dictyBase/apihelpers/aphgrpc"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/app/service"
	"github.com/dictyBase/modware-content/internal/message"
	"github.com/dictyBase/modware-content/internal/message/nats"
	"github.com/dictyBase/modware-content/internal/repository"
	"github.com/dictyBase/modware-content/internal/repository/arangodb"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const ExitError = 2
const Timeout = 10

type serverParams struct {
	repo repository.ContentRepository
	msg  message.Publisher
}

func RunServer(clt *cli.Context) error {
	spn, err := repoAndNatsConn(clt)
	if err != nil {
		return cli.NewExitError(err.Error(), ExitError)
	}
	grpcS := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(getLogger(clt)),
		),
	)
	srv, err := service.NewContentService(
		&service.Params{
			Repository: spn.repo,
			Publisher:  spn.msg,
			Group:      "groups",
			Options:    getGrpcOpt(),
		})
	if err != nil {
		return cli.NewExitError(err.Error(), ExitError)
	}
	content.RegisterContentServiceServer(grpcS, srv)
	reflection.Register(grpcS)
	// create listener
	endP := fmt.Sprintf(":%s", clt.String("port"))
	lis, err := net.Listen("tcp", endP)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to listen %s", err), ExitError,
		)
	}
	log.Printf("starting grpc server on %s", endP)
	if err := grpcS.Serve(lis); err != nil {
		return cli.NewExitError(err.Error(), ExitError)
	}

	return nil
}

func getLogger(cltx *cli.Context) *logrus.Entry {
	log := logrus.New()
	log.Out = os.Stderr
	switch cltx.GlobalString("log-format") {
	case "text":
		log.Formatter = &logrus.TextFormatter{
			TimestampFormat: "02/Jan/2006:15:04:05",
		}
	case "json":
		log.Formatter = &logrus.JSONFormatter{
			TimestampFormat: "02/Jan/2006:15:04:05",
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

func allParams(
	clt *cli.Context,
) *manager.ConnectParams {
	arPort, _ := strconv.Atoi(clt.String("arangodb-port"))

	return &manager.ConnectParams{
			User:     clt.String("arangodb-user"),
			Pass:     clt.String("arangodb-pass"),
			Database: clt.String("arangodb-database"),
			Host:     clt.String("arangodb-host"),
			Port:     arPort,
			Istls:    clt.Bool("is-secure"),
		}		}
}

func getGrpcOpt() []aphgrpc.Option {
	return []aphgrpc.Option{
		aphgrpc.TopicsOption(map[string]string{
			"contentCreate": "ContentService.Create",
			"contentDelete": "ContentService.Delete",
			"contentUpdate": "ContentService.Update",
		}),
	}
}

func repoAndNatsConn(clt *cli.Context) (*serverParams, error) {
	anrepo, err := arangodb.NewContentRepo(allParams(clt),clt.String("content-collection"))
	if err != nil {
		return &serverParams{},
			fmt.Errorf(
				"cannot connect to arangodb annotation repository %s",
				err,
			)
	}
	msp, err := nats.NewPublisher(
		clt.String("nats-host"), clt.String("nats-port"),
		gnats.MaxReconnects(-1), gnats.ReconnectWait(waitTime*time.Second),
	)
	if err != nil {
		return &serverParams{},
			fmt.Errorf("cannot connect to messaging server %s", err)
	}

	return &serverParams{
		repo: anrepo,
		msg:  msp,
	}, nil
}
