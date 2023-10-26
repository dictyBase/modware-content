package message

import (
	"github.com/dictyBase/go-genproto/dictybaseapis/content"
)

// Publisher manages publishing of message.
type Publisher interface {
	// Publis publishes the annotation object using the given subject
	Publish(subject string, cont *content.Content) error
	// Close closes the connection to the underlying messaging server
	Close() error
}
