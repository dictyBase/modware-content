package service

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/dictyBase/aphgrpc"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/repository/arangodb"
	"github.com/dictyBase/modware-content/internal/testutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type MockMessage struct{}

func (msn *MockMessage) Publish(subject string, cont *content.Content) error {
	return nil
}

func (msn *MockMessage) Close() error {
	return nil
}

func setup(t *testing.T) (content.ContentServiceClient, *require.Assertions) {
	t.Helper()
	assert := require.New(t)
	tra, err := testarango.NewTestArangoFromEnv(true)
	assert.NoError(err, "expect no error from creating an arangodb instance")
	repo, err := arangodb.NewContentRepo(
		&manager.ConnectParams{
			User:     tra.User,
			Pass:     tra.Pass,
			Database: tra.Database,
			Host:     tra.Host,
			Port:     tra.Port,
			Istls:    false,
		}, manager.RandomString(16, 19),
	)
	assert.NoErrorf(
		err,
		"expect no error connecting to annotation repository, received %s",
		err,
	)
	baseServer := grpc.NewServer()
	srv, err := NewContentService(&Params{
		Repository: repo,
		Publisher:  &MockMessage{},
		Group:      "groups",
		Options: []aphgrpc.Option{
			aphgrpc.TopicsOption(map[string]string{
				"contentCreate": "ContentService.Create",
				"contentDelete": "ContentService.Delete",
				"contentUpdate": "ContentService.Update",
			}),
		},
	})
	assert.NoError(err, "expect no error from creating service")
	content.RegisterContentServiceServer(baseServer, srv)
	listener := bufconn.Listen(1024 * 1024)
	go func() {
		if err := baseServer.Serve(listener); err != nil {
			t.Logf("error in listener %s", err)
			os.Exit(1)
		}
	}()
	dialer := func(context.Context, string) (net.Conn, error) {
		conn, err := listener.Dial()
		assert.NoError(err, "expect no error from creating listener")

		return conn, nil
	}
	conn, err := grpc.DialContext(
		context.Background(),
		"",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(err, "expect no error in creating grpc client")
	t.Cleanup(func() {
		_ = repo.Dbh().Drop()
		conn.Close()
		listener.Close()
		baseServer.Stop()
	})

	return content.NewContentServiceClient(conn), assert
}
