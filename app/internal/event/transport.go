package event

import (
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/transport"
)

var (
	_ transport.Transporter = (*Transport)(nil)
	_ ReplyCarrier          = (*reply)(nil)
)

type ReplyCarrier interface {
	GetHeader() transport.Header
	GetBody() interface{}
}

// Transport is an event transport.
type Transport struct {
	endpoint    string
	operation   string
	reqHeader   headerCarrier
	replyHeader headerCarrier
}

func (tr *Transport) Endpoint() string {
	return tr.endpoint
}

func (tr *Transport) Operation() string {
	return tr.operation
}

func (tr *Transport) RequestHeader() transport.Header {
	return tr.reqHeader
}

func (tr *Transport) ReplyHeader() transport.Header {
	return tr.replyHeader
}

// Kind returns the transport kind.
func (tr *Transport) Kind() transport.Kind {
	return "event"
}

type headerCarrier metadata.Metadata

// Get returns the value associated with the passed key.
func (hc headerCarrier) Get(key string) string {
	return metadata.Metadata(hc).Get(key)
}

// Set stores the key-value pair.
func (hc headerCarrier) Set(key string, value string) {
	metadata.Metadata(hc).Set(key, value)
}

// Add adds the key, value pair to the header.

func (hc headerCarrier) Add(key string, value string) {
	metadata.Metadata(hc).Add(key, value)
}

func (hc headerCarrier) Values(key string) []string {
	return metadata.Metadata(hc).Values(key)
}

// Keys lists the keys stored in this carrier.
func (hc headerCarrier) Keys() []string {
	keys := make([]string, 0, len(hc))
	for k := range hc {
		keys = append(keys, k)
	}
	return keys
}

type reply struct {
	body   interface{}
	header headerCarrier
}

func (r *reply) GetHeader() transport.Header {
	return r.header
}

func (r *reply) GetBody() interface{} {
	return r.body
}
