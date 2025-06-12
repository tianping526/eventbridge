package server

import (
	validateV2 "github.com/go-kratos/kratos/contrib/middleware/validate/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"go.opentelemetry.io/otel/trace"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/service/internal/conf"
	"github.com/tianping526/eventbridge/app/service/internal/data"
	"github.com/tianping526/eventbridge/app/service/internal/service"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	bc *conf.Bootstrap,
	logger log.Logger,
	m *data.Metric,
	s *service.EventBridgeService,
	_ trace.TracerProvider, // otel.SetTracerProvider(provider) instead, but need to declare to wire injection
) *grpc.Server {
	cs := bc.Server
	// ca := bc.Auth
	opts := []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			metrics.Server(
				metrics.WithSeconds(m.DurationSec),
				metrics.WithRequests(m.CodeTotal),
			),
			recovery.Recovery(),
			logging.Server(logger),
			ratelimit.Server(),
			validateV2.ProtoValidate(),
			// jwt.Server(func(token *jwtV4.Token) (interface{}, error) {
			//	return []byte(ca.Key), nil
			// }),
		),
	}
	if cs.Grpc.Network != "" {
		opts = append(opts, grpc.Network(cs.Grpc.Network))
	}
	if cs.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(cs.Grpc.Addr))
	}
	if cs.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(cs.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterEventBridgeServiceServer(srv, s)
	return srv
}
