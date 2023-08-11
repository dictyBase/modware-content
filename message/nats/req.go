package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/dictyBase/go-genproto/dictybaseapis/pubsub"
	"github.com/dictyBase/modware-identity/message"
	gnats "github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/encoders/protobuf"
)

type natsRequest struct {
	econn *gnats.EncodedConn
}

func NewRequest(
	host, port string,
	options ...gnats.Option,
) (message.Request, error) {
	gnc, err := gnats.Connect(
		fmt.Sprintf("nats://%s:%s", host, port),
		options...)
	if err != nil {
		return &natsRequest{}, fmt.Errorf("error in connecting nats %s", err)
	}
	ec, err := gnats.NewEncodedConn(gnc, protobuf.PROTOBUF_ENCODER)
	if err != nil {
		return &natsRequest{}, fmt.Errorf("error in encoding for nats %s", err)
	}

	return &natsRequest{econn: ec}, nil
}

func (n *natsRequest) UserRequest(
	subj string,
	r *pubsub.IdRequest,
	timeout time.Duration,
) (*pubsub.UserReply, error) {
	reply := &pubsub.UserReply{}
	err := n.econn.Request(subj, r, reply, timeout)
	if err != nil {
		return reply, fmt.Errorf("error in getting reply %s", err)
	}

	return reply, nil
}

func (n *natsRequest) UserRequestWithContext(
	ctx context.Context,
	subj string,
	r *pubsub.IdRequest,
) (*pubsub.UserReply, error) {
	reply := &pubsub.UserReply{}
	err := n.econn.RequestWithContext(ctx, subj, r, reply)
	if err != nil {
		return reply, fmt.Errorf("error in getting reply %s", err)
	}

	return reply, nil
}
