package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"

	pb "github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/go-genproto/dictybaseapis/pubsub"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/pressly/goose"
	"google.golang.org/grpc"

	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
	"gopkg.in/src-d/go-git.v4"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/pressly/goose"
)

var db *sql.DB
var schemaRepo string = "https://github.com/dictybase-docker/dictycontent-schema"
var pgAddr = fmt.Sprintf("%s:%s", os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"))
var pgConn = fmt.Sprintf(
	"postgres://%s:%s@%s/%s?sslmode=disable",
	os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), pgAddr, os.Getenv("POSTGRES_DB"))

const (
	port   = ":9596"
	schema = "content"
)

type fakeRequest struct {
	name string
}
type fakeRequest struct{}

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

func CheckPostgresEnv() error {
	envs := []string{
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		"POSTGRES_HOST",
	}
	for _, e := range envs {
		if len(os.Getenv(e)) == 0 {
			return fmt.Errorf("env %s is not set", e)
		}
	}

	return nil
}

type TestPostgres struct {
	DB *sql.DB
}

func NewTestPostgresFromEnv() (*TestPostgres, error) {
	pg := new(TestPostgres)
	pgt := new(TestPostgres)
	if err := CheckPostgresEnv(); err != nil {
		return pg, err
		return pgt, err
	}
	dbh, err := sql.Open("pgx", pgConn)
	if err != nil {
		return pgt, fmt.Errorf("error in opening db connection %s", err)
	}
	timeout, err := time.ParseDuration("28s")
	if err != nil {
		return pgt, fmt.Errorf("error in parsing time %s", err)
	}
	t1 := time.Now()
	for {
		if err := dbh.Ping(); err != nil {
			if time.Since(t1).Seconds() > timeout.Seconds() {
				return pgt, errors.New("timed out, no connection retrieved")
			}

			continue
		}

		break
	}
	pgt.DB = dbh

	return pgt, nil
}

func cloneDbSchemaRepo() (string, error) {
	path, err := ioutil.TempDir("", "content")
	if err != nil {
		return path, fmt.Errorf("error in creating temp dir %s", err)
	}
	_, err = git.PlainClone(path, false, &git.CloneOptions{URL: schemaRepo})
	if err != nil {
		return path, fmt.Errorf("error in cloing to path %s", err)
	}

	return path, nil
}

func runGRPCServer(db *sql.DB) {
	dbh := runner.NewDB(db, "postgres")
	grpcS := grpc.NewServer()
	pb.RegisterContentServiceServer(grpcS, NewContentService(dbh, &fakeRequest{}))
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("error listening to grpc port %s", err)
	}
	log.Printf("starting grpc server at port %s", port)
	if err := grpcS.Serve(lis); err != nil {
		log.Fatalf("error serving %s", err)
	}
}

func TestMain(m *testing.M) {
	pg, err := NewTestPostgresFromEnv()
	pgt, err := NewTestPostgresFromEnv()
	if err != nil {
		log.Fatalf(
			"unable to construct new NewTestPostgresFromEnv instance %s",
			err,
		)
	}
	dbh = pgt.DB
	// create schema for this application
	_, err = dbh.Exec(fmt.Sprintf("CREATE SCHEMA %s", schema))
	if err != nil {
		log.Fatal(err)
	}
	_, err = dbh.Exec(fmt.Sprintf("SET search_path TO %s", schema))
	if err != nil {
		log.Fatal(err)
	}
	dir, err := cloneDbSchemaRepo()
	defer os.RemoveAll(dir)
	if err != nil {
		log.Fatalf("issue with cloning %s repo %s\n", schemaRepo, err)
	}
	if err := goose.Up(dbh, dir); err != nil {
		log.Fatalf("issue with running database migration %s\n", err)
	}
	go runGRPCServer(dbh)
	os.Exit(m.Run())
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "localhost"+port, grpc.WithInsecure(), grpc.WithBlock())
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
		t.Errorf("expected id %d did not match %d", ct.Data.Id, nct.Data.Id)
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
		t.Errorf("expected id %d did not match %d", ct.Data.Id, nct.Data.Id)
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
