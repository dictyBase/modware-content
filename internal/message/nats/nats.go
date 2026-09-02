package nats

import (
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/message"
	gnats "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type natsPublisher struct {
	conn *gnats.Conn
}

func NewPublisher(
	host, port string,
	options ...gnats.Option,
) (message.Publisher, error) {
	ncr, err := gnats.Connect(
		fmt.Sprintf("nats://%s:%s", host, port),
		options...)
	if err != nil {
		return &natsPublisher{}, fmt.Errorf(
			"error in connecting to nats server %s",
			err,
		)
	}

	return &natsPublisher{conn: ncr}, nil
}

func (n *natsPublisher) Publish(
	subj string,
	cont *content.Content,
) error {
	data, err := proto.Marshal(cont)
	if err != nil {
		return fmt.Errorf("error in marshaling content %s", err)
	}
	if err := n.conn.Publish(subj, data); err != nil {
		return fmt.Errorf("error in publishing through nats %s", err)
	}

	return nil
}

func (n *natsPublisher) Close() error {
	n.conn.Close()

	return nil
}
