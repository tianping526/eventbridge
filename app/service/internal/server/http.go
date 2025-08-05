package server

import (
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
	"github.com/go-kratos/kratos/v2/transport/http"
	jwtV5 "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
	"github.com/tianping526/eventbridge/app/service/internal/conf"
	"github.com/tianping526/eventbridge/app/service/internal/data"
	"github.com/tianping526/eventbridge/app/service/internal/service"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	bc *conf.Bootstrap,
	logger log.Logger,
	m *data.Metric,
	s *service.EventBridgeService,
	_ trace.TracerProvider, // otel.SetTracerProvider(provider) instead, but need to declare to wire injection
) *http.Server {
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
	opts := []http.ServerOption{
		http.Middleware(
			middlewares...,
		),
		http.Filter(handlers.CORS(
			handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}),
			handlers.AllowedOrigins([]string{"*"}),
		)),
	}
	if cs.Http.Network != "" {
		opts = append(opts, http.Network(cs.Http.Network))
	}
	if cs.Http.Addr != "" {
		opts = append(opts, http.Address(cs.Http.Addr))
	}
	if cs.Http.Timeout != nil {
		opts = append(opts, http.Timeout(cs.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	srv.Handle("/metrics", promhttp.Handler())
	v1.RegisterEventBridgeServiceHTTPServer(srv, s)
	return srv
}
