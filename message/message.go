package message

import (
	"context"
	"time"

	"github.com/dictyBase/go-genproto/dictybaseapis/pubsub"
)

type Request interface {
	UserRequest(string, *pubsub.IdRequest, time.Duration) (*pubsub.UserReply, error)
	UserRequestWithContext(context.Context, string, *pubsub.IdRequest) (*pubsub.UserReply, error)
}
