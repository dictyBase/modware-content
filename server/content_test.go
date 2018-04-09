package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"

	pb "github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/go-genproto/dictybaseapis/pubsub"
	"google.golang.org/grpc"

	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
	"gopkg.in/src-d/go-git.v4"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/pressly/goose"
)

var db *sql.DB
var schemaRepo string = "https://github.com/dictybase-docker/dictycontent-schema"

const (
	port = ":9596"
)

type fakeRequest struct {
	name string
}

func (f *fakeRequest) UserRequest(s string, r *pubsub.IdRequest, t time.Duration) (*pubsub.UserReply, error) {
	return &pubsub.UserReply{Exist: true}, nil
}

func (f *fakeRequest) UserRequestWithContext(ctx context.Context, s string, r *pubsub.IdRequest) (*pubsub.UserReply, error) {
	return &pubsub.UserReply{Exist: true}, nil
}

type ContentJSON struct {
	Paragraph string `json:"paragraph"`
	Text      string `json:"text"`
}

type PgDocker struct {
	Client   *client.Client
	Image    string
	Pass     string
	User     string
	Database string
	Debug    bool
	ContJSON types.ContainerJSON
}

func NewPgDocker() (*PgDocker, error) {
	pg := &PgDocker{}
	if len(os.Getenv("DOCKER_HOST")) == 0 {
		return pg, errors.New("DOCKER_HOST is not set")
	}
	if len(os.Getenv("DOCKER_API_VERSION")) == 0 {
		return pg, errors.New("DOCKER_API is not set")
	}
	cl, err := client.NewEnvClient()
	if err != nil {
		return pg, err
	}
	pg.Client = cl
	pg.Image = "postgres:9.6.6-alpine"
	pg.Pass = "pgdocker"
	pg.User = "pguser"
	pg.Database = "pgtest"
	return pg, nil
}

func (d *PgDocker) Run() (container.ContainerCreateCreatedBody, error) {
	cli := d.Client
	out, err := cli.ImagePull(context.Background(), d.Image, types.ImagePullOptions{})
	if err != nil {
		return container.ContainerCreateCreatedBody{}, err
	}
	if d.Debug {
		io.Copy(os.Stdout, out)
	}
	resp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: d.Image,
		Env: []string{
			"POSTGRES_PASSWORD=" + d.Pass,
			"POSTGRES_USER=" + d.User,
			"POSTGRES_DB=" + d.Database,
		},
	}, nil, nil, "")
	if err != nil {
		return container.ContainerCreateCreatedBody{}, err
	}
	if err := cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		return container.ContainerCreateCreatedBody{}, err
	}
	cjson, err := cli.ContainerInspect(context.Background(), resp.ID)
	if err != nil {
		return container.ContainerCreateCreatedBody{}, err
	}
	d.ContJSON = cjson
	return resp, nil
}

func (d *PgDocker) RetryDbConnection() (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		d.User, d.Pass, d.GetIP(), d.GetPort(), d.Database,
	)
	dbh, err := sql.Open("pgx", connStr)
	if err != nil {
		return dbh, err
	}
	timeout, err := time.ParseDuration("28s")
	t1 := time.Now()
	for {
		if err := dbh.Ping(); err != nil {
			if time.Now().Sub(t1).Seconds() > timeout.Seconds() {
				return dbh, errors.New("timed out, no connection retrieved")
			}
			continue
		}
		break
	}
	return dbh, nil
}

func (d *PgDocker) GetIP() string {
	return d.ContJSON.NetworkSettings.IPAddress
}

func (d *PgDocker) GetPort() string {
	return "5432"
}

func (d *PgDocker) Purge(resp container.ContainerCreateCreatedBody) error {
	cli := d.Client
	if err := cli.ContainerStop(context.Background(), resp.ID, nil); err != nil {
		return err
	}
	if err := cli.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{}); err != nil {
		return err
	}
	return nil
}

func cloneDbSchemaRepo() (string, error) {
	path, err := ioutil.TempDir("", "content")
	if err != nil {
		return path, err
	}
	_, err = git.PlainClone(path, false, &git.CloneOptions{URL: schemaRepo})
	return path, err
}

func runGRPCServer(db *sql.DB) {
	dbh := runner.NewDB(db, "postgres")
	grpcS := grpc.NewServer()
	pb.RegisterContentServiceServer(grpcS, NewContentService(dbh, &fakeRequest{}))
	lis, err := net.Listen("tcp", port)
	if err != nil {
		panic(err)
	}
	log.Printf("starting grpc server at port %s", port)
	if err := grpcS.Serve(lis); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	pg, err := NewPgDocker()
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	resource, err := pg.Run()
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	db, err = pg.RetryDbConnection()
	if err != nil {
		log.Fatal(err)
	}
	dir, err := cloneDbSchemaRepo()
	defer os.RemoveAll(dir)
	if err != nil {
		log.Fatalf("issue with cloning %s repo %s\n", schemaRepo, err)
	}
	if err := goose.Up(db, dir); err != nil {
		log.Fatalf("issue with running database migration %s\n", err)
	}
	go runGRPCServer(db)
	code := m.Run()
	if err = pg.Purge(resource); err != nil {
		log.Fatalf("unable to remove container %s\n", err)
	}
	os.Exit(code)
}

func NewStoreContent(name string) *pb.StoreContentRequest {
	cdata, _ := json.Marshal(ContentJSON{
		Paragraph: "paragraph",
		Text:      "text",
	})
	return &pb.StoreContentRequest{
		Data: &pb.StoreContentRequest_Data{
			Type: "contents",
			Attributes: &pb.NewContentAttributes{
				Name:      name,
				CreatedBy: rand.Int63n(100) + 1,
				Content:   string(cdata),
				Namespace: "stockcenter",
			},
		},
	}

}

func TestCreate(t *testing.T) {
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewContentServiceClient(conn)
	nct, err := client.StoreContent(context.Background(), NewStoreContent("catalog"))
	if err != nil {
		t.Fatalf("could not store the content %s\n", err)
	}
	if nct.Data.Id < 1 {
		t.Fatalf("No id attribute value %d", nct.Data.Id)
	}
	if nct.Links.Self != nct.Data.Links.Self {
		t.Fatalf("top link %s does not match resource link %s", nct.Links.Self, nct.Data.Links.Self)
	}
	if nct.Data.Attributes.Slug != "stockcenter-catalog" {
		t.Fatalf("expected slug did not match with %s", nct.Data.Attributes.Slug)
	}
}

func TestGetBySlug(t *testing.T) {
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewContentServiceClient(conn)
	nct, err := client.StoreContent(context.Background(), NewStoreContent("payment information"))
	if err != nil {
		t.Fatalf("could not store the content %s\n", err)
	}
	ct, err := client.GetContentBySlug(context.Background(), &pb.ContentRequest{Slug: "stockcenter-payment-information"})
	if err != nil {
		t.Fatalf("could not retrieve content by %d Id", nct.Data.Id)
	}
	if nct.Data.Id != ct.Data.Id {
		t.Errorf("expected id %d did not match %d", ct.Data.Id)
	}
	if ct.Data.Attributes.Slug != "stockcenter-payment-information" {
		t.Fatalf("expected slug did not match %s", ct.Data.Attributes.Slug)
	}
}

func TestGet(t *testing.T) {
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewContentServiceClient(conn)
	nct, err := client.StoreContent(context.Background(), NewStoreContent("order information"))
	if err != nil {
		t.Fatalf("could not store the content %s\n", err)
	}
	ct, err := client.GetContent(context.Background(), &pb.ContentIdRequest{Id: nct.Data.Id})
	if err != nil {
		t.Fatalf("could not retrieve content by %d Id", nct.Data.Id)
	}
	if nct.Data.Id != ct.Data.Id {
		t.Errorf("expected id %d did not match %d", ct.Data.Id)
	}
	if ct.Data.Attributes.Slug != "stockcenter-order-information" {
		t.Fatalf("expected slug did not match %s", ct.Data.Attributes.Slug)
	}
}

func TestUpdate(t *testing.T) {
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewContentServiceClient(conn)
	nct, err := client.StoreContent(context.Background(), NewStoreContent("plasmid catalog"))
	if err != nil {
		t.Fatalf("could not store the content %s\n", err)
	}
	uct := &pb.UpdateContentRequest{
		Id: nct.Data.Id,
		Data: &pb.UpdateContentRequest_Data{
			Type: nct.Data.Type,
			Id:   nct.Data.Id,
			Attributes: &pb.ExistingContentAttributes{
				UpdatedBy: nct.Data.Attributes.UpdatedBy + 5,
				Content:   nct.Data.Attributes.Content,
			},
		},
	}
	ct, err := client.UpdateContent(context.Background(), uct)
	if err != nil {
		t.Fatalf("could not update content %s", err)
	}
	if ct.Data.Attributes.Namespace != nct.Data.Attributes.Namespace {
		t.Fatalf(
			"expected namespace %s did not match %s",
			nct.Data.Attributes.Namespace,
			ct.Data.Attributes.Namespace,
		)
	}
	if ct.Data.Attributes.UpdatedBy != nct.Data.Attributes.UpdatedBy+5 {
		t.Fatalf(
			"expected updated_by %d did not match %d",
			nct.Data.Attributes.UpdatedBy+5,
			ct.Data.Attributes.UpdatedBy,
		)
	}
	cjson := &ContentJSON{}
	err = json.Unmarshal([]byte(ct.Data.Attributes.Content), cjson)
	if err != nil {
		t.Fatalf("unable to decode content json %s\n", err)
	}
	if cjson.Text != "text" {
		t.Fatalf("expected json field text %s did not match %s", "text", cjson.Text)
	}
}

func TesDelete(t *testing.T) {
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewContentServiceClient(conn)
	nct, err := client.StoreContent(context.Background(), NewStoreContent("deposit information"))
	if err != nil {
		t.Fatalf("could not store the content %s\n", err)
	}
	_, err = client.DeleteContent(context.Background(), &pb.ContentIdRequest{Id: nct.Data.Id})
	if err != nil {
		t.Fatalf("could not delete content by %d Id", nct.Data.Id)
	}
}
