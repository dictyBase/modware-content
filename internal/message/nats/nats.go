package nats

import (
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/content"
	"github.com/dictyBase/modware-content/internal/message"
	gnats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/encoders/protobuf"
)

type natsPublisher struct {
	econn *gnats.EncodedConn
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
	ec, err := gnats.NewEncodedConn(ncr, protobuf.PROTOBUF_ENCODER)
	if err != nil {
		return &natsPublisher{}, fmt.Errorf("error in encoding %s", err)
	}

	return &natsPublisher{econn: ec}, nil
}

func (n *natsPublisher) Publish(
	subj string,
	cont *content.Content,
) error {
	if err := n.econn.Publish(subj, cont); err != nil {
		return fmt.Errorf("error in publishing through nats %s", err)
	}

	return nil
}

func (n *natsPublisher) Close() error {
	n.econn.Close()

	return nil
}
