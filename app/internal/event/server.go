package event

import (
	"context"
	"time"

	"github.com/tianping526/eventbridge/app/internal/rule"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

var _ transport.Server = (*Server)(nil)

// ServerOption is an event server option.
type ServerOption func(*Server)

// WithTimeout with server timeout, default 1 s
func WithTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

// WithEndpoint with server endpoint.
func WithEndpoint(endpoint string) ServerOption {
	return func(s *Server) {
		s.endpoint = endpoint
	}
}

// WithOperation with server operation.
func WithOperation(operation string) ServerOption {
	return func(s *Server) {
		s.operation = operation
	}
}

// WithMiddleware with a service middleware option.
func WithMiddleware(m ...middleware.Middleware) ServerOption {
	return func(o *Server) {
		o.middleware = append(o.middleware, m...)
	}
}

// Server is an event server wrapper.
type Server struct {
	endpoint   string
	operation  string
	receiver   Receiver
	handler    Handler
	timeout    time.Duration
	middleware []middleware.Middleware
}

// NewServer creates an event server by options.
func NewServer(receiver Receiver, handler Handler, opts ...ServerOption) *Server {
	srv := &Server{
		receiver:   receiver,
		timeout:    1 * time.Second,
		middleware: make([]middleware.Middleware, 0),
	}
	for _, o := range opts {
		o(srv)
	}

	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		evt := req.(*rule.EventExt)
		return handler(ctx, evt)
	}
	if len(srv.middleware) > 0 {
		h = middleware.Chain(srv.middleware...)(h)
	}

	srv.handler = func(ctx context.Context, e *rule.EventExt) (interface{}, error) {
		md, _ := metadata.FromServerContext(ctx)
		replyHeader := headerCarrier{}
		tr := &Transport{
			endpoint:    srv.endpoint,
			operation:   srv.operation,
			reqHeader:   headerCarrier(md),
			replyHeader: replyHeader,
		}
		ctx = transport.NewServerContext(ctx, tr)
		if srv.timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, srv.timeout)
			defer cancel()
		}
		body, err := h(ctx, e)
		if err != nil {
			return nil, err
		}
		return &reply{header: replyHeader, body: body}, nil
	}

	return srv
}

// Start the event server.
func (s *Server) Start(ctx context.Context) error {
	log.Infof("[event] server receiving from: %s", s.endpoint)
	return s.receiver.Receive(ctx, s.handler)
}

// Stop the event server.
func (s *Server) Stop(ctx context.Context) error {
	log.Info("[event] server stopping")
	return s.receiver.Close(ctx)
}
