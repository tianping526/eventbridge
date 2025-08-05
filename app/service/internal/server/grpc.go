package server

import (
	"context"

	validateV2 "github.com/go-kratos/kratos/contrib/middleware/validate/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	jwtV5 "github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel/trace"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/service/internal/conf"
	"github.com/tianping526/eventbridge/app/service/internal/data"
	"github.com/tianping526/eventbridge/app/service/internal/service"
)

func NewWhiteListMatcher() selector.MatchFunc {
	whiteList := make(map[string]struct{})
	whiteList["/eventbridge.service.v1.EventBridgeService/ListDispatcherSchema"] = struct{}{}
	whiteList["/grpc.health.v1.Health/Check"] = struct{}{}
	return func(_ context.Context, operation string) bool {
		if _, ok := whiteList[operation]; ok {
			return false
		}
		return true
	}
}

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	bc *conf.Bootstrap,
	logger log.Logger,
	m *data.Metric,
	s *service.EventBridgeService,
	_ trace.TracerProvider, // otel.SetTracerProvider(provider) instead, but need to declare to wire injection
) *grpc.Server {
	cs := bc.Server
	ca := bc.Auth
	middlewares := []middleware.Middleware{
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
	}
	if ca != nil && ca.Key != "" {
		selector.Server()
		middlewares = append(middlewares, selector.Server(
			jwt.Server(func(_ *jwtV5.Token) (interface{}, error) {
				return []byte(ca.Key), nil
			}, jwt.WithSigningMethod(jwtV5.SigningMethodHS256), jwt.WithClaims(func() jwtV5.Claims {
				return &jwtV5.MapClaims{}
			})),
		).
			Match(NewWhiteListMatcher()).
			Build())
	}
	opts := []grpc.ServerOption{
		grpc.Middleware(
			middlewares...,
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
