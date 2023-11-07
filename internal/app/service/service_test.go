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

func TestStoreContent(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	nct, err := client.StoreContent(
		context.Background(),
		&content.StoreContentRequest{
			Data: &content.StoreContentRequest_Data{
				Attributes: testutils.NewStoreContent("catalog", "dsc"),
			},
		},
	)
	assert.NoError(err, "expect no error from storing content")
	assert.Equal(nct.Data.Attributes.Name, "catalog", "name should match")
	assert.Equal(nct.Data.Attributes.Namespace, "dsc", "namespace should match")
	assert.Equal(nct.Data.Attributes.Slug, "catalog-dsc", "slug should match")
	assert.Equal(
		nct.Data.Attributes.CreatedBy,
		"content@content.org",
		"should match created_by",
	)
	assert.Equal(
		nct.Data.Attributes.CreatedAt,
		nct.Data.Attributes.UpdatedAt,
		"created_at should match updated_at",
	)
	assert.True(
		nct.Data.Attributes.CreatedAt.AsTime().Before(time.Now()),
		"should have created before the current time",
	)
	ctnt, err := testutils.ContentFromStore(nct.Data.Attributes.Content)
	assert.NoError(err, "should not have any error with json unmarshaling")
	assert.Equal(
		ctnt,
		&testutils.ContentJSON{Paragraph: "paragraph", Text: "text"},
		"should match the content",
	)
}
func TestGetContentBySlug(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	nct, err := client.StoreContent(
		context.Background(),
		&content.StoreContentRequest{
			Data: &content.StoreContentRequest_Data{
				Attributes: testutils.NewStoreContent("catalog", "dsc"),
			},
		},
	)
	assert.NoError(err, "expect no error from storing content")
	sct, err := client.GetContentBySlug(
		context.Background(),
		&content.ContentRequest{Slug: nct.Data.Attributes.Slug},
	)
	assert.NoError(err, "expect no error from fetching content by slug")
	testContentProperties(assert, sct, nct)
}
func testContentProperties(
	assert *require.Assertions,
	sct, nct *content.Content,
) {
	assert.Equal(
		sct.Data.Attributes.Name,
		nct.Data.Attributes.Name,
		"name should match",
	)
	assert.Equal(
		sct.Data.Attributes.Namespace,
		nct.Data.Attributes.Namespace,
		"namespace should match",
	)
	assert.Equal(
		sct.Data.Attributes.Slug,
		nct.Data.Attributes.Slug,
		"slug should match",
	)
	assert.Equal(
		sct.Data.Attributes.CreatedBy,
		nct.Data.Attributes.CreatedBy,
		"should match created_by",
	)
	assert.Equal(
		sct.Data.Attributes.CreatedAt,
		nct.Data.Attributes.CreatedAt,
		"created_at should match",
	)
	assert.Equal(
		sct.Data.Attributes.UpdatedAt,
		nct.Data.Attributes.UpdatedAt,
		"updated_at should match",
	)
	assert.Equal(
		sct.Data.Attributes.Content,
		nct.Data.Attributes.Content,
		"should match raw conent",
	)
}
